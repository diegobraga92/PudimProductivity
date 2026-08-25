package featureflag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Feature flags are per-deployment (not per-user), with a local in-memory cache
type Service struct {
	repo        Repository
	cache       map[string]bool // flag name → enabled state
	cacheMu     sync.RWMutex
	cacheTTL    time.Duration // 0 disables caching entirely
	lastRefresh time.Time
}

func NewService(repo Repository, cacheTTL time.Duration) *Service {
	return &Service{
		repo:     repo,
		cache:    make(map[string]bool),
		cacheTTL: cacheTTL, // Set to 0 to disable
	}
}

func (s *Service) IsEnabled(ctx context.Context, name string) (bool, error) {
	if s.cacheTTL > 0 {
		s.cacheMu.RLock()
		if time.Since(s.lastRefresh) < s.cacheTTL {
			if enabled, ok := s.cache[name]; ok {
				s.cacheMu.RUnlock()
				return enabled, nil
			}
		}
		s.cacheMu.RUnlock()
	}

	flag, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return false, fmt.Errorf("check feature flag %q: %w", name, err)
	}

	if flag == nil {
		return false, nil
	}

	if s.cacheTTL > 0 {
		s.cacheMu.Lock()
		s.cache[name] = flag.Enabled
		s.lastRefresh = time.Now()
		s.cacheMu.Unlock()
	}

	return flag.Enabled, nil
}

func (s *Service) ListEnabled(ctx context.Context) ([]FeatureFlag, error) {
	return s.repo.ListEnabled(ctx)
}

func (s *Service) RefreshCache() {
	if s.cacheTTL > 0 {
		s.cacheMu.Lock()
		s.lastRefresh = time.Time{}
		s.cacheMu.Unlock()
	}
}

func (s *Service) GetByName(ctx context.Context, name string) (*FeatureFlag, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *Service) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if err := s.repo.SetEnabled(ctx, id, enabled); err != nil {
		return fmt.Errorf("set feature flag: %w", err)
	}

	s.RefreshCache()

	log.Info().
		Str("flag_id", id).
		Bool("enabled", enabled).
		Msg("feature flag toggled")

	return nil
}
