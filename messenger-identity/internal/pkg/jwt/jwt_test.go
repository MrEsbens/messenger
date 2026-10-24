package jwt_test

import (
	"testing"
	"time"

	"github.com/MrEsbens/messenger-identity/internal/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Generate_Success(t *testing.T) {
	svc := jwt.NewService([]byte("test-secret"), 15*time.Minute, 24*time.Hour)

	accessToken, refreshToken, err := svc.Generate("user-123")

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
}

func TestService_Verify_Success(t *testing.T) {
	svc := jwt.NewService([]byte("test-secret"), 15*time.Minute, 24*time.Hour)
	token, _, _ := svc.Generate("user-123")

	userID, err := svc.Verify(token)

	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestService_Verify_Expired(t *testing.T) {
	svc := jwt.NewService([]byte("test-secret"), 1*time.Millisecond, 24*time.Hour)
	token, _, _ := svc.Generate("user-123")

	time.Sleep(10 * time.Millisecond)

	userID, err := svc.Verify(token)

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestService_Verify_InvalidToken(t *testing.T) {
	svc := jwt.NewService([]byte("test-secret"), 15*time.Minute, 24*time.Hour)

	userID, err := svc.Verify("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestService_Verify_WrongSecret(t *testing.T) {
	svc1 := jwt.NewService([]byte("secret-1"), 15*time.Minute, 24*time.Hour)
	svc2 := jwt.NewService([]byte("secret-2"), 15*time.Minute, 24*time.Hour)

	token, _, _ := svc1.Generate("user-123")

	userID, err := svc2.Verify(token)

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestService_VerifyRefresh_Success(t *testing.T) {
	svc := jwt.NewService([]byte("test-secret"), 15*time.Minute, 24*time.Hour)
	_, refreshToken, _ := svc.Generate("user-123")

	userID, err := svc.VerifyRefresh(refreshToken)

	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestService_Generate_EmptyUserID(t *testing.T) {
	svc := jwt.NewService([]byte("test-secret"), 15*time.Minute, 24*time.Hour)

	accessToken, refreshToken, err := svc.Generate("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userID is required")
	assert.Empty(t, accessToken)
	assert.Empty(t, refreshToken)
}
