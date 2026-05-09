package main

import (
	"auth/controller"
	"auth/db"
	"auth/middlewares"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"buf.build/go/protovalidate"
	auth "github.com/Aditya-0011/common/contracts/go/auth"
	"google.golang.org/grpc"
)

func main() {
	database, err := db.Setup()
	if err != nil {
		slog.Error("Failed to setup databases", "error", err)
		os.Exit(1)
	}
	defer database.Cleanup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "7295"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	validator, err := protovalidate.New()
	if err != nil {
		slog.Error("failed to initialize validator", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(middlewares.ValidationInterceptor(validator)))

	auth.RegisterAuthServiceServer(s, controller.NewServer(controller.ServerParams{
		Postgres: database.Postgres,
	}))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("gRPC server listening", "address", lis.Addr())
		if err := s.Serve(lis); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	<-quit
	slog.Info("Interrupt received. Starting graceful shutdown...")

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-time.After(5 * time.Second):
		slog.Info("Timeout reached (5s). Forcing server shutdown...")
		s.Stop()
	case <-stopped:
		slog.Info("Server gracefully stopped within timeout.")
	}

	slog.Info("Shutdown complete. Exiting main...")
}
