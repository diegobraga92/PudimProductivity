package recipe

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a recipe does not exist.
var ErrNotFound = errors.New("recipe not found")

// ListFilter holds the optional filters for Repository.List. Empty values are
// ignored. Cursor is a keyset cursor encoded as (createdAt,id).
type ListFilter struct {
	Search     string
	Tags       []string
	Difficulty string
	Cursor     *time.Time
	CursorID   string
	Limit      int
}

// Repository persists recipes and their children.
type Repository interface {
	// Create persists the recipe plus tags, ingredients and steps atomically.
	Create(ctx context.Context, recipe *Recipe) error
	// GetByID returns the full recipe with all children; ErrNotFound if missing.
	GetByID(ctx context.Context, id string) (*Recipe, error)
	// List returns recipes matching the filter. Each result carries Tags (the
	// list view); children are NOT loaded. Ordered created_at DESC, id DESC.
	List(ctx context.Context, filter ListFilter) ([]*Recipe, error)
	// Update replaces the recipe row and its children atomically.
	Update(ctx context.Context, recipe *Recipe) error
	// Delete removes the recipe and (via CASCADE) its children.
	Delete(ctx context.Context, id string) error
}
