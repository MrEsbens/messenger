package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrEsbens/messenger-identity/internal/app"
	"github.com/MrEsbens/messenger-identity/internal/config"
	"github.com/MrEsbens/messenger-identity/internal/transport/grpcserver"
)

func main() {
	cfg := config.Load()

	application := app.New(cfg)
	defer application.Close()

	grpcServer, err := grpcserver.NewServer(
		fmt.Sprintf(":%d", cfg.Port),
		application.IdentityService,
	)
	if err != nil {
		log.Fatalf("failed to create grpc server: %v", err)
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("🛑 Received shutdown signal")

		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Start(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
