package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresParams struct {
	Pool *pgxpool.Pool
}

func connectPostgres(url string) (*PostgresParams, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Successfully connected to Postgres pool")

	return &PostgresParams{Pool: pool}, nil
}

func (p *PostgresParams) disconnectPostgres() error {
	log.Println("Disconnecting from Postgres...")
	p.Pool.Close()
	return nil
}
