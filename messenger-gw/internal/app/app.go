package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/MrEsbens/messenger-gw/internal/config"
	"github.com/MrEsbens/messenger-gw/internal/pubsub"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/chats"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/file"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/identity"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/messages"
	"github.com/MrEsbens/messenger-gw/internal/transport/grpcclient/servers"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg            *config.Config
	wsServer       *websocket.Server
	messagesClient messages.MessagesClientInterface
	identityClient identity.IdentityClientInterface
	chatsClient    chats.ChatsClientInterface
	serversClient  servers.ServersClientInterface
	fileClient     file.FileClientInterface
	redisClient    *redis.Client
	redisPubSub    *pubsub.RedisManager
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log.Println("🔄 Initializing Gateway Service")

	// ─── 1. Redis ─────────────────────────────────────────────
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	redisClient := redis.NewClient(redisOpt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	log.Println("✅ Connected to Redis")

	// ─── 2. gRPC Clients ──────────────────────────────────────
	messagesClient, err := messages.NewMessagesClient(cfg.MessagesEndpoint, 5*time.Second)
	if err != nil {
		log.Printf("⚠️ Failed to create messages client: %v", err)
	} else {
		log.Println("✅ Connected to Messages Service")
	}

	identityClient, err := identity.NewIdentityClient(cfg.IdentityEndpoint, 5*time.Second)
	if err != nil {
		log.Printf("⚠️ Failed to create identity client: %v", err)
	} else {
		log.Println("✅ Connected to Identity Service")
	}

	chatsClient, err := chats.NewChatsClient(cfg.ChatsEndpoint, 5*time.Second)
	if err != nil {
		log.Printf("⚠️ Failed to create chats client: %v", err)
	} else {
		log.Println("✅ Connected to Chats Service")
	}

	serversClient, err := servers.NewServersClient(cfg.ServersEndpoint, 5*time.Second)
	if err != nil {
		log.Printf("⚠️ Failed to create servers client: %v", err)
	} else {
		log.Println("✅ Connected to Servers Service")
	}

	fileClient, err := file.NewFileClient(cfg.FileEndpoint, 5*time.Second)
	if err != nil {
		log.Printf("⚠️ Failed to create file client: %v", err)
	} else {
		log.Println("✅ Connected to File Service")
	}

	// ─── 3. WS Server & Pub/Sub ───────────────────────────────
	wsServer := websocket.NewServer(cfg, messagesClient, identityClient, chatsClient, serversClient, fileClient)
	redisPubSub := pubsub.NewRedisManager(redisClient, wsServer.GetHandler())

	return &App{
		cfg:            cfg,
		wsServer:       wsServer,
		messagesClient: messagesClient,
		identityClient: identityClient,
		chatsClient:    chatsClient,
		serversClient:  serversClient,
		fileClient:     fileClient,
		redisClient:    redisClient,
		redisPubSub:    redisPubSub,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("🚀 Gateway HTTP server starting on :%s", a.cfg.HTTPPort)

	// Запускаем фоновые воркеры
	go a.redisPubSub.Start(ctx)
	a.wsServer.Start(ctx)

	<-ctx.Done()
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("🛑 Shutting down Gateway Service...")

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	// Хелпер для параллельного закрытия ресурсов
	runClose := func(name string, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				log.Printf("⚠️ %s close error: %v", name, err)
				errCh <- fmt.Errorf("%s: %w", name, err)
			}
		}()
	}

	// 1. Graceful shutdown HTTP/WS
	runClose("ws_server", func() error { return a.wsServer.Shutdown(ctx) })

	// 2. Закрываем gRPC соединения
	if a.messagesClient != nil {
		runClose("messages_client", a.messagesClient.Close)
	}
	if a.identityClient != nil {
		runClose("identity_client", a.identityClient.Close)
	}
	if a.chatsClient != nil {
		runClose("chats_client", a.chatsClient.Close)
	}
	if a.serversClient != nil {
		runClose("servers_client", a.serversClient.Close)
	}
	if a.fileClient != nil {
		runClose("file_client", a.fileClient.Close)
	}

	// 3. Закрываем Redis
	if a.redisClient != nil {
		runClose("redis_client", a.redisClient.Close)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}

	log.Println("✅ Gateway Service shut down")
	return nil
}

// WaitForSignal возвращает контекст, который отменяется по SIGINT/SIGTERM
func WaitForSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("📡 Received signal: %v", sig)
		cancel()
	}()
	return ctx
}
