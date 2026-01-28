package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// AnthropicAPIURL is the endpoint for Claude API requests.
	AnthropicAPIURL = "https://api.anthropic.com/v1/messages"

	// DefaultModel is the default Claude model to use.
	DefaultModel = "claude-sonnet-4-20250514"

	// DefaultMaxTokens is the default max tokens for classification responses.
	DefaultMaxTokens = 256
)

// ClaudeEngine implements DecisionEngine using the Anthropic Claude API.
type ClaudeEngine struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// ClaudeEngineOption configures a ClaudeEngine.
type ClaudeEngineOption func(*ClaudeEngine)

// WithModel sets the Claude model to use.
func WithModel(model string) ClaudeEngineOption {
	return func(e *ClaudeEngine) {
		e.model = model
	}
}

// WithMaxTokens sets the max tokens for responses.
func WithMaxTokens(maxTokens int) ClaudeEngineOption {
	return func(e *ClaudeEngine) {
		e.maxTokens = maxTokens
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClaudeEngineOption {
	return func(e *ClaudeEngine) {
		e.httpClient = client
	}
}

// NewClaudeEngine creates a new Claude-based decision engine.
// It reads the API key from ANTHROPIC_API_KEY environment variable if not provided.
func NewClaudeEngine(apiKey string, opts ...ClaudeEngineOption) (*ClaudeEngine, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
	}

	engine := &ClaudeEngine{
		apiKey:    apiKey,
		model:     DefaultModel,
		maxTokens: DefaultMaxTokens,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(engine)
	}

	return engine, nil
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
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Classify sends the section to Claude for classification.
func (c *ClaudeEngine) Classify(fileName, epubType, snippet string) (*ClassificationResult, error) {
	prompt := BuildPrompt(fileName, epubType, snippet)

	reqBody := claudeRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", AnthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if claudeResp.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	// Extract the text response
	responseText := claudeResp.Content[0].Text

	// Parse the LLM response
	llmResp, err := ParseLLMResponse(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return &ClassificationResult{
		Decision:   strings.ToUpper(llmResp.Decision),
		Confidence: 0.85, // LLM decisions get moderate-high confidence
		Reason:     llmResp.Reason,
		Source:     SourceLLM,
		NeedsLLM:   false,
	}, nil
}

// TestConnection verifies the API key works by making a minimal request.
func (c *ClaudeEngine) TestConnection() error {
	reqBody := claudeRequest{
		Model:     c.model,
		MaxTokens: 10,
		Messages: []claudeMessage{
			{Role: "user", Content: "Say 'ok'"},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", AnthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
