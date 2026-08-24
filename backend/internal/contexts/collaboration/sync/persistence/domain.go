// Package persistence serves the incremental offline-sync endpoint. Offline
// clients (mobile Room DB) use this to converge with the server after being disconnected.
package persistence

import (
	"context"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/tasklist"
)

// Repository reads changed rows for the incremental sync bundle.
type Repository interface {
	// Bundle returns everything that changed after `since` plus the current server timestamp.
	Bundle(ctx context.Context, since time.Time) (*ChangeSet, error)
}

// ChangeSet is the domain result of an incremental sync query: every row created
// or updated after `since`, plus the IDs/keys of rows soft-deleted after `since`.
//
// It is deliberately domain-shaped (task.Task, tasklist.Share, ...) rather than
// a mirror of the sync HTTP response. The handler in this package (handler.go)
// maps it to the wire Bundle using the canonical API response types, so the
// sync endpoint cannot silently drift from the task/tasklist endpoints.
type ChangeSet struct {
	Timestamp            time.Time
	Tasks                []*task.Task
	DeletedTaskIDs       []string
	Completions          []*task.TaskCompletion
	DeletedCompletionIDs []string
	TaskLists            []*tasklist.TaskList
	DeletedTaskListIDs   []string
	Shares               []*tasklist.Share
	DeletedShareKeys     []string
}
