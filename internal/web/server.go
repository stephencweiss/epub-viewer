package web

import (
	"fmt"
	"html/template"
	"net/http"

	"epub-reader/pkg/storage"
)

// Server is the web dashboard server.
type Server struct {
	store     *storage.SQLiteStore
	mux       *http.ServeMux
	templates map[string]*template.Template // One template set per page
	staticDir string
}

// NewServer creates a new web server.
func NewServer(store *storage.SQLiteStore, staticDir string) (*Server, error) {
	s := &Server{
		store:     store,
		mux:       http.NewServeMux(),
		staticDir: staticDir,
	}

	if err := s.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	s.registerRoutes()
	return s, nil
}

// ServeHTTP implements http.Handler with middleware.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := RecoveryMiddleware(LoggingMiddleware(s.mux))
	handler.ServeHTTP(w, r)
}

// loadTemplates loads and parses all HTML templates.
// Each page template gets its own template set to avoid namespace conflicts.
func (s *Server) loadTemplates() error {
	funcMap := template.FuncMap{
		"formatPercent": formatPercent,
		"formatFloat":   formatFloat,
		"formatWordsK":  formatWordsK,
		"truncate":      truncateString,
		"add":           func(a, b int) int { return a + b },
	}

	// Parse base template and partials first
	base, err := template.New("").Funcs(funcMap).ParseFiles(
		"internal/web/templates/base.html",
	)
	if err != nil {
		return fmt.Errorf("failed to parse base template: %w", err)
	}

	// Add partials to base
	base, err = base.ParseGlob("internal/web/templates/partials/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse partials: %w", err)
	}

	// Page templates that need their own namespace
	pages := []string{"library", "book", "authors", "author", "audit", "compare", "error", "reader"}
	s.templates = make(map[string]*template.Template)

	for _, page := range pages {
		// Clone the base template
		pageTemplate, err := base.Clone()
		if err != nil {
			return fmt.Errorf("failed to clone base for %s: %w", page, err)
		}

		// Parse the page-specific template into the cloned set
		pageTemplate, err = pageTemplate.ParseFiles(
			fmt.Sprintf("internal/web/templates/%s.html", page),
		)
		if err != nil {
			return fmt.Errorf("failed to parse %s template: %w", page, err)
		}

		s.templates[page] = pageTemplate
	}

	return nil
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	// Static files
	fs := http.FileServer(http.Dir(s.staticDir))
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Library views
	s.mux.HandleFunc("GET /", s.handleLibrary)
	s.mux.HandleFunc("GET /books/{id}", s.handleBookDetail)
	s.mux.HandleFunc("GET /books/{id}/read", s.handleBookReader)
	s.mux.HandleFunc("GET /books/{id}/chapters/{num}", s.handleChapterContent)
	s.mux.HandleFunc("POST /books/{id}/reassign", s.handleReassignBook)
	s.mux.HandleFunc("POST /books/{id}/edit", s.handleEditBook)

	// Author views
	s.mux.HandleFunc("GET /authors", s.handleAuthors)
	s.mux.HandleFunc("GET /authors/{id}", s.handleAuthorDetail)
	s.mux.HandleFunc("POST /authors/{id}/rename", s.handleRenameAuthor)
	s.mux.HandleFunc("POST /authors/{id}/delete", s.handleDeleteAuthor)
	s.mux.HandleFunc("POST /authors/merge", s.handleMergeAuthors)

	// Audit views
	s.mux.HandleFunc("GET /audit", s.handleAuditList)
	s.mux.HandleFunc("GET /audit/book/{id}", s.handleBookAudit)
	s.mux.HandleFunc("POST /audit/{id}/overrule", s.handleOverrule)

	// Compare view
	s.mux.HandleFunc("GET /compare", s.handleCompare)

	// API endpoints for Chart.js
	s.mux.HandleFunc("GET /api/authors/{id}/metrics", s.apiAuthorMetrics)
	s.mux.HandleFunc("GET /api/compare", s.apiCompare)
}

// render executes a template and writes the response.
func (s *Server) render(w http.ResponseWriter, name string, data any) {
	tmpl, ok := s.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderError renders an error page.
func (s *Server) renderError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	titles := map[int]string{
		400: "Bad Request",
		404: "Not Found",
		500: "Internal Server Error",
	}
	title := titles[code]
	if title == "" {
		title = "Error"
	}
	s.render(w, "error", map[string]any{
		"Code":    code,
		"Title":   title,
		"Message": message,
	})
}

// Template helper functions

func formatPercent(f float64) string {
	return fmt.Sprintf("%.1f%%", f*100)
}

func formatFloat(f float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, f)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatWordsK(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%dk", (n+500)/1000)
	}
	return fmt.Sprintf("%d", n)
}

