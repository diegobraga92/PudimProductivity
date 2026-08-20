package library

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a library item does not exist.
var ErrNotFound = errors.New("library item not found")

// ListFilter carries the optional filters for List. Empty/zero values mean
// "no filter" for that dimension.
type ListFilter struct {
	MediaType string // "" = all media types
	Done      *bool  // nil = any status
	Subtype   string // "" = all subtypes (case-insensitive exact match when set)
}

// Repository persists library items.
type Repository interface {
	// Create persists a single item.
	Create(ctx context.Context, item *Item) error
	// GetByID returns an item or ErrNotFound.
	GetByID(ctx context.Context, id string) (*Item, error)
	// List returns items ordered most-recently-added first, filtered by the
	// given ListFilter (zero values = no filter).
	List(ctx context.Context, filter ListFilter) ([]*Item, error)
	// DistinctSubtypes returns the distinct non-empty subtype values, sorted
	// alphabetically, optionally scoped to a media type ("" = all types).
	DistinctSubtypes(ctx context.Context, mediaType string) ([]string, error)
	// Update overwrites the editable fields of an item. Returns ErrNotFound
	// when the item does not exist.
	Update(ctx context.Context, item *Item) error
	// Delete removes an item. Returns ErrNotFound when it does not exist.
	Delete(ctx context.Context, id string) error
	// Import persists a batch of items in a single transaction.
	Import(ctx context.Context, items []*Item) error
}
