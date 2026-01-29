package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MetadataEdit contains the metadata fields to update.
type MetadataEdit struct {
	Title     *string
	Author    *string
	Language  *string
	Publisher *string
}

// ModifyMetadata creates a new EPUB file with modified metadata.
// It copies the original EPUB, updating only the metadata in the OPF file.
func ModifyMetadata(inputPath, outputPath string, edit MetadataEdit) error {
	// Open the original EPUB
	zr, err := zip.OpenReader(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open epub: %w", err)
	}
	defer zr.Close()

	// Find the OPF path
	opfPath, err := findRootFile(zr)
	if err != nil {
		return fmt.Errorf("failed to find root file: %w", err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	// Copy all files, modifying the OPF
	for _, f := range zr.File {
		if f.Name == opfPath {
			// Read and modify OPF
			content, err := readZipFile(f)
			if err != nil {
				return fmt.Errorf("failed to read OPF: %w", err)
			}

			modified, err := modifyOPFMetadata(content, edit)
			if err != nil {
				return fmt.Errorf("failed to modify OPF: %w", err)
			}

			// Write modified OPF
			w, err := zw.Create(f.Name)
			if err != nil {
				return fmt.Errorf("failed to create OPF in output: %w", err)
			}
			if _, err := w.Write(modified); err != nil {
				return fmt.Errorf("failed to write OPF: %w", err)
			}
		} else {
			// Copy file unchanged
			if err := copyZipFile(zw, f); err != nil {
				return fmt.Errorf("failed to copy %s: %w", f.Name, err)
			}
		}
	}

	return nil
}

// readZipFile reads the contents of a zip file entry.
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// copyZipFile copies a file from one zip to another.
func copyZipFile(zw *zip.Writer, f *zip.File) error {
	// Read file content first
	rc, err := f.Open()
	if err != nil {
		return err
	}
	content, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return err
	}

	// The mimetype file must be stored uncompressed and first in EPUB archives
	if f.Name == "mimetype" || f.Method == zip.Store {
		// Create a new header for stored (uncompressed) file
		header := &zip.FileHeader{
			Name:   f.Name,
			Method: zip.Store,
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err
	}

	// For other files, use default compression
	w, err := zw.Create(f.Name)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

// modifyOPFMetadata updates metadata in the OPF XML content.
// We use careful string manipulation to preserve the original XML structure.
func modifyOPFMetadata(content []byte, edit MetadataEdit) ([]byte, error) {
	result := string(content)

	// Update title
	if edit.Title != nil {
		result = replaceMetadataElement(result, "dc:title", *edit.Title)
		result = replaceMetadataElement(result, "title", *edit.Title)
	}

	// Update creator/author
	if edit.Author != nil {
		result = replaceCreator(result, *edit.Author)
	}

	// Update language
	if edit.Language != nil {
		result = replaceMetadataElement(result, "dc:language", *edit.Language)
		result = replaceMetadataElement(result, "language", *edit.Language)
	}

	// Update publisher
	if edit.Publisher != nil {
		result = replaceMetadataElement(result, "dc:publisher", *edit.Publisher)
		result = replaceMetadataElement(result, "publisher", *edit.Publisher)
	}

	return []byte(result), nil
}

// replaceMetadataElement replaces the content of a metadata element.
func replaceMetadataElement(content, element, newValue string) string {
	// Match <element ...>content</element> or <dc:element ...>content</dc:element>
	// Handle both with and without attributes
	patterns := []string{
		fmt.Sprintf(`(<%s[^>]*>)[^<]*(</\s*%s\s*>)`, regexp.QuoteMeta(element), regexp.QuoteMeta(element)),
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, "${1}"+escapeXML(newValue)+"${2}")
			break
		}
	}

	return content
}

// replaceCreator handles the dc:creator element which may have attributes.
func replaceCreator(content, newValue string) string {
	// Match <dc:creator ...>content</dc:creator> preserving attributes
	patterns := []string{
		`(<dc:creator[^>]*>)[^<]*(</\s*dc:creator\s*>)`,
		`(<creator[^>]*>)[^<]*(</\s*creator\s*>)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, "${1}"+escapeXML(newValue)+"${2}")
		}
	}

	return content
}

// escapeXML escapes special XML characters.
func escapeXML(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// DefaultOutputPath generates the default output path for a modified EPUB.
// e.g., "book.epub" -> "book_edited.epub"
func DefaultOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	return base + "_edited" + ext
}
