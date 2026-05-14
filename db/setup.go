package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type DatabaseParams struct {
	Postgres *PostgresParams
}

func Setup(ctx context.Context) (*DatabaseParams, error) {
	postgresUrl := os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		return nil, fmt.Errorf("POSTGRES_URL environment variable is not set")
	}

	postgres, err := connectPostgres(ctx, postgresUrl)
	if err != nil {
		return nil, err
	}

	slog.Info("Postgres initialized")

	return &DatabaseParams{
		Postgres: postgres,
	}, nil
}

func (s *DatabaseParams) Cleanup() error {
	slog.Info("Closing database connections")

	if err := s.Postgres.disconnectPostgres(); err != nil {
		slog.Error("Error disconnecting Postgres", "error", err)
	}

	return nil
}
