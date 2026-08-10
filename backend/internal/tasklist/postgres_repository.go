package tasklist

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTaskListRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTaskListRepository(pool *pgxpool.Pool) *PostgresTaskListRepository {
	return &PostgresTaskListRepository{pool: pool}
}

func (r *PostgresTaskListRepository) Create(ctx context.Context, list *TaskList) error {
	query := `
		INSERT INTO task_lists (id, name, description, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		list.ID,
		list.Name,
		list.Description,
		list.OwnerID,
		list.CreatedAt,
		list.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task list: %w", err)
	}

	return nil
}

func (r *PostgresTaskListRepository) GetByID(ctx context.Context, id string) (*TaskList, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM task_lists
		WHERE id = $1
	`

	list := &TaskList{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&list.ID,
		&list.Name,
		&list.Description,
		&list.OwnerID,
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

func (r *PostgresTaskListRepository) List(ctx context.Context) ([]*TaskList, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at
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
			&list.OwnerID,
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

// --- Phase 8: collaboration / sharing ---

func (r *PostgresTaskListRepository) GetMemberRole(ctx context.Context, listID, userID string) (Role, error) {
	// Owner check first.
	var ownerID string
	err := r.pool.QueryRow(ctx,
		`SELECT owner_id FROM task_lists WHERE id = $1`, listID,
	).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrTaskListNotFound
		}
		return "", fmt.Errorf("get task list owner: %w", err)
	}
	if ownerID == userID {
		return RoleOwner, nil
	}

	// Share check.
	var role string
	err = r.pool.QueryRow(ctx,
		`SELECT role FROM task_list_shares WHERE list_id = $1 AND shared_with = $2`,
		listID, userID,
	).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrTaskListAccessDenied
		}
		return "", fmt.Errorf("get share role: %w", err)
	}
	return Role(role), nil
}

func (r *PostgresTaskListRepository) CreateShare(ctx context.Context, share *Share) error {
	query := `
		INSERT INTO task_list_shares (list_id, shared_with, role, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (list_id, shared_with) DO NOTHING
	`

	result, err := r.pool.Exec(ctx, query,
		share.ListID,
		share.SharedWith,
		string(share.Role),
		share.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task list share: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrShareExists
	}
	return nil
}

func (r *PostgresTaskListRepository) DeleteShare(ctx context.Context, listID, userID string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM task_list_shares WHERE list_id = $1 AND shared_with = $2`,
		listID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete task list share: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrShareNotFound
	}
	return nil
}

func (r *PostgresTaskListRepository) ListShares(ctx context.Context, listID string) ([]*Share, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT list_id, shared_with, role, created_at
		 FROM task_list_shares
		 WHERE list_id = $1
		 ORDER BY created_at ASC`, listID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task list shares: %w", err)
	}
	defer rows.Close()

	var shares []*Share
	for rows.Next() {
		s := &Share{}
		if err := rows.Scan(&s.ListID, &s.SharedWith, &s.Role, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan share row: %w", err)
		}
		shares = append(shares, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate share rows: %w", err)
	}
	if shares == nil {
		shares = make([]*Share, 0)
	}
	return shares, nil
}

func (r *PostgresTaskListRepository) ListListsForUser(ctx context.Context, userID string) ([]*TaskList, error) {
	query := `
		SELECT DISTINCT l.id, l.name, l.description, l.owner_id, l.created_at, l.updated_at
		FROM task_lists l
		LEFT JOIN task_list_shares s ON s.list_id = l.id
		WHERE l.owner_id = $1 OR s.shared_with = $1
		ORDER BY l.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list task lists for user: %w", err)
	}
	defer rows.Close()

	var lists []*TaskList
	for rows.Next() {
		list := &TaskList{}
		if err := rows.Scan(
			&list.ID, &list.Name, &list.Description,
			&list.OwnerID, &list.CreatedAt, &list.UpdatedAt,
		); err != nil {
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

func (r *PostgresTaskListRepository) ListListIDsForUser(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT list_id FROM (
			SELECT id AS list_id FROM task_lists WHERE owner_id = $1
			UNION
			SELECT list_id FROM task_list_shares WHERE shared_with = $1
		) t
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list task list ids for user: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan list id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate list id rows: %w", err)
	}
	return ids, nil
}

func (r *PostgresTaskListRepository) ListMemberUserIDs(ctx context.Context, listID string) ([]string, error) {
	query := `
		SELECT DISTINCT user_id FROM (
			SELECT owner_id AS user_id FROM task_lists WHERE id = $1
			UNION
			SELECT shared_with AS user_id FROM task_list_shares WHERE list_id = $1
		) t
	`

	rows, err := r.pool.Query(ctx, query, listID)
	if err != nil {
		return nil, fmt.Errorf("list member user ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan member id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate member id rows: %w", err)
	}
	return ids, nil
}
