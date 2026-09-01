// Package db owns Postgres connectivity for the Cyto AI API.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a pgx connection pool, applies conservative MVP pool
// defaults, and verifies connectivity with a Ping. It returns an error rather
// than a pool when the database is unreachable.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}

	// MVP defaults; tune per environment later.
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// Ready reports whether the pool can currently reach Postgres. The Phase 7
// /readyz handler can call this directly.
func Ready(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
