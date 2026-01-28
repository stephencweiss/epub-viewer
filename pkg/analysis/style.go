package analysis

import (
	"regexp"
	"strings"
)

// styleStats holds style-related statistics.
type styleStats struct {
	readability   float64
	dialogueRatio float64
	syllables     int
	avgSyllables  float64
}

// analyzeStyle performs style analysis on text.
func analyzeStyle(text string, wordStats wordStats, sentenceStats sentenceStats) styleStats {
	stats := styleStats{}

	// Calculate total syllables
	words := tokenizeWords(text)
	for _, word := range words {
		stats.syllables += countSyllables(word)
	}

	if wordStats.total > 0 {
		stats.avgSyllables = float64(stats.syllables) / float64(wordStats.total)
	}

	// Flesch-Kincaid Reading Ease
	stats.readability = calculateFleschKincaid(
		wordStats.total,
		sentenceStats.count,
		stats.syllables,
	)

	// Dialogue ratio
	stats.dialogueRatio = calculateDialogueRatio(text)

	return stats
}

// calculateFleschKincaid calculates the Flesch-Kincaid Reading Ease score.
// Higher scores indicate easier reading:
// 90-100: 5th grade (very easy)
// 80-90: 6th grade (easy)
// 70-80: 7th grade (fairly easy)
// 60-70: 8th-9th grade (plain English)
// 50-60: 10th-12th grade (fairly difficult)
// 30-50: College level (difficult)
// 0-30: Graduate level (very difficult)
func calculateFleschKincaid(words, sentences, syllables int) float64 {
	if words == 0 || sentences == 0 {
		return 0
	}

	wordsPerSentence := float64(words) / float64(sentences)
	syllablesPerWord := float64(syllables) / float64(words)

	// Flesch Reading Ease formula
	score := 206.835 - (1.015 * wordsPerSentence) - (84.6 * syllablesPerWord)

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// calculateDialogueRatio estimates the ratio of dialogue to total text.
func calculateDialogueRatio(text string) float64 {
	// Count text within quotes
	dialoguePatterns := []string{
		`"[^"]*"`,      // Double quotes
		`'[^']*'`,      // Single quotes
		`"[^"]*"`,      // Smart double quotes
		`'[^']*'`,      // Smart single quotes
		`«[^»]*»`,      // Guillemets
	}

	totalDialogue := 0
	for _, pattern := range dialoguePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(text, -1)
		for _, match := range matches {
			words := tokenizeWords(match)
			totalDialogue += len(words)
		}
	}

	totalWords := len(tokenizeWords(text))
	if totalWords == 0 {
		return 0
	}

	return float64(totalDialogue) / float64(totalWords)
}

// ReadabilityLevel returns a human-readable description of the readability score.
func ReadabilityLevel(score float64) string {
	switch {
	case score >= 90:
		return "Very Easy (5th grade)"
	case score >= 80:
		return "Easy (6th grade)"
	case score >= 70:
		return "Fairly Easy (7th grade)"
	case score >= 60:
		return "Standard (8th-9th grade)"
	case score >= 50:
		return "Fairly Difficult (10th-12th grade)"
	case score >= 30:
		return "Difficult (College level)"
	default:
		return "Very Difficult (Graduate level)"
	}
}

// AnalyzeVocabularyComplexity provides additional vocabulary metrics.
func AnalyzeVocabularyComplexity(text string) map[string]float64 {
	words := tokenizeWords(text)
	if len(words) == 0 {
		return nil
	}

	metrics := make(map[string]float64)

	// Calculate percentages by word length
	shortWords := 0   // 1-4 letters
	mediumWords := 0  // 5-8 letters
	longWords := 0    // 9+ letters

	for _, word := range words {
		l := len(word)
		switch {
		case l <= 4:
			shortWords++
		case l <= 8:
			mediumWords++
		default:
			longWords++
		}
	}

	total := float64(len(words))
	metrics["short_word_ratio"] = float64(shortWords) / total
	metrics["medium_word_ratio"] = float64(mediumWords) / total
	metrics["long_word_ratio"] = float64(longWords) / total

	// Polysyllabic words (3+ syllables)
	polysyllabic := 0
	for _, word := range words {
		if countSyllables(word) >= 3 {
			polysyllabic++
		}
	}
	metrics["polysyllabic_ratio"] = float64(polysyllabic) / total

	return metrics
}

// DetectPOVStyle attempts to detect the narrative point of view.
func DetectPOVStyle(text string) string {
	textLower := strings.ToLower(text)
	words := tokenizeWords(textLower)

	// Count pronouns
	firstPerson := 0  // I, me, my, we, us, our
	secondPerson := 0 // You, your
	thirdPerson := 0  // He, she, they, his, her, their

	firstPersonWords := map[string]bool{"i": true, "me": true, "my": true, "mine": true, "we": true, "us": true, "our": true, "ours": true}
	secondPersonWords := map[string]bool{"you": true, "your": true, "yours": true}
	thirdPersonWords := map[string]bool{"he": true, "she": true, "him": true, "her": true, "his": true, "hers": true, "they": true, "them": true, "their": true, "theirs": true}

	for _, word := range words {
		if firstPersonWords[word] {
			firstPerson++
		}
		if secondPersonWords[word] {
			secondPerson++
		}
		if thirdPersonWords[word] {
			thirdPerson++
		}
	}

	// Determine dominant POV
	total := firstPerson + secondPerson + thirdPerson
	if total == 0 {
		return "Unknown"
	}

	if firstPerson > secondPerson && firstPerson > thirdPerson {
		if float64(firstPerson)/float64(total) > 0.5 {
			return "First Person"
		}
	}
	if secondPerson > firstPerson && secondPerson > thirdPerson {
		if float64(secondPerson)/float64(total) > 0.5 {
			return "Second Person"
		}
	}
	if thirdPerson > firstPerson && thirdPerson > secondPerson {
		return "Third Person"
	}

	return "Mixed"
}
