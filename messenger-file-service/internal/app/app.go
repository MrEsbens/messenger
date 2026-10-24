package app

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/config"
	"github.com/MrEsbens/messenger-file-service/internal/db"
	"github.com/MrEsbens/messenger-file-service/internal/pkg/minio"
	"github.com/MrEsbens/messenger-file-service/internal/repository"
	"github.com/MrEsbens/messenger-file-service/internal/service"
)

type App struct {
	cfg           *config.Config
	FileService   *service.FileService // 🔥 Экспортированное поле (большая буква)
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
	cleanupWg     sync.WaitGroup
}

func New(cfg *config.Config) *App {
	pg := db.NewPostgres(cfg.Database)

	minioCfg := minio.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		UseSSL:    cfg.MinIO.UseSSL,
	}
	minioClient, err := minio.NewClient(minioCfg)
	if err != nil {
		log.Fatalf("failed to create minio client: %v", err)
	}

	fileRepo := repository.NewFileRepository(pg.DB)
	fileService := service.NewFileService(minioClient, fileRepo)

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	app := &App{
		cfg:           cfg,
		FileService:   fileService, // 🔥 Теперь экспортировано
		cleanupCtx:    cleanupCtx,
		cleanupCancel: cleanupCancel,
	}

	app.startCleanupLoop()

	return app
}

func (a *App) startCleanupLoop() {
	a.cleanupWg.Add(1)
	go func() {
		defer a.cleanupWg.Done()

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		log.Println("🧹 Pending files cleanup loop started (interval: 15m)")

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(a.cleanupCtx, 2*time.Minute)
				deleted, err := a.FileService.CleanupPendingFiles(ctx, 30*time.Minute)
				cancel()

				if err != nil {
					log.Printf("⚠️ Cleanup error: %v", err)
				} else if deleted > 0 {
					log.Printf("✅ Cleaned up %d pending files", deleted)
				}

			case <-a.cleanupCtx.Done():
				log.Println("🛑 Cleanup loop stopped")
				return
			}
		}
	}()
}

// 🔥 Close теперь принимает контекст для graceful shutdown
func (a *App) Close(ctx context.Context) error {
	log.Println("🛑 Shutting down File Service...")

	if a.cleanupCancel != nil {
		a.cleanupCancel()

		// Ждём завершения горутины с таймаутом из контекста
		done := make(chan struct{})
		go func() {
			a.cleanupWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Println("✅ Cleanup loop stopped")
		case <-ctx.Done():
			log.Println("⚠️ Cleanup loop shutdown timed out")
		}
	}

	log.Println("✅ File Service shut down")
	return nil
}
