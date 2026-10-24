package config

import (
	"time"

	"github.com/joho/godotenv"
)

// Config — главная структура конфигурации приложения
type Config struct {
	GRPCPort int
	WSPort   int

	Database DatabaseConfig

	Redis RedisConfig

	External ExternalConfig

	RabbitMQConfig RabbitMQConfig
}

type DatabaseConfig struct {
	URL            string
	Schema         string
	MigrationsPath string
}

type RedisConfig struct {
	URL string

	// Префикс для ключей (опционально, для изоляции)
	KeyPrefix string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type ExternalConfig struct {
	FileServiceEndpoint       string
	ChatServiceEndpoint       string
	ServersServiceEndpoint    string
	ModerationServiceEndpoint string
	GRPCDialTimeout           time.Duration
	GRPCTimeout               time.Duration
}

type RabbitMQConfig struct {
	URL                 string `env:"RABBITMQ_URL" envDefault:"amqp://messenger:messengerpass@rabbitmq:5672/"`
	ModerationExchange  string `env:"MODERATION_EXCHANGE" envDefault:"moderation.events"`
	ModerationQueueName string `env:"MODERATION_QUEUE_NAME" envDefault:"moderation.requests"`
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		GRPCPort: getEnvInt("GRPC_PORT", 50053),
		WSPort:   getEnvInt("WS_PORT", 50054),

		Database: DatabaseConfig{
			URL:            getEnvRequired("DATABASE_URL"),
			Schema:         getEnv("DB_SCHEMA", "messages"),
			MigrationsPath: getEnv("MIGRATIONS_PATH", "internal/db/migrations"),
		},

		Redis: RedisConfig{
			URL:          getEnvRequired("REDIS_URL"),
			KeyPrefix:    getEnv("REDIS_KEY_PREFIX", "msg:"),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", "5s"),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", "5s"),
		},

		External: ExternalConfig{
			FileServiceEndpoint:       getEnv("FILE_SERVICE_ENDPOINT", "localhost:50052"),
			ChatServiceEndpoint:       getEnv("CHAT_SERVICE_ENDPOINT", "localhost:50055"),
			ServersServiceEndpoint:    getEnv("SERVERS_SERVICE_ENDPOINT", "localhost:50055"),
			ModerationServiceEndpoint: getEnv("MODERATION_SERVICE_ENDPOINT", "localhost:50057"),
			GRPCDialTimeout:           getEnvDuration("GRPC_DIAL_TIMEOUT_SEC", "5s"),
			GRPCTimeout:               getEnvDuration("GRPC_TIMEOUT_SEC", "10s"),
		},

		RabbitMQConfig: RabbitMQConfig{
            URL:                 getEnv("RABBITMQ_URL", "amqp://messenger:messengerpass@rabbitmq:5672/"),
            ModerationExchange:  getEnv("MODERATION_EXCHANGE", "moderation.events"),
            ModerationQueueName: getEnv("MODERATION_QUEUE_NAME", "moderation.requests"),
        },
	}
}
