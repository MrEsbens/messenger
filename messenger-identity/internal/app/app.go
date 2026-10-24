// internal/app/app.go

package app

import (
	"log"

	"github.com/MrEsbens/messenger-identity/internal/config"
	"github.com/MrEsbens/messenger-identity/internal/db"
	"github.com/MrEsbens/messenger-identity/internal/pkg/jwt"
	"github.com/MrEsbens/messenger-identity/internal/repository"
	"github.com/MrEsbens/messenger-identity/internal/service"
	"github.com/MrEsbens/messenger-identity/internal/transport/grpcclient"
)

type App struct {
	IdentityService *service.IdentityService
	fileClient      *grpcclient.FileClient
}

// New создаёт и инициализирует все сервисы
func New(cfg *config.Config) *App {
	// 1. БД
	pg := db.NewPostgres(cfg.Database)

	// 2. Репозитории
	userRepo := repository.NewUserRepository(pg.DB)
	contactRepo := repository.NewContactRepository(pg.DB, userRepo)

	// 3. gRPC клиент к File Service
	fileClient, err := grpcclient.NewFileClient(cfg.FileServiceEndpoint)
	if err != nil {
		log.Fatalf("failed to create file client: %v", err)
	}

	// 4. JWT сервис (пока не используется в interceptor'ах, но нужен для Auth)
	jwtService := jwt.NewService(
		[]byte(cfg.JWTSecret),
		cfg.JWTAccessTTL,
		cfg.JWTRefreshTTL,
	)

	// 5. Бизнес-сервисы
	authService := service.NewAuthService(userRepo, jwtService)
	userService := service.NewUserService(userRepo, fileClient)
	contactService := service.NewContactService(userRepo, contactRepo)

	identityService := service.NewIdentityService(authService, userService, contactService)

	return &App{
		IdentityService: identityService,
		fileClient:      fileClient,
	}
}

func (a *App) Close() error {
	if a.fileClient != nil {
		return a.fileClient.Close()
	}
	return nil
}
