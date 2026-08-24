package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/sync/persistence"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/tasklist"
)

type SyncRepository struct {
	pool *pgxpool.Pool
}

func NewSyncRepository(pool *pgxpool.Pool) *SyncRepository {
	return &SyncRepository{pool: pool}
}

func (r *SyncRepository) Bundle(ctx context.Context, since time.Time) (*persistence.ChangeSet, error) {
	cs := &persistence.ChangeSet{Timestamp: time.Now().UTC()}

	if err := r.loadTasks(ctx, since, cs); err != nil {
		return nil, err
	}
	if err := r.loadCompletions(ctx, since, cs); err != nil {
		return nil, err
	}
	if err := r.loadTaskLists(ctx, since, cs); err != nil {
		return nil, err
	}
	if err := r.loadShares(ctx, since, cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (r *SyncRepository) loadTasks(ctx context.Context, since time.Time, cs *persistence.ChangeSet) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, status, recurrence_days, list_id, start_time, end_time,
		       color, scheduled_date, alarm_minutes, created_at, updated_at
		FROM tasks
		WHERE deleted_at IS NULL AND (created_at > $1 OR updated_at > $1)
		ORDER BY updated_at ASC
	`, since)
	if err != nil {
		return fmt.Errorf("sync tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		t := &task.Task{}
		// pgx cannot scan a DATE column into *string directly in binary mode —
		// mirror task.scanTask and use an intermediate *time.Time for
		// scheduled_date (start_time/end_time TIME columns scan into *string fine).
		var scheduledDate *time.Time
		if err := rows.Scan(
			&t.ID, &t.Title, (*string)(&t.Status), &t.RecurrenceDays, &t.ListID,
			&t.StartTime, &t.EndTime, &t.Color, &scheduledDate, &t.AlarmMinutes,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return fmt.Errorf("scan sync task: %w", err)
		}
		if scheduledDate != nil {
			s := scheduledDate.Format("2006-01-02")
			t.ScheduledDate = &s
		}
		cs.Tasks = append(cs.Tasks, t)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sync tasks: %w", err)
	}

	dRows, err := r.pool.Query(ctx, `
		SELECT id FROM tasks WHERE deleted_at IS NOT NULL AND deleted_at > $1
	`, since)
	if err != nil {
		return fmt.Errorf("sync deleted tasks: %w", err)
	}
	defer dRows.Close()
	for dRows.Next() {
		var id string
		if err := dRows.Scan(&id); err != nil {
			return fmt.Errorf("scan deleted task id: %w", err)
		}
		cs.DeletedTaskIDs = append(cs.DeletedTaskIDs, id)
	}
	return dRows.Err()
}

func (r *SyncRepository) loadCompletions(ctx context.Context, since time.Time, cs *persistence.ChangeSet) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE deleted_at IS NULL AND created_at > $1
		ORDER BY created_at ASC
	`, since)
	if err != nil {
		return fmt.Errorf("sync completions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		c := &task.TaskCompletion{}
		if err := rows.Scan(&c.ID, &c.TaskID, &c.CompletedDate, &c.CreatedAt); err != nil {
			return fmt.Errorf("scan sync completion: %w", err)
		}
		cs.Completions = append(cs.Completions, c)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sync completions: %w", err)
	}

	dRows, err := r.pool.Query(ctx, `
		SELECT id FROM task_completions WHERE deleted_at IS NOT NULL AND deleted_at > $1
	`, since)
	if err != nil {
		return fmt.Errorf("sync deleted completions: %w", err)
	}
	defer dRows.Close()
	for dRows.Next() {
		var id string
		if err := dRows.Scan(&id); err != nil {
			return fmt.Errorf("scan deleted completion id: %w", err)
		}
		cs.DeletedCompletionIDs = append(cs.DeletedCompletionIDs, id)
	}
	return dRows.Err()
}

func (r *SyncRepository) loadTaskLists(ctx context.Context, since time.Time, cs *persistence.ChangeSet) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM task_lists
		WHERE deleted_at IS NULL AND (created_at > $1 OR updated_at > $1)
		ORDER BY updated_at ASC
	`, since)
	if err != nil {
		return fmt.Errorf("sync task lists: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		l := &tasklist.TaskList{}
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.OwnerID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return fmt.Errorf("scan sync task list: %w", err)
		}
		cs.TaskLists = append(cs.TaskLists, l)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sync task lists: %w", err)
	}

	dRows, err := r.pool.Query(ctx, `
		SELECT id FROM task_lists WHERE deleted_at IS NOT NULL AND deleted_at > $1
	`, since)
	if err != nil {
		return fmt.Errorf("sync deleted task lists: %w", err)
	}
	defer dRows.Close()
	for dRows.Next() {
		var id string
		if err := dRows.Scan(&id); err != nil {
			return fmt.Errorf("scan deleted task list id: %w", err)
		}
		cs.DeletedTaskListIDs = append(cs.DeletedTaskListIDs, id)
	}
	return dRows.Err()
}

func (r *SyncRepository) loadShares(ctx context.Context, since time.Time, cs *persistence.ChangeSet) error {
	rows, err := r.pool.Query(ctx, `
		SELECT list_id, shared_with, role, created_at
		FROM task_list_shares
		WHERE deleted_at IS NULL AND created_at > $1
		ORDER BY created_at ASC
	`, since)
	if err != nil {
		return fmt.Errorf("sync shares: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		s := &tasklist.Share{}
		if err := rows.Scan(&s.ListID, &s.SharedWith, &s.Role, &s.CreatedAt); err != nil {
			return fmt.Errorf("scan sync share: %w", err)
		}
		cs.Shares = append(cs.Shares, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sync shares: %w", err)
	}

	dRows, err := r.pool.Query(ctx, `
		SELECT list_id, shared_with FROM task_list_shares
		WHERE deleted_at IS NOT NULL AND deleted_at > $1
	`, since)
	if err != nil {
		return fmt.Errorf("sync deleted shares: %w", err)
	}
	defer dRows.Close()
	for dRows.Next() {
		var listID, sharedWith string
		if err := dRows.Scan(&listID, &sharedWith); err != nil {
			return fmt.Errorf("scan deleted share key: %w", err)
		}
		cs.DeletedShareKeys = append(cs.DeletedShareKeys, listID+":"+sharedWith)
	}
	return dRows.Err()
}
