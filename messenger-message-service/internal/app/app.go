package app

import (
	"context"
	"errors"
	"fmt"
	"log"

	rediscache "github.com/MrEsbens/messenger-message-service/internal/cache/redis"
	"github.com/MrEsbens/messenger-message-service/internal/config"
	"github.com/MrEsbens/messenger-message-service/internal/db"
	"github.com/MrEsbens/messenger-message-service/internal/repository"
	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcserver"
	"github.com/MrEsbens/messenger-message-service/internal/transport/rabbitmq"
	"github.com/MrEsbens/messenger-message-service/internal/transport/websocket"
	"github.com/MrEsbens/messenger-message-service/pkg/redis"
)

type App struct {
	Config      *config.Config
	Postgres    *db.Postgres
	RedisClient *redis.Client
	FileClient  *grpcclient.FileClient

	ServersClient    grpcclient.ServersClientInterface
	ModerationClient grpcclient.ModerationClientInterface
	ModConfigCache   rediscache.ModerationConfigCacheInterface
	ModProducer      rabbitmq.ModerationProducerInterface

	MessageService service.MessageServiceInterface
	GRPCServer     *grpcserver.Server
	WSServer       *websocket.Server
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	ctx := context.Background()

	pg := db.NewPostgres(cfg.Database)

	redisClient := redis.NewClient(ctx, cfg.Redis.URL)

	fileClient, err := grpcclient.NewFileClient(cfg.External.FileServiceEndpoint)
	if err != nil {
		_ = redisClient.Close()
		_ = pg.Close()
		return nil, fmt.Errorf("failed to init file client: %w", err)
	}

	var serversClient grpcclient.ServersClientInterface
	if cfg.External.ServersServiceEndpoint != "" {
		client, err := grpcclient.NewServersClient(
			cfg.External.ServersServiceEndpoint,
			cfg.External.GRPCTimeout,
		)
		if err != nil {
			log.Printf("⚠️  Failed to create servers client: %v", err)
		} else {
			serversClient = client
			log.Printf("✅ Connected to Servers Service at %s", cfg.External.ServersServiceEndpoint)
		}
	}

	var moderationClient grpcclient.ModerationClientInterface
	if cfg.External.ModerationServiceEndpoint != "" {
		client, err := grpcclient.NewModerationClient(
			cfg.External.ModerationServiceEndpoint,
			cfg.External.GRPCTimeout,
		)
		if err != nil {
			log.Printf("⚠️  Failed to create moderation client: %v", err)
		} else {
			moderationClient = client
			log.Printf("✅ Connected to Moderation Service at %s", cfg.External.ModerationServiceEndpoint)
		}
	}

	var modConfigCache rediscache.ModerationConfigCacheInterface
	if redisClient != nil && cfg.Redis.URL != "" {
		modConfigCache = rediscache.NewModerationConfigCache(redisClient.Raw())
	}

	var chatClient grpcclient.ChatClientInterface
	if cfg.External.ChatServiceEndpoint != "" {
		client, err := grpcclient.NewChatClient(cfg.External.ChatServiceEndpoint)
		if err != nil {
			log.Printf("⚠️  Failed to create chat client: %v", err)
		} else {
			chatClient = client
			log.Printf("✅ Connected to Chats Service at %s", cfg.External.ChatServiceEndpoint)
		}
	}

	var modProducer rabbitmq.ModerationProducerInterface
	if cfg.RabbitMQConfig.URL != "" {
		prod, err := rabbitmq.NewModerationProducer(cfg.RabbitMQConfig.URL, cfg.RabbitMQConfig.ModerationExchange)
		if err != nil {
			log.Printf("❌ Failed to create moderation producer: %v", err)
		} else {
			modProducer = prod
			log.Printf("✅ Moderation producer created: %T (nil=%v)", modProducer, modProducer == nil)
			log.Println("✅ Moderation producer connected to RabbitMQ")
		}
	}

	msgRepo := repository.NewMessageRepository(pg.DB)
	statusRepo := repository.NewStatusRepository(pg.DB)

	messageService := service.NewMessageService(
		msgRepo,
		statusRepo,
		redisClient,
		fileClient,
		chatClient,
		serversClient,
		moderationClient,
		modProducer,
		modConfigCache,
	)

	grpcServer, err := grpcserver.NewServer(cfg.GRPCPort, messageService)
	if err != nil {
		_ = fileClient.Close()
		if serversClient != nil {
			_ = serversClient.Close()
		}
		if moderationClient != nil {
			_ = moderationClient.Close()
		}
		_ = redisClient.Close()
		_ = pg.Close()
		return nil, fmt.Errorf("failed to init grpc server: %w", err)
	}

	wsServer := websocket.NewWebSocketServer(cfg.WSPort, redisClient, messageService, cfg.Redis.KeyPrefix)

	return &App{
		Config:           cfg,
		Postgres:         pg,
		RedisClient:      redisClient,
		FileClient:       fileClient,
		ServersClient:    serversClient,
		ModerationClient: moderationClient,
		ModConfigCache:   modConfigCache,
		ModProducer:      modProducer,
		MessageService:   messageService,
		GRPCServer:       grpcServer,
		WSServer:         wsServer,
	}, nil
}

func (a *App) Close() error {
	log.Println("🛑 Closing application resources...")

	var firstErr error

	if a != nil {
		if a.WSServer != nil {
			if err := a.WSServer.Stop(context.Background()); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		if a.GRPCServer != nil {
			a.GRPCServer.Stop()
		}

		if a.FileClient != nil {
			if err := a.FileClient.Close(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		if a.ServersClient != nil {
			if err := a.ServersClient.Close(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		if a.ModerationClient != nil {
			if err := a.ModerationClient.Close(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		if a.RedisClient != nil {
			if err := a.RedisClient.Close(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		if a.Postgres != nil {
			if err := a.Postgres.Close(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		if a.ModProducer != nil {
			if err := a.ModProducer.Close(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}

	log.Println("✅ Application resources closed")
	return firstErr
}
