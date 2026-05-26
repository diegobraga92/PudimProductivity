package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func ConnectPool(ctx context.Context, envConfig shared.Config) (*pgxpool.Pool, error) {
	dbConfig, err := pgxpool.ParseConfig(envConfig.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	dbConfig.MaxConns = int32(envConfig.DatabaseMaxConns)
	dbConfig.MinConns = int32(envConfig.DatabaseMinConns)
	dbConfig.MaxConnLifetime = time.Duration(envConfig.DatabaseMaxConnLifetime) * time.Minute
	dbConfig.MaxConnIdleTime = time.Duration(envConfig.DatabaseMaxConnIdletime) * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info().Int32("max_conns", dbConfig.MaxConns).Msg("database connection pool established")
	return pool, nil
}
