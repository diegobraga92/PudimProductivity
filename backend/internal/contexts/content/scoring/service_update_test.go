package scoring

import (
	"context"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

func TestService_Update_PersistsAndReloads(t *testing.T) {
	repo := &fakeRepo{providers: []Provider{{Name: "rawg", APIKey: "old"}}}
	reloader := &fakeReloader{}
	svc := NewService(repo, nil, reloader, nil, config.ScoreProviderConfig{})

	req := UpdateConfigRequest{
		GameProvider:  "rawg",
		MovieProvider: "none",
		LookupEnabled: boolPtr(true),
		Providers:     []UpdateProviderRequest{{Name: "rawg", APIKey: strPtr("new")}},
	}
	api, err := svc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if api.GameProvider != "rawg" {
		t.Fatalf("Update result GameProvider = %q", api.GameProvider)
	}
	if reloader.last.Game != "rawg" || reloader.last.Keys["rawg"] != "new" {
		t.Fatalf("reloaded config = %+v", reloader.last)
	}
	if repo.cfg.GameProvider != "rawg" || repo.cfg.SavedAt == nil {
		t.Fatalf("config not persisted as saved: %+v", repo.cfg)
	}
	if got := repo.providers[0].APIKey; got != "new" {
		t.Fatalf("stored rawg key = %q, want new", got)
	}
}

func TestService_Update_NilAPIKeyKeepsExisting(t *testing.T) {
	repo := &fakeRepo{providers: []Provider{{Name: "rawg", APIKey: "keep-me"}}}
	reloader := &fakeReloader{}
	svc := NewService(repo, nil, reloader, nil, config.ScoreProviderConfig{})

	req := UpdateConfigRequest{
		GameProvider: "rawg",
		Providers:    []UpdateProviderRequest{{Name: "rawg"}}, // no api_key → keep
	}
	if _, err := svc.Update(context.Background(), req); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := repo.providers[0].APIKey; got != "keep-me" {
		t.Fatalf("rawg key = %q, want keep-me", got)
	}
}

func TestService_Update_InvalidConfigDoesNotPersist(t *testing.T) {
	repo := &fakeRepo{providers: []Provider{{Name: "rawg", APIKey: "k"}}}
	reloader := &fakeReloader{}
	svc := NewService(repo, nil, reloader, nil, config.ScoreProviderConfig{})

	req := UpdateConfigRequest{
		MovieProvider: "rawg", // rawg cannot serve movies
		Providers:     []UpdateProviderRequest{{Name: "rawg"}},
	}
	if _, err := svc.Update(context.Background(), req); err == nil {
		t.Fatal("expected validation error")
	}
	if repo.cfg.SavedAt != nil {
		t.Fatal("invalid config must not be persisted")
	}
	if reloader.last.Game != "" {
		t.Fatalf("invalid config must not be reloaded: %+v", reloader.last)
	}
}

func TestService_Update_ManagerIntegration(t *testing.T) {
	// Use the real scoring.Manager as the reloader and confirm the runtime
	// lookup becomes configured after a save.
	repo := &fakeRepo{providers: []Provider{{Name: "rawg"}}}
	manager := NewManager(nil)
	svc := NewService(repo, nil, manager, nil, config.ScoreProviderConfig{})

	if manager.Configured() {
		t.Fatal("manager must start unconfigured")
	}
	req := UpdateConfigRequest{
		GameProvider: "rawg",
		Providers:    []UpdateProviderRequest{{Name: "rawg", APIKey: strPtr("k")}},
	}
	if _, err := svc.Update(context.Background(), req); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !manager.Configured() {
		t.Fatal("manager must be configured after a valid save")
	}
}
