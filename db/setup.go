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

	slog.LogAttrs(context.Background(), slog.LevelInfo, "Postgres initialized")

	return &DatabaseParams{
		Postgres: postgres,
	}, nil
}

func (s *DatabaseParams) Cleanup() error {
	slog.LogAttrs(context.Background(), slog.LevelInfo, "Closing database connections")

	if err := s.Postgres.disconnectPostgres(); err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "Error disconnecting Postgres", slog.String("error", err.Error()))
	}

	return nil
}
