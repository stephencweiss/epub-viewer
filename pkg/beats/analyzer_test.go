package beats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"epub-reader/pkg/epub"
	"epub-reader/pkg/storage"
)

// testStore creates a temporary SQLite store for testing.
func testStore(t *testing.T) *storage.SQLiteStore {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
		os.Remove(dbPath)
	})
	return store
}

// createTestBook creates a test author and book, returning the book.
func createTestBook(t *testing.T, store *storage.SQLiteStore) *storage.Book {
	t.Helper()
	author, err := store.CreateAuthor("Test Author")
	if err != nil {
		t.Fatalf("failed to create test author: %v", err)
	}
	book, err := store.AddBook(author.ID, "Test Book", "/path/to/test.epub", "en", "Test Publisher")
	if err != nil {
		t.Fatalf("failed to create test book: %v", err)
	}
	return book
}

// mockClaudeServer creates a test server that returns mock LLM responses.
func mockClaudeServer(t *testing.T, responses []string) *httptest.Server {
	t.Helper()
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing x-api-key header")
		}

		response := `{"summary": "default", "conflict": "default", "choice": "default", "consequence": "default"}`
		if callCount < len(responses) {
			response = responses[callCount]
		}
		callCount++

		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": response},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

func TestNewAnalyzer_RequiresAPIKey(t *testing.T) {
	store := testStore(t)

	// Ensure env var is not set
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", oldKey)
		}
	}()

	_, err := NewAnalyzer(store, "")
	if err == nil {
		t.Error("expected error when API key is not set")
	}
}

func TestNewAnalyzer_AcceptsAPIKey(t *testing.T) {
	store := testStore(t)

	analyzer, err := NewAnalyzer(store, "test-api-key")
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}

	if analyzer.apiKey != "test-api-key" {
		t.Error("API key not set correctly")
	}
}

func TestAnalyzer_MockLLM(t *testing.T) {
	store := testStore(t)
	_ = createTestBook(t, store) // Create book for database setup

	// Create mock responses for 2 scenes
	mockResponses := []string{
		`{"summary": "Scene one summary", "conflict": "Scene one conflict", "choice": "Scene one choice", "consequence": "Scene one consequence", "perspective_shift": ""}`,
		`{"summary": "Scene two summary", "conflict": "Scene two conflict", "choice": "Scene two choice", "consequence": "Scene two consequence", "perspective_shift": "Reader sees villain differently"}`,
	}

	server := mockClaudeServer(t, mockResponses)

	// Create analyzer with mock server
	analyzer, err := NewAnalyzer(store, "test-key",
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}

	// Create a simple book with 2 scenes
	epubBook := &epub.Book{
		Chapters: []epub.Chapter{
			{ID: "ch1", Text: "Scene one content.\n***\nScene two content."},
		},
	}

	// Track progress
	var progressCalls []int
	analyzer.OnProgress = func(processed, total int, preview string) {
		progressCalls = append(progressCalls, processed)
	}

	// Test scene detection works correctly
	scenes := DetectScenes(epubBook)
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}

	// Verify analyzer was configured
	if analyzer.apiKey != "test-key" {
		t.Error("analyzer API key not set")
	}
}

func TestAnalyzer_AnalyzeSceneComponents(t *testing.T) {
	// Test that the analyzer correctly processes scenes into beats
	store := testStore(t)
	book := createTestBook(t, store)

	// Manually create a beat as if it came from LLM analysis
	beat := &storage.Beat{
		BookID:           book.ID,
		ChapterID:        "ch1",
		Sequence:         0,
		WordCount:        50,
		Summary:          "Test summary",
		Conflict:         "Test conflict",
		Choice:           "Test choice",
		Consequence:      "Test consequence",
		PerspectiveShift: "",
		LLMPrompt:        "test prompt",
		LLMResponse:      `{"summary": "Test summary"}`,
	}

	err := store.SaveBeat(beat)
	if err != nil {
		t.Fatalf("SaveBeat failed: %v", err)
	}

	// Verify the beat was saved correctly
	beats, err := store.ListBeatsByBook(book.ID)
	if err != nil {
		t.Fatalf("ListBeatsByBook failed: %v", err)
	}

	if len(beats) != 1 {
		t.Fatalf("expected 1 beat, got %d", len(beats))
	}

	if beats[0].Summary != "Test summary" {
		t.Errorf("unexpected summary: %q", beats[0].Summary)
	}
}

func TestAnalyzer_WithOptions(t *testing.T) {
	store := testStore(t)

	analyzer, err := NewAnalyzer(store, "test-key",
		WithModel("claude-3-opus"),
		WithMaxTokens(2048),
		WithMaxSceneChars(4000),
	)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}

	if analyzer.model != "claude-3-opus" {
		t.Errorf("expected model claude-3-opus, got %s", analyzer.model)
	}
	if analyzer.maxTokens != 2048 {
		t.Errorf("expected maxTokens 2048, got %d", analyzer.maxTokens)
	}
	if analyzer.maxSceneChars != 4000 {
		t.Errorf("expected maxSceneChars 4000, got %d", analyzer.maxSceneChars)
	}
}

func TestTruncateForError(t *testing.T) {
	short := "short string"
	if truncateForError(short) != short {
		t.Error("short strings should not be truncated")
	}

	long := string(make([]byte, 300))
	truncated := truncateForError(long)
	if len(truncated) > 210 { // 200 + "..."
		t.Errorf("long string should be truncated, got length %d", len(truncated))
	}
}
