package audit

import (
	"context"
	"time"
)

type Repository interface {
	// Insert writes a new audit log entry.
	Insert(ctx context.Context, entry *Entry) error

	// ListByResource returns audit entries for a specific resource, ordered by most recent first.
	ListByResource(ctx context.Context, resource string, resourceID string, limit, offset int) ([]Entry, error)

	// ListByActor returns audit entries for a specific actor, ordered by most recent first.
	ListByActor(ctx context.Context, actorID string, limit, offset int) ([]Entry, error)

	// ListByAction returns audit entries for a specific action type, ordered by most recent first.
	ListByAction(ctx context.Context, action string, since time.Time, limit, offset int) ([]Entry, error)
}
