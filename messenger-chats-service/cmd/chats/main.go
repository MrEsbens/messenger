package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrEsbens/messenger-chats-service/internal/app"
	"github.com/MrEsbens/messenger-chats-service/internal/config"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize app: %v", err)
	}

	go func() {
		if err := application.Run(ctx); err != nil {
			log.Printf("❌ Application error: %v", err)
			cancel()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("🛑 Shutting down...")
	cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Shutdown error: %v", err)
	}

	log.Println("✅ Shutdown complete")
}
