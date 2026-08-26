package postgres_test

import (
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/scoring"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
)

func TestPostgresRepository_ConfigLifecycle(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	repo := postgres.NewScoringRepository(pool)

	// Migration seeds a never-saved empty config + the two providers.
	cfg, err := repo.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.SavedAt != nil {
		t.Fatalf("config starts saved: %+v", cfg)
	}
	providers, err := repo.GetProviders(ctx)
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	names := map[string]bool{}
	for _, p := range providers {
		names[p.Name] = true
	}
	if !names["omdb"] || !names["rawg"] {
		t.Fatalf("providers seeded = %+v, want omdb+rawg", names)
	}

	// Save a config + provider. Verify it persists and marks saved.
	now := time.Now().UTC()
	want := scoring.Config{
		MovieProvider:  "omdb",
		SeriesProvider: "omdb",
		GameProvider:   "rawg",
		BookProvider:   "none",
		SavedAt:        &now,
	}
	if err := repo.SaveConfig(ctx, want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := repo.SaveProvider(ctx, scoring.Provider{Name: "rawg", APIKey: "secret-key", BaseURL: "https://rawg.example"}); err != nil {
		t.Fatalf("SaveProvider: %v", err)
	}

	got, err := repo.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got.MovieProvider != "omdb" || got.GameProvider != "rawg" || got.BookProvider != "none" {
		t.Fatalf("saved config = %+v", got)
	}
	if got.SavedAt == nil {
		t.Fatal("config must be marked saved after SaveConfig")
	}

	gotProviders, err := repo.GetProviders(ctx)
	if err != nil {
		t.Fatalf("GetProviders: %v", err)
	}
	var rawg *scoring.Provider
	for i := range gotProviders {
		if gotProviders[i].Name == "rawg" {
			rawg = &gotProviders[i]
		}
	}
	if rawg == nil || rawg.APIKey != "secret-key" || rawg.BaseURL != "https://rawg.example" {
		t.Fatalf("rawg provider = %+v", rawg)
	}
}
