package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MembershipRepository answers "which task lists can this user access?" from
// task_lists.owner_id + task_list_shares.shared_with. It implements
// membership.Repository and satisfies the sync hub's MembershipResolver.
type MembershipRepository struct {
	pool *pgxpool.Pool
}

// NewMembershipRepository builds a repository bound to the app pool.
func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{pool: pool}
}

// ListIDsForUser returns the IDs of the task lists the user owns or is a
// member of. Admins see every list (dev-mode role model).
func (r *MembershipRepository) ListIDsForUser(ctx context.Context, userID, role string) ([]string, error) {
	if userID == "" {
		return []string{}, nil
	}

	var (
		rows pgx.Rows
		err  error
	)
	if role == "admin" {
		// This branch has no $1 placeholder, so userID must not be passed:
		// pgx uses the extended protocol whenever arguments are present, and
		// PostgreSQL rejects a bind that supplies more parameters than the
		// statement declares.
		rows, err = r.pool.Query(ctx, `SELECT id FROM task_lists WHERE deleted_at IS NULL`)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT DISTINCT list_id FROM (
				SELECT id AS list_id FROM task_lists WHERE owner_id = $1 AND deleted_at IS NULL
				UNION
				SELECT list_id FROM task_list_shares WHERE shared_with = $1 AND deleted_at IS NULL
			) t
		`, userID)
	}
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
