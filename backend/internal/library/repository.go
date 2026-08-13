package library

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a library item does not exist.
var ErrNotFound = errors.New("library item not found")

// Repository persists library items.
type Repository interface {
	// Create persists a single item.
	Create(ctx context.Context, item *Item) error
	// GetByID returns an item or ErrNotFound.
	GetByID(ctx context.Context, id string) (*Item, error)
	// List returns items ordered most-recently-added first, optionally
	// filtered by media type and/or done status (nil = no filter).
	List(ctx context.Context, mediaType string, done *bool) ([]*Item, error)
	// Update overwrites the editable fields of an item. Returns ErrNotFound
	// when the item does not exist.
	Update(ctx context.Context, item *Item) error
	// Delete removes an item. Returns ErrNotFound when it does not exist.
	Delete(ctx context.Context, id string) error
	// Import persists a batch of items in a single transaction.
	Import(ctx context.Context, items []*Item) error
}
