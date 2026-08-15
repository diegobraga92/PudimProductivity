package scoringsettings

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func setupScoringsettingsPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("pudimproductivity"),
		postgres.WithUsername("pudim"),
		postgres.WithPassword("pudim_dev"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := db.ConnectPool(ctx, shared.DatabaseConfig{
		URL:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("ConnectPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}

func TestPostgresRepository_ConfigLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupScoringsettingsPostgres(t)
	repo := NewPostgresRepository(pool)

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

	// Save a config + provider; verify it persists and marks saved.
	now := time.Now().UTC()
	want := Config{
		MovieProvider:  "omdb",
		SeriesProvider: "omdb",
		GameProvider:   "rawg",
		BookProvider:   "none",
		SavedAt:        &now,
	}
	if err := repo.SaveConfig(ctx, want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := repo.SaveProvider(ctx, Provider{Name: "rawg", APIKey: "secret-key", BaseURL: "https://rawg.example"}); err != nil {
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
	var rawg *Provider
	for i := range gotProviders {
		if gotProviders[i].Name == "rawg" {
			rawg = &gotProviders[i]
		}
	}
	if rawg == nil || rawg.APIKey != "secret-key" || rawg.BaseURL != "https://rawg.example" {
		t.Fatalf("rawg provider = %+v", rawg)
	}
}
