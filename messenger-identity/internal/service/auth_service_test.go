package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/MrEsbens/messenger-identity/internal/pkg/jwt"
	"github.com/MrEsbens/messenger-identity/internal/repository/mocks"
	"github.com/MrEsbens/messenger-identity/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTestAuthService(t *testing.T) (*service.AuthService, *mocks.MockUserRepository) {
	userRepo := mocks.NewMockUserRepository(t)
	jwtService := jwt.NewService(
		[]byte("test-secret-key-12345678"),
		15*time.Minute,
		24*time.Hour,
	)

	return service.NewAuthService(userRepo, jwtService), userRepo
}

func TestAuthService_Register_Success(t *testing.T) {
	svc, userRepo := newTestAuthService(t)

	req := &identityv1.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	userRepo.On("GetByEmail", mock.Anything, req.Email).
		Return(nil, errors.New("not found")).Once()

	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
		Return(nil).Once()

	resp, err := svc.Register(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, req.Username, resp.User.Username)
	assert.Equal(t, req.Email, resp.User.Email)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Register_EmptyFields(t *testing.T) {
	svc, _ := newTestAuthService(t)

	testCases := []struct {
		name string
		req  *identityv1.RegisterRequest
	}{
		{"empty email", &identityv1.RegisterRequest{Username: "user", Password: "pass123"}},
		{"empty password", &identityv1.RegisterRequest{Username: "user", Email: "test@test.com"}},
		{"empty username", &identityv1.RegisterRequest{Email: "test@test.com", Password: "pass123"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.Register(context.Background(), tc.req)
			assert.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}

func TestAuthService_Register_ShortPassword(t *testing.T) {
	svc, _ := newTestAuthService(t)

	req := &identityv1.RegisterRequest{
		Username: "user",
		Email:    "test@test.com",
		Password: "12345",
	}

	resp, err := svc.Register(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "6 characters")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, userRepo := newTestAuthService(t)

	req := &identityv1.RegisterRequest{
		Username: "user",
		Email:    "test@example.com",
		Password: "password123",
	}

	existingUser := &domain.User{
		ID:    uuid.New(),
		Email: req.Email,
	}
	userRepo.On("GetByEmail", mock.Anything, req.Email).
		Return(existingUser, nil).Once()

	resp, err := svc.Register(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "AlreadyExists")

	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, userRepo := newTestAuthService(t)

	req := &identityv1.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Username:     "testuser",
		Status:       "online",
	}

	userRepo.On("GetByEmail", mock.Anything, req.Email).
		Return(user, nil).Once()

	resp, err := svc.Login(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, user.Username, resp.User.Username)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	svc, userRepo := newTestAuthService(t)

	req := &identityv1.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	userRepo.On("GetByEmail", mock.Anything, req.Email).
		Return(nil, errors.New("not found")).Once()

	resp, err := svc.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "NotFound")

	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, userRepo := newTestAuthService(t)

	req := &identityv1.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
	}

	userRepo.On("GetByEmail", mock.Anything, req.Email).
		Return(user, nil).Once()

	resp, err := svc.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "NotFound")

	userRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	svc, _ := newTestAuthService(t)

	testUserID := uuid.New().String()
	jwtService := jwt.NewService(
		[]byte("test-secret-key-12345678"),
		15*time.Minute,
		24*time.Hour,
	)
	_, refreshToken, err := jwtService.Generate(testUserID)
	require.NoError(t, err)

	req := &identityv1.RefreshRequest{
		RefreshToken: refreshToken,
	}

	resp, err := svc.Refresh(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestAuthService_Refresh_EmptyToken(t *testing.T) {
	svc, _ := newTestAuthService(t)

	req := &identityv1.RefreshRequest{
		RefreshToken: "",
	}

	resp, err := svc.Refresh(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "refresh token is required")
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	svc, _ := newTestAuthService(t)

	req := &identityv1.RefreshRequest{
		RefreshToken: "invalid-token",
	}

	resp, err := svc.Refresh(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Unauthenticated")
}
