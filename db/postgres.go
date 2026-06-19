package db

import (
	"auth/internal/timeout"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresParams struct {
	Pool *pgxpool.Pool
}

func connectPostgres(c context.Context, url string) (*PostgresParams, error) {
	ctx, cancel := context.WithTimeout(c, timeout.Duration)
	defer cancel()

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pool config: %w", err)
	}

	config.MaxConns = 3
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.MaxConnLifetimeJitter = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	slog.LogAttrs(ctx, slog.LevelInfo, "Successfully connected to Postgres pool",
		slog.Int("maxConns", int(config.MaxConns)),
		slog.Int("minConns", int(config.MinConns)),
	)

	return &PostgresParams{Pool: pool}, nil
}

func (p *PostgresParams) disconnectPostgres() error {
	slog.LogAttrs(context.Background(), slog.LevelInfo, "Disconnecting from Postgres...")
	p.Pool.Close()
	return nil
}
