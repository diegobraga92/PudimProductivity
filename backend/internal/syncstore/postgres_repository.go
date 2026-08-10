package syncstore

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository reads changed rows for the incremental sync bundle.
type Repository interface {
	// Bundle returns everything that changed after `since` plus the current
	// server timestamp.
	Bundle(ctx context.Context, since time.Time) (*Bundle, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Bundle(ctx context.Context, since time.Time) (*Bundle, error) {
	b := &Bundle{Timestamp: time.Now().UTC().Format(time.RFC3339)}

	if err := r.loadTasks(ctx, since, b); err != nil {
		return nil, err
	}
	if err := r.loadCompletions(ctx, since, b); err != nil {
		return nil, err
	}
	if err := r.loadTaskLists(ctx, since, b); err != nil {
		return nil, err
	}
	if err := r.loadShares(ctx, since, b); err != nil {
		return nil, err
	}

	if b.Tasks == nil {
		b.Tasks = make([]TaskDTO, 0)
	}
	if b.DeletedTaskIDs == nil {
		b.DeletedTaskIDs = make([]string, 0)
	}
	if b.Completions == nil {
		b.Completions = make([]CompletionDTO, 0)
	}
	if b.DeletedCompletionIDs == nil {
		b.DeletedCompletionIDs = make([]string, 0)
	}
	if b.TaskLists == nil {
		b.TaskLists = make([]TaskListDTO, 0)
	}
	if b.DeletedTaskListIDs == nil {
		b.DeletedTaskListIDs = make([]string, 0)
	}
	if b.Shares == nil {
		b.Shares = make([]ShareDTO, 0)
	}
	if b.DeletedShareKeys == nil {
		b.DeletedShareKeys = make([]string, 0)
	}
	return b, nil
}

func (r *PostgresRepository) loadTasks(ctx context.Context, since time.Time, b *Bundle) error {
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
		var t TaskDTO
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Status, &t.RecurrenceDays, &t.ListID,
			&t.StartTime, &t.EndTime, &t.Color, &t.ScheduledDate, &t.AlarmMinutes,
			&createdAt, &updatedAt,
		); err != nil {
			return fmt.Errorf("scan sync task: %w", err)
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		t.UpdatedAt = updatedAt.Format(time.RFC3339)
		b.Tasks = append(b.Tasks, t)
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
		b.DeletedTaskIDs = append(b.DeletedTaskIDs, id)
	}
	return dRows.Err()
}

func (r *PostgresRepository) loadCompletions(ctx context.Context, since time.Time, b *Bundle) error {
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
		var c CompletionDTO
		var date, createdAt time.Time
		if err := rows.Scan(&c.ID, &c.TaskID, &date, &createdAt); err != nil {
			return fmt.Errorf("scan sync completion: %w", err)
		}
		c.CompletedDate = date.Format("2006-01-02")
		c.CreatedAt = createdAt.Format(time.RFC3339)
		b.Completions = append(b.Completions, c)
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
		b.DeletedCompletionIDs = append(b.DeletedCompletionIDs, id)
	}
	return dRows.Err()
}

func (r *PostgresRepository) loadTaskLists(ctx context.Context, since time.Time, b *Bundle) error {
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
		var l TaskListDTO
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.OwnerID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return fmt.Errorf("scan sync task list: %w", err)
		}
		b.TaskLists = append(b.TaskLists, l)
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
		b.DeletedTaskListIDs = append(b.DeletedTaskListIDs, id)
	}
	return dRows.Err()
}

func (r *PostgresRepository) loadShares(ctx context.Context, since time.Time, b *Bundle) error {
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
		var s ShareDTO
		if err := rows.Scan(&s.ListID, &s.SharedWith, &s.Role, &s.CreatedAt); err != nil {
			return fmt.Errorf("scan sync share: %w", err)
		}
		b.Shares = append(b.Shares, s)
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
		b.DeletedShareKeys = append(b.DeletedShareKeys, listID+":"+sharedWith)
	}
	return dRows.Err()
}
