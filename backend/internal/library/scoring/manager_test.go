package scoring

import (
	"context"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func TestManager_StartsNoop(t *testing.T) {
	m := NewManager(nil)
	if m.Configured() {
		t.Fatal("manager must start unconfigured")
	}
	cands, err := m.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame})
	if err != nil || len(cands) != 0 {
		t.Fatalf("noop Search = %v (err %v), want empty", cands, err)
	}
}

func TestManager_Reload_Configures(t *testing.T) {
	m := NewManager(nil)
	cfg := shared.ScoreProviderConfig{
		Game: "rawg",
		Keys: map[string]string{"rawg": "k"},
	}
	if err := m.Reload(context.Background(), cfg); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !m.Configured() {
		t.Fatal("manager must be configured after valid reload")
	}
	// A media type without a provider returns no candidates.
	cands, err := m.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie})
	if err != nil || len(cands) != 0 {
		t.Fatalf("Search(disabled movie) = %v (err %v), want empty", cands, err)
	}
}

func TestManager_ReloadEmpty_Deconfigures(t *testing.T) {
	m := NewManager(nil)
	cfg := shared.ScoreProviderConfig{
		Game: "rawg",
		Keys: map[string]string{"rawg": "k"},
	}
	if err := m.Reload(context.Background(), cfg); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if err := m.Reload(context.Background(), shared.ScoreProviderConfig{}); err != nil {
		t.Fatalf("Reload(empty): %v", err)
	}
	if m.Configured() {
		t.Fatal("empty config must deconfigure the manager")
	}
}

func TestManager_ReloadInvalid_KeepsPrevious(t *testing.T) {
	m := NewManager(nil)
	valid := shared.ScoreProviderConfig{
		Game: "rawg",
		Keys: map[string]string{"rawg": "k"},
	}
	if err := m.Reload(context.Background(), valid); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	invalid := shared.ScoreProviderConfig{Movie: "netflix", Keys: map[string]string{"netflix": "k"}}
	if err := m.Reload(context.Background(), invalid); err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !m.Configured() {
		t.Fatal("failed reload must keep the previous client active")
	}
}
