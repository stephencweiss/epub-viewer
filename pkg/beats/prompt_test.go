package beats

import (
	"strings"
	"testing"
)

func TestBuildBeatPrompt(t *testing.T) {
	sceneText := "The protagonist walked into the room and found a letter on the table."

	prompt := BuildBeatPrompt(sceneText, DefaultMaxSceneChars)

	// Verify prompt contains the scene text
	if !strings.Contains(prompt, sceneText) {
		t.Error("prompt should contain the scene text")
	}

	// Verify prompt contains key instructions
	if !strings.Contains(prompt, "SUMMARY") {
		t.Error("prompt should contain SUMMARY instruction")
	}
	if !strings.Contains(prompt, "CONFLICT") {
		t.Error("prompt should contain CONFLICT instruction")
	}
	if !strings.Contains(prompt, "CHOICE") {
		t.Error("prompt should contain CHOICE instruction")
	}
	if !strings.Contains(prompt, "CONSEQUENCE") {
		t.Error("prompt should contain CONSEQUENCE instruction")
	}
	if !strings.Contains(prompt, "PERSPECTIVE_SHIFT") {
		t.Error("prompt should contain PERSPECTIVE_SHIFT instruction")
	}

	// Verify JSON format is specified
	if !strings.Contains(prompt, `"summary"`) {
		t.Error("prompt should specify JSON format with summary field")
	}
}

func TestBuildBeatPrompt_Truncation(t *testing.T) {
	// Create a scene text longer than max chars
	longText := strings.Repeat("word ", 2000) // ~10000 chars
	maxChars := 100

	prompt := BuildBeatPrompt(longText, maxChars)

	// Verify truncation happened
	if !strings.Contains(prompt, "[Scene truncated for analysis...]") {
		t.Error("long scene should be truncated")
	}

	// The prompt itself will be longer than maxChars because of the template,
	// but the scene text portion should be limited
	sceneStart := strings.Index(prompt, "---\n") + 4
	sceneEnd := strings.Index(prompt[sceneStart:], "\n---")

	if sceneEnd > maxChars+50 { // Allow some buffer for truncation message
		t.Errorf("truncated scene portion too long: %d chars", sceneEnd)
	}
}

func TestBuildBeatPrompt_DefaultMaxChars(t *testing.T) {
	sceneText := "Short text"

	// Pass 0 to use default
	prompt := BuildBeatPrompt(sceneText, 0)

	if !strings.Contains(prompt, sceneText) {
		t.Error("prompt should contain scene text when using default max chars")
	}
}

func TestParseBeatResponse_Valid(t *testing.T) {
	response := `{
		"summary": "The hero discovers a hidden map.",
		"conflict": "Internal struggle between duty and curiosity.",
		"choice": "The hero decides to follow the map's path.",
		"consequence": "This leads to an unexpected encounter.",
		"perspective_shift": "Reader sees the hero's adventurous side."
	}`

	result, err := ParseBeatResponse(response)
	if err != nil {
		t.Fatalf("ParseBeatResponse failed: %v", err)
	}

	if result.Summary != "The hero discovers a hidden map." {
		t.Errorf("unexpected summary: %q", result.Summary)
	}
	if result.Conflict != "Internal struggle between duty and curiosity." {
		t.Errorf("unexpected conflict: %q", result.Conflict)
	}
	if result.Choice != "The hero decides to follow the map's path." {
		t.Errorf("unexpected choice: %q", result.Choice)
	}
	if result.Consequence != "This leads to an unexpected encounter." {
		t.Errorf("unexpected consequence: %q", result.Consequence)
	}
	if result.PerspectiveShift != "Reader sees the hero's adventurous side." {
		t.Errorf("unexpected perspective_shift: %q", result.PerspectiveShift)
	}
}

func TestParseBeatResponse_Wrapped(t *testing.T) {
	// LLM sometimes adds extra text around the JSON
	response := `Here's the analysis:

{
	"summary": "A brief meeting occurs.",
	"conflict": "Tension between characters.",
	"choice": "One character walks away.",
	"consequence": "The relationship changes.",
	"perspective_shift": ""
}

I hope this helps!`

	result, err := ParseBeatResponse(response)
	if err != nil {
		t.Fatalf("ParseBeatResponse failed: %v", err)
	}

	if result.Summary != "A brief meeting occurs." {
		t.Errorf("unexpected summary: %q", result.Summary)
	}
}

func TestParseBeatResponse_Invalid(t *testing.T) {
	invalidResponses := []string{
		"This is not JSON at all",
		"{ incomplete json",
		"",
		"   ",
	}

	for _, response := range invalidResponses {
		_, err := ParseBeatResponse(response)
		if err == nil {
			t.Errorf("expected error for invalid response: %q", response)
		}
	}
}

func TestParseBeatResponse_EmptyPerspectiveShift(t *testing.T) {
	response := `{
		"summary": "Something happens.",
		"conflict": "Some conflict.",
		"choice": "A choice is made.",
		"consequence": "Result follows.",
		"perspective_shift": ""
	}`

	result, err := ParseBeatResponse(response)
	if err != nil {
		t.Fatalf("ParseBeatResponse failed: %v", err)
	}

	if result.PerspectiveShift != "" {
		t.Errorf("expected empty perspective_shift, got: %q", result.PerspectiveShift)
	}
}

func TestParseBeatResponse_MissingOptionalField(t *testing.T) {
	// perspective_shift is optional
	response := `{
		"summary": "Something happens.",
		"conflict": "Some conflict.",
		"choice": "A choice is made.",
		"consequence": "Result follows."
	}`

	result, err := ParseBeatResponse(response)
	if err != nil {
		t.Fatalf("ParseBeatResponse failed: %v", err)
	}

	if result.Summary != "Something happens." {
		t.Errorf("unexpected summary: %q", result.Summary)
	}
}
