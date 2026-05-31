package db

import (
	"auth/utils"
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresParams struct {
	Pool *pgxpool.Pool
}

func connectPostgres(c context.Context, url string) (*PostgresParams, error) {
	ctx, cancel := context.WithTimeout(c, utils.TimeoutDuration)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	slog.Info("Successfully connected to Postgres pool")

	return &PostgresParams{Pool: pool}, nil
}

func (p *PostgresParams) disconnectPostgres() error {
	slog.Info("Disconnecting from Postgres...")
	p.Pool.Close()
	return nil
}
