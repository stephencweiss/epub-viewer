package beats

import (
	"testing"

	"epub-reader/pkg/epub"
)

func TestDetectScenes_BlankLines(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "First scene with some content.\n\n\nSecond scene after blank lines.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}

	if scenes[0].BreakType != BreakBlankLines {
		t.Errorf("expected first scene break type to be blank_lines, got %s", scenes[0].BreakType)
	}

	if scenes[0].Text != "First scene with some content." {
		t.Errorf("unexpected first scene text: %q", scenes[0].Text)
	}

	if scenes[1].Text != "Second scene after blank lines." {
		t.Errorf("unexpected second scene text: %q", scenes[1].Text)
	}
}

func TestDetectScenes_Asterisks(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "triple asterisks",
			text:     "First scene.\n***\nSecond scene.",
			expected: 2,
		},
		{
			name:     "spaced asterisks",
			text:     "First scene.\n* * *\nSecond scene.",
			expected: 2,
		},
		{
			name:     "asterisks with whitespace",
			text:     "First scene.\n  ***  \nSecond scene.",
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			book := &epub.Book{
				Chapters: []epub.Chapter{{ID: "ch1", Text: tc.text}},
			}

			scenes := DetectScenes(book)
			if len(scenes) != tc.expected {
				t.Errorf("expected %d scenes, got %d", tc.expected, len(scenes))
			}

			if len(scenes) >= 1 && scenes[0].BreakType != BreakAsterisks {
				t.Errorf("expected break type asterisks, got %s", scenes[0].BreakType)
			}
		})
	}
}

func TestDetectScenes_Dashes(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "triple dashes",
			text:     "First scene.\n---\nSecond scene.",
			expected: 2,
		},
		{
			name:     "spaced dashes",
			text:     "First scene.\n- - -\nSecond scene.",
			expected: 2,
		},
		{
			name:     "em dashes",
			text:     "First scene.\n———\nSecond scene.",
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			book := &epub.Book{
				Chapters: []epub.Chapter{{ID: "ch1", Text: tc.text}},
			}

			scenes := DetectScenes(book)
			if len(scenes) != tc.expected {
				t.Errorf("expected %d scenes, got %d", tc.expected, len(scenes))
			}

			if len(scenes) >= 1 && scenes[0].BreakType != BreakDashes {
				t.Errorf("expected break type dashes, got %s", scenes[0].BreakType)
			}
		})
	}
}

func TestDetectScenes_Mixed(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "Scene one.\n***\nScene two.\n\n\nScene three.\n---\nScene four.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 4 {
		t.Fatalf("expected 4 scenes, got %d", len(scenes))
	}

	// Verify break types in order
	expectedBreaks := []SceneBreakType{BreakAsterisks, BreakBlankLines, BreakDashes, BreakChapter}
	for i, expected := range expectedBreaks {
		if scenes[i].BreakType != expected {
			t.Errorf("scene %d: expected break type %s, got %s", i, expected, scenes[i].BreakType)
		}
	}
}

func TestDetectScenes_NoBreaks(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "This is a single chapter with no scene breaks. It should be treated as one scene.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 1 {
		t.Fatalf("expected 1 scene, got %d", len(scenes))
	}

	if scenes[0].BreakType != BreakChapter {
		t.Errorf("expected break type chapter, got %s", scenes[0].BreakType)
	}

	if scenes[0].ChapterID != "ch1" {
		t.Errorf("expected chapter ID ch1, got %s", scenes[0].ChapterID)
	}
}

func TestDetectScenes_MultipleChapters(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{ID: "ch1", Text: "Chapter 1 scene 1.\n***\nChapter 1 scene 2."},
			{ID: "ch2", Text: "Chapter 2 scene 1.\n---\nChapter 2 scene 2."},
			{ID: "ch3", Text: "Chapter 3 only one scene."},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 5 {
		t.Fatalf("expected 5 scenes, got %d", len(scenes))
	}

	// Verify global indexing
	for i, scene := range scenes {
		if scene.Index != i {
			t.Errorf("scene %d: expected Index %d, got %d", i, i, scene.Index)
		}
	}

	// Verify chapter distribution
	expectedChapterNums := []int{1, 1, 2, 2, 3}
	for i, expected := range expectedChapterNums {
		if scenes[i].ChapterNum != expected {
			t.Errorf("scene %d: expected ChapterNum %d, got %d", i, expected, scenes[i].ChapterNum)
		}
	}
}

func TestDetectScenes_EmptyChapter(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{ID: "ch1", Text: "Content in chapter 1."},
			{ID: "ch2", Text: "   "},   // Whitespace only
			{ID: "ch3", Text: ""},      // Empty
			{ID: "ch4", Text: "Content in chapter 4."},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes (skipping empty chapters), got %d", len(scenes))
	}

	// Verify only non-empty chapters are included
	if scenes[0].ChapterID != "ch1" {
		t.Errorf("expected first scene from ch1, got %s", scenes[0].ChapterID)
	}
	if scenes[1].ChapterID != "ch4" {
		t.Errorf("expected second scene from ch4, got %s", scenes[1].ChapterID)
	}
}

func TestDetectScenes_WordCount(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "One two three four five.\n***\nSix seven eight.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}

	if scenes[0].WordCount != 5 {
		t.Errorf("expected first scene word count 5, got %d", scenes[0].WordCount)
	}

	if scenes[1].WordCount != 3 {
		t.Errorf("expected second scene word count 3, got %d", scenes[1].WordCount)
	}
}

func TestDetectScenes_Hashes(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "First scene.\n###\nSecond scene.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}

	if scenes[0].BreakType != BreakHashes {
		t.Errorf("expected break type hashes, got %s", scenes[0].BreakType)
	}
}

func TestDetectScenes_HTMLHorizontalRule(t *testing.T) {
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "First scene.<hr/>Second scene.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}

	if scenes[0].BreakType != BreakHorizontal {
		t.Errorf("expected break type horizontal, got %s", scenes[0].BreakType)
	}
}

func TestDetectScenes_OverlappingBreaks(t *testing.T) {
	// When asterisks appear within blank lines, asterisks should take priority
	book := &epub.Book{
		Chapters: []epub.Chapter{
			{
				ID:   "ch1",
				Text: "First scene.\n\n***\n\nSecond scene.",
			},
		},
	}

	scenes := DetectScenes(book)

	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}

	// Asterisks should take priority over blank lines
	if scenes[0].BreakType != BreakAsterisks {
		t.Errorf("expected break type asterisks (priority), got %s", scenes[0].BreakType)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"   ", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one\ntwo\nthree", 3},
	}

	for _, tc := range tests {
		got := countWords(tc.text)
		if got != tc.expected {
			t.Errorf("countWords(%q) = %d, want %d", tc.text, got, tc.expected)
		}
	}
}
