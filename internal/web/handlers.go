package web

import (
	"net/http"
	"strconv"
	"strings"

	"epub-reader/pkg/epub"
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
	authors, _ := s.store.ListAuthors()

	s.render(w, "book", map[string]any{
		"Book":     book,
		"Analysis": analysis,
		"Authors":  authors,
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
	allAuthors, _ := s.store.ListAuthors()
	bookCount, _ := s.store.CountBooksByAuthor(id)

	// Attach analysis to each book
	booksWithAnalysis := make([]BookWithAnalysis, len(books))
	for i, book := range books {
		analysis, _ := s.store.GetAnalysis(book.ID)
		booksWithAnalysis[i] = BookWithAnalysis{Book: book, Analysis: analysis}
	}

	s.render(w, "author", map[string]any{
		"Author":     author,
		"Corpus":     corpus,
		"Books":      booksWithAnalysis,
		"AllAuthors": allAuthors,
		"BookCount":  bookCount,
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

// handleOverrule flips a decision and returns the updated row for HTMX.
func (s *Server) handleOverrule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid decision ID", http.StatusBadRequest)
		return
	}

	updated, err := s.store.OverruleDecision(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return just the updated row partial for HTMX swap
	// Use the audit template set which has the audit_row partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["audit"].ExecuteTemplate(w, "audit_row", updated); err != nil {
		http.Error(w, "Failed to render row", http.StatusInternalServerError)
	}
}

// handleCompare renders the author comparison page with radar chart.
func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	authors, err := s.store.ListAuthors()
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "Failed to load authors")
		return
	}

	// Pre-select from query params if provided
	var author1ID, author2ID int64
	if v := r.URL.Query().Get("author1"); v != "" {
		author1ID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := r.URL.Query().Get("author2"); v != "" {
		author2ID, _ = strconv.ParseInt(v, 10, 64)
	}

	s.render(w, "compare", map[string]any{
		"Authors":   authors,
		"Author1ID": author1ID,
		"Author2ID": author2ID,
	})
}

// handleBookReader renders the book reader page with chapter list.
func (s *Server) handleBookReader(w http.ResponseWriter, r *http.Request) {
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

	// Parse the EPUB to get chapters
	parsed, err := epub.Parse(book.Path)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "Failed to parse EPUB file")
		return
	}

	s.render(w, "reader", map[string]any{
		"Book":     book,
		"Chapters": parsed.Chapters,
	})
}

// handleChapterContent returns the content of a specific chapter (HTMX partial).
func (s *Server) handleChapterContent(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	chapterNum, err := strconv.Atoi(r.PathValue("num"))
	if err != nil || chapterNum < 1 {
		http.Error(w, "Invalid chapter number", http.StatusBadRequest)
		return
	}

	book, err := s.store.GetBook(bookID)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	parsed, err := epub.Parse(book.Path)
	if err != nil {
		http.Error(w, "Failed to parse EPUB", http.StatusInternalServerError)
		return
	}

	if chapterNum > len(parsed.Chapters) {
		http.Error(w, "Chapter not found", http.StatusNotFound)
		return
	}

	chapter := parsed.Chapters[chapterNum-1]
	wordCount := len(strings.Fields(chapter.Text))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["reader"].ExecuteTemplate(w, "chapter_content", map[string]any{
		"Chapter":   chapter,
		"Number":    chapterNum,
		"WordCount": wordCount,
	}); err != nil {
		http.Error(w, "Failed to render chapter", http.StatusInternalServerError)
	}
}

// handleReassignBook moves a book to a different author (HTMX).
func (s *Server) handleReassignBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	newAuthorID, err := strconv.ParseInt(r.FormValue("author_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	if err := s.store.ReassignBook(bookID, newAuthorID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to book page
	w.Header().Set("HX-Redirect", "/books/"+strconv.FormatInt(bookID, 10))
	w.WriteHeader(http.StatusOK)
}

// handleEditBook edits book metadata (HTMX).
func (s *Server) handleEditBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	language := strings.TrimSpace(r.FormValue("language"))
	publisher := strings.TrimSpace(r.FormValue("publisher"))

	if title == "" {
		http.Error(w, "Title cannot be empty", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateBook(bookID, title, language, publisher); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/books/"+strconv.FormatInt(bookID, 10))
	w.WriteHeader(http.StatusOK)
}

// handleRenameAuthor renames an author (HTMX).
func (s *Server) handleRenameAuthor(w http.ResponseWriter, r *http.Request) {
	authorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	newName := strings.TrimSpace(r.FormValue("name"))
	if newName == "" {
		http.Error(w, "Name cannot be empty", http.StatusBadRequest)
		return
	}

	if err := s.store.RenameAuthor(authorID, newName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/authors/"+strconv.FormatInt(authorID, 10))
	w.WriteHeader(http.StatusOK)
}

// handleDeleteAuthor deletes an author with no books (HTMX).
func (s *Server) handleDeleteAuthor(w http.ResponseWriter, r *http.Request) {
	authorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteAuthor(authorID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Redirect", "/authors")
	w.WriteHeader(http.StatusOK)
}

// handleInfo renders the statistics info page.
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	s.render(w, "info", nil)
}

// handleMergeAuthors merges two authors (HTMX).
func (s *Server) handleMergeAuthors(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	sourceID, err := strconv.ParseInt(r.FormValue("source_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid source author ID", http.StatusBadRequest)
		return
	}

	targetID, err := strconv.ParseInt(r.FormValue("target_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid target author ID", http.StatusBadRequest)
		return
	}

	if err := s.store.MergeAuthors(sourceID, targetID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/authors/"+strconv.FormatInt(targetID, 10))
	w.WriteHeader(http.StatusOK)
}
