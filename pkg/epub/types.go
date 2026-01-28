package epub

import "time"

// Book represents a parsed EPUB file.
type Book struct {
	// Metadata
	Title       string
	Authors     []string
	Language    string
	Publisher   string
	Description string
	Date        time.Time
	Identifier  string

	// Content
	Chapters []Chapter

	// Internal
	basePath string
}

// Chapter represents a single content document in the EPUB.
type Chapter struct {
	ID       string
	Title    string
	Href     string
	Content  string // Raw XHTML content
	Text     string // Plain text (HTML stripped)
	EpubType string // epub:type attribute (e.g., "chapter", "frontmatter")
	Order    int
}

// Manifest represents all items in the EPUB package.
type Manifest struct {
	Items []ManifestItem
}

// ManifestItem represents a single resource in the EPUB.
type ManifestItem struct {
	ID         string
	Href       string
	MediaType  string
	Properties string // EPUB 3 properties (e.g., "nav", "cover-image")
}

// Spine represents the reading order of the EPUB.
type Spine struct {
	ItemRefs []SpineItemRef
}

// SpineItemRef references a manifest item in reading order.
type SpineItemRef struct {
	IDRef  string
	Linear bool
}
