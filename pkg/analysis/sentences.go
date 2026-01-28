package analysis

import (
	"regexp"
	"strings"
)

// sentenceStats holds sentence-level statistics.
type sentenceStats struct {
	count      int
	avgLength  float64
	lengthDist map[int]int // Word count -> number of sentences
}

// analyzeSentences performs sentence-level analysis.
func analyzeSentences(text string) sentenceStats {
	stats := sentenceStats{
		lengthDist: make(map[int]int),
	}

	sentences := tokenizeSentences(text)
	stats.count = len(sentences)

	if stats.count == 0 {
		return stats
	}

	totalWords := 0
	for _, sentence := range sentences {
		words := tokenizeWords(sentence)
		wordCount := len(words)
		totalWords += wordCount

		// Bucket by sentence length
		bucket := (wordCount / 5) * 5 // 0-4, 5-9, 10-14, etc.
		stats.lengthDist[bucket]++
	}

	stats.avgLength = float64(totalWords) / float64(stats.count)

	return stats
}

// tokenizeSentences splits text into sentences.
func tokenizeSentences(text string) []string {
	// Handle common abbreviations to avoid false splits
	text = protectAbbreviations(text)

	// Split on sentence-ending punctuation
	sentenceEndRegex := regexp.MustCompile(`[.!?]+[\s]+`)
	parts := sentenceEndRegex.Split(text, -1)

	var sentences []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = restoreAbbreviations(part)
		if part != "" && len(tokenizeWords(part)) > 0 {
			sentences = append(sentences, part)
		}
	}

	return sentences
}

// protectAbbreviations temporarily replaces periods in abbreviations.
func protectAbbreviations(text string) string {
	abbrevs := []string{
		"Mr.", "Mrs.", "Ms.", "Dr.", "Prof.",
		"Sr.", "Jr.", "St.", "Rev.", "Gen.",
		"Lt.", "Col.", "Sgt.", "Capt.",
		"Inc.", "Ltd.", "Corp.", "Co.",
		"i.e.", "e.g.", "etc.", "vs.", "et al.",
		"a.m.", "p.m.", "A.M.", "P.M.",
		"Jan.", "Feb.", "Mar.", "Apr.", "Jun.",
		"Jul.", "Aug.", "Sep.", "Sept.", "Oct.", "Nov.", "Dec.",
	}

	for _, abbr := range abbrevs {
		placeholder := strings.ReplaceAll(abbr, ".", "##PERIOD##")
		text = strings.ReplaceAll(text, abbr, placeholder)
	}

	// Protect single letters with periods (initials)
	initialRegex := regexp.MustCompile(`\b([A-Z])\.\s*([A-Z])\.`)
	text = initialRegex.ReplaceAllString(text, "$1##PERIOD## $2##PERIOD##")

	return text
}

// restoreAbbreviations restores protected periods.
func restoreAbbreviations(text string) string {
	return strings.ReplaceAll(text, "##PERIOD##", ".")
}

// paragraphStats holds paragraph-level statistics.
type paragraphStats struct {
	count        int
	avgSentences float64
}

// analyzeParagraphs performs paragraph-level analysis.
func analyzeParagraphs(text string) paragraphStats {
	stats := paragraphStats{}

	paragraphs := tokenizeParagraphs(text)
	stats.count = len(paragraphs)

	if stats.count == 0 {
		return stats
	}

	totalSentences := 0
	for _, para := range paragraphs {
		sentences := tokenizeSentences(para)
		totalSentences += len(sentences)
	}

	stats.avgSentences = float64(totalSentences) / float64(stats.count)

	return stats
}

// tokenizeParagraphs splits text into paragraphs.
func tokenizeParagraphs(text string) []string {
	// Split on double newlines or more
	paraRegex := regexp.MustCompile(`\n\s*\n`)
	parts := paraRegex.Split(text, -1)

	var paragraphs []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && len(tokenizeWords(part)) > 0 {
			paragraphs = append(paragraphs, part)
		}
	}

	return paragraphs
}

// GetSentences returns all sentences from text.
func GetSentences(text string) []string {
	return tokenizeSentences(text)
}

// GetParagraphs returns all paragraphs from text.
func GetParagraphs(text string) []string {
	return tokenizeParagraphs(text)
}
