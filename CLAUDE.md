# EPUB Reader - Project Guidelines

## Version Control: jj (Jujutsu)

This project uses **jj** (Jujutsu) for version control with stacked diffs.

### Workflow

1. **Each feature/step = one atomic commit**
2. **Commits must be focused on a single logical change**
3. **Use stacked diffs to build features incrementally**

### Common Commands

```bash
# Create new change on top of current
jj new -m "description"

# Edit current change message
jj describe -m "description"

# Show commit stack
jj log

# Show status
jj status

# Squash current change into parent
jj squash

# Split a change into multiple
jj split

# Rebase stack
jj rebase -d <destination>

# Push stack for review
jj git push

# Fetch from remote
jj git fetch
```

### Commit Message Format

```
stepN: short description

Longer explanation if needed.
```

Example:
```
step1: add section filtering database schema

Add section_rules, decision_audit, and sections tables
to support content filtering and LLM decision tracking.
```

### Fallback

If `jj` is not available, use standard `git` commands:
```bash
git add -A && git commit -m "description"
```

## Project Structure

```
epub-reader/
├── cmd/
│   └── epub-reader/        # CLI application
├── pkg/
│   ├── epub/               # EPUB parsing
│   ├── markdown/           # Markdown conversion
│   ├── analysis/           # Text analysis
│   ├── filter/             # Section filtering (NEW)
│   └── storage/            # SQLite persistence
└── internal/
    └── web/                # Web dashboard (future)
```

## Build & Test

```bash
# Build
go build -o epub-reader ./cmd/epub-reader

# Run
./epub-reader --help

# Test
go test ./...
```

## Plan Design

When designing a plan to implement a feature / structuring a commit - these should be considered "vertical" slices that fully implement the feature - from the client (as necessary) all the way to the data layer.

For example: If a plan includes adding a "Edit" and "Delete" functionality, this would involve
- UI (to indicate in the client which behavior to take)
- Route handlers (on the server for accepting the request)
- Controllers to manage the request and isolate the business logic
- Data/storage layer to communicate with the database

In this case, we would have two commits:
1/ Edit: includes any necessary UI, route handler, controller, and data changes necessary to support the "edit" feature
2/ Delete: includes any necessary UI, route handler, controller, and data changes necessary to support the "delete" feature.