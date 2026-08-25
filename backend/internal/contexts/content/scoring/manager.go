package scoring

import (
	"context"
	"fmt"
	"sync"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/scoringapi"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

// Manager is a reloadable library.ScoreLookupProvider, so provider configuration
// changes take effect immediately without a backend restart.
type Manager struct {
	mu      sync.RWMutex
	current library.ScoreLookupProvider
}

func NewManager(initial library.ScoreLookupProvider) *Manager {
	if initial == nil {
		initial = library.NoopScoreLookup{}
	}
	return &Manager{current: initial}
}

// Search calls the provider's search.
func (m *Manager) Search(ctx context.Context, query library.ScoreQuery) ([]library.ScoreCandidate, error) {
	m.mu.RLock()
	current := m.current
	m.mu.RUnlock()
	if current == nil {
		return nil, nil
	}
	return current.Search(ctx, query)
}

// Configured reports whether the active client can serve lookups.
func (m *Manager) Configured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current != nil && m.current.Configured()
}

// Current returns the active provider.
func (m *Manager) Current() library.ScoreLookupProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Reload rebuilds the composite from cfg and swaps it in atomically.
func (m *Manager) Reload(ctx context.Context, cfg config.ScoreProviderConfig) error {
	client, err := scoringapi.NewComposite(ctx, cfg)
	if err != nil {
		return fmt.Errorf("reload score providers: %w", err)
	}
	provider, ok := client.(library.ScoreLookupProvider)
	if !ok {
		return fmt.Errorf("reload score providers: client %T is not a ScoreLookupProvider", client)
	}
	m.mu.Lock()
	m.current = provider
	m.mu.Unlock()
	return nil
}
