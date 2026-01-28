package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
)

// containerXML represents the META-INF/container.xml file.
type containerXML struct {
	XMLName   xml.Name `xml:"container"`
	Version   string   `xml:"version,attr"`
	RootFiles []struct {
		FullPath  string `xml:"full-path,attr"`
		MediaType string `xml:"media-type,attr"`
	} `xml:"rootfiles>rootfile"`
}

// findRootFile locates the OPF file path from container.xml.
func findRootFile(zr *zip.ReadCloser) (string, error) {
	containerFile, err := findFileInZip(zr, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("container.xml not found: %w", err)
	}

	rc, err := containerFile.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open container.xml: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read container.xml: %w", err)
	}

	var container containerXML
	if err := xml.Unmarshal(data, &container); err != nil {
		return "", fmt.Errorf("failed to parse container.xml: %w", err)
	}

	if len(container.RootFiles) == 0 {
		return "", fmt.Errorf("no rootfile found in container.xml")
	}

	// Find the OPF file (application/oebps-package+xml)
	for _, rf := range container.RootFiles {
		if rf.MediaType == "application/oebps-package+xml" {
			return rf.FullPath, nil
		}
	}

	// Fallback to the first rootfile
	return container.RootFiles[0].FullPath, nil
}

// findFileInZip locates a file in the zip archive.
func findFileInZip(zr *zip.ReadCloser, name string) (*zip.File, error) {
	for _, f := range zr.File {
		if f.Name == name || filepath.Clean(f.Name) == filepath.Clean(name) {
			return f, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// readFileFromZip reads the contents of a file in the zip archive.
func readFileFromZip(zr *zip.ReadCloser, name string) ([]byte, error) {
	f, err := findFileInZip(zr, name)
	if err != nil {
		return nil, err
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", name, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}
