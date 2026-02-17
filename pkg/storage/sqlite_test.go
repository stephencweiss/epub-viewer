package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// testStore creates a temporary SQLite store for testing.
func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewSQLiteStore(dbPath)
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
func createTestBook(t *testing.T, store *SQLiteStore) *Book {
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

func TestSaveBeat(t *testing.T) {
	store := testStore(t)
	book := createTestBook(t, store)

	beat := &Beat{
		BookID:           book.ID,
		ChapterID:        "chapter1",
		Sequence:         0,
		WordCount:        500,
		Summary:          "The protagonist discovers a hidden letter.",
		Conflict:         "Internal conflict about whether to read the letter.",
		Choice:           "The protagonist decides to read the letter despite reservations.",
		Consequence:      "Learning a family secret that changes everything.",
		PerspectiveShift: "Reader now sees the father in a different light.",
		LLMPrompt:        "Analyze this scene...",
		LLMResponse:      `{"summary": "..."}`,
	}

	err := store.SaveBeat(beat)
	if err != nil {
		t.Fatalf("SaveBeat failed: %v", err)
	}

	if beat.ID == 0 {
		t.Error("expected beat ID to be assigned after save")
	}

	// Verify we can retrieve it
	retrieved, err := store.GetBeat(beat.ID)
	if err != nil {
		t.Fatalf("GetBeat failed: %v", err)
	}

	if retrieved.Summary != beat.Summary {
		t.Errorf("summary mismatch: got %q, want %q", retrieved.Summary, beat.Summary)
	}
	if retrieved.Conflict != beat.Conflict {
		t.Errorf("conflict mismatch: got %q, want %q", retrieved.Conflict, beat.Conflict)
	}
	if retrieved.Choice != beat.Choice {
		t.Errorf("choice mismatch: got %q, want %q", retrieved.Choice, beat.Choice)
	}
	if retrieved.Consequence != beat.Consequence {
		t.Errorf("consequence mismatch: got %q, want %q", retrieved.Consequence, beat.Consequence)
	}
	if retrieved.PerspectiveShift != beat.PerspectiveShift {
		t.Errorf("perspective_shift mismatch: got %q, want %q", retrieved.PerspectiveShift, beat.PerspectiveShift)
	}
}

func TestListBeatsByBook(t *testing.T) {
	store := testStore(t)
	book := createTestBook(t, store)

	// Create beats out of order to verify sorting
	beats := []*Beat{
		{BookID: book.ID, Sequence: 2, Summary: "Third beat", Conflict: "c3", Choice: "ch3", Consequence: "co3"},
		{BookID: book.ID, Sequence: 0, Summary: "First beat", Conflict: "c1", Choice: "ch1", Consequence: "co1"},
		{BookID: book.ID, Sequence: 1, Summary: "Second beat", Conflict: "c2", Choice: "ch2", Consequence: "co2"},
	}

	for _, b := range beats {
		if err := store.SaveBeat(b); err != nil {
			t.Fatalf("SaveBeat failed: %v", err)
		}
	}

	// Retrieve and verify order
	result, err := store.ListBeatsByBook(book.ID)
	if err != nil {
		t.Fatalf("ListBeatsByBook failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 beats, got %d", len(result))
	}

	expectedOrder := []string{"First beat", "Second beat", "Third beat"}
	for i, expected := range expectedOrder {
		if result[i].Summary != expected {
			t.Errorf("beat %d: got summary %q, want %q", i, result[i].Summary, expected)
		}
		if result[i].Sequence != i {
			t.Errorf("beat %d: got sequence %d, want %d", i, result[i].Sequence, i)
		}
	}
}

func TestDeleteBeatsByBook(t *testing.T) {
	store := testStore(t)
	book := createTestBook(t, store)

	// Create some beats
	for i := 0; i < 3; i++ {
		beat := &Beat{
			BookID:      book.ID,
			Sequence:    i,
			Summary:     "summary",
			Conflict:    "conflict",
			Choice:      "choice",
			Consequence: "consequence",
		}
		if err := store.SaveBeat(beat); err != nil {
			t.Fatalf("SaveBeat failed: %v", err)
		}
	}

	// Verify beats exist
	has, _ := store.HasBeats(book.ID)
	if !has {
		t.Fatal("expected book to have beats")
	}

	// Delete beats
	err := store.DeleteBeatsByBook(book.ID)
	if err != nil {
		t.Fatalf("DeleteBeatsByBook failed: %v", err)
	}

	// Verify beats are gone
	has, _ = store.HasBeats(book.ID)
	if has {
		t.Error("expected book to have no beats after deletion")
	}

	beats, _ := store.ListBeatsByBook(book.ID)
	if len(beats) != 0 {
		t.Errorf("expected 0 beats after deletion, got %d", len(beats))
	}
}

func TestHasBeats(t *testing.T) {
	store := testStore(t)
	book := createTestBook(t, store)

	// Initially no beats
	has, err := store.HasBeats(book.ID)
	if err != nil {
		t.Fatalf("HasBeats failed: %v", err)
	}
	if has {
		t.Error("expected HasBeats to return false for book with no beats")
	}

	// Add a beat
	beat := &Beat{
		BookID:      book.ID,
		Sequence:    0,
		Summary:     "summary",
		Conflict:    "conflict",
		Choice:      "choice",
		Consequence: "consequence",
	}
	if err := store.SaveBeat(beat); err != nil {
		t.Fatalf("SaveBeat failed: %v", err)
	}

	// Now has beats
	has, err = store.HasBeats(book.ID)
	if err != nil {
		t.Fatalf("HasBeats failed: %v", err)
	}
	if !has {
		t.Error("expected HasBeats to return true after adding a beat")
	}
}

func TestBeatsCascadeDelete(t *testing.T) {
	store := testStore(t)
	book := createTestBook(t, store)

	// Create beats for the book
	for i := 0; i < 3; i++ {
		beat := &Beat{
			BookID:      book.ID,
			Sequence:    i,
			Summary:     "summary",
			Conflict:    "conflict",
			Choice:      "choice",
			Consequence: "consequence",
		}
		if err := store.SaveBeat(beat); err != nil {
			t.Fatalf("SaveBeat failed: %v", err)
		}
	}

	// Verify beats exist
	count, _ := store.CountBeatsByBook(book.ID)
	if count != 3 {
		t.Fatalf("expected 3 beats, got %d", count)
	}

	// Delete the book
	err := store.RemoveBook(book.ID)
	if err != nil {
		t.Fatalf("RemoveBook failed: %v", err)
	}

	// Verify beats are cascade deleted
	count, _ = store.CountBeatsByBook(book.ID)
	if count != 0 {
		t.Errorf("expected 0 beats after book deletion, got %d", count)
	}
}

func TestGetBeat_NotFound(t *testing.T) {
	store := testStore(t)

	_, err := store.GetBeat(99999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCountBeatsByBook(t *testing.T) {
	store := testStore(t)
	book := createTestBook(t, store)

	// Initially 0
	count, err := store.CountBeatsByBook(book.ID)
	if err != nil {
		t.Fatalf("CountBeatsByBook failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 beats, got %d", count)
	}

	// Add beats
	for i := 0; i < 5; i++ {
		beat := &Beat{
			BookID:      book.ID,
			Sequence:    i,
			Summary:     "summary",
			Conflict:    "conflict",
			Choice:      "choice",
			Consequence: "consequence",
		}
		store.SaveBeat(beat)
	}

	count, _ = store.CountBeatsByBook(book.ID)
	if count != 5 {
		t.Errorf("expected 5 beats, got %d", count)
	}
}
