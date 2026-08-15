package scoring

import (
	"context"
	"fmt"
	"sync"

	"github.com/diegobraga92/pudimproductivity/backend/internal/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Manager is a reloadable library.ScoreLookupProvider. It holds the current
// composite behind a mutex and swaps it atomically on Reload, so provider
// configuration changes made through the admin UI take effect immediately
// without a backend restart.
type Manager struct {
	mu      sync.RWMutex
	current library.ScoreLookupProvider
}

// NewManager builds a Manager starting from initial (nil means Noop, i.e.
// "not configured"). The initial client is typically the result of a startup
// load, and can be replaced later via Reload.
func NewManager(initial library.ScoreLookupProvider) *Manager {
	if initial == nil {
		initial = library.NoopScoreLookup{}
	}
	return &Manager{current: initial}
}

// Search delegates to the active client. Satisfies library.ScoreLookupClient.
func (m *Manager) Search(ctx context.Context, query library.ScoreQuery) ([]library.ScoreCandidate, error) {
	m.mu.RLock()
	current := m.current
	m.mu.RUnlock()
	if current == nil {
		return nil, nil
	}
	return current.Search(ctx, query)
}

// Configured reports whether the active client can serve lookups. Satisfies
// library.ScoreLookupProvider.
func (m *Manager) Configured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current != nil && m.current.Configured()
}

// Current returns the active client (readers may also use Search directly).
func (m *Manager) Current() library.ScoreLookupProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Reload rebuilds the composite from cfg and swaps it in atomically. On error
// the previous client stays active (a failed config change never degrades the
// running lookup).
func (m *Manager) Reload(ctx context.Context, cfg shared.ScoreProviderConfig) error {
	client, err := NewComposite(ctx, cfg)
	if err != nil {
		return fmt.Errorf("reload score providers: %w", err)
	}
	provider, ok := client.(library.ScoreLookupProvider)
	if !ok {
		// NewComposite only ever returns *composite or NoopScoreLookup, both of
		// which implement the provider interface. Defensive fallback.
		return fmt.Errorf("reload score providers: client %T is not a ScoreLookupProvider", client)
	}
	m.mu.Lock()
	m.current = provider
	m.mu.Unlock()
	return nil
}
