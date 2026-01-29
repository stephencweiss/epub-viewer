# EPUB Reader

A command-line tool for reading, analyzing, and comparing EPUB files. Includes a web dashboard for browsing your library.

## Features

- **EPUB Parsing**: Extract metadata, chapters, and text content from EPUB files
- **Text Analysis**: Word counts, vocabulary richness, readability scores, dialogue ratio
- **Author Comparison**: Compare writing styles across authors' corpora
- **Section Filtering**: Classify sections (front matter, chapters, etc.) with optional LLM support
- **Markdown Export**: Convert EPUBs to Markdown (single file or per-chapter)
- **Web Dashboard**: Browse library, view stats, audit decisions, compare authors with radar charts
- **SQLite Storage**: Persistent library with analysis results

## Installation

```bash
go install ./cmd/epub-reader
go install ./cmd/epub-server
```

## CLI Usage

### Adding Books

```bash
# Add a single EPUB
epub-reader add book.epub

# Add all EPUBs in a directory
epub-reader add /path/to/epubs/

# Recursively scan subdirectories
epub-reader add /path/to/library -r

# Batch mode (auto-create authors, no prompts)
epub-reader add /path/to/library -r -y

# Override author for all books
epub-reader add /path/to/epubs --author "Jane Doe" -y
```

### Library Management

```bash
# List all books
epub-reader list

# List books by author
epub-reader list --author "Dickens"

# Show book details
epub-reader info <book-id>

# Remove a book
epub-reader remove <book-id>

# List authors
epub-reader authors
```

### Analysis

```bash
# Analyze without storing
epub-reader analyze book.epub

# Compare two authors
epub-reader compare "Author One" "Author Two"
```

### Export

```bash
# Export to Markdown (one file per chapter)
epub-reader export book.epub output/

# Export as single file
epub-reader export book.epub output/ --single
```

### Section Filtering

```bash
# Show section classifications
epub-reader filter <book-id>

# Use LLM for ambiguous sections
epub-reader filter <book-id> --llm

# Manage filtering rules
epub-reader rules list
epub-reader rules add "copyright" deny
epub-reader rules remove <rule-id>
```

### Audit & Overrule

```bash
# View LLM decisions for a book
epub-reader audit <book-id>

# Flip a decision (ALLOW <-> DENY)
epub-reader overrule <decision-id>
```

## Web Dashboard

Start the web server:

```bash
epub-reader-server --port 8080
```

Or run directly:

```bash
go run ./cmd/epub-server --port 8080
```

Then visit http://localhost:8080

### Dashboard Features

- **Library View**: Browse all books with word counts and readability scores
- **Author View**: See corpus statistics and list of books per author
- **Audit View**: Review and overrule section classification decisions (HTMX-powered)
- **Compare View**: Radar chart comparison of two authors' writing metrics

## Configuration

The default database location is `~/.epub-reader/library.db`. Override with:

```bash
epub-reader --db /path/to/library.db <command>
epub-server --db /path/to/library.db
```

## Environment Variables

- `ANTHROPIC_API_KEY`: Required for LLM-based section classification (`--llm` flag)

## Tech Stack

- **CLI**: Go + Cobra
- **Storage**: SQLite
- **Web**: Go `html/template` + HTMX + PicoCSS + Chart.js
- **EPUB Parsing**: Custom parser using `golang.org/x/net/html`
