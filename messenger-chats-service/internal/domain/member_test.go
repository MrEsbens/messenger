package domain_test

import (
	"testing"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChatMember_Validate_Success(t *testing.T) {
	member, err := domain.NewChatMember(uuid.New(), uuid.New(), domain.RoleMember)

	assert.NoError(t, err)
	assert.NoError(t, member.Validate())
	assert.True(t, member.IsActive())
}

func TestChatMember_Validate_Fail_InvalidRole(t *testing.T) {
	member := &domain.ChatMember{
		ID:               uuid.New(),
		ChatID:           uuid.New(),
		UserID:           uuid.New(),
		Role:             "invalid_role",
		JoinedAt:         time.Now(),
		NotificationMode: domain.NotifyAll,
	}

	err := member.Validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestChatMember_Leave(t *testing.T) {
	member, _ := domain.NewChatMember(uuid.New(), uuid.New(), domain.RoleMember)

	assert.True(t, member.IsActive())

	member.Leave()

	assert.False(t, member.IsActive())
	assert.NotNil(t, member.LeftAt)
}

func TestChatMember_UpdateRole(t *testing.T) {
	member, _ := domain.NewChatMember(uuid.New(), uuid.New(), domain.RoleMember)

	member.UpdateRole(domain.RoleAdmin)

	assert.Equal(t, domain.RoleAdmin, member.Role)
	assert.Nil(t, member.IsMutedUntil) // Мут сбрасывается
}

func TestChatMember_IsMuted(t *testing.T) {
	member, _ := domain.NewChatMember(uuid.New(), uuid.New(), domain.RoleMuted)

	// Не заглушен (нет IsMutedUntil)
	assert.False(t, member.IsMuted())

	// Заглушен (будущее время)
	future := time.Now().Add(1 * time.Hour)
	member.IsMutedUntil = &future
	assert.True(t, member.IsMuted())

	// Мут истёк (прошлое время)
	past := time.Now().Add(-1 * time.Hour)
	member.IsMutedUntil = &past
	assert.False(t, member.IsMuted())
}
