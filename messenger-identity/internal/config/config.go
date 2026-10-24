package config

import (
	"log"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     int
	Database DatabaseConfig

	JWTSecret      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration

	FileServiceEndpoint string
}

type DatabaseConfig struct {
	URL string
	Schema string
	MigrationsPath string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{}

	portStr := getEnv("PORT", "50051")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid PORT value: %v", err)
	}
	cfg.Port = port

	cfg.Database.URL = getEnv("DATABASE_URL", "")
	if cfg.Database.URL == "" {
		log.Fatalf("DATABASE_URL is required")
	}

	cfg.Database.Schema = getEnv("DB_SCHEMA", "identity")
	cfg.Database.MigrationsPath = getEnv("MIGRATIONS_PATH", "internal/db/migrations")

	cfg.JWTSecret = getEnv("JWT_SECRET", "dev-secret-change-in-prod")

	accessTTLStr := getEnv("JWT_ACCESS_TTL", "15m")
	cfg.JWTAccessTTL = parseDuration(accessTTLStr, 15*time.Minute)

	refreshTTLStr := getEnv("JWT_REFRESH_TTL", "168h") // 7 дней
	cfg.JWTRefreshTTL = parseDuration(refreshTTLStr, 7*24*time.Hour)

	cfg.FileServiceEndpoint = getEnv("FILE_SERVICE_ENDPOINT", "localhost:50052")

	return cfg
}

func parseDuration(val string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}
