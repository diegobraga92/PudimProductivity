package features

import (
	"context"
	"sync"
	"time"
)

// TODO: Check and rewrite all 'features' code (it was generated automatically)

// FeatureFlag represents a toggleable feature.
type FeatureFlag struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FeatureStore defines the interface for feature flag persistence.
type FeatureStore interface {
	// GetAll returns all feature flags.
	GetAll(ctx context.Context) ([]FeatureFlag, error)
	// GetByName returns a specific feature flag by name.
	GetByName(ctx context.Context, name string) (*FeatureFlag, error)
}

// CachedFeatureStore wraps a FeatureStore with an in-memory cache.
type CachedFeatureStore struct {
	store    FeatureStore
	mu       sync.RWMutex
	cache    map[string]FeatureFlag
	lastSync time.Time
	ttl      time.Duration
}

// NewCachedFeatureStore creates a new CachedFeatureStore.
func NewCachedFeatureStore(store FeatureStore, ttl time.Duration) *CachedFeatureStore {
	return &CachedFeatureStore{
		store: store,
		cache: make(map[string]FeatureFlag),
		ttl:   ttl,
	}
}

// GetAll returns all feature flags, using cache if fresh.
func (c *CachedFeatureStore) GetAll(ctx context.Context) ([]FeatureFlag, error) {
	if err := c.refreshIfStale(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	flags := make([]FeatureFlag, 0, len(c.cache))
	for _, f := range c.cache {
		flags = append(flags, f)
	}
	return flags, nil
}

// GetByName returns a specific feature flag by name.
func (c *CachedFeatureStore) GetByName(ctx context.Context, name string) (*FeatureFlag, error) {
	if err := c.refreshIfStale(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	flag, ok := c.cache[name]
	if !ok {
		return nil, nil
	}
	return &flag, nil
}

// IsEnabled checks if a feature is enabled by name.
func (c *CachedFeatureStore) IsEnabled(ctx context.Context, name string) (bool, error) {
	flag, err := c.GetByName(ctx, name)
	if err != nil {
		return false, err
	}
	if flag == nil {
		return false, nil
	}
	return flag.Enabled, nil
}

func (c *CachedFeatureStore) refreshIfStale(ctx context.Context) error {
	c.mu.RLock()
	needsRefresh := time.Since(c.lastSync) > c.ttl
	c.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	flags, err := c.store.GetAll(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]FeatureFlag, len(flags))
	for _, f := range flags {
		c.cache[f.Name] = f
	}
	c.lastSync = time.Now()

	return nil
}
