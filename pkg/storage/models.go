package storage

import (
	"time"

	"epub-reader/pkg/analysis"
)

// Author represents an author in the database.
type Author struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// Book represents a book in the database.
type Book struct {
	ID        int64
	AuthorID  int64
	Title     string
	Path      string
	Language  string
	Publisher string
	AddedAt   time.Time

	// Joined fields
	AuthorName string
}

// StoredAnalysis represents stored analysis results.
type StoredAnalysis struct {
	ID        int64
	BookID    int64
	CreatedAt time.Time

	// Word metrics
	TotalWords     int
	UniqueWords    int
	VocabularyRich float64
	HapaxLegomena  int
	AvgWordLen     float64

	// Sentence metrics
	TotalSentences int
	AvgSentenceLen float64

	// Paragraph metrics
	TotalParagraphs int
	AvgParagraphLen float64

	// Style metrics
	ReadabilityScore float64
	DialogueRatio    float64
	TotalSyllables   int
	AvgSyllables     float64
}

// ToAnalysis converts a StoredAnalysis to an analysis.Analysis.
func (s *StoredAnalysis) ToAnalysis() *analysis.Analysis {
	return &analysis.Analysis{
		TotalWords:       s.TotalWords,
		UniqueWords:      s.UniqueWords,
		VocabularyRich:   s.VocabularyRich,
		HapaxLegomena:    s.HapaxLegomena,
		AverageWordLen:   s.AvgWordLen,
		TotalSentences:   s.TotalSentences,
		AvgSentenceLen:   s.AvgSentenceLen,
		TotalParagraphs:  s.TotalParagraphs,
		AvgParagraphLen:  s.AvgParagraphLen,
		ReadabilityScore: s.ReadabilityScore,
		DialogueRatio:    s.DialogueRatio,
		TotalSyllables:   s.TotalSyllables,
		AvgSyllablesWord: s.AvgSyllables,
	}
}

// FromAnalysis creates a StoredAnalysis from an analysis.Analysis.
func FromAnalysis(bookID int64, a *analysis.Analysis) *StoredAnalysis {
	return &StoredAnalysis{
		BookID:           bookID,
		TotalWords:       a.TotalWords,
		UniqueWords:      a.UniqueWords,
		VocabularyRich:   a.VocabularyRich,
		HapaxLegomena:    a.HapaxLegomena,
		AvgWordLen:       a.AverageWordLen,
		TotalSentences:   a.TotalSentences,
		AvgSentenceLen:   a.AvgSentenceLen,
		TotalParagraphs:  a.TotalParagraphs,
		AvgParagraphLen:  a.AvgParagraphLen,
		ReadabilityScore: a.ReadabilityScore,
		DialogueRatio:    a.DialogueRatio,
		TotalSyllables:   a.TotalSyllables,
		AvgSyllables:     a.AvgSyllablesWord,
	}
}

// CorpusAnalysis represents aggregated analysis for an author's corpus.
type CorpusAnalysis struct {
	Author     Author
	BookCount  int
	TotalWords int

	// Averaged metrics
	AvgVocabularyRich float64
	AvgReadability    float64
	AvgSentenceLen    float64
	AvgWordLen        float64
	AvgDialogueRatio  float64

	// Totals
	TotalUniqueWords int
}

// ComparisonResult represents a comparison between two authors.
type ComparisonResult struct {
	Author1 Author
	Author2 Author
	Corpus1 CorpusAnalysis
	Corpus2 CorpusAnalysis

	// Differences (Author2 - Author1)
	VocabularyDiff    float64
	ReadabilityDiff   float64
	SentenceLenDiff   float64
	WordLenDiff       float64
	DialogueRatioDiff float64
}

// SectionRule represents a persistent allow/deny rule for section filtering.
type SectionRule struct {
	ID         int64
	Pattern    string  // Filename pattern or epub:type value
	Decision   string  // "ALLOW" or "DENY"
	Confidence float64 // 0.0-1.0, higher = more certain
	Source     string  // "manual", "llm", "default"
	CreatedAt  time.Time
}

// DecisionAudit stores LLM decisions for evaluation and transparency.
type DecisionAudit struct {
	ID               int64
	BookID           int64
	FileName         string
	EpubType         string // epub:type attribute if present
	TextSnippet      string // First ~500 chars sent to LLM
	LLMPrompt        string // Full prompt for reproducibility
	LLMResponse      string // Raw LLM response
	FinalDecision    string // "ALLOW" or "DENY"
	Reason           string // LLM's reasoning
	ManuallyVerified bool   // Has a human reviewed this?
	CreatedAt        time.Time
}

// Section represents an individual content section from an EPUB.
type Section struct {
	ID          int64
	BookID      int64
	FileName    string
	EpubType    string // epub:type attribute
	Title       string
	TextContent string // Full text content
	Status      string // "ALLOW", "DENY", "PENDING"
	Order       int    // Reading order from spine
	CreatedAt   time.Time
}

// Beat represents a narrative beat extracted from a book.
type Beat struct {
	ID               int64
	BookID           int64
	ChapterID        string // Reference to chapter if applicable
	Sequence         int    // Order within the book (0-indexed)
	WordCount        int
	Summary          string
	Conflict         string
	Choice           string
	Consequence      string
	PerspectiveShift string // Optional
	LLMPrompt        string // Full prompt for reproducibility
	LLMResponse      string // Raw LLM response
	CreatedAt        time.Time
}
