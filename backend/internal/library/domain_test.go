package library

import (
	"testing"
	"time"
)

// TestNewItem_SetsTimestamps guards the create-response bug where CreatedAt /
// UpdatedAt were left at the zero time ("0001-01-01T00:00:00Z") because NewItem
// never initialized them, so POST /api/v1/library returned wrong timestamps.
func TestNewItem_SetsTimestamps(t *testing.T) {
	item, err := NewItem("id-1", "The Matrix", MediaTypeMovie, nil, false, "", nil, "", "")
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	if item.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set, got zero time")
	}
	if item.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set, got zero time")
	}
	if time.Since(item.CreatedAt) > time.Minute {
		t.Fatalf("CreatedAt not recent: %v", item.CreatedAt)
	}
}
