package db

import (
	"log"
	"os"
)

type DatabaseParams struct {
	Postgres *PostgresParams
}

func Setup() (*DatabaseParams, error) {
	postgresUrl := os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		log.Fatal("POSTGRES_URL environment variable is not set")
	}

	postgres, err := connectPostgres(postgresUrl)
	if err != nil {
		return nil, err
	}

	log.Println("Postgres initialized")

	return &DatabaseParams{
		Postgres: postgres,
	}, nil
}

func (s *DatabaseParams) Cleanup() error {
	log.Println("Closing database connections")

	if err := s.Postgres.disconnectPostgres(); err != nil {
		log.Printf("Error disconnecting Postgres: %v", err)
	}

	return nil
}
