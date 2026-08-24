package persistence

import (
	"context"
	"time"
)

// Repository reads changed rows for the incremental sync bundle.
type Repository interface {
	// Bundle returns everything that changed after `since` plus the current server timestamp.
	Bundle(ctx context.Context, since time.Time) (*ChangeSet, error)
}
