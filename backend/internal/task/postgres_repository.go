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
		INSERT INTO tasks (id, title, description, status, priority, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		task.ID,
		task.Title,
		task.Description,
		string(task.Status),
		string(task.Priority),
		task.DueDate,
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
		SELECT id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	task := &Task{}
	var description *string
	var dueDate *time.Time

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&description,
		(*string)(&task.Status),
		(*string)(&task.Priority),
		&dueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	task.Description = description
	task.DueDate = dueDate

	return task, nil
}

// List returns all tasks, optionally filtered by status and/or priority.
func (r *PostgresTaskRepository) List(ctx context.Context, statusFilter, priorityFilter string) ([]*Task, error) {
	query := `
		SELECT id, title, description, status, priority, due_date, created_at, updated_at
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
	if priorityFilter != "" {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, priorityFilter)
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
		var description *string
		var dueDate *time.Time

		err := rows.Scan(
			&task.ID,
			&task.Title,
			&description,
			(*string)(&task.Status),
			(*string)(&task.Priority),
			&dueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}

		task.Description = description
		task.DueDate = dueDate
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
		SET title = $1, description = $2, status = $3, priority = $4, due_date = $5, updated_at = $6
		WHERE id = $7
	`

	result, err := r.pool.Exec(ctx, query,
		task.Title,
		task.Description,
		string(task.Status),
		string(task.Priority),
		task.DueDate,
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
