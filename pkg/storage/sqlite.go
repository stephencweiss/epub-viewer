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

	CREATE TABLE IF NOT EXISTS section_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pattern TEXT NOT NULL UNIQUE,
		decision TEXT NOT NULL CHECK(decision IN ('ALLOW', 'DENY')),
		confidence REAL DEFAULT 1.0,
		source TEXT DEFAULT 'manual',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS decision_audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		file_name TEXT NOT NULL,
		epub_type TEXT,
		text_snippet TEXT,
		llm_prompt TEXT,
		llm_response TEXT,
		final_decision TEXT NOT NULL CHECK(final_decision IN ('ALLOW', 'DENY')),
		reason TEXT,
		manually_verified BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS sections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		file_name TEXT NOT NULL,
		epub_type TEXT,
		title TEXT,
		text_content TEXT,
		status TEXT NOT NULL CHECK(status IN ('ALLOW', 'DENY', 'PENDING')),
		section_order INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
		UNIQUE(book_id, file_name)
	);

	CREATE INDEX IF NOT EXISTS idx_decision_audit_book ON decision_audit(book_id);
	CREATE INDEX IF NOT EXISTS idx_sections_book ON sections(book_id);
	CREATE INDEX IF NOT EXISTS idx_sections_status ON sections(status);
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

// --- Section Rules ---

