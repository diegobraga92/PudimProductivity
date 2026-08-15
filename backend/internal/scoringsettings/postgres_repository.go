package scoringsettings

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetConfig(ctx context.Context) (Config, error) {
	var cfg Config
	err := r.pool.QueryRow(ctx, `
		SELECT movie_provider, series_provider, game_provider, book_provider, saved_at
		FROM score_provider_config
		WHERE id = 1
	`).Scan(&cfg.MovieProvider, &cfg.SeriesProvider, &cfg.GameProvider, &cfg.BookProvider, &cfg.SavedAt)
	if err != nil {
		return Config{}, fmt.Errorf("get score provider config: %w", err)
	}
	return cfg, nil
}

func (r *PostgresRepository) SaveConfig(ctx context.Context, cfg Config) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO score_provider_config
			(id, movie_provider, series_provider, game_provider, book_provider, saved_at, updated_at)
		VALUES (1, $1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			movie_provider  = EXCLUDED.movie_provider,
			series_provider = EXCLUDED.series_provider,
			game_provider   = EXCLUDED.game_provider,
			book_provider   = EXCLUDED.book_provider,
			saved_at        = COALESCE(score_provider_config.saved_at, NOW()),
			updated_at      = NOW()
	`, cfg.MovieProvider, cfg.SeriesProvider, cfg.GameProvider, cfg.BookProvider)
	if err != nil {
		return fmt.Errorf("save score provider config: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetProviders(ctx context.Context) ([]Provider, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT name, api_key, base_url
		FROM score_providers
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list score providers: %w", err)
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.Name, &p.APIKey, &p.BaseURL); err != nil {
			return nil, fmt.Errorf("scan score provider row: %w", err)
		}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate score providers: %w", err)
	}
	if providers == nil {
		providers = make([]Provider, 0)
	}
	return providers, nil
}

func (r *PostgresRepository) SaveProvider(ctx context.Context, p Provider) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO score_providers (name, api_key, base_url, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (name) DO UPDATE SET
			api_key    = EXCLUDED.api_key,
			base_url   = EXCLUDED.base_url,
			updated_at = NOW()
	`, p.Name, p.APIKey, p.BaseURL)
	if err != nil {
		return fmt.Errorf("save score provider %q: %w", p.Name, err)
	}
	return nil
}
