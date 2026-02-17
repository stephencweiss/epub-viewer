package beats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"epub-reader/pkg/epub"
	"epub-reader/pkg/storage"
)

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	defaultModel     = "claude-sonnet-4-20250514"
	defaultMaxTokens = 1024
)

// Analyzer orchestrates beat extraction for a book.
type Analyzer struct {
	store         *storage.SQLiteStore
	apiKey        string
	model         string
	maxTokens     int
	maxSceneChars int
	httpClient    *http.Client

	// Progress callback called after each scene is analyzed.
	// Arguments: processed count, total count, current scene preview
	OnProgress func(processed, total int, preview string)
}

// AnalyzerOption configures an Analyzer.
type AnalyzerOption func(*Analyzer)

// WithModel sets the Claude model to use.
func WithModel(model string) AnalyzerOption {
	return func(a *Analyzer) {
		a.model = model
	}
}

// WithMaxTokens sets the max tokens for responses.
func WithMaxTokens(maxTokens int) AnalyzerOption {
	return func(a *Analyzer) {
		a.maxTokens = maxTokens
	}
}

// WithMaxSceneChars sets the max characters to send per scene.
func WithMaxSceneChars(maxChars int) AnalyzerOption {
	return func(a *Analyzer) {
		a.maxSceneChars = maxChars
	}
}

// WithHTTPClient sets a custom HTTP client (useful for testing).
func WithHTTPClient(client *http.Client) AnalyzerOption {
	return func(a *Analyzer) {
		a.httpClient = client
	}
}

// NewAnalyzer creates a new beat analyzer.
// It reads the API key from ANTHROPIC_API_KEY environment variable if not provided.
func NewAnalyzer(store *storage.SQLiteStore, apiKey string, opts ...AnalyzerOption) (*Analyzer, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
	}

	a := &Analyzer{
		store:         store,
		apiKey:        apiKey,
		model:         defaultModel,
		maxTokens:     defaultMaxTokens,
		maxSceneChars: DefaultMaxSceneChars,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(a)
	}

	return a, nil
}

// AnalyzeBook extracts all beats from a book.
// It detects scenes, sends each to the LLM for analysis, and stores the results.
func (a *Analyzer) AnalyzeBook(bookID int64, book *epub.Book) ([]storage.Beat, error) {
	// Detect scenes
	scenes := DetectScenes(book)
	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes detected in book")
	}

	var beats []storage.Beat

	for i, scene := range scenes {
		// Progress callback
		if a.OnProgress != nil {
			preview := scene.Text
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			a.OnProgress(i+1, len(scenes), preview)
		}

		// Analyze scene
		beat, err := a.analyzeScene(bookID, scene)
		if err != nil {
			// Log error but continue with next scene
			// Could add error callback here for more sophisticated handling
			continue
		}

		beats = append(beats, *beat)
	}

	return beats, nil
}

// analyzeScene sends a scene to the LLM and stores the resulting beat.
func (a *Analyzer) analyzeScene(bookID int64, scene Scene) (*storage.Beat, error) {
	prompt := BuildBeatPrompt(scene.Text, a.maxSceneChars)

	// Call LLM
	response, err := a.callLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	result, err := ParseBeatResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	beat := &storage.Beat{
		BookID:           bookID,
		ChapterID:        scene.ChapterID,
		Sequence:         scene.Index,
		WordCount:        scene.WordCount,
		Summary:          result.Summary,
		Conflict:         result.Conflict,
		Choice:           result.Choice,
		Consequence:      result.Consequence,
		PerspectiveShift: result.PerspectiveShift,
		LLMPrompt:        prompt,
		LLMResponse:      response,
	}

	// Save to database
	if err := a.store.SaveBeat(beat); err != nil {
		return nil, fmt.Errorf("failed to save beat: %w", err)
	}

	return beat, nil
}

// claudeRequest is the request body for the Claude API.
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse is the response from the Claude API.
type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// callLLM sends a prompt to Claude and returns the response text.
func (a *Analyzer) callLLM(prompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", anthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}

// ParseBeatResponse parses the LLM response into a BeatAnalysisResult.
// It handles both clean JSON and JSON embedded in other text.
func ParseBeatResponse(response string) (*BeatAnalysisResult, error) {
	response = strings.TrimSpace(response)

	var result BeatAnalysisResult

	// Try direct parse first
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		return &result, nil
	}

	// Try to extract JSON from response (LLM sometimes adds extra text)
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx != -1 && endIdx > startIdx {
		jsonStr := response[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return &result, nil
		}
	}

	return nil, fmt.Errorf("could not parse beat response: %s", truncateForError(response))
}

// truncateForError truncates a string for error messages.
func truncateForError(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
