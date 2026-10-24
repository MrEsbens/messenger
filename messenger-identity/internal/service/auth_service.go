package service

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/MrEsbens/messenger-identity/internal/pkg/jwt"
	"github.com/MrEsbens/messenger-identity/internal/repository"
	"github.com/google/uuid"
)

type AuthService struct {
	identityv1.UnimplementedIdentityServiceServer

	userRepo   repository.UserRepository
	jwtService *jwt.Service
}

func NewAuthService(
	userRepo repository.UserRepository,
	jwtService *jwt.Service,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (s *AuthService) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.AuthResponse, error) {
	if req.Email == "" || req.Password == "" || req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password and username are required")
	}
	if len(req.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 6 characters")
	}

	if _, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	newUser := &domain.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Status:       "online",
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		if strings.Contains(err.Error(), "unique_violation") || strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	accessToken, refreshToken, err := s.jwtService.Generate(newUser.ID.String())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate tokens")
	}

	return &identityv1.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserDTOWithAvatar(newUser, ""),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.NotFound, "invalid credentials")
	}

	accessToken, refreshToken, err := s.jwtService.Generate(user.ID.String())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate tokens")
	}

	return &identityv1.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserDTOWithAvatar(user, ""),
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req *identityv1.RefreshRequest) (*identityv1.RefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	userID, err := s.jwtService.VerifyRefresh(req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	newAccessToken, _, err := s.jwtService.Generate(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &identityv1.RefreshResponse{
		AccessToken: newAccessToken,
	}, nil
}
