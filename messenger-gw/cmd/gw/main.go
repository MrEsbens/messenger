package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrEsbens/messenger-gw/internal/app"
	"github.com/MrEsbens/messenger-gw/internal/config"
)

func main() {
	cfg := config.Load()

	ctx := waitForSignal()

	gw, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize gateway: %v", err)
	}

	if err := gw.Run(ctx); err != nil && err != context.Canceled {
		log.Printf("❌ Gateway runtime error: %v", err)
	}

	if err := gw.Shutdown(context.Background()); err != nil {
		log.Printf("⚠️ Shutdown error: %v", err)
	}

	log.Println("✅ Gateway stopped")
}

func waitForSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("📶 Received signal: %v", sig)
		cancel()
	}()

	return ctx
}
