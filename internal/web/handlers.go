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

// AuthorWithStats combines an author with their corpus analysis.
type AuthorWithStats struct {
	storage.Author
	Corpus *storage.CorpusAnalysis
}

// handleAuthors renders the authors listing page.
func (s *Server) handleAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := s.store.ListAuthors()
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "Failed to load authors")
		return
	}

	data := struct {
		Authors []AuthorWithStats
	}{
		Authors: make([]AuthorWithStats, len(authors)),
	}

	for i, author := range authors {
		corpus, _ := s.store.GetCorpusAnalysis(author.ID)
		data.Authors[i] = AuthorWithStats{Author: author, Corpus: corpus}
	}

	s.render(w, "authors", data)
}

// handleAuthorDetail renders the detail page for a single author.
func (s *Server) handleAuthorDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid author ID")
		return
	}

	author, err := s.store.GetAuthor(id)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Author not found")
		return
	}

	corpus, _ := s.store.GetCorpusAnalysis(id)
	books, _ := s.store.ListBooksByAuthor(id)

	// Attach analysis to each book
	booksWithAnalysis := make([]BookWithAnalysis, len(books))
	for i, book := range books {
		analysis, _ := s.store.GetAnalysis(book.ID)
		booksWithAnalysis[i] = BookWithAnalysis{Book: book, Analysis: analysis}
	}

	s.render(w, "author", map[string]any{
		"Author": author,
		"Corpus": corpus,
		"Books":  booksWithAnalysis,
	})
}

// handleAuditList renders the list of unverified decisions.
func (s *Server) handleAuditList(w http.ResponseWriter, r *http.Request) {
	decisions, err := s.store.ListUnverifiedDecisions()
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "Failed to load decisions")
		return
	}

	s.render(w, "audit", map[string]any{
		"Decisions": decisions,
		"Title":     "Unverified Decisions",
	})
}

// handleBookAudit renders the decisions for a specific book.
func (s *Server) handleBookAudit(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	book, err := s.store.GetBook(bookID)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Book not found")
		return
	}

	decisions, err := s.store.ListDecisionAuditByBook(bookID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "Failed to load decisions")
		return
	}

	s.render(w, "audit", map[string]any{
		"Decisions": decisions,
		"Book":      book,
		"Title":     "Audit: " + book.Title,
	})
}
