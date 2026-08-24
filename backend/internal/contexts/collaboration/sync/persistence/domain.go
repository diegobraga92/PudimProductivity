// Package persistence serves the incremental offline-sync endpoint. Offline
// clients (mobile Room DB) use this to converge with the server after being disconnected.
package persistence

import (
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/tasklist"
)

// ChangeSet is the domain result of an incremental sync query: every row created
// or updated after `since`, plus the IDs/keys of rows soft-deleted after `since`.
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
