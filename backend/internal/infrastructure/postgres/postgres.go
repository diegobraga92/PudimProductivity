package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

const dbPingTimeout = 5 * time.Second

func ConnectPool(ctx context.Context, dbCfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dbCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(dbCfg.MaxConns)
	poolConfig.MinConns = int32(dbCfg.MinConns)
	poolConfig.MaxConnLifetime = dbCfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = dbCfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, dbPingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info().Int32("max_conns", poolConfig.MaxConns).Msg("database connection pool established")
	return pool, nil
}
