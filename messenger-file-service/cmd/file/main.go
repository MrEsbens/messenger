package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/app"
	"github.com/MrEsbens/messenger-file-service/internal/config"
	"github.com/MrEsbens/messenger-file-service/internal/transport/grpcserver"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application := app.New(cfg)

	grpcServer, err := grpcserver.NewServer(
		fmt.Sprintf(":%d", cfg.Port),
		application.FileService, // 🔥 Теперь работает — поле экспортировано
	)
	if err != nil {
		log.Fatalf("failed to create grpc server: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 File gRPC server starting on port %d", cfg.Port)
		if err := grpcServer.Start(); err != nil {
			log.Printf("⚠️ gRPC server error: %v", err)
		}
	}()

	<-sigCh
	log.Println("🛑 Received shutdown signal, initiating graceful shutdown...")

	grpcServer.GracefulStop()
	log.Println("✅ gRPC server stopped")

	// 🔥 Создаём контекст с таймаутом и передаём в Close()
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	if err := application.Close(shutdownCtx); err != nil { // 🔥 Передаём контекст
		log.Printf("⚠️ Application close error: %v", err)
	}
	log.Println("✅ Application shut down complete")
}
