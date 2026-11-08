package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"auth/controller"
	"auth/db"

	"buf.build/go/protovalidate"
	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/Aditya-0011/common/contracts/go/grpcx"
)

const (
	defaultPort     = "7295"
	startupTimeout  = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("auth service failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	slog.Info("runtime config",
		slog.Int("GOMAXPROCS", runtime.GOMAXPROCS(0)),
		slog.Int64("GOMEMLIMIT_MiB", debug.SetMemoryLimit(-1)/1024/1024),
	)

	serverConfig, err := grpcx.ServerConfigFromEnv(defaultPort)
	if err != nil {
		return err
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		return errors.New("POSTGRES_URL is not set")
	}
	allowBootstrap := os.Getenv("AUTH_ALLOW_BOOTSTRAP") == "true"
	if allowBootstrap {
		slog.Warn("AUTH_ALLOW_BOOTSTRAP is enabled: the first login creates the account if none exists")
	}

	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	pool, err := db.Connect(ctx, postgresURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("connected to postgres")

	validator, err := protovalidate.New()
	if err != nil {
		return err
	}

	srv, err := grpcx.NewServer(serverConfig, validator)
	if err != nil {
		return err
	}
	auth.RegisterAuthServiceServer(srv, controller.NewServer(pool, allowBootstrap))

	return srv.Run(shutdownTimeout)
}
