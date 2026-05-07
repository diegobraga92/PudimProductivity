package task

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTaskRepository implements TaskRepository using PostgreSQL.
type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTaskRepository creates a new PostgresTaskRepository.
func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

// Create persists a new task.
func (r *PostgresTaskRepository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, title, status, recurrence_days, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		task.ID,
		task.Title,
		string(task.Status),
		task.RecurrenceDays,
		task.CreatedAt,
		task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by its ID.
func (r *PostgresTaskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	query := `
		SELECT id, title, status, recurrence_days, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	task := &Task{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		(*string)(&task.Status),
		&task.RecurrenceDays,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}

// List returns all tasks, optionally filtered by status.
func (r *PostgresTaskRepository) List(ctx context.Context, statusFilter string) ([]*Task, error) {
	query := `
		SELECT id, title, status, recurrence_days, created_at, updated_at
		FROM tasks
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1

	if statusFilter != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, statusFilter)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}

		err := rows.Scan(
			&task.ID,
			&task.Title,
			(*string)(&task.Status),
			&task.RecurrenceDays,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}

	if tasks == nil {
		tasks = make([]*Task, 0)
	}

	return tasks, nil
}

// Update persists changes to an existing task.
func (r *PostgresTaskRepository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks
		SET title = $1, status = $2, recurrence_days = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := r.pool.Exec(ctx, query,
		task.Title,
		string(task.Status),
		task.RecurrenceDays,
		task.UpdatedAt,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// Delete removes a task by its ID.
func (r *PostgresTaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// CreateCompletion records a habit completion for a specific date.
func (r *PostgresTaskRepository) CreateCompletion(ctx context.Context, completion *TaskCompletion) error {
	query := `
		INSERT INTO task_completions (id, task_id, completed_date, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, completed_date) DO NOTHING
	`

	result, err := r.pool.Exec(ctx, query,
		completion.ID,
		completion.TaskID,
		completion.CompletedDate,
		completion.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task completion: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("completion already exists for task %s on %s", completion.TaskID, completion.CompletedDate.Format("2006-01-02"))
	}

	return nil
}

// DeleteCompletion removes a habit completion for a specific task+date.
func (r *PostgresTaskRepository) DeleteCompletion(ctx context.Context, taskID string, date time.Time) error {
	query := `DELETE FROM task_completions WHERE task_id = $1 AND completed_date = $2`

	result, err := r.pool.Exec(ctx, query, taskID, date)
	if err != nil {
		return fmt.Errorf("delete task completion: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// GetCompletion retrieves a single completion for a task on a specific date.
func (r *PostgresTaskRepository) GetCompletion(ctx context.Context, taskID string, date time.Time) (*TaskCompletion, error) {
	query := `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE task_id = $1 AND completed_date = $2
	`

	completion := &TaskCompletion{}

	err := r.pool.QueryRow(ctx, query, taskID, date).Scan(
		&completion.ID,
		&completion.TaskID,
		&completion.CompletedDate,
		&completion.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get task completion: %w", err)
	}

	return completion, nil
}

// ListCompletions returns all completions for a task within a date range (inclusive).
func (r *PostgresTaskRepository) ListCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*TaskCompletion, error) {
	query := `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE task_id = $1 AND completed_date >= $2 AND completed_date <= $3
		ORDER BY completed_date ASC
	`

	rows, err := r.pool.Query(ctx, query, taskID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list task completions: %w", err)
	}
	defer rows.Close()

	var completions []*TaskCompletion
	for rows.Next() {
		c := &TaskCompletion{}

		err := rows.Scan(
			&c.ID,
			&c.TaskID,
			&c.CompletedDate,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task completion row: %w", err)
		}

		completions = append(completions, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task completion rows: %w", err)
	}

	if completions == nil {
		completions = make([]*TaskCompletion, 0)
	}

	return completions, nil
}
