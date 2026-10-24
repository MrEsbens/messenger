package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPCPort int
	WSPort   int
	Database DatabaseConfig
	Redis    RedisConfig
	External ExternalConfig
}

type DatabaseConfig struct {
	URL            string
	Schema         string
	MigrationsPath string
}

type RedisConfig struct {
	URL          string
	KeyPrefix    string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type ExternalConfig struct {
	IdentityServiceEndpoint string
	MessageServiceEndpoint  string
	FileServiceEndpoint     string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ godotenv.Load() error: %v", err)
	}

	return &Config{
		GRPCPort: getEnvInt("GRPC_PORT", 50055),
		WSPort:   getEnvInt("WS_PORT", 50056),
		Database: DatabaseConfig{
			URL:            getEnvRequired("DATABASE_URL"),
			Schema:         getEnv("DB_SCHEMA", "chats"),
			MigrationsPath: getEnv("MIGRATIONS_PATH", "internal/db/migrations"),
		},
		Redis: RedisConfig{
			URL:          getEnvRequired("REDIS_URL"),
			KeyPrefix:    getEnv("REDIS_KEY_PREFIX", "chat:"),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", "5s"),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", "5s"),
		},
		External: ExternalConfig{
			IdentityServiceEndpoint: getEnv("IDENTITY_SERVICE_ENDPOINT", "localhost:50051"),
			MessageServiceEndpoint:  getEnv("MESSAGE_SERVICE_ENDPOINT", "localhost:50053"),
			FileServiceEndpoint:     getEnv("FILE_SERVICE_ENDPOINT", "localhost:50052"),
		},
	}
}
