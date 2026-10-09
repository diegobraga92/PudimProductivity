package mealplan

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a meal plan item does not exist.
var ErrNotFound = errors.New("meal plan item not found")

// Repository persists the recurring weekly meal plan.
type Repository interface {
	// CreateBatch persists all items atomically.
	CreateBatch(ctx context.Context, items []*Item) error
	// List returns every item of the weekly template.
	List(ctx context.Context) ([]*Item, error)
	// Update replaces the mutable fields of an existing item.
	Update(ctx context.Context, item *Item) error
	// Delete removes an item by id.
	Delete(ctx context.Context, id string) error
}
