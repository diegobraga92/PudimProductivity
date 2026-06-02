package featureflag

import "context"

type Repository interface {
	// ListEnabled returns all feature flags that are currently enabled.
	ListEnabled(ctx context.Context) ([]FeatureFlag, error)

	// GetByName retrieves a single feature flag by its name.
	GetByName(ctx context.Context, name string) (*FeatureFlag, error)

	// GetByID retrieves a single feature flag by its ID.
	GetByID(ctx context.Context, id string) (*FeatureFlag, error)

	// SetEnabled toggles a feature flag on or off.
	SetEnabled(ctx context.Context, id string, enabled bool) error
}