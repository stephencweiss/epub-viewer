package epub

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// extractText extracts plain text from XHTML content.
func extractText(xhtmlContent string) string {
	doc, err := html.Parse(strings.NewReader(xhtmlContent))
	if err != nil {
		// Fallback: strip tags with regex
		return stripHTMLTags(xhtmlContent)
	}

	var buf bytes.Buffer
	extractTextFromNode(doc, &buf)

	text := buf.String()

	// Clean up whitespace
	text = normalizeWhitespace(text)

	return text
}

// extractTextFromNode recursively extracts text from HTML nodes.
func extractTextFromNode(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}

	// Add spacing for block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr":
			buf.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextFromNode(c, buf)
	}

	// Add newline after block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			buf.WriteString("\n")
		}
	}
}

// stripHTMLTags removes HTML tags as a fallback.
func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

// normalizeWhitespace cleans up whitespace in text.
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`[ \t]+`)
	s = spaceRe.ReplaceAllString(s, " ")

	// Replace multiple newlines with double newline (paragraph break)
	nlRe := regexp.MustCompile(`\n{3,}`)
	s = nlRe.ReplaceAllString(s, "\n\n")

	// Trim spaces around newlines
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, "\n")

	return strings.TrimSpace(s)
}

// extractChapterTitle attempts to extract a title from XHTML content.
func extractChapterTitle(xhtmlContent string) string {
	doc, err := html.Parse(strings.NewReader(xhtmlContent))
	if err != nil {
		return ""
	}

	// Look for title in <title> tag or first heading
	title := findTitle(doc)
	if title == "" {
		title = findFirstHeading(doc)
	}

	return strings.TrimSpace(title)
}

// findTitle finds the <title> element.
func findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		return getTextContent(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := findTitle(c); title != "" {
			return title
		}
	}
	return ""
}

// findFirstHeading finds the first h1-h6 element.
func findFirstHeading(n *html.Node) string {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return getTextContent(n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if heading := findFirstHeading(c); heading != "" {
			return heading
		}
	}
	return ""
}

// getTextContent extracts all text from a node and its children.
func getTextContent(n *html.Node) string {
	var buf bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return buf.String()
}

// renderHTML renders an HTML node back to string.
func renderHTML(n *html.Node) (string, error) {
	var buf bytes.Buffer
	if err := html.Render(&buf, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// findBody finds the <body> element in an HTML document.
func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if body := findBody(c); body != nil {
			return body
		}
	}
	return nil
}

// parseXHTML parses XHTML content and returns the body content.
func parseXHTML(content []byte) (*html.Node, error) {
	return html.Parse(bytes.NewReader(content))
}

// isContentType checks if a media type represents document content.
func isContentType(mediaType string) bool {
	return mediaType == "application/xhtml+xml" ||
		mediaType == "text/html" ||
		mediaType == "application/x-dtbncx+xml"
}

// extractEpubType extracts the epub:type attribute from XHTML content.
// It looks for epub:type on body, section, or article elements.
func extractEpubType(xhtmlContent string) string {
	doc, err := html.Parse(strings.NewReader(xhtmlContent))
	if err != nil {
		return ""
	}

	epubType := findEpubType(doc)
	return epubType
}

// findEpubType recursively searches for epub:type attribute.
func findEpubType(n *html.Node) string {
	if n.Type == html.ElementNode {
		// Check elements that commonly have epub:type
		switch n.Data {
		case "body", "section", "article", "div", "nav", "aside":
			for _, attr := range n.Attr {
				// epub:type can be namespaced or not
				if attr.Key == "epub:type" || attr.Key == "type" && attr.Namespace == "http://www.idpf.org/2007/ops" {
					return attr.Val
				}
			}
		}
	}

	// Search children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if epubType := findEpubType(c); epubType != "" {
			return epubType
		}
	}

	return ""
}

// Discard variable for io.Copy operations
var _ = io.Discard
