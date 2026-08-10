// Package booktrack implements the Book Tracking module (Phase 5): books are
// added by ISBN (metadata looked up via the Google Books API through the
// googlebooks adapter) or manually, then tracked through
// want_to_read / reading / read.
package booktrack

import (
	"fmt"
	"strings"
	"time"
)

// BookStatus is the reading-progress state of a book.
type BookStatus string

const (
	StatusWantToRead BookStatus = "want_to_read"
	StatusReading    BookStatus = "reading"
	StatusRead       BookStatus = "read"
)

func (s BookStatus) Valid() bool {
	switch s {
	case StatusWantToRead, StatusReading, StatusRead:
		return true
	default:
		return false
	}
}

// Book is a tracked book.
type Book struct {
	ID            string
	ISBN          string
	Title         string
	Authors       []string
	Publisher     string
	PublishedDate string
	Description   string
	PageCount     int
	ThumbnailURL  string
	Status        BookStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewBook validates and builds a book.
func NewBook(id, isbn, title string, authors []string, publisher, publishedDate, description string, pageCount int, thumbnailURL string, status BookStatus) (*Book, error) {
	if id == "" {
		return nil, fmt.Errorf("book id cannot be empty")
	}
	isbn = NormalizeISBN(isbn)
	if isbn == "" {
		return nil, fmt.Errorf("book isbn cannot be empty")
	}
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("book title cannot be empty")
	}
	if !status.Valid() {
		return nil, fmt.Errorf("invalid book status %q", status)
	}
	if pageCount < 0 {
		return nil, fmt.Errorf("page count cannot be negative")
	}
	if authors == nil {
		authors = []string{}
	}
	return &Book{
		ID:            id,
		ISBN:          isbn,
		Title:         title,
		Authors:       authors,
		Publisher:     publisher,
		PublishedDate: publishedDate,
		Description:   description,
		PageCount:     pageCount,
		ThumbnailURL:  thumbnailURL,
		Status:        status,
	}, nil
}

// NormalizeISBN strips separators/spaces and uppercases (for ISBN-10 X).
func NormalizeISBN(isbn string) string {
	isbn = strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r == 'X', r == 'x':
			return r
		default:
			return -1
		}
	}, strings.TrimSpace(isbn))
	return strings.ToUpper(isbn)
}
