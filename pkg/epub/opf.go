package epub

import (
	"encoding/xml"
	"strings"
	"time"
)

// packageOPF represents the OPF package document.
type packageOPF struct {
	XMLName  xml.Name    `xml:"package"`
	Version  string      `xml:"version,attr"`
	Metadata metadataOPF `xml:"metadata"`
	Manifest manifestOPF `xml:"manifest"`
	Spine    spineOPF    `xml:"spine"`
}

type metadataOPF struct {
	Titles      []string     `xml:"title"`
	Creators    []creatorOPF `xml:"creator"`
	Languages   []string     `xml:"language"`
	Publishers  []string     `xml:"publisher"`
	Dates       []string     `xml:"date"`
	Identifiers []string     `xml:"identifier"`
	Description []string     `xml:"description"`
}

type creatorOPF struct {
	Name   string `xml:",chardata"`
	Role   string `xml:"role,attr"`
	FileAs string `xml:"file-as,attr"`
}

type manifestOPF struct {
	Items []manifestItemOPF `xml:"item"`
}

type manifestItemOPF struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type spineOPF struct {
	Toc      string         `xml:"toc,attr"`
	ItemRefs []spineItemRef `xml:"itemref"`
}

type spineItemRef struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

// parseOPF parses the OPF package document.
func parseOPF(data []byte) (*packageOPF, error) {
	var pkg packageOPF
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

// extractMetadata converts OPF metadata to Book fields.
func extractMetadata(pkg *packageOPF) (title string, authors []string, language, publisher, description, identifier string, date time.Time) {
	// Title
	if len(pkg.Metadata.Titles) > 0 {
		title = pkg.Metadata.Titles[0]
	}

	// Authors - prefer creators with role "aut", fallback to all creators
	for _, c := range pkg.Metadata.Creators {
		role := strings.ToLower(c.Role)
		if role == "" || role == "aut" || role == "author" {
			if c.Name != "" {
				authors = append(authors, strings.TrimSpace(c.Name))
			}
		}
	}
	// If no authors found with role filtering, include all creators
	if len(authors) == 0 {
		for _, c := range pkg.Metadata.Creators {
			if c.Name != "" {
				authors = append(authors, strings.TrimSpace(c.Name))
			}
		}
	}

	// Language
	if len(pkg.Metadata.Languages) > 0 {
		language = pkg.Metadata.Languages[0]
	}

	// Publisher
	if len(pkg.Metadata.Publishers) > 0 {
		publisher = pkg.Metadata.Publishers[0]
	}

	// Description
	if len(pkg.Metadata.Description) > 0 {
		description = pkg.Metadata.Description[0]
	}

	// Identifier
	if len(pkg.Metadata.Identifiers) > 0 {
		identifier = pkg.Metadata.Identifiers[0]
	}

	// Date - try to parse various formats
	if len(pkg.Metadata.Dates) > 0 {
		dateStr := pkg.Metadata.Dates[0]
		formats := []string{
			"2006-01-02",
			"2006-01",
			"2006",
			time.RFC3339,
			"January 2, 2006",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				date = t
				break
			}
		}
	}

	return
}

// buildManifest converts OPF manifest to our Manifest type.
func buildManifest(pkg *packageOPF) Manifest {
	m := Manifest{
		Items: make([]ManifestItem, len(pkg.Manifest.Items)),
	}
	for i, item := range pkg.Manifest.Items {
		m.Items[i] = ManifestItem{
			ID:        item.ID,
			Href:      item.Href,
			MediaType: item.MediaType,
		}
	}
	return m
}

// buildSpine converts OPF spine to our Spine type.
func buildSpine(pkg *packageOPF) Spine {
	s := Spine{
		ItemRefs: make([]SpineItemRef, len(pkg.Spine.ItemRefs)),
	}
	for i, ref := range pkg.Spine.ItemRefs {
		s.ItemRefs[i] = SpineItemRef{
			IDRef:  ref.IDRef,
			Linear: ref.Linear != "no",
		}
	}
	return s
}

// manifestItemByID finds a manifest item by its ID.
func manifestItemByID(manifest Manifest, id string) *ManifestItem {
	for _, item := range manifest.Items {
		if item.ID == id {
			return &item
		}
	}
	return nil
}
