package main

import (
	"auth/controller"
	"auth/db"
	"auth/middlewares"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"buf.build/go/protovalidate"
	auth "github.com/Aditya-0011/common/contracts/go"
	"google.golang.org/grpc"
)

func main() {
	database, err := db.Setup()
	if err != nil {
		log.Fatal("Failed to setup databases:", err)
	}
	defer database.Cleanup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "7295"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	validator, err := protovalidate.New()
	if err != nil {
		log.Fatalf("failed to initialize validator: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(middlewares.ValidationInterceptor(validator)))

	auth.RegisterAuthServiceServer(s, controller.AuthController(controller.AuthControllerParams{
		Postgres: database.Postgres,
	}))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Interrupt received. Starting graceful shutdown...")

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-time.After(5 * time.Second):
		log.Println("Timeout reached (5s). Forcing server shutdown...")
		s.Stop()
	case <-stopped:
		log.Println("Server gracefully stopped within timeout.")
	}

	log.Println("Shutdown complete. Exiting main...")
}
