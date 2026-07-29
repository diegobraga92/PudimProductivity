package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Insert(ctx context.Context, entry *Entry) error {
	query := `
		INSERT INTO audit_log (id, actor_id, action, resource, resource_id, old_values, new_values, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	oldValues := entry.OldValues
	if oldValues == nil {
		oldValues = json.RawMessage("null")
	}
	newValues := entry.NewValues
	if newValues == nil {
		newValues = json.RawMessage("null")
	}

	_, err := r.pool.Exec(ctx, query,
		entry.ID,
		entry.ActorID,
		entry.Action,
		entry.Resource,
		entry.ResourceID,
		oldValues,
		newValues,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert audit log entry: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListByResource(ctx context.Context, resource string, resourceID string, limit, offset int) ([]Entry, error) {
	query := `
		SELECT id, actor_id, action, resource, resource_id, old_values, new_values, created_at
		FROM audit_log
		WHERE resource = $1 AND resource_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, resource, resourceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit log by resource: %w", err)
	}
	defer rows.Close()

	entries, err := scanEntries(rows)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *PostgresRepository) ListByActor(ctx context.Context, actorID string, limit, offset int) ([]Entry, error) {
	query := `
		SELECT id, actor_id, action, resource, resource_id, old_values, new_values, created_at
		FROM audit_log
		WHERE actor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, actorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit log by actor: %w", err)
	}
	defer rows.Close()

	entries, err := scanEntries(rows)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *PostgresRepository) ListByAction(ctx context.Context, action string, since time.Time, limit, offset int) ([]Entry, error) {
	var query string
	var args []any

	if action == "" {
		query = `
			SELECT id, actor_id, action, resource, resource_id, old_values, new_values, created_at
			FROM audit_log
			WHERE created_at >= $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []any{since, limit, offset}
	} else {
		query = `
			SELECT id, actor_id, action, resource, resource_id, old_values, new_values, created_at
			FROM audit_log
			WHERE action = $1 AND created_at >= $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []any{action, since, limit, offset}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit log by action: %w", err)
	}
	defer rows.Close()

	entries, err := scanEntries(rows)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func scanEntries(rows pgx.Rows) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		var e Entry
		err := rows.Scan(
			&e.ID,
			&e.ActorID,
			&e.Action,
			&e.Resource,
			&e.ResourceID,
			&e.OldValues,
			&e.NewValues,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan audit log entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit log rows: %w", err)
	}

	if entries == nil {
		entries = make([]Entry, 0)
	}

	return entries, nil
}
