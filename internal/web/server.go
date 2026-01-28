package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"epub-reader/pkg/storage"
)

// Server is the web dashboard server.
type Server struct {
	store     *storage.SQLiteStore
	mux       *http.ServeMux
	templates *template.Template
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
func (s *Server) loadTemplates() error {
	funcMap := template.FuncMap{
		"formatPercent": formatPercent,
		"formatFloat":   formatFloat,
		"truncate":      truncateString,
	}

	// Parse base templates
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("internal/web/templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	// Parse partials
	tmpl, err = tmpl.ParseGlob("internal/web/templates/partials/*.html")
	if err != nil {
		// Partials might not exist yet, which is okay
		if !filepath.IsAbs(err.Error()) {
			// Ignore glob errors for missing partials directory
		}
	}

	s.templates = tmpl
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

	// Author views
	s.mux.HandleFunc("GET /authors", s.handleAuthors)
	s.mux.HandleFunc("GET /authors/{id}", s.handleAuthorDetail)

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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
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

