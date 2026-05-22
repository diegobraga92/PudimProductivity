package tasklist

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTaskListRepository implements TaskListRepository using PostgreSQL.
type PostgresTaskListRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTaskListRepository creates a new PostgresTaskListRepository.
func NewPostgresTaskListRepository(pool *pgxpool.Pool) *PostgresTaskListRepository {
	return &PostgresTaskListRepository{pool: pool}
}

// Create persists a new task list.
func (r *PostgresTaskListRepository) Create(ctx context.Context, list *TaskList) error {
	query := `
		INSERT INTO task_lists (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		list.ID,
		list.Name,
		list.Description,
		list.CreatedAt,
		list.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task list: %w", err)
	}

	return nil
}

// GetByID retrieves a task list by its ID.
func (r *PostgresTaskListRepository) GetByID(ctx context.Context, id string) (*TaskList, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM task_lists
		WHERE id = $1
	`

	list := &TaskList{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&list.ID,
		&list.Name,
		&list.Description,
		&list.CreatedAt,
		&list.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTaskListNotFound
		}
		return nil, fmt.Errorf("get task list by id: %w", err)
	}

	return list, nil
}

// List returns all task lists.
func (r *PostgresTaskListRepository) List(ctx context.Context) ([]*TaskList, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM task_lists
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list task lists: %w", err)
	}
	defer rows.Close()

	var lists []*TaskList
	for rows.Next() {
		list := &TaskList{}

		err := rows.Scan(
			&list.ID,
			&list.Name,
			&list.Description,
			&list.CreatedAt,
			&list.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task list row: %w", err)
		}

		lists = append(lists, list)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task list rows: %w", err)
	}

	if lists == nil {
		lists = make([]*TaskList, 0)
	}

	return lists, nil
}

// Update persists changes to an existing task list.
func (r *PostgresTaskListRepository) Update(ctx context.Context, list *TaskList) error {
	query := `
		UPDATE task_lists
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := r.pool.Exec(ctx, query,
		list.Name,
		list.Description,
		list.UpdatedAt,
		list.ID,
	)
	if err != nil {
		return fmt.Errorf("update task list: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskListNotFound
	}

	return nil
}

// Delete removes a task list by its ID.
func (r *PostgresTaskListRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM task_lists WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task list: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskListNotFound
	}

	return nil
}
