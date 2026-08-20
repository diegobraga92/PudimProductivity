package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
)

type FeatureFlagRepository struct {
	pool *pgxpool.Pool
}

func NewFeatureFlagRepository(pool *pgxpool.Pool) *FeatureFlagRepository {
	return &FeatureFlagRepository{pool: pool}
}

func (r *FeatureFlagRepository) ListEnabled(ctx context.Context) ([]featureflag.FeatureFlag, error) {
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

	var flags []featureflag.FeatureFlag
	for rows.Next() {
		var f featureflag.FeatureFlag
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
		flags = make([]featureflag.FeatureFlag, 0)
	}

	return flags, nil
}

func (r *FeatureFlagRepository) GetByName(ctx context.Context, name string) (*featureflag.FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE name = $1
	`

	var f featureflag.FeatureFlag
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

func (r *FeatureFlagRepository) GetByID(ctx context.Context, id string) (*featureflag.FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE id = $1
	`

	var f featureflag.FeatureFlag
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

func (r *FeatureFlagRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
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
