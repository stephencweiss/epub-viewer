package storage

import "errors"

// Common errors
var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrAuthorHasBooks = errors.New("author has books and cannot be deleted")
)

// Store defines the interface for book storage operations.
type Store interface {
	// Author operations
	CreateAuthor(name string) (*Author, error)
	GetAuthor(id int64) (*Author, error)
	GetAuthorByName(name string) (*Author, error)
	ListAuthors() ([]Author, error)
	FindSimilarAuthors(name string) ([]Author, error)

	// Book operations
	AddBook(authorID int64, title, path, language, publisher string) (*Book, error)
	GetBook(id int64) (*Book, error)
	GetBookByPath(path string) (*Book, error)
	ListBooks() ([]Book, error)
	ListBooksByAuthor(authorID int64) ([]Book, error)
	RemoveBook(id int64) error

	// Analysis operations
	SaveAnalysis(analysis *StoredAnalysis) error
	GetAnalysis(bookID int64) (*StoredAnalysis, error)

	// Corpus operations
	GetCorpusAnalysis(authorID int64) (*CorpusAnalysis, error)

	// Lifecycle
	Close() error
}
