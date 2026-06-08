package task

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

func (r *PostgresTaskRepository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, title, status, recurrence_days, list_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		task.ID,
		task.Title,
		string(task.Status),
		task.RecurrenceDays,
		task.ListID,
		task.CreatedAt,
		task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

func (r *PostgresTaskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	query := `
		SELECT id, title, status, recurrence_days, list_id, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	task := &Task{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		(*string)(&task.Status),
		&task.RecurrenceDays,
		&task.ListID,
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

func (r *PostgresTaskRepository) List(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error) {
	query := `
		SELECT id, title, status, recurrence_days, list_id, created_at, updated_at
		FROM tasks
		WHERE list_id IS NULL
	`
	args := make([]any, 0)

	if statusFilter != "" {
		query += " AND status = $1"
		args = append(args, statusFilter)
	}

	if typeFilter != "" {
		switch typeFilter {
		case "one-off":
			query += " AND recurrence_days IS NULL"
		case "habit":
			query += " AND recurrence_days IS NOT NULL"
		}
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
			&task.ListID,
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

func (r *PostgresTaskRepository) ListByListID(ctx context.Context, listID, typeFilter string) ([]*Task, error) {
	query := `
		SELECT id, title, status, recurrence_days, list_id, created_at, updated_at
		FROM tasks
		WHERE list_id = $1
	`
	args := []any{listID}

	if typeFilter != "" {
		switch typeFilter {
		case "one-off":
			query += " AND recurrence_days IS NULL"
		case "habit":
			query += " AND recurrence_days IS NOT NULL"
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks by list id: %w", err)
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
			&task.ListID,
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

func (r *PostgresTaskRepository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks
		SET title = $1, status = $2, recurrence_days = $3, list_id = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := r.pool.Exec(ctx, query,
		task.Title,
		string(task.Status),
		task.RecurrenceDays,
		task.ListID,
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
		return ErrCompletionAlreadyExists
	}

	return nil
}

func (r *PostgresTaskRepository) DeleteCompletion(ctx context.Context, taskID string, date time.Time) error {
	query := `DELETE FROM task_completions WHERE task_id = $1 AND completed_date = $2`

	result, err := r.pool.Exec(ctx, query, taskID, date)
	if err != nil {
		return fmt.Errorf("delete task completion: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCompletionNotFound
	}

	return nil
}

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

func (r *PostgresTaskRepository) ListAllCompletions(ctx context.Context, from, to time.Time) ([]*TaskCompletion, error) {
	query := `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE completed_date >= $1 AND completed_date <= $2
		ORDER BY task_id, completed_date ASC
	`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("list all task completions: %w", err)
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
