package analysis

import (
	"epub-reader/pkg/epub"
)

// Analysis contains all text analysis results.
type Analysis struct {
	// Word metrics
	TotalWords       int
	UniqueWords      int
	VocabularyRich   float64 // Type-token ratio
	HapaxLegomena    int     // Words appearing only once
	TopWords         []WordFreq
	AverageWordLen   float64

	// Sentence metrics
	TotalSentences   int
	AvgSentenceLen   float64 // Words per sentence
	SentenceLenDist  map[int]int // Length -> count

	// Paragraph metrics
	TotalParagraphs  int
	AvgParagraphLen  float64 // Sentences per paragraph

	// Style metrics
	ReadabilityScore float64 // Flesch-Kincaid
	DialogueRatio    float64 // Ratio of dialogue to narrative
	TotalSyllables   int
	AvgSyllablesWord float64
}

// WordFreq represents a word and its frequency.
type WordFreq struct {
	Word  string
	Count int
}

// Analyzer analyzes text for various metrics.
type Analyzer struct {
	// TopWordsCount is the number of top words to include.
	TopWordsCount int
}

// NewAnalyzer creates a new Analyzer with default settings.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		TopWordsCount: 50,
	}
}

// AnalyzeBook analyzes an entire EPUB book.
func (a *Analyzer) AnalyzeBook(book *epub.Book) *Analysis {
	return a.AnalyzeText(book.FullText())
}

// AnalyzeText performs comprehensive text analysis.
func (a *Analyzer) AnalyzeText(text string) *Analysis {
	analysis := &Analysis{
		SentenceLenDist: make(map[int]int),
	}

	// Word analysis
	wordStats := analyzeWords(text)
	analysis.TotalWords = wordStats.total
	analysis.UniqueWords = wordStats.unique
	analysis.VocabularyRich = wordStats.typeTokenRatio
	analysis.HapaxLegomena = wordStats.hapaxCount
	analysis.TopWords = getTopWords(wordStats.frequencies, a.TopWordsCount)
	analysis.AverageWordLen = wordStats.avgWordLength

	// Sentence analysis
	sentenceStats := analyzeSentences(text)
	analysis.TotalSentences = sentenceStats.count
	analysis.AvgSentenceLen = sentenceStats.avgLength
	analysis.SentenceLenDist = sentenceStats.lengthDist

	// Paragraph analysis
	paragraphStats := analyzeParagraphs(text)
	analysis.TotalParagraphs = paragraphStats.count
	analysis.AvgParagraphLen = paragraphStats.avgSentences

	// Style analysis
	styleStats := analyzeStyle(text, wordStats, sentenceStats)
	analysis.ReadabilityScore = styleStats.readability
	analysis.DialogueRatio = styleStats.dialogueRatio
	analysis.TotalSyllables = styleStats.syllables
	analysis.AvgSyllablesWord = styleStats.avgSyllables

	return analysis
}

// CompareAnalyses compares two analyses and returns differences.
func CompareAnalyses(a1, a2 *Analysis) map[string]float64 {
	diff := make(map[string]float64)

	if a1.TotalWords > 0 && a2.TotalWords > 0 {
		diff["words_ratio"] = float64(a2.TotalWords) / float64(a1.TotalWords)
	}
	diff["vocab_diff"] = a2.VocabularyRich - a1.VocabularyRich
	diff["avg_sentence_diff"] = a2.AvgSentenceLen - a1.AvgSentenceLen
	diff["readability_diff"] = a2.ReadabilityScore - a1.ReadabilityScore
	diff["dialogue_diff"] = a2.DialogueRatio - a1.DialogueRatio
	diff["avg_word_len_diff"] = a2.AverageWordLen - a1.AverageWordLen

	return diff
}
