package main

import (
	"auth/controller"
	"auth/db"
	"auth/middlewares"
	"auth/utils"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"buf.build/go/protovalidate"
	"github.com/Aditya-0011/common/contracts/go/auth"
	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.LogAttrs(context.Background(), slog.LevelInfo, "runtime config",
		slog.Int("GOMAXPROCS", runtime.GOMAXPROCS(0)),
		slog.Int("NumCPU", runtime.NumCPU()),
		slog.Int64("GOMEMLIMIT_MiB", debug.SetMemoryLimit(-1)/1024/1024),
	)

	setupCtx, setupCancel := context.WithTimeout(context.Background(), utils.TimeoutDuration)
	defer setupCancel()

	database, err := db.Setup(setupCtx)
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "Failed to setup databases", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer database.Cleanup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "7295"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	validator, err := protovalidate.New()
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "failed to initialize validator", slog.String("error", err.Error()))
		os.Exit(1)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(middlewares.ValidationInterceptor(validator)),
		grpc.MaxConcurrentStreams(50),
	)

	auth.RegisterAuthServiceServer(s, controller.NewServer(controller.ServerParams{
		Postgres: database.Postgres,
	}))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)

	go func() {
		slog.LogAttrs(context.Background(), slog.LevelInfo, "gRPC server listening", slog.String("address", lis.Addr().String()))
		if err := s.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-quit:
		slog.Info("Interrupt received. Starting graceful shutdown...")
	case err := <-errChan:
		slog.LogAttrs(context.Background(), slog.LevelError, "gRPC server failed", slog.String("error", err.Error()))
		slog.Info("Starting graceful shutdown due to server error...")
	}

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	shutdownTimer := time.NewTimer(utils.TimeoutDuration)
	defer shutdownTimer.Stop()

	select {
	case <-shutdownTimer.C:
		slog.Info("Timeout reached. Forcing server shutdown...")
		s.Stop()
	case <-stopped:
		slog.Info("Server gracefully stopped within timeout.")
	}

	slog.Info("Shutdown complete. Exiting main...")
}
