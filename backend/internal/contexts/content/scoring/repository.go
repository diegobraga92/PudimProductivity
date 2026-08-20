package scoring

import "context"

// Repository persists the score-provider configuration.
type Repository interface {
	// GetConfig returns the singleton media-type → provider mapping.
	GetConfig(ctx context.Context) (Config, error)

	// SaveConfig persists the mapping and marks it as explicitly saved, after
	// which environment defaults are no longer overlaid.
	SaveConfig(ctx context.Context, cfg Config) error

	// GetProviders returns every registered provider row (name, api_key, base_url).
	GetProviders(ctx context.Context) ([]Provider, error)

	// SaveProvider upserts a single provider row.
	SaveProvider(ctx context.Context, p Provider) error
}
