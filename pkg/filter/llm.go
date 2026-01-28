package filter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"epub-reader/pkg/storage"
)

// ClassificationPromptTemplate is the prompt used for LLM-based section classification.
// The LLM should return a JSON response with decision and reason.
const ClassificationPromptTemplate = `You are an EPUB content classifier. Your task is to determine whether a section of an EPUB file contains PRIMARY AUTHOR CONTENT or SUPPLEMENTARY MATERIAL.

PRIMARY AUTHOR CONTENT (classify as ALLOW):
- Main story chapters
- Author's narrative text
- Dialogue and scenes written by the author
- Prologue, epilogue written by the author

SUPPLEMENTARY MATERIAL (classify as DENY):
- Copyright pages and legal notices
- Table of contents
- Introductions written by someone OTHER than the author
- Forewords by editors or other authors
- Publisher advertisements
- Sample chapters from OTHER books
- Library of Congress cataloging data
- Dedication pages
- Acknowledgments
- Appendices with reference material
- Glossaries and indexes

SECTION INFORMATION:
- File name: %s
- epub:type attribute: %s
- Content snippet (first ~500 chars):
---
%s
---

Based on the above, classify this section.

Respond with ONLY a JSON object in this exact format:
{"decision": "ALLOW" or "DENY", "reason": "Brief explanation of why"}

JSON response:`

// LLMResponse represents the expected JSON response from the LLM.
type LLMResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// ParseLLMResponse extracts the classification decision from an LLM response.
// It handles both clean JSON and JSON embedded in other text.
func ParseLLMResponse(response string) (*LLMResponse, error) {
	response = strings.TrimSpace(response)

	// Try direct parse first
	var result LLMResponse
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		if isValidDecision(result.Decision) {
			return &result, nil
		}
	}

	// Try to extract JSON from the response
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		jsonStr := response[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			if isValidDecision(result.Decision) {
				return &result, nil
			}
		}
	}

	// Fallback: look for ALLOW or DENY keywords
	upperResponse := strings.ToUpper(response)
	if strings.Contains(upperResponse, "DENY") {
		return &LLMResponse{
			Decision: DecisionDeny,
			Reason:   "Extracted from non-JSON response",
		}, nil
	}
	if strings.Contains(upperResponse, "ALLOW") {
		return &LLMResponse{
			Decision: DecisionAllow,
			Reason:   "Extracted from non-JSON response",
		}, nil
	}

	return nil, fmt.Errorf("could not parse LLM response: %s", response)
}

func isValidDecision(d string) bool {
	upper := strings.ToUpper(d)
	return upper == DecisionAllow || upper == DecisionDeny
}

// BuildPrompt creates a classification prompt for the given section.
func BuildPrompt(fileName, epubType, snippet string) string {
	if epubType == "" {
		epubType = "(not specified)"
	}
	return fmt.Sprintf(ClassificationPromptTemplate, fileName, epubType, snippet)
}

// AuditingEngine wraps a DecisionEngine to log all decisions to the database.
type AuditingEngine struct {
	engine DecisionEngine
	store  *storage.SQLiteStore
	bookID int64
}

// NewAuditingEngine creates a new auditing wrapper around a DecisionEngine.
func NewAuditingEngine(engine DecisionEngine, store *storage.SQLiteStore, bookID int64) *AuditingEngine {
	return &AuditingEngine{
		engine: engine,
		store:  store,
		bookID: bookID,
	}
}

// Classify calls the underlying engine and logs the decision.
func (a *AuditingEngine) Classify(fileName, epubType, snippet string) (*ClassificationResult, error) {
	prompt := BuildPrompt(fileName, epubType, snippet)

	result, err := a.engine.Classify(fileName, epubType, snippet)
	if err != nil {
		// Log the failed attempt
		if a.store != nil {
			_ = a.logDecision(fileName, epubType, snippet, prompt, err.Error(), DecisionAllow, "LLM error, defaulting to ALLOW")
		}
		return nil, err
	}

	// Log the successful decision
	if a.store != nil {
		_ = a.logDecision(fileName, epubType, snippet, prompt, "", result.Decision, result.Reason)
	}

	return result, nil
}

func (a *AuditingEngine) logDecision(fileName, epubType, snippet, prompt, llmResponse, decision, reason string) error {
	audit := &storage.DecisionAudit{
		BookID:        a.bookID,
		FileName:      fileName,
		EpubType:      epubType,
		TextSnippet:   snippet,
		LLMPrompt:     prompt,
		LLMResponse:   llmResponse,
		FinalDecision: decision,
		Reason:        reason,
		CreatedAt:     time.Now(),
	}
	return a.store.SaveDecisionAudit(audit)
}

// MockEngine is a simple decision engine for testing that always returns a fixed decision.
type MockEngine struct {
	DefaultDecision string
	DefaultReason   string
}

// NewMockEngine creates a mock engine for testing.
func NewMockEngine(decision, reason string) *MockEngine {
	return &MockEngine{
		DefaultDecision: decision,
		DefaultReason:   reason,
	}
}

// Classify returns the mock's configured decision.
func (m *MockEngine) Classify(fileName, epubType, snippet string) (*ClassificationResult, error) {
	return &ClassificationResult{
		Decision:   m.DefaultDecision,
		Confidence: 0.5,
		Reason:     m.DefaultReason,
		Source:     "mock",
		NeedsLLM:   false,
	}, nil
}
