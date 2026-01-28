package filter

import (
	"epub-reader/pkg/epub"
	"epub-reader/pkg/storage"
)

// FilteredChapter contains a chapter with its classification result.
type FilteredChapter struct {
	epub.Chapter
	Classification *ClassificationResult
}

// FilteredBook contains the full book with filtered chapters.
type FilteredBook struct {
	*epub.Book
	FilteredChapters []FilteredChapter
	AllowedChapters  []epub.Chapter
	DeniedChapters   []epub.Chapter
}

// ParseAndFilter parses an EPUB and classifies all chapters.
// It returns the full book with classification results for each chapter.
func ParseAndFilter(path string, classifier *Classifier) (*FilteredBook, error) {
	book, err := epub.Parse(path)
	if err != nil {
		return nil, err
	}

	return FilterBook(book, classifier), nil
}

// FilterBook applies classification to all chapters in a parsed book.
func FilterBook(book *epub.Book, classifier *Classifier) *FilteredBook {
	fb := &FilteredBook{
		Book:             book,
		FilteredChapters: make([]FilteredChapter, 0, len(book.Chapters)),
		AllowedChapters:  make([]epub.Chapter, 0),
		DeniedChapters:   make([]epub.Chapter, 0),
	}

	for _, chapter := range book.Chapters {
		snippet := GetSnippet(chapter.Text, 500)
		result := classifier.Classify(chapter.Href, chapter.EpubType, snippet)

		fc := FilteredChapter{
			Chapter:        chapter,
			Classification: result,
		}
		fb.FilteredChapters = append(fb.FilteredChapters, fc)

		if result.Decision == DecisionAllow {
			fb.AllowedChapters = append(fb.AllowedChapters, chapter)
		} else {
			fb.DeniedChapters = append(fb.DeniedChapters, chapter)
		}
	}

	return fb
}

// AllowedText returns the concatenated text of all allowed chapters.
func (fb *FilteredBook) AllowedText() string {
	var text string
	for i, chapter := range fb.AllowedChapters {
		if chapter.Text != "" {
			if i > 0 {
				text += "\n\n"
			}
			text += chapter.Text
		}
	}
	return text
}

// StoreFilteredSections saves all classified sections to the database.
func StoreFilteredSections(fb *FilteredBook, bookID int64, store *storage.SQLiteStore) error {
	for _, fc := range fb.FilteredChapters {
		section := &storage.Section{
			BookID:      bookID,
			FileName:    fc.Href,
			EpubType:    fc.EpubType,
			Title:       fc.Title,
			TextContent: fc.Text,
			Status:      fc.Classification.Decision,
			Order:       fc.Order,
		}

		if err := store.SaveSection(section); err != nil {
			return err
		}
	}
	return nil
}

// Summary returns a summary of the filtering results.
type FilterSummary struct {
	TotalChapters   int
	AllowedCount    int
	DeniedCount     int
	NeedsLLMReview  int
	AllowedBySource map[string]int
	DeniedBySource  map[string]int
}

// GetSummary generates a summary of the filtering results.
func (fb *FilteredBook) GetSummary() *FilterSummary {
	summary := &FilterSummary{
		TotalChapters:   len(fb.FilteredChapters),
		AllowedCount:    len(fb.AllowedChapters),
		DeniedCount:     len(fb.DeniedChapters),
		AllowedBySource: make(map[string]int),
		DeniedBySource:  make(map[string]int),
	}

	for _, fc := range fb.FilteredChapters {
		if fc.Classification.NeedsLLM {
			summary.NeedsLLMReview++
		}

		if fc.Classification.Decision == DecisionAllow {
			summary.AllowedBySource[fc.Classification.Source]++
		} else {
			summary.DeniedBySource[fc.Classification.Source]++
		}
	}

	return summary
}
