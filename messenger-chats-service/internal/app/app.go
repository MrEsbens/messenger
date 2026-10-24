package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/MrEsbens/messenger-chats-service/internal/config"
	"github.com/MrEsbens/messenger-chats-service/internal/db"
	"github.com/MrEsbens/messenger-chats-service/internal/repository"
	"github.com/MrEsbens/messenger-chats-service/internal/service"
	"github.com/MrEsbens/messenger-chats-service/internal/transport/grpcclient"
	"github.com/MrEsbens/messenger-chats-service/internal/transport/grpcserver"
	"github.com/MrEsbens/messenger-chats-service/internal/transport/websocket"
	"github.com/MrEsbens/messenger-chats-service/pkg/redis"
)

type App struct {
	cfg            *config.Config
	db             *db.Postgres
	redis          *redis.Client
	fileClient     grpcclient.FileClientInterface
	identityClient grpcclient.IdentityClientInterface

	chatRepo   repository.ChatRepository
	configRepo repository.ChatConfigRepository
	memberRepo repository.MemberRepository
	inviteRepo repository.InviteRepository

	service service.ChatServiceInterface
	handler *grpcserver.GRPCHandler

	grpcServer *grpcserver.Server
	wsHub      *websocket.Hub
	wsServer   *websocket.Server
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	pg := db.NewPostgres(cfg.Database)

	redisClient := redis.NewClient(ctx, cfg.Redis.URL)

	chatRepo := repository.NewChatRepository(pg.DB, cfg.Database.Schema)
	configRepo := repository.NewChatConfigRepository(pg.DB, cfg.Database.Schema)
	memberRepo := repository.NewMemberRepository(pg.DB, cfg.Database.Schema)
	inviteRepo := repository.NewInviteRepository(pg.DB, cfg.Database.Schema)

	var fileClient grpcclient.FileClientInterface
	if cfg.External.FileServiceEndpoint != "" {
		client, err := grpcclient.NewFileClient(cfg.External.FileServiceEndpoint)
		if err != nil {
			log.Printf("⚠️ Failed to create file client: %v", err)
		} else {
			fileClient = client
			log.Printf("✅ Connected to File Service at %s", cfg.External.FileServiceEndpoint)
		}
	}

	var identityClient grpcclient.IdentityClientInterface
	if cfg.External.IdentityServiceEndpoint != "" {
		client, err := grpcclient.NewIdentityClient(cfg.External.IdentityServiceEndpoint)
		if err != nil {
			log.Printf("⚠️ Failed to create identity client: %v", err)
		} else {
			identityClient = client
			log.Printf("✅ Connected to Identity Service at %s", cfg.External.IdentityServiceEndpoint)
		}
	}

	chatService := service.NewChatService(
		chatRepo,
		configRepo,
		memberRepo,
		inviteRepo,
		redisClient,
		cfg.Redis.KeyPrefix,
		fileClient,
		identityClient,
	)

	handler := grpcserver.NewGRPCHandler(chatService)
	grpcServer := grpcserver.NewServer(handler, cfg.GRPCPort)

	wsHub := websocket.NewHub(redisClient, cfg.Redis.KeyPrefix)
	wsServer := websocket.NewServer(wsHub, cfg.WSPort)

	return &App{
		cfg:            cfg,
		db:             pg,
		redis:          redisClient,
		fileClient:     fileClient,
		identityClient: identityClient,

		chatRepo:   chatRepo,
		configRepo: configRepo,
		memberRepo: memberRepo,
		inviteRepo: inviteRepo,

		service: chatService,
		handler: handler,

		grpcServer: grpcServer,
		wsHub:      wsHub,
		wsServer:   wsServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("🚀 Chats Service starting")
	log.Printf("   gRPC:        localhost:%d", a.cfg.GRPCPort)
	log.Printf("   WebSocket:   ws://localhost:%d/ws", a.cfg.WSPort)

	if a.fileClient != nil {
		log.Printf("   File Service:    %s", a.cfg.External.FileServiceEndpoint)
	}
	if a.identityClient != nil {
		log.Printf("   Identity Service: %s", a.cfg.External.IdentityServiceEndpoint)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("📡 Starting gRPC server on port %d", a.cfg.GRPCPort)
		if err := a.grpcServer.Start(ctx); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("🔌 Starting WebSocket server on port %d", a.cfg.WSPort)
		if err := a.wsServer.Start(ctx); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("WebSocket server error: %w", err)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errChan:
		log.Printf("❌ Server error: %v", err)
		return err
	case <-ctx.Done():
		log.Println("🛑 Context cancelled, shutting down...")
		return nil
	case <-done:
		log.Println("🛑 All servers stopped")
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("🔌 Shutting down services...")

	if a.fileClient != nil {
		if err := a.fileClient.Close(); err != nil {
			log.Printf("⚠️ File client close error: %v", err)
		} else {
			log.Println("✅ File client closed")
		}
	}

	if a.identityClient != nil {
		if err := a.identityClient.Close(); err != nil {
			log.Printf("⚠️ Identity client close error: %v", err)
		} else {
			log.Println("✅ Identity client closed")
		}
	}

	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			log.Printf("⚠️ Redis close error: %v", err)
		} else {
			log.Println("✅ Redis closed")
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("⚠️ Database close error: %v", err)
		} else {
			log.Println("✅ Database closed")
		}
	}

	log.Println("✅ Shutdown complete")
	return nil
}
