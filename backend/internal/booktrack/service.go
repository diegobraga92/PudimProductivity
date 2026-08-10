package booktrack

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/booktrack/googlebooks"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// LookupClient resolves ISBN metadata from an external source. Google Books is
// the production implementation; tests use a stub.
type LookupClient interface {
	LookupByISBN(ctx context.Context, isbn string) (*googlebooks.BookInfo, error)
}

// AddInput is the manual-book entry payload.
type AddInput struct {
	ISBN          string
	Title         string
	Authors       []string
	Publisher     string
	PublishedDate string
	Description   string
	PageCount     int
	ThumbnailURL  string
	Status        BookStatus
}

// Service coordinates the book-tracking module.
type BookService struct {
	repo   Repository
	lookup LookupClient // nil = external lookup disabled (degraded mode)
	audit  audit.Logger
	bus    eventbus.Bus
}

func NewBookService(repo Repository, lookup LookupClient, auditLogger audit.Logger, bus eventbus.Bus) *BookService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &BookService{repo: repo, lookup: lookup, audit: auditLogger, bus: bus}
}

// AddByISBN looks up metadata from Google Books and persists the book.
func (s *BookService) AddByISBN(ctx context.Context, isbn string) (*Book, error) {
	if s.lookup == nil {
		return nil, errors.New("external book lookup is not configured")
	}
	isbn = NormalizeISBN(isbn)
	if isbn == "" {
		return nil, errors.New("isbn cannot be empty")
	}

	info, err := s.lookup.LookupByISBN(ctx, isbn)
	if err != nil {
		return nil, err
	}

	book, err := NewBook(
		shared.NewUUID(), info.ISBN, info.Title, info.Authors,
		info.Publisher, info.PublishedDate, info.Description,
		info.PageCount, info.ThumbnailURL, StatusWantToRead,
	)
	if err != nil {
		return nil, fmt.Errorf("add book by isbn: %w", err)
	}
	return s.persist(ctx, book)
}

// AddManual persists a book from client-provided fields (no external lookup).
func (s *BookService) AddManual(ctx context.Context, in AddInput) (*Book, error) {
	status := in.Status
	if status == "" {
		status = StatusWantToRead
	}
	book, err := NewBook(
		shared.NewUUID(), in.ISBN, in.Title, in.Authors,
		in.Publisher, in.PublishedDate, in.Description,
		in.PageCount, in.ThumbnailURL, status,
	)
	if err != nil {
		return nil, err
	}
	return s.persist(ctx, book)
}

func (s *BookService) persist(ctx context.Context, book *Book) (*Book, error) {
	if err := s.repo.Create(ctx, book); err != nil {
		return nil, err // ErrDuplicateISBN passes through
	}

	log.Info().Ctx(ctx).Str("book_id", book.ID).Str("isbn", book.ISBN).Str("title", book.Title).Msg("book added")
	s.audit.Log(ctx, audit.ActionBookAdded, audit.ResourceBooks, book.ID, nil, map[string]any{
		"isbn":  book.ISBN,
		"title": book.Title,
	})
	s.publish(ctx, eventbus.EventBookAdded, toResponse(book))
	return book, nil
}

func (s *BookService) Get(ctx context.Context, id string) (*Book, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BookService) List(ctx context.Context, status string) ([]*Book, error) {
	return s.repo.List(ctx, status)
}

func (s *BookService) UpdateStatus(ctx context.Context, id string, status BookStatus) error {
	if !status.Valid() {
		return fmt.Errorf("invalid book status %q", status)
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return err // ErrNotFound passes through
	}
	s.audit.Log(ctx, audit.ActionBookUpdated, audit.ResourceBooks, id, nil, map[string]any{"status": status})
	return nil
}

func (s *BookService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err // ErrNotFound passes through
	}
	s.audit.Log(ctx, audit.ActionBookDeleted, audit.ResourceBooks, id, nil, nil)
	return nil
}

func (s *BookService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish book event")
	}
}
