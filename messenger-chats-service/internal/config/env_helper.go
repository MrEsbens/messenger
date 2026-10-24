package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getEnvRequired(key string) string {
	value := getEnv(key, "")
	if value == "" {
		log.Fatalf("❌ Required environment variable %s is not set", key)
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("⚠️ Invalid int value for %s=%s, using fallback %d", key, value, fallback)
		return fallback
	}
	return intVal
}

func getEnvDuration(key, fallback string) time.Duration {
	value := getEnv(key, fallback)

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("⚠️ Invalid duration value for %s=%s, using fallback %s", key, value, fallback)
		d, _ := time.ParseDuration(fallback)
		return d
	}
	return duration
}
