package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/scoring"
)

type ScoringRepository struct {
	pool *pgxpool.Pool
}

func NewScoringRepository(pool *pgxpool.Pool) *ScoringRepository {
	return &ScoringRepository{pool: pool}
}

func (r *ScoringRepository) GetConfig(ctx context.Context) (scoring.Config, error) {
	var cfg scoring.Config
	err := r.pool.QueryRow(ctx, `
		SELECT movie_provider, series_provider, game_provider, book_provider, saved_at
		FROM score_provider_config
		WHERE id = 1
	`).Scan(&cfg.MovieProvider, &cfg.SeriesProvider, &cfg.GameProvider, &cfg.BookProvider, &cfg.SavedAt)
	if err != nil {
		return scoring.Config{}, fmt.Errorf("get score provider config: %w", err)
	}
	return cfg, nil
}

func (r *ScoringRepository) SaveConfig(ctx context.Context, cfg scoring.Config) error {
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

func (r *ScoringRepository) GetProviders(ctx context.Context) ([]scoring.Provider, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT name, api_key, base_url
		FROM score_providers
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list score providers: %w", err)
	}
	defer rows.Close()

	var providers []scoring.Provider
	for rows.Next() {
		var p scoring.Provider
		if err := rows.Scan(&p.Name, &p.APIKey, &p.BaseURL); err != nil {
			return nil, fmt.Errorf("scan score provider row: %w", err)
		}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate score providers: %w", err)
	}
	if providers == nil {
		providers = make([]scoring.Provider, 0)
	}
	return providers, nil
}

func (r *ScoringRepository) SaveProvider(ctx context.Context, p scoring.Provider) error {
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
