// Package postgrestest provides a disposable Postgres container for
// integration tests.
package postgrestest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	testpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

// postgresImage should match the version used by docker-compose.yml and CI
const postgresImage = "postgres:18-alpine"

// SkipIfShort skips the calling test when run with -short. Every test that
// boots a container should call it before Setup or SetupPool.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
}

// Setup starts a fresh Postgres container and returns its config.
func Setup(t *testing.T) (context.Context, config.DatabaseConfig) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pgContainer, err := testpg.Run(ctx, postgresImage,
		testpg.WithDatabase("pudimproductivity"),
		testpg.WithUsername("pudim"),
		testpg.WithPassword("pudim_dev"),
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

	return ctx, config.DatabaseConfig{
		URL:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	}
}

// SetupPool is Setup plus a connected, migrated pool.
func SetupPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, dbCfg := Setup(t)

	pool, err := postgres.ConnectPool(ctx, dbCfg)
	if err != nil {
		t.Fatalf("ConnectPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}
