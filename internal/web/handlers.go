package web

import (
	"net/http"
	"strconv"

	"epub-reader/pkg/storage"
)

// BookWithAnalysis combines a book with its analysis for display.
type BookWithAnalysis struct {
	storage.Book
	Analysis *storage.StoredAnalysis
}

// handleLibrary renders the library page with all books.
func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	// Only handle exact root path
	if r.URL.Path != "/" {
		s.renderError(w, http.StatusNotFound, "Page not found")
		return
	}

	books, err := s.store.ListBooks()
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "Failed to load books")
		return
	}

	data := struct {
		Books []BookWithAnalysis
	}{
		Books: make([]BookWithAnalysis, len(books)),
	}

	for i, book := range books {
		analysis, _ := s.store.GetAnalysis(book.ID)
		data.Books[i] = BookWithAnalysis{Book: book, Analysis: analysis}
	}

	s.render(w, "library", data)
}

// handleBookDetail renders the detail page for a single book.
func (s *Server) handleBookDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	book, err := s.store.GetBook(id)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Book not found")
		return
	}

	analysis, _ := s.store.GetAnalysis(id)

	s.render(w, "book", map[string]any{
		"Book":     book,
		"Analysis": analysis,
	})
}
