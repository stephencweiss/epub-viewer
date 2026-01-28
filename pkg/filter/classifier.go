package filter

import (
	"strings"

	"epub-reader/pkg/storage"
)

// Decision constants
const (
	DecisionAllow = "ALLOW"
	DecisionDeny  = "DENY"
)

// Source constants for tracking where rules came from
const (
	SourceDefault = "default"
	SourceManual  = "manual"
	SourceLLM     = "llm"
)

// DefaultAllowList contains patterns that typically indicate author content.
var DefaultAllowList = []string{
	"chapter",
	"part",
	"body",
	"text",
	"content",
	"section",
	"story",
	"prologue",
	"epilogue",
}

// DefaultDenyList contains patterns that typically indicate non-author content.
var DefaultDenyList = []string{
	"copyright",
	"toc",
	"nav",
	"ncx",
	"colophon",
	"dedication",
	"epigraph",
	"foreword",
	"preface",
	"introduction",
	"afterword",
	"appendix",
	"glossary",
	"index",
	"bibliography",
	"advert",
	"advertisement",
	"sample",
	"preview",
	"titlepage",
	"halftitle",
	"cover",
	"frontmatter",
	"backmatter",
	"acknowledgment",
	"acknowledgement",
}

// EpubTypeAllowList contains epub:type values that indicate author content.
var EpubTypeAllowList = []string{
	"bodymatter",
	"chapter",
	"part",
	"volume",
	"division",
	"subchapter",
}

// EpubTypeDenyList contains epub:type values that indicate non-author content.
var EpubTypeDenyList = []string{
	"frontmatter",
	"backmatter",
	"toc",
	"landmarks",
	"loi", // list of illustrations
	"lot", // list of tables
	"copyright-page",
	"colophon",
	"dedication",
	"epigraph",
	"foreword",
	"preface",
	"introduction",
	"afterword",
	"appendix",
	"glossary",
	"index",
	"bibliography",
	"titlepage",
	"halftitlepage",
	"cover",
	"acknowledgments",
}

// ClassificationResult represents the outcome of classifying a section.
type ClassificationResult struct {
	Decision   string  // "ALLOW" or "DENY"
	Confidence float64 // 0.0-1.0
	Reason     string  // Human-readable explanation
	Source     string  // "default", "manual", "llm"
	NeedsLLM   bool    // True if LLM should make the final decision
}

// Classifier determines whether EPUB sections should be included in analysis.
type Classifier struct {
	store          *storage.SQLiteStore
	decisionEngine DecisionEngine // Interface for LLM fallback

	// In-memory lists loaded from defaults + database
	allowPatterns []string
	denyPatterns  []string
}

// DecisionEngine interface for LLM-based classification.
// This allows pluggable backends (Claude, OpenAI, local models, etc.)
type DecisionEngine interface {
	Classify(fileName, epubType, snippet string) (*ClassificationResult, error)
}

// NewClassifier creates a new section classifier.
func NewClassifier(store *storage.SQLiteStore) *Classifier {
	c := &Classifier{
		store:         store,
		allowPatterns: make([]string, len(DefaultAllowList)),
		denyPatterns:  make([]string, len(DefaultDenyList)),
	}

	// Copy default lists
	copy(c.allowPatterns, DefaultAllowList)
	copy(c.denyPatterns, DefaultDenyList)

	// Load rules from database if store is provided
	if store != nil {
		c.loadRulesFromDB()
	}

	return c
}

// SetDecisionEngine sets the LLM backend for ambiguous cases.
func (c *Classifier) SetDecisionEngine(engine DecisionEngine) {
	c.decisionEngine = engine
}

// loadRulesFromDB loads persistent rules from the database.
func (c *Classifier) loadRulesFromDB() {
	rules, err := c.store.ListSectionRules()
	if err != nil {
		return // Silently continue with defaults
	}

	for _, rule := range rules {
		if rule.Decision == DecisionAllow {
			c.allowPatterns = append(c.allowPatterns, rule.Pattern)
		} else {
			c.denyPatterns = append(c.denyPatterns, rule.Pattern)
		}
	}
}

