package domain_test

import (
	"testing"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewChatInvite_Success(t *testing.T) {
	invite, err := domain.NewChatInvite(uuid.New(), uuid.New(), nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, invite)
	assert.NotEmpty(t, invite.Token)
	assert.Equal(t, 0, invite.UsesCount)
	assert.Nil(t, invite.UsedAt)
	assert.True(t, invite.IsActive())
}

func TestNewChatInvite_WithExpiry(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, err := domain.NewChatInvite(uuid.New(), uuid.New(), &expiresAt, nil)

	assert.NoError(t, err)
	assert.NotNil(t, invite.ExpiresAt)
	assert.True(t, invite.IsActive())
}

func TestNewChatInvite_WithMaxUses(t *testing.T) {
	maxUses := 5
	invite, err := domain.NewChatInvite(uuid.New(), uuid.New(), nil, &maxUses)

	assert.NoError(t, err)
	assert.NotNil(t, invite.MaxUses)
	assert.Equal(t, 5, *invite.MaxUses)
	assert.Equal(t, 0, invite.UsesCount)
}

func TestChatInvite_Validate_Fail_EmptyToken(t *testing.T) {
	invite := &domain.ChatInvite{
		Token: "",
	}

	err := invite.Validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInvite)
}

func TestChatInvite_IsActive_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), &past, nil)

	assert.False(t, invite.IsActive())
}

func TestChatInvite_IsActive_MaxUsesReached(t *testing.T) {
	maxUses := 1
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), nil, &maxUses)
	invite.UsesCount = 1

	assert.False(t, invite.IsActive())
}

func TestChatInvite_IsActive_UsedAt_SingleUse(t *testing.T) {
	maxUses := 1
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), nil, &maxUses)
	now := time.Now()
	invite.UsedAt = &now

	assert.False(t, invite.IsActive())
}

func TestChatInvite_IsActive_UsedAt_MultiUse(t *testing.T) {
	maxUses := 5
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), nil, &maxUses)
	now := time.Now()
	invite.UsedAt = &now
	invite.UsesCount = 2

	// Для мульти-использования used_at не блокирует
	assert.True(t, invite.IsActive())
}

func TestChatInvite_Use_Success(t *testing.T) {
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), nil, nil)

	err := invite.Use()

	assert.NoError(t, err)
	assert.Equal(t, 1, invite.UsesCount)
	assert.Nil(t, invite.UsedAt) // Для мульти-использования не ставится
}

func TestChatInvite_Use_Success_SingleUse(t *testing.T) {
	maxUses := 1
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), nil, &maxUses)

	err := invite.Use()

	assert.NoError(t, err)
	assert.Equal(t, 1, invite.UsesCount)
	assert.NotNil(t, invite.UsedAt) // Для одноразовых ставится
}

func TestChatInvite_Use_Fail_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), &past, nil)

	err := invite.Use()

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInviteExpired)
}

func TestChatInvite_Use_Fail_MaxUsesReached(t *testing.T) {
	maxUses := 1
	invite, _ := domain.NewChatInvite(uuid.New(), uuid.New(), nil, &maxUses)
	invite.UsesCount = 1

	err := invite.Use()

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInviteMaxUses)
}

func TestGenerateToken(t *testing.T) {
	token1, err := domain.GenerateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token1)

	token2, err := domain.GenerateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token2)

	// Токены уникальны
	assert.NotEqual(t, token1, token2)
}
