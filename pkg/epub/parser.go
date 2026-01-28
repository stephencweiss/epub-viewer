package epub

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"
)

// Parse reads and parses an EPUB file.
func Parse(path string) (*Book, error) {
	// Open the EPUB file (it's a ZIP archive)
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open epub: %w", err)
	}
	defer zr.Close()

	// Find the root OPF file via container.xml
	opfPath, err := findRootFile(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to find root file: %w", err)
	}

	// Read and parse the OPF file
	opfData, err := readFileFromZip(zr, opfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPF file: %w", err)
	}

	pkg, err := parseOPF(opfData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OPF: %w", err)
	}

	// Extract metadata
	title, authors, language, publisher, description, identifier, date := extractMetadata(pkg)

	// Build manifest and spine
	manifest := buildManifest(pkg)
	spine := buildSpine(pkg)

	// Determine the base path for content files (relative to OPF location)
	basePath := filepath.Dir(opfPath)

	// Extract chapters based on spine order
	chapters, err := extractChapters(zr, manifest, spine, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract chapters: %w", err)
	}

	book := &Book{
		Title:       title,
		Authors:     authors,
		Language:    language,
		Publisher:   publisher,
		Description: description,
		Identifier:  identifier,
		Date:        date,
		Chapters:    chapters,
		basePath:    basePath,
	}

	return book, nil
}

// extractChapters reads and processes all content documents in spine order.
func extractChapters(zr *zip.ReadCloser, manifest Manifest, spine Spine, basePath string) ([]Chapter, error) {
	var chapters []Chapter

	for i, itemRef := range spine.ItemRefs {
		// Find the manifest item
		item := manifestItemByID(manifest, itemRef.IDRef)
		if item == nil {
			continue
		}

		// Only process content documents
		if !isContentType(item.MediaType) {
			continue
		}

		// Skip NCX files (navigation)
		if strings.HasSuffix(strings.ToLower(item.Href), ".ncx") {
			continue
		}

		// Build the full path to the content file
		contentPath := item.Href
		if basePath != "" && basePath != "." {
			contentPath = filepath.Join(basePath, item.Href)
		}
		// Normalize path separators for zip
		contentPath = filepath.ToSlash(contentPath)

		// Read the content file
		content, err := readFileFromZip(zr, contentPath)
		if err != nil {
			// Try without base path
			content, err = readFileFromZip(zr, item.Href)
			if err != nil {
				// Skip files that can't be read
				continue
			}
		}

		contentStr := string(content)

		// Extract title and plain text
		title := extractChapterTitle(contentStr)
		text := extractText(contentStr)

		chapter := Chapter{
			ID:      item.ID,
			Title:   title,
			Href:    item.Href,
			Content: contentStr,
			Text:    text,
			Order:   i,
		}

		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

// FullText returns the complete plain text of the book.
func (b *Book) FullText() string {
	var parts []string
	for _, ch := range b.Chapters {
		if ch.Text != "" {
			parts = append(parts, ch.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// Author returns the primary author or empty string if none.
func (b *Book) Author() string {
	if len(b.Authors) > 0 {
		return b.Authors[0]
	}
	return ""
}

// ChapterCount returns the number of chapters.
func (b *Book) ChapterCount() int {
	return len(b.Chapters)
}
