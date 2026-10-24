package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort    string
	JWTSecret   []byte

	RedisURL    string
	RedisPrefix string

	IdentityEndpoint  string
	ChatsEndpoint     string
	MessagesEndpoint  string
	ServersEndpoint   string
	FileEndpoint      string
}

func Load() *Config {
	_ = godotenv.Load()

	getEnv := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}

	cfg := &Config{
		HTTPPort:  getEnv("HTTP_PORT", "8080"),
		JWTSecret: []byte(getEnv("JWT_SECRET", "super-secret-key-change-in-production")),

		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPrefix: getEnv("REDIS_KEY_PREFIX", "gw:"),

		IdentityEndpoint:  getEnv("IDENTITY_SERVICE_ENDPOINT", "localhost:50052"),
		ChatsEndpoint:     getEnv("CHATS_SERVICE_ENDPOINT", "localhost:50053"),
		MessagesEndpoint:  getEnv("MESSAGES_SERVICE_ENDPOINT", "localhost:50056"),
		ServersEndpoint:   getEnv("SERVERS_SERVICE_ENDPOINT", "localhost:50055"),
		FileEndpoint:      getEnv("FILE_SERVICE_ENDPOINT", "localhost:50051"),
	}

	if len(cfg.JWTSecret) < 16 {
		log.Fatal("❌ JWT_SECRET must be at least 16 characters")
	}

	return cfg
}