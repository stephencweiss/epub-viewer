package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"

	"epub-reader/pkg/epub"
)

// Converter converts EPUB content to Markdown.
type Converter struct {
	// IncludeChapterTitles adds chapter titles as headings.
	IncludeChapterTitles bool
}

// NewConverter creates a new Markdown converter with default settings.
func NewConverter() *Converter {
	return &Converter{
		IncludeChapterTitles: true,
	}
}

// ConvertBook converts an entire EPUB book to Markdown.
func (c *Converter) ConvertBook(book *epub.Book) (string, error) {
	var buf bytes.Buffer

	// Book title
	if book.Title != "" {
		buf.WriteString(fmt.Sprintf("# %s\n\n", book.Title))
	}

	// Author info
	if len(book.Authors) > 0 {
		buf.WriteString(fmt.Sprintf("*By %s*\n\n", strings.Join(book.Authors, ", ")))
	}

	buf.WriteString("---\n\n")

	// Convert each chapter
	for i, chapter := range book.Chapters {
		chapterMd, err := c.ConvertChapter(chapter)
		if err != nil {
			return "", fmt.Errorf("failed to convert chapter %d: %w", i, err)
		}

		buf.WriteString(chapterMd)
		buf.WriteString("\n\n")
	}

	return buf.String(), nil
}

// ConvertChapter converts a single chapter to Markdown.
func (c *Converter) ConvertChapter(chapter epub.Chapter) (string, error) {
	var buf bytes.Buffer

	// Add chapter title if available and enabled
	if c.IncludeChapterTitles && chapter.Title != "" {
		buf.WriteString(fmt.Sprintf("## %s\n\n", chapter.Title))
	}

	// Parse the HTML content
	doc, err := html.Parse(strings.NewReader(chapter.Content))
	if err != nil {
		// Fallback to plain text
		return chapter.Text, nil
	}

	// Find the body element
	body := findBody(doc)
	if body == nil {
		// Convert entire document if no body found
		body = doc
	}

	// Convert HTML to Markdown
	md := c.convertNode(body, &conversionContext{})

	// Clean up the output
	md = cleanMarkdown(md)

	buf.WriteString(md)
	return buf.String(), nil
}

// conversionContext tracks state during conversion.
type conversionContext struct {
	listDepth  int
	inPre      bool
	inLink     bool
	linkHref   string
	orderedIdx int
}

// convertNode recursively converts an HTML node to Markdown.
func (c *Converter) convertNode(n *html.Node, ctx *conversionContext) string {
	if n == nil {
		return ""
	}

	var buf bytes.Buffer

	switch n.Type {
	case html.TextNode:
		text := n.Data
		if !ctx.inPre {
			// Normalize whitespace but preserve intentional spacing
			text = strings.ReplaceAll(text, "\n", " ")
			text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
		}
		buf.WriteString(text)

	case html.ElementNode:
		switch n.Data {
		case "h1":
			buf.WriteString("\n# ")
			buf.WriteString(c.getTextContent(n))
			buf.WriteString("\n\n")

		case "h2":
			buf.WriteString("\n## ")
			buf.WriteString(c.getTextContent(n))
			buf.WriteString("\n\n")

		case "h3":
			buf.WriteString("\n### ")
			buf.WriteString(c.getTextContent(n))
			buf.WriteString("\n\n")

		case "h4":
			buf.WriteString("\n#### ")
			buf.WriteString(c.getTextContent(n))
			buf.WriteString("\n\n")

		case "h5":
			buf.WriteString("\n##### ")
			buf.WriteString(c.getTextContent(n))
			buf.WriteString("\n\n")

		case "h6":
			buf.WriteString("\n###### ")
			buf.WriteString(c.getTextContent(n))
			buf.WriteString("\n\n")

		case "p":
			text := c.convertChildren(n, ctx)
			text = strings.TrimSpace(text)
			if text != "" {
				buf.WriteString("\n")
				buf.WriteString(text)
				buf.WriteString("\n")
			}

		case "br":
			buf.WriteString("  \n")

		case "hr":
			buf.WriteString("\n---\n")

		case "strong", "b":
			text := c.convertChildren(n, ctx)
			text = strings.TrimSpace(text)
			if text != "" {
				buf.WriteString("**")
				buf.WriteString(text)
				buf.WriteString("**")
			}

		case "em", "i":
			text := c.convertChildren(n, ctx)
			text = strings.TrimSpace(text)
			if text != "" {
				buf.WriteString("*")
				buf.WriteString(text)
				buf.WriteString("*")
			}

		case "code":
			text := c.getTextContent(n)
			if !ctx.inPre {
				buf.WriteString("`")
				buf.WriteString(text)
				buf.WriteString("`")
			} else {
				buf.WriteString(text)
			}

		case "pre":
			ctx.inPre = true
			buf.WriteString("\n```\n")
			buf.WriteString(c.convertChildren(n, ctx))
			buf.WriteString("\n```\n")
			ctx.inPre = false

		case "blockquote":
			text := c.convertChildren(n, ctx)
			lines := strings.Split(strings.TrimSpace(text), "\n")
			buf.WriteString("\n")
			for _, line := range lines {
				buf.WriteString("> ")
				buf.WriteString(strings.TrimSpace(line))
				buf.WriteString("\n")
			}

		case "ul":
			ctx.listDepth++
			buf.WriteString("\n")
			buf.WriteString(c.convertChildren(n, ctx))
			ctx.listDepth--

		case "ol":
			ctx.listDepth++
			oldIdx := ctx.orderedIdx
			ctx.orderedIdx = 1
			buf.WriteString("\n")
			buf.WriteString(c.convertChildren(n, ctx))
			ctx.orderedIdx = oldIdx
			ctx.listDepth--

		case "li":
			indent := strings.Repeat("  ", ctx.listDepth-1)
			text := strings.TrimSpace(c.convertChildren(n, ctx))
			if ctx.orderedIdx > 0 {
				buf.WriteString(fmt.Sprintf("%s%d. %s\n", indent, ctx.orderedIdx, text))
				ctx.orderedIdx++
			} else {
				buf.WriteString(fmt.Sprintf("%s- %s\n", indent, text))
			}

		case "a":
			href := getAttr(n, "href")
			text := c.getTextContent(n)
			if href != "" && text != "" {
				buf.WriteString(fmt.Sprintf("[%s](%s)", text, href))
			} else if text != "" {
				buf.WriteString(text)
			}

		case "img":
			src := getAttr(n, "src")
			alt := getAttr(n, "alt")
			if src != "" {
				buf.WriteString(fmt.Sprintf("![%s](%s)", alt, src))
			}

		case "table":
			buf.WriteString("\n")
			buf.WriteString(c.convertTable(n))
			buf.WriteString("\n")

		case "div", "section", "article", "main", "aside", "nav", "header", "footer":
			// Block elements - just convert children with newlines
			text := c.convertChildren(n, ctx)
			if strings.TrimSpace(text) != "" {
				buf.WriteString("\n")
				buf.WriteString(text)
				buf.WriteString("\n")
			}

		case "span":
			buf.WriteString(c.convertChildren(n, ctx))

		case "script", "style", "head", "meta", "link":
			// Skip these elements

		default:
			// For unknown elements, just convert children
			buf.WriteString(c.convertChildren(n, ctx))
		}

	case html.DocumentNode:
		buf.WriteString(c.convertChildren(n, ctx))
	}

	return buf.String()
}

// convertChildren converts all child nodes.
func (c *Converter) convertChildren(n *html.Node, ctx *conversionContext) string {
	var buf bytes.Buffer
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		buf.WriteString(c.convertNode(child, ctx))
	}
	return buf.String()
}

// getTextContent extracts plain text from a node.
func (c *Converter) getTextContent(n *html.Node) string {
	var buf bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			f(child)
		}
	}
	f(n)
	return strings.TrimSpace(buf.String())
}

// convertTable converts an HTML table to Markdown.
func (c *Converter) convertTable(n *html.Node) string {
	var rows [][]string

	var processNode func(*html.Node)
	processNode = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "thead":
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					processNode(child)
				}
				return
			case "tr":
				var cells []string
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
						cells = append(cells, strings.TrimSpace(c.getTextContent(child)))
					}
				}
				if len(cells) > 0 {
					rows = append(rows, cells)
				}
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			processNode(child)
		}
	}

	processNode(n)

	if len(rows) == 0 {
		return ""
	}

	// Find max columns
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	var buf bytes.Buffer
	for i, row := range rows {
		// Pad row to max columns
		for len(row) < maxCols {
			row = append(row, "")
		}

		buf.WriteString("| ")
		buf.WriteString(strings.Join(row, " | "))
		buf.WriteString(" |\n")

		// Add header separator after first row
		if i == 0 {
			buf.WriteString("|")
			for j := 0; j < maxCols; j++ {
				buf.WriteString(" --- |")
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// getAttr gets an attribute value from an HTML node.
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// findBody finds the body element in an HTML document.
func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if body := findBody(child); body != nil {
			return body
		}
	}
	return nil
}

// cleanMarkdown cleans up the generated Markdown.
func cleanMarkdown(md string) string {
	// Remove excessive blank lines
	md = regexp.MustCompile(`\n{3,}`).ReplaceAllString(md, "\n\n")

	// Trim lines
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	md = strings.Join(lines, "\n")

	// Trim start and end
	md = strings.TrimSpace(md)

	return md
}
