package featureflag

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListEnabled(ctx context.Context) ([]FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE enabled = true
		ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list enabled feature flags: %w", err)
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		err := rows.Scan(
			&f.ID,
			&f.Name,
			&f.Description,
			&f.Enabled,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feature flag row: %w", err)
		}
		flags = append(flags, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature flag rows: %w", err)
	}

	if flags == nil {
		flags = make([]FeatureFlag, 0)
	}

	return flags, nil
}

func (r *PostgresRepository) GetByName(ctx context.Context, name string) (*FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE name = $1
	`

	var f FeatureFlag
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&f.ID,
		&f.Name,
		&f.Description,
		&f.Enabled,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get feature flag by name: %w", err)
	}

	return &f, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE id = $1
	`

	var f FeatureFlag
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID,
		&f.Name,
		&f.Description,
		&f.Enabled,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get feature flag by id: %w", err)
	}

	return &f, nil
}

func (r *PostgresRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	query := `
		UPDATE feature_flags
		SET enabled = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, enabled, id)
	if err != nil {
		return fmt.Errorf("set feature flag enabled: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("feature flag not found: %s", id)
	}

	return nil
}
