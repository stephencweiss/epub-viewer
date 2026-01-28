package analysis

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// wordStats holds word-level statistics.
type wordStats struct {
	total          int
	unique         int
	frequencies    map[string]int
	typeTokenRatio float64
	hapaxCount     int
	avgWordLength  float64
}

// analyzeWords performs word-level analysis on text.
func analyzeWords(text string) wordStats {
	stats := wordStats{
		frequencies: make(map[string]int),
	}

	words := tokenizeWords(text)
	stats.total = len(words)

	if stats.total == 0 {
		return stats
	}

	// Count frequencies and calculate average word length
	totalLength := 0
	for _, word := range words {
		lower := strings.ToLower(word)
		stats.frequencies[lower]++
		totalLength += len(word)
	}

	stats.unique = len(stats.frequencies)
	stats.avgWordLength = float64(totalLength) / float64(stats.total)

	// Type-token ratio (vocabulary richness)
	stats.typeTokenRatio = float64(stats.unique) / float64(stats.total)

	// Hapax legomena (words appearing only once)
	for _, count := range stats.frequencies {
		if count == 1 {
			stats.hapaxCount++
		}
	}

	return stats
}

// tokenizeWords extracts words from text.
func tokenizeWords(text string) []string {
	// Match sequences of letters and apostrophes (for contractions)
	wordRegex := regexp.MustCompile(`[\p{L}]+(?:'[\p{L}]+)?`)
	return wordRegex.FindAllString(text, -1)
}

// getTopWords returns the N most frequent words.
func getTopWords(frequencies map[string]int, n int) []WordFreq {
	var words []WordFreq
	for word, count := range frequencies {
		// Skip very short words and common stop words
		if len(word) > 2 && !isStopWord(word) {
			words = append(words, WordFreq{Word: word, Count: count})
		}
	}

	// Sort by frequency (descending)
	sort.Slice(words, func(i, j int) bool {
		return words[i].Count > words[j].Count
	})

	if n > len(words) {
		n = len(words)
	}
	return words[:n]
}

// isStopWord checks if a word is a common stop word.
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"as": true, "is": true, "was": true, "are": true, "were": true,
		"been": true, "be": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
		"i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "me": true, "him": true, "her": true,
		"us": true, "them": true, "my": true, "your": true, "his": true,
		"its": true, "our": true, "their": true, "what": true, "which": true,
		"who": true, "whom": true, "whose": true, "when": true, "where": true,
		"why": true, "how": true, "all": true, "each": true, "every": true,
		"both": true, "few": true, "more": true, "most": true, "other": true,
		"some": true, "such": true, "no": true, "not": true, "only": true,
		"same": true, "so": true, "than": true, "too": true, "very": true,
		"just": true, "into": true, "over": true, "after": true, "before": true,
		"then": true, "once": true, "here": true, "there": true, "about": true,
		"up": true, "down": true, "out": true, "off": true, "again": true,
	}
	return stopWords[word]
}

// countSyllables estimates the number of syllables in a word.
func countSyllables(word string) int {
	word = strings.ToLower(word)
	if len(word) == 0 {
		return 0
	}

	// Simple syllable counting heuristic
	count := 0
	prevVowel := false

	for _, r := range word {
		isVowel := unicode.In(r, unicode.Ll, unicode.Lu) &&
			(r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u' || r == 'y')

		if isVowel && !prevVowel {
			count++
		}
		prevVowel = isVowel
	}

	// Handle silent 'e'
	if strings.HasSuffix(word, "e") && count > 1 {
		count--
	}

	// Handle special endings
	if strings.HasSuffix(word, "le") && len(word) > 2 {
		prevChar := word[len(word)-3]
		if !isVowelChar(rune(prevChar)) {
			count++
		}
	}

	if count == 0 {
		count = 1
	}

	return count
}

// isVowelChar checks if a rune is a vowel.
func isVowelChar(r rune) bool {
	r = unicode.ToLower(r)
	return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u'
}

// GetWordFrequencies returns word frequencies without filtering.
func GetWordFrequencies(text string) map[string]int {
	stats := analyzeWords(text)
	return stats.frequencies
}

// GetUniqueWords returns a list of all unique words.
func GetUniqueWords(text string) []string {
	stats := analyzeWords(text)
	words := make([]string, 0, len(stats.frequencies))
	for word := range stats.frequencies {
		words = append(words, word)
	}
	sort.Strings(words)
	return words
}
