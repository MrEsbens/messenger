package config

import (
	"log"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     int
	Database DatabaseConfig

	MinIO *MinIOConfig
}

type DatabaseConfig struct {
	URL            string
	Schema         string
	MigrationsPath string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{}

	portStr := getEnv("PORT", "50052")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid PORT value: %v", err)
	}
	cfg.Port = port

	cfg.Database.URL = getEnv("DATABASE_URL", "")
	if cfg.Database.URL == "" {
		log.Fatalf("DATABASE_URL is required")
	}

	cfg.Database.Schema = getEnv("DB_SCHEMA", "storage")
	cfg.Database.MigrationsPath = getEnv("MIGRATIONS_PATH", "internal/db/migrations")

	cfg.MinIO = LoadMinIOConfig()

	return cfg
}
