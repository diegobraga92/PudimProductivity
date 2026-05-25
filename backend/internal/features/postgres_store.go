package features

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresFeatureStore implements FeatureStore using PostgreSQL.
type PostgresFeatureStore struct {
	pool *pgxpool.Pool
}

// NewPostgresFeatureStore creates a new PostgresFeatureStore.
func NewPostgresFeatureStore(pool *pgxpool.Pool) *PostgresFeatureStore {
	return &PostgresFeatureStore{pool: pool}
}

// GetAll returns all feature flags from the database.
func (s *PostgresFeatureStore) GetAll(ctx context.Context) ([]FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		ORDER BY name
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query feature flags: %w", err)
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan feature flag: %w", err)
		}
		flags = append(flags, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature flags: %w", err)
	}

	if flags == nil {
		flags = make([]FeatureFlag, 0)
	}

	return flags, nil
}

// GetByName returns a specific feature flag by name.
func (s *PostgresFeatureStore) GetByName(ctx context.Context, name string) (*FeatureFlag, error) {
	query := `
		SELECT id, name, description, enabled, created_at, updated_at
		FROM feature_flags
		WHERE name = $1
	`

	var f FeatureFlag
	err := s.pool.QueryRow(ctx, query, name).Scan(
		&f.ID, &f.Name, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query feature flag by name: %w", err)
	}

	return &f, nil
}
