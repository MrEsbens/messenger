package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrEsbens/messenger-message-service/internal/app"
	"github.com/MrEsbens/messenger-message-service/internal/config"
)

func main() {
	cfg := config.Load()
	log.Println("📋 Configuration loaded")

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize application: %v", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("⚠️ Error during app close: %v", err)
		}
	}()

	grpcServer := application.GRPCServer
	wsServer := application.WSServer

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("🛑 Received shutdown signal")

		if err := wsServer.Stop(shutdownCtx); err != nil {
			log.Printf("⚠️ Error stopping WebSocket server: %v", err)
		}

		grpcServer.GracefulStop()

		shutdownCancel()
	}()

	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatalf("❌ Failed to serve gRPC: %v", err)
		}
	}()

	if err := wsServer.Start(shutdownCtx); err != nil {
		log.Fatalf("❌ Failed to serve WebSocket: %v", err)
	}

	log.Println("✅ Messages service stopped gracefully")
}
