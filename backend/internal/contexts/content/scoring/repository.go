package scoring

import "context"

// Repository persists the score-provider configuration.
type Repository interface {
	// GetConfig returns the singleton media-type → provider mapping.
	GetConfig(ctx context.Context) (Config, error)

	// SaveConfig persists the mapping.
	SaveConfig(ctx context.Context, cfg Config) error

	// GetProviders returns every registered provider row.
	GetProviders(ctx context.Context) ([]Provider, error)

	// SaveProvider saves a single provider row.
	SaveProvider(ctx context.Context, p Provider) error
}