// Classify determines whether a section should be included.
// It checks in order: epub:type, allowlist, denylist, then optionally LLM.
func (c *Classifier) Classify(fileName, epubType, snippet string) *ClassificationResult {
	fileNameLower := strings.ToLower(fileName)
	epubTypeLower := strings.ToLower(epubType)

	// 1. Check epub:type first (most reliable signal)
	if epubTypeLower != "" {
		if result := c.checkEpubType(epubTypeLower); result != nil {
			return result
		}
	}

	// 2. Check allowlist (high confidence allow)
	if result := c.checkAllowList(fileNameLower, epubTypeLower); result != nil {
		return result
	}

	// 3. Check denylist (high confidence deny)
	if result := c.checkDenyList(fileNameLower, epubTypeLower); result != nil {
		return result
	}

	// 4. If we have an LLM engine, ask it
	if c.decisionEngine != nil {
		result, err := c.decisionEngine.Classify(fileName, epubType, snippet)
		if err == nil && result != nil {
			return result
		}
	}

	// 5. Default: mark as needing review (conservative approach)
	return &ClassificationResult{
		Decision:   DecisionAllow, // Default to allow to not lose content
		Confidence: 0.3,
		Reason:     "No matching pattern found, defaulting to allow",
		Source:     SourceDefault,
		NeedsLLM:   true, // Flag for later LLM review
	}
}

// checkEpubType checks the epub:type attribute against known values.
func (c *Classifier) checkEpubType(epubType string) *ClassificationResult {
	// Check allow types
	for _, allowType := range EpubTypeAllowList {
		if strings.Contains(epubType, allowType) {
			return &ClassificationResult{
				Decision:   DecisionAllow,
				Confidence: 0.95,
				Reason:     "epub:type '" + epubType + "' matches allow pattern '" + allowType + "'",
				Source:     SourceDefault,
				NeedsLLM:   false,
			}
		}
	}

	// Check deny types
	for _, denyType := range EpubTypeDenyList {
		if strings.Contains(epubType, denyType) {
			return &ClassificationResult{
				Decision:   DecisionDeny,
				Confidence: 0.95,
				Reason:     "epub:type '" + epubType + "' matches deny pattern '" + denyType + "'",
				Source:     SourceDefault,
				NeedsLLM:   false,
			}
		}
	}

	return nil
}

// checkAllowList checks filename against allow patterns.
func (c *Classifier) checkAllowList(fileName, epubType string) *ClassificationResult {
	for _, pattern := range c.allowPatterns {
		if strings.Contains(fileName, pattern) || strings.Contains(epubType, pattern) {
			return &ClassificationResult{
				Decision:   DecisionAllow,
				Confidence: 0.8,
				Reason:     "Filename/type matches allow pattern '" + pattern + "'",
				Source:     SourceDefault,
				NeedsLLM:   false,
			}
		}
	}
	return nil
}

// checkDenyList checks filename against deny patterns.
func (c *Classifier) checkDenyList(fileName, epubType string) *ClassificationResult {
	for _, pattern := range c.denyPatterns {
		if strings.Contains(fileName, pattern) || strings.Contains(epubType, pattern) {
			return &ClassificationResult{
				Decision:   DecisionDeny,
				Confidence: 0.8,
				Reason:     "Filename/type matches deny pattern '" + pattern + "'",
				Source:     SourceDefault,
				NeedsLLM:   false,
			}
		}
	}
	return nil
}

// AddRule adds a persistent rule to the classifier.
func (c *Classifier) AddRule(pattern, decision string, confidence float64, source string) error {
	if c.store == nil {
		return nil
	}

	_, err := c.store.CreateSectionRule(pattern, decision, source, confidence)
	if err != nil {
		return err
	}

	// Update in-memory lists
	if decision == DecisionAllow {
		c.allowPatterns = append(c.allowPatterns, pattern)
	} else {
		c.denyPatterns = append(c.denyPatterns, pattern)
	}

	return nil
}

// ShouldProcess is a convenience method that returns true if content should be analyzed.
func (c *Classifier) ShouldProcess(fileName, epubType, snippet string) bool {
	result := c.Classify(fileName, epubType, snippet)
	return result.Decision == DecisionAllow
}

// GetSnippet extracts the first N characters from text for LLM classification.
func GetSnippet(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Try to cut at a word boundary
	snippet := text[:maxLen]
	lastSpace := strings.LastIndex(snippet, " ")
	if lastSpace > maxLen/2 {
		snippet = snippet[:lastSpace]
	}
	return snippet + "..."
}
