# EPUB Reader

A command-line tool for reading, analyzing, and comparing EPUB files. Includes a web dashboard for browsing your library.

## Features

- **EPUB Parsing**: Extract metadata, chapters, and text content from EPUB files
- **Text Analysis**: Word counts, vocabulary richness, readability scores, dialogue ratio
- **Author Comparison**: Compare writing styles across authors' corpora
- **Section Filtering**: Classify sections (front matter, chapters, etc.) with optional LLM support
- **Beats Analysis**: Extract narrative beats (conflict, choice, consequence) using scene detection and LLM analysis
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

Section filtering uses a **tiered classification approach** to separate author-written content from front/back matter:

1. **epub:type attributes** (95% confidence) - Most reliable signal
2. **Built-in pattern matching** (80% confidence) - Filename/type heuristics
3. **Custom rules from database** - User-defined patterns
4. **LLM classification** (optional) - For ambiguous cases only
5. **Default fallback** - Allow with low confidence, flagged for review

**Most books work perfectly without the `--llm` flag.** Only use it if you see many "needs review" sections.

```bash
# Show section classifications (uses heuristics only)
epub-reader filter <book-id>

# Use LLM for ambiguous sections (requires ANTHROPIC_API_KEY)
epub-reader filter <book-id> --llm

# Store classifications to database
epub-reader filter <book-id> --store

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

### Beats Analysis

Extract narrative beats from books using automatic scene detection and LLM analysis. Each beat captures:
- **Summary**: What happens in the scene
- **Conflict**: The tension or problem
- **Choice**: Decision made by characters
- **Consequence**: Result of that choice
- **Perspective Shift**: Changes in POV or understanding

```bash
# Analyze beats for a book (requires ANTHROPIC_API_KEY)
epub-reader beats analyze <book-id>

# Re-analyze and replace existing beats
epub-reader beats analyze <book-id> --force

# List all beats for a book
epub-reader beats list <book-id>

# Show detailed beat information
epub-reader beats show <beat-id>

# Include LLM prompt and response in output
epub-reader beats show <beat-id> --prompt
```

**Scene Detection**: Automatically identifies scene breaks using:
- Blank lines between paragraphs
- Common scene separators (`***`, `---`, `• • •`, etc.)
- Chapter boundaries

## Web Dashboard

Start the web server:

```bash
epub-server --port 8080
```

Or run directly:

```bash
go run ./cmd/epub-server --port 8080
```

Then visit http://localhost:8080

### Dashboard Features

- **Library View**: Browse all books with word counts and readability scores
- **Book View**: Detailed book information with quick actions and beats access
- **Beats Timeline**: Interactive chart showing story beats across the narrative arc
- **Author View**: See corpus statistics and list of books per author
- **Audit View**: Review and overrule section classification decisions (HTMX-powered)
- **Compare View**: Radar chart comparison of two authors' writing metrics

## Development

### Running Locally

During development, use `go run` to automatically rebuild on changes:

```bash
# CLI tool
go run ./cmd/epub-reader <command>

# Web server (rebuilds automatically)
go run ./cmd/epub-server --port 8080
```

The server will automatically find an available port if 8080 is busy (tries 8080-8089).

### Building for Production

Install the binaries:

```bash
go install ./cmd/epub-reader
go install ./cmd/epub-server
```

Then run them directly:

```bash
epub-reader <command>
epub-server --port 8080
```

**Note:** After making code changes, you must rebuild (`go install`) or use `go run` to see the changes.

## Configuration

The default database location is `~/.epub-reader/library.db`. Override with:

```bash
epub-reader --db /path/to/library.db <command>
epub-server --db /path/to/library.db
```

## Environment Variables

- `ANTHROPIC_API_KEY`: Required for LLM-powered features:
  - Section filtering with `epub-reader filter <book-id> --llm`
  - Beats analysis with `epub-reader beats analyze <book-id>`
  
  **Note**: Most functionality works without an API key. Section filtering uses heuristics by default, and beats analysis is optional.

## Tech Stack

- **CLI**: Go + Cobra
- **Storage**: SQLite
- **Web**: Go `html/template` + HTMX + PicoCSS + Chart.js
- **EPUB Parsing**: Custom parser using `golang.org/x/net/html`