// CreateSectionRule creates a new section rule.
func (s *SQLiteStore) CreateSectionRule(pattern, decision, source string, confidence float64) (*SectionRule, error) {
	result, err := s.db.Exec(
		"INSERT INTO section_rules (pattern, decision, source, confidence) VALUES (?, ?, ?, ?)",
		pattern, decision, source, confidence,
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

	return &SectionRule{
		ID:         id,
		Pattern:    pattern,
		Decision:   decision,
		Source:     source,
		Confidence: confidence,
		CreatedAt:  time.Now(),
	}, nil
}

// GetSectionRule retrieves a rule by pattern.
func (s *SQLiteStore) GetSectionRule(pattern string) (*SectionRule, error) {
	var r SectionRule
	err := s.db.QueryRow(
		"SELECT id, pattern, decision, confidence, source, created_at FROM section_rules WHERE pattern = ?",
		pattern,
	).Scan(&r.ID, &r.Pattern, &r.Decision, &r.Confidence, &r.Source, &r.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListSectionRules returns all section rules.
func (s *SQLiteStore) ListSectionRules() ([]SectionRule, error) {
	rows, err := s.db.Query(
		"SELECT id, pattern, decision, confidence, source, created_at FROM section_rules ORDER BY pattern",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []SectionRule
	for rows.Next() {
		var r SectionRule
		if err := rows.Scan(&r.ID, &r.Pattern, &r.Decision, &r.Confidence, &r.Source, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// ListSectionRulesByDecision returns rules filtered by decision type.
func (s *SQLiteStore) ListSectionRulesByDecision(decision string) ([]SectionRule, error) {
	rows, err := s.db.Query(
		"SELECT id, pattern, decision, confidence, source, created_at FROM section_rules WHERE decision = ? ORDER BY pattern",
		decision,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []SectionRule
	for rows.Next() {
		var r SectionRule
		if err := rows.Scan(&r.ID, &r.Pattern, &r.Decision, &r.Confidence, &r.Source, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// DeleteSectionRule removes a rule by ID.
func (s *SQLiteStore) DeleteSectionRule(id int64) error {
	result, err := s.db.Exec("DELETE FROM section_rules WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Decision Audit ---

// SaveDecisionAudit stores an LLM decision for audit purposes.
func (s *SQLiteStore) SaveDecisionAudit(audit *DecisionAudit) error {
	result, err := s.db.Exec(`
		INSERT INTO decision_audit (book_id, file_name, epub_type, text_snippet, llm_prompt, llm_response, final_decision, reason, manually_verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, audit.BookID, audit.FileName, audit.EpubType, audit.TextSnippet, audit.LLMPrompt, audit.LLMResponse, audit.FinalDecision, audit.Reason, audit.ManuallyVerified)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	audit.ID = id
	return nil
}

// GetDecisionAudit retrieves a decision audit by ID.
func (s *SQLiteStore) GetDecisionAudit(id int64) (*DecisionAudit, error) {
	var a DecisionAudit
	err := s.db.QueryRow(`
		SELECT id, book_id, file_name, epub_type, text_snippet, llm_prompt, llm_response, final_decision, reason, manually_verified, created_at
		FROM decision_audit WHERE id = ?
	`, id).Scan(&a.ID, &a.BookID, &a.FileName, &a.EpubType, &a.TextSnippet, &a.LLMPrompt, &a.LLMResponse, &a.FinalDecision, &a.Reason, &a.ManuallyVerified, &a.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListDecisionAuditByBook returns all decision audits for a book.
func (s *SQLiteStore) ListDecisionAuditByBook(bookID int64) ([]DecisionAudit, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, file_name, epub_type, text_snippet, llm_prompt, llm_response, final_decision, reason, manually_verified, created_at
		FROM decision_audit WHERE book_id = ? ORDER BY created_at
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []DecisionAudit
	for rows.Next() {
		var a DecisionAudit
		if err := rows.Scan(&a.ID, &a.BookID, &a.FileName, &a.EpubType, &a.TextSnippet, &a.LLMPrompt, &a.LLMResponse, &a.FinalDecision, &a.Reason, &a.ManuallyVerified, &a.CreatedAt); err != nil {
			return nil, err
		}
		audits = append(audits, a)
	}
	return audits, rows.Err()
}

// OverruleDecision flips a decision and marks it as manually verified.
func (s *SQLiteStore) OverruleDecision(id int64) (*DecisionAudit, error) {
	// First get the current decision
	audit, err := s.GetDecisionAudit(id)
	if err != nil {
		return nil, err
	}

	// Flip the decision
	newDecision := "DENY"
	if audit.FinalDecision == "DENY" {
		newDecision = "ALLOW"
	}

	_, err = s.db.Exec(`
		UPDATE decision_audit SET final_decision = ?, manually_verified = TRUE WHERE id = ?
	`, newDecision, id)
	if err != nil {
		return nil, err
	}

	audit.FinalDecision = newDecision
	audit.ManuallyVerified = true
	return audit, nil
}

// ListUnverifiedDecisions returns decisions that haven't been manually verified.
func (s *SQLiteStore) ListUnverifiedDecisions() ([]DecisionAudit, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, file_name, epub_type, text_snippet, llm_prompt, llm_response, final_decision, reason, manually_verified, created_at
		FROM decision_audit WHERE manually_verified = FALSE ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []DecisionAudit
	for rows.Next() {
		var a DecisionAudit
		if err := rows.Scan(&a.ID, &a.BookID, &a.FileName, &a.EpubType, &a.TextSnippet, &a.LLMPrompt, &a.LLMResponse, &a.FinalDecision, &a.Reason, &a.ManuallyVerified, &a.CreatedAt); err != nil {
			return nil, err
		}
		audits = append(audits, a)
	}
	return audits, rows.Err()
}

// --- Sections ---

// SaveSection stores a section from an EPUB.
func (s *SQLiteStore) SaveSection(section *Section) error {
	result, err := s.db.Exec(`
		INSERT OR REPLACE INTO sections (book_id, file_name, epub_type, title, text_content, status, section_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, section.BookID, section.FileName, section.EpubType, section.Title, section.TextContent, section.Status, section.Order)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	section.ID = id
	return nil
}

// GetSection retrieves a section by ID.
func (s *SQLiteStore) GetSection(id int64) (*Section, error) {
	var sec Section
	err := s.db.QueryRow(`
		SELECT id, book_id, file_name, epub_type, title, text_content, status, section_order, created_at
		FROM sections WHERE id = ?
	`, id).Scan(&sec.ID, &sec.BookID, &sec.FileName, &sec.EpubType, &sec.Title, &sec.TextContent, &sec.Status, &sec.Order, &sec.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sec, nil
}

// ListSectionsByBook returns all sections for a book.
func (s *SQLiteStore) ListSectionsByBook(bookID int64) ([]Section, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, file_name, epub_type, title, text_content, status, section_order, created_at
		FROM sections WHERE book_id = ? ORDER BY section_order
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var sec Section
		if err := rows.Scan(&sec.ID, &sec.BookID, &sec.FileName, &sec.EpubType, &sec.Title, &sec.TextContent, &sec.Status, &sec.Order, &sec.CreatedAt); err != nil {
			return nil, err
		}
		sections = append(sections, sec)
	}
	return sections, rows.Err()
}

// ListSectionsByBookAndStatus returns sections filtered by status.
func (s *SQLiteStore) ListSectionsByBookAndStatus(bookID int64, status string) ([]Section, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, file_name, epub_type, title, text_content, status, section_order, created_at
		FROM sections WHERE book_id = ? AND status = ? ORDER BY section_order
	`, bookID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var sec Section
		if err := rows.Scan(&sec.ID, &sec.BookID, &sec.FileName, &sec.EpubType, &sec.Title, &sec.TextContent, &sec.Status, &sec.Order, &sec.CreatedAt); err != nil {
			return nil, err
		}
		sections = append(sections, sec)
	}
	return sections, rows.Err()
}

// UpdateSectionStatus updates the status of a section.
func (s *SQLiteStore) UpdateSectionStatus(id int64, status string) error {
	result, err := s.db.Exec("UPDATE sections SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetAllowedTextForBook returns concatenated text from all ALLOW sections.
func (s *SQLiteStore) GetAllowedTextForBook(bookID int64) (string, error) {
	rows, err := s.db.Query(`
		SELECT text_content FROM sections
		WHERE book_id = ? AND status = 'ALLOW'
		ORDER BY section_order
	`, bookID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var texts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", err
		}
		texts = append(texts, text)
	}
	return strings.Join(texts, "\n\n"), rows.Err()
}
