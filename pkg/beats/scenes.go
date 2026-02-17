package beats

import (
	"regexp"
	"sort"
	"strings"

	"epub-reader/pkg/epub"
)

// Scene break detection patterns
var (
	// Multiple blank lines (3+ newlines = 2+ blank lines)
	blankLinePattern = regexp.MustCompile(`\n\s*\n\s*\n`)

	// Asterisk separators: ***, * * *, ⁂, etc.
	asteriskPattern = regexp.MustCompile(`(?m)^\s*[\*⁂]{3,}\s*$|^\s*\*\s+\*\s+\*\s*$`)

	// Dash separators: ---, - - -, etc.
	dashPattern = regexp.MustCompile(`(?m)^\s*[-–—]{3,}\s*$|^\s*-\s+-\s+-\s*$`)

	// Hash separators: ###, # # #, etc.
	hashPattern = regexp.MustCompile(`(?m)^\s*[#]{3,}\s*$|^\s*#\s+#\s+#\s*$`)

	// HTML horizontal rules (in case text still has some HTML artifacts)
	hrPattern = regexp.MustCompile(`(?i)<hr\s*/?>`)
)

// sceneBreak represents a detected break point in text.
type sceneBreak struct {
	offset    int
	length    int
	breakType SceneBreakType
}

// DetectScenes extracts scenes from a book based on scene breaks.
func DetectScenes(book *epub.Book) []Scene {
	var scenes []Scene
	globalIndex := 0

	for chapterNum, chapter := range book.Chapters {
		chapterScenes := detectScenesInChapter(chapter, chapterNum+1)

		// Re-index scenes globally
		for i := range chapterScenes {
			chapterScenes[i].Index = globalIndex
			globalIndex++
		}

		scenes = append(scenes, chapterScenes...)
	}

	return scenes
}

// detectScenesInChapter finds scene breaks within a single chapter.
func detectScenesInChapter(chapter epub.Chapter, chapterNum int) []Scene {
	text := chapter.Text
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// Find all scene breaks
	breaks := findSceneBreaks(text)

	// If no breaks found, treat entire chapter as one scene
	if len(breaks) == 0 {
		return []Scene{{
			ChapterID:   chapter.ID,
			ChapterNum:  chapterNum,
			StartOffset: 0,
			EndOffset:   len(text),
			Text:        strings.TrimSpace(text),
			WordCount:   countWords(text),
			BreakType:   BreakChapter,
		}}
	}

	// Split chapter into scenes based on breaks
	return splitByBreaks(chapter, chapterNum, text, breaks)
}

// findSceneBreaks locates all potential scene breaks in the text.
func findSceneBreaks(text string) []sceneBreak {
	var breaks []sceneBreak

	// Check each pattern (order matters for priority)
	patterns := []struct {
		re        *regexp.Regexp
		breakType SceneBreakType
	}{
		{asteriskPattern, BreakAsterisks},
		{dashPattern, BreakDashes},
		{hashPattern, BreakHashes},
		{hrPattern, BreakHorizontal},
		{blankLinePattern, BreakBlankLines},
	}

	for _, p := range patterns {
		matches := p.re.FindAllStringIndex(text, -1)
		for _, match := range matches {
			breaks = append(breaks, sceneBreak{
				offset:    match[0],
				length:    match[1] - match[0],
				breakType: p.breakType,
			})
		}
	}

	// Sort by offset and deduplicate overlapping breaks
	return deduplicateBreaks(breaks)
}

// deduplicateBreaks sorts breaks by offset and removes overlapping ones.
// When breaks overlap, keeps the more specific one (non-blank-line takes priority).
func deduplicateBreaks(breaks []sceneBreak) []sceneBreak {
	if len(breaks) == 0 {
		return breaks
	}

	// Sort by offset
	sort.Slice(breaks, func(i, j int) bool {
		return breaks[i].offset < breaks[j].offset
	})

	// Remove overlapping breaks, keeping more specific types
	result := make([]sceneBreak, 0, len(breaks))
	for _, b := range breaks {
		if len(result) == 0 {
			result = append(result, b)
			continue
		}

		last := &result[len(result)-1]
		// Check for overlap
		if b.offset < last.offset+last.length {
			// Overlapping - keep the more specific one (non-blank-line)
			if last.breakType == BreakBlankLines && b.breakType != BreakBlankLines {
				*last = b
			}
			// Otherwise keep existing (first one found)
			continue
		}

		result = append(result, b)
	}

	return result
}

// splitByBreaks divides chapter text into scenes using detected breaks.
func splitByBreaks(chapter epub.Chapter, chapterNum int, text string, breaks []sceneBreak) []Scene {
	var scenes []Scene
	currentStart := 0

	for _, brk := range breaks {
		// Extract text before this break
		sceneText := text[currentStart:brk.offset]
		sceneText = strings.TrimSpace(sceneText)

		if sceneText != "" {
			scenes = append(scenes, Scene{
				ChapterID:   chapter.ID,
				ChapterNum:  chapterNum,
				StartOffset: currentStart,
				EndOffset:   brk.offset,
				Text:        sceneText,
				WordCount:   countWords(sceneText),
				BreakType:   brk.breakType,
			})
		}

		// Move past the break
		currentStart = brk.offset + brk.length
	}

	// Handle text after the last break
	if currentStart < len(text) {
		sceneText := text[currentStart:]
		sceneText = strings.TrimSpace(sceneText)

		if sceneText != "" {
			scenes = append(scenes, Scene{
				ChapterID:   chapter.ID,
				ChapterNum:  chapterNum,
				StartOffset: currentStart,
				EndOffset:   len(text),
				Text:        sceneText,
				WordCount:   countWords(sceneText),
				BreakType:   BreakChapter, // Last segment uses chapter as break type
			})
		}
	}

	return scenes
}

// countWords returns the number of words in text.
func countWords(text string) int {
	return len(strings.Fields(text))
}
