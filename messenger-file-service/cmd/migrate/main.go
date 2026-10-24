package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/config"
	"github.com/MrEsbens/messenger-file-service/internal/db/migrator"
	"github.com/MrEsbens/messenger-file-service/internal/pkg/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	cfg := config.Load()

	cmd := flag.String("cmd", "up", "up | down | force | buckets")
	steps := flag.Int("steps", 1, "steps for down")
	version := flag.Int("version", 1, "version for force")
	flag.Parse()

	m := migrator.New(cfg.Database.URL, cfg.Database.MigrationsPath)

	switch *cmd {
	case "up":
		m.Up()
	case "down":
		m.Down(*steps)
	case "force":
		m.Force(*version)
	case "buckets":
		if err := initMinIOBuckets(cfg.MinIO, storage.AllBucketTypes()); err != nil {
			log.Fatalf("failed to init MinIO buckets: %v", err)
		}
		log.Println("✓ MinIO buckets initialized")
	default:
		log.Fatalf("unknown command: %s", *cmd)
	}
}

func initMinIOBuckets(cfg *config.MinIOConfig, buckets []storage.BucketType) error {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, bucket := range buckets {
		exists, err := client.BucketExists(ctx, string(bucket))
		if err != nil {
			return fmt.Errorf("failed to check bucket %s: %w", bucket, err)
		}
		if !exists {
			err := client.MakeBucket(ctx, string(bucket), minio.MakeBucketOptions{Region: cfg.Region})
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
			log.Printf("✓ Created bucket: %s", bucket)
		} else {
			log.Printf("✓ Bucket exists: %s", bucket)
		}
	}
	return nil
}
