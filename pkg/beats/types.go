package beats

// SceneBreakType represents the type of scene break detected.
type SceneBreakType string

const (
	BreakBlankLines SceneBreakType = "blank_lines" // Multiple blank lines
	BreakAsterisks  SceneBreakType = "asterisks"   // ***, * * *, etc.
	BreakDashes     SceneBreakType = "dashes"      // ---, - - -, etc.
	BreakHashes     SceneBreakType = "hashes"      // ###, # # #, etc.
	BreakHorizontal SceneBreakType = "horizontal"  // <hr> or similar HTML
	BreakChapter    SceneBreakType = "chapter"     // Chapter boundary (no internal breaks)
)

// Scene represents a detected scene within a book.
type Scene struct {
	Index       int            // Global 0-indexed position across all chapters
	ChapterID   string         // Chapter this scene belongs to
	ChapterNum  int            // Chapter number (1-indexed)
	StartOffset int            // Character offset start within chapter
	EndOffset   int            // Character offset end within chapter
	Text        string         // Scene content (plain text)
	WordCount   int            // Number of words in the scene
	BreakType   SceneBreakType // How this scene was delimited
}

// BeatAnalysisResult contains the LLM-extracted beat data.
type BeatAnalysisResult struct {
	Summary          string `json:"summary"`
	Conflict         string `json:"conflict"`
	Choice           string `json:"choice"`
	Consequence      string `json:"consequence"`
	PerspectiveShift string `json:"perspective_shift,omitempty"`
}
