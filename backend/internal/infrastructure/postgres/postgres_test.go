package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

func TestConnectPool_Select1(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, dbCfg := postgrestest.Setup(t)

	pool, err := postgres.ConnectPool(ctx, dbCfg)
	if err != nil {
		t.Fatalf("ConnectPool failed: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	var result int32
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("SELECT 1 failed: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}

	t.Log("SELECT 1 returned 1 — database connectivity verified")
}

func TestConnectPool_QueryWithPoolConfig(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, dbCfg := postgrestest.Setup(t)

	// Build a pool manually to verify the config parsing works
	config, err := pgxpool.ParseConfig(dbCfg.URL)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	config.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	defer pool.Close()

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := pool.Ping(pingCtx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Run a simple query to confirm the pool works
	var now time.Time
	err = pool.QueryRow(ctx, "SELECT NOW()").Scan(&now)
	if err != nil {
		t.Fatalf("SELECT NOW() failed: %v", err)
	}

	t.Logf("Database time: %s", now.Format(time.RFC3339))
}

func TestConnectPool_InvalidConnectionString(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbCfg := config.DatabaseConfig{
		URL: "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable",
	}

	// An invalid connection string should cause ConnectPool to return an error
	_, err := postgres.ConnectPool(ctx, dbCfg)
	if err == nil {
		t.Fatal("expected an error for an invalid connection string, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

func TestConnectPool_ConnectionRefused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbCfg := config.DatabaseConfig{
		URL: "postgres://pudim:pudim_dev@localhost:15432/pudimproductivity?sslmode=disable",
	}

	// A valid-looking URL pointing to a port where nothing is listening
	_, err := postgres.ConnectPool(ctx, dbCfg)
	if err == nil {
		t.Fatal("expected an error when connection is refused, got nil")
	}

	t.Logf("Got expected error: %v", err)
}
