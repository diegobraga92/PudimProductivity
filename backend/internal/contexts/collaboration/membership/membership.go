// Package membership (Phase 8) wires collaboration infrastructure into the server:
// a Postgres-backed task-list membership resolver for the sync hub (event
// scoping + presence) and the REST presence endpoint.
package membership

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresMembershipResolver answers "which task lists can this user access?"
// from task_lists.owner_id + task_list_shares.shared_with.
type PostgresMembershipResolver struct {
	pool *pgxpool.Pool
}

// NewPostgresMembershipResolver builds a resolver bound to the app pool.
func NewPostgresMembershipResolver(pool *pgxpool.Pool) *PostgresMembershipResolver {
	return &PostgresMembershipResolver{pool: pool}
}

// ListIDsForUser returns the IDs of the task lists the user owns or is a
// member of. Admins see every list (dev-mode role model).
func (r *PostgresMembershipResolver) ListIDsForUser(ctx context.Context, userID, role string) ([]string, error) {
	if userID == "" {
		return []string{}, nil
	}

	var query string
	if role == "admin" {
		query = `SELECT id FROM task_lists WHERE deleted_at IS NULL`
	} else {
		query = `
			SELECT DISTINCT list_id FROM (
				SELECT id AS list_id FROM task_lists WHERE owner_id = $1 AND deleted_at IS NULL
				UNION
				SELECT list_id FROM task_list_shares WHERE shared_with = $1 AND deleted_at IS NULL
			) t
		`
	}

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve membership: %w", err)
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
		return nil, fmt.Errorf("iterate list ids: %w", err)
	}
	if ids == nil {
		ids = make([]string, 0)
	}
	return ids, nil
}
