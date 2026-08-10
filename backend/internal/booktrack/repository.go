package booktrack

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a book does not exist.
var ErrNotFound = errors.New("book not found")

// ErrDuplicateISBN is returned when adding a book whose ISBN already exists.
var ErrDuplicateISBN = errors.New("book with this ISBN already exists")

// Repository persists books.
type Repository interface {
	// Create persists a book. A duplicate ISBN returns ErrDuplicateISBN.
	Create(ctx context.Context, book *Book) error
	// GetByID returns a book or ErrNotFound.
	GetByID(ctx context.Context, id string) (*Book, error)
	// GetByISBN returns a book or ErrNotFound.
	GetByISBN(ctx context.Context, isbn string) (*Book, error)
	// List returns books ordered most-recently-added first, optionally
	// filtered by reading status.
	List(ctx context.Context, status string) ([]*Book, error)
	// UpdateStatus changes the reading status.
	UpdateStatus(ctx context.Context, id string, status BookStatus) error
	// Delete removes a book.
	Delete(ctx context.Context, id string) error
}
