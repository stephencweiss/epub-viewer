package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database tables.
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS authors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		author_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		language TEXT,
		publisher TEXT,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (author_id) REFERENCES authors(id)
	);

	CREATE TABLE IF NOT EXISTS analyses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		total_words INTEGER,
		unique_words INTEGER,
		vocabulary_rich REAL,
		hapax_legomena INTEGER,
		avg_word_len REAL,
		total_sentences INTEGER,
		avg_sentence_len REAL,
		total_paragraphs INTEGER,
		avg_paragraph_len REAL,
		readability_score REAL,
		dialogue_ratio REAL,
		total_syllables INTEGER,
		avg_syllables REAL,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_books_author ON books(author_id);
	CREATE INDEX IF NOT EXISTS idx_analyses_book ON analyses(book_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateAuthor creates a new author.
func (s *SQLiteStore) CreateAuthor(name string) (*Author, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	result, err := s.db.Exec("INSERT INTO authors (name) VALUES (?)", name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Author{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

// GetAuthor retrieves an author by ID.
func (s *SQLiteStore) GetAuthor(id int64) (*Author, error) {
	var author Author
	err := s.db.QueryRow(
		"SELECT id, name, created_at FROM authors WHERE id = ?",
		id,
	).Scan(&author.ID, &author.Name, &author.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &author, nil
}

// GetAuthorByName retrieves an author by exact name.
func (s *SQLiteStore) GetAuthorByName(name string) (*Author, error) {
	var author Author
	err := s.db.QueryRow(
		"SELECT id, name, created_at FROM authors WHERE name = ?",
		name,
	).Scan(&author.ID, &author.Name, &author.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &author, nil
}

// ListAuthors returns all authors.
func (s *SQLiteStore) ListAuthors() ([]Author, error) {
	rows, err := s.db.Query("SELECT id, name, created_at FROM authors ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}

	return authors, rows.Err()
}

// FindSimilarAuthors finds authors with similar names.
func (s *SQLiteStore) FindSimilarAuthors(name string) ([]Author, error) {
	// Simple LIKE-based search
	pattern := "%" + strings.TrimSpace(name) + "%"
	rows, err := s.db.Query(
		"SELECT id, name, created_at FROM authors WHERE name LIKE ? ORDER BY name",
		pattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}

	return authors, rows.Err()
}

// AddBook adds a new book to the database.
func (s *SQLiteStore) AddBook(authorID int64, title, path, language, publisher string) (*Book, error) {
	result, err := s.db.Exec(
		"INSERT INTO books (author_id, title, path, language, publisher) VALUES (?, ?, ?, ?, ?)",
		authorID, title, path, language, publisher,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Book{
		ID:        id,
		AuthorID:  authorID,
		Title:     title,
		Path:      path,
		Language:  language,
		Publisher: publisher,
		AddedAt:   time.Now(),
	}, nil
}

// GetBook retrieves a book by ID.
func (s *SQLiteStore) GetBook(id int64) (*Book, error) {
	var book Book
	err := s.db.QueryRow(`
		SELECT b.id, b.author_id, b.title, b.path, b.language, b.publisher, b.added_at, a.name
		FROM books b
		JOIN authors a ON b.author_id = a.id
		WHERE b.id = ?
	`, id).Scan(&book.ID, &book.AuthorID, &book.Title, &book.Path, &book.Language, &book.Publisher, &book.AddedAt, &book.AuthorName)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &book, nil
}

// GetBookByPath retrieves a book by file path.
func (s *SQLiteStore) GetBookByPath(path string) (*Book, error) {
	var book Book
	err := s.db.QueryRow(`
		SELECT b.id, b.author_id, b.title, b.path, b.language, b.publisher, b.added_at, a.name
		FROM books b
		JOIN authors a ON b.author_id = a.id
		WHERE b.path = ?
	`, path).Scan(&book.ID, &book.AuthorID, &book.Title, &book.Path, &book.Language, &book.Publisher, &book.AddedAt, &book.AuthorName)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &book, nil
}

// ListBooks returns all books.
func (s *SQLiteStore) ListBooks() ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.author_id, b.title, b.path, b.language, b.publisher, b.added_at, a.name
		FROM books b
		JOIN authors a ON b.author_id = a.id
		ORDER BY a.name, b.title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.AuthorID, &b.Title, &b.Path, &b.Language, &b.Publisher, &b.AddedAt, &b.AuthorName); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	return books, rows.Err()
}

// ListBooksByAuthor returns all books by a specific author.
func (s *SQLiteStore) ListBooksByAuthor(authorID int64) ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.author_id, b.title, b.path, b.language, b.publisher, b.added_at, a.name
		FROM books b
		JOIN authors a ON b.author_id = a.id
		WHERE b.author_id = ?
		ORDER BY b.title
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.AuthorID, &b.Title, &b.Path, &b.Language, &b.Publisher, &b.AddedAt, &b.AuthorName); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	return books, rows.Err()
}

// RemoveBook removes a book and its analysis.
func (s *SQLiteStore) RemoveBook(id int64) error {
	result, err := s.db.Exec("DELETE FROM books WHERE id = ?", id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// SaveAnalysis saves or updates analysis for a book.
func (s *SQLiteStore) SaveAnalysis(analysis *StoredAnalysis) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO analyses (
			book_id, total_words, unique_words, vocabulary_rich, hapax_legomena,
			avg_word_len, total_sentences, avg_sentence_len, total_paragraphs,
			avg_paragraph_len, readability_score, dialogue_ratio, total_syllables, avg_syllables
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		analysis.BookID, analysis.TotalWords, analysis.UniqueWords,
		analysis.VocabularyRich, analysis.HapaxLegomena, analysis.AvgWordLen,
		analysis.TotalSentences, analysis.AvgSentenceLen, analysis.TotalParagraphs,
		analysis.AvgParagraphLen, analysis.ReadabilityScore, analysis.DialogueRatio,
		analysis.TotalSyllables, analysis.AvgSyllables,
	)
	return err
}

// GetAnalysis retrieves analysis for a book.
func (s *SQLiteStore) GetAnalysis(bookID int64) (*StoredAnalysis, error) {
	var a StoredAnalysis
	err := s.db.QueryRow(`
		SELECT id, book_id, created_at, total_words, unique_words, vocabulary_rich,
			hapax_legomena, avg_word_len, total_sentences, avg_sentence_len,
			total_paragraphs, avg_paragraph_len, readability_score, dialogue_ratio,
			total_syllables, avg_syllables
		FROM analyses WHERE book_id = ?
	`, bookID).Scan(
		&a.ID, &a.BookID, &a.CreatedAt, &a.TotalWords, &a.UniqueWords,
		&a.VocabularyRich, &a.HapaxLegomena, &a.AvgWordLen, &a.TotalSentences,
		&a.AvgSentenceLen, &a.TotalParagraphs, &a.AvgParagraphLen,
		&a.ReadabilityScore, &a.DialogueRatio, &a.TotalSyllables, &a.AvgSyllables,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

// GetCorpusAnalysis returns aggregated analysis for an author's corpus.
func (s *SQLiteStore) GetCorpusAnalysis(authorID int64) (*CorpusAnalysis, error) {
	author, err := s.GetAuthor(authorID)
	if err != nil {
		return nil, err
	}

	var corpus CorpusAnalysis
	corpus.Author = *author

	err = s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(a.total_words), 0),
			COALESCE(AVG(a.vocabulary_rich), 0),
			COALESCE(AVG(a.readability_score), 0),
			COALESCE(AVG(a.avg_sentence_len), 0),
			COALESCE(AVG(a.avg_word_len), 0),
			COALESCE(AVG(a.dialogue_ratio), 0),
			COALESCE(SUM(a.unique_words), 0)
		FROM books b
		LEFT JOIN analyses a ON b.id = a.book_id
		WHERE b.author_id = ?
	`, authorID).Scan(
		&corpus.BookCount,
		&corpus.TotalWords,
		&corpus.AvgVocabularyRich,
		&corpus.AvgReadability,
		&corpus.AvgSentenceLen,
		&corpus.AvgWordLen,
		&corpus.AvgDialogueRatio,
		&corpus.TotalUniqueWords,
	)

	if err != nil {
		return nil, err
	}

	return &corpus, nil
}

// DefaultDBPath returns the default database path.
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "epub-reader.db"
	}
	return filepath.Join(home, ".epub-reader", "library.db")
}
