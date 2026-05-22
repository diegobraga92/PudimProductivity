package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations executes all pending SQL migration files in order.
// It creates a migrations tracking table and applies any files that
// have not yet been run.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Ensure the migrations tracking table exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename    TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Read embedded migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	// Sort by filename to ensure order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Check if already applied
		var applied bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)",
			entry.Name(),
		).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			log.Debug().Str("migration", entry.Name()).Msg("migration already applied, skipping")
			continue
		}

		// Read and execute the migration
		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", entry.Name(), err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin transaction for %s: %w", entry.Name(), err)
		}

		_, err = tx.Exec(ctx, string(content))
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}

		_, err = tx.Exec(ctx,
			"INSERT INTO schema_migrations (filename) VALUES ($1)",
			entry.Name(),
		)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}

		log.Info().Str("migration", entry.Name()).Msg("migration applied successfully")
	}

	return nil
}
