package domain_test

import (
	"testing"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChat_Validate_Success_Group(t *testing.T) {
	name := "Test Group"
	chat, err := domain.NewChat(domain.ChatTypeGroup, &name, nil, uuid.New())

	assert.NoError(t, err)
	assert.NoError(t, chat.Validate())
	assert.Equal(t, domain.ChatTypeGroup, chat.Type)
}

func TestChat_Validate_Success_Private(t *testing.T) {
	// Private чат может быть без имени
	chat, err := domain.NewChat(domain.ChatTypePrivate, nil, nil, uuid.New())

	assert.NoError(t, err)
	assert.NoError(t, chat.Validate())
}

func TestChat_Validate_Fail_GroupNoName(t *testing.T) {
	// Group чат требует имя
	chat, err := domain.NewChat(domain.ChatTypeGroup, nil, nil, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, chat)
	assert.ErrorIs(t, err, domain.ErrInvalidChatName)
}

func TestChat_Validate_Fail_InvalidType(t *testing.T) {
	name := "Test"
	chat := &domain.Chat{
		ID:        uuid.New(),
		Type:      "invalid_type",
		Name:      &name,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := chat.Validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidChatType)
}

func TestChat_SoftDelete(t *testing.T) {
	name := "Test"
	chat, _ := domain.NewChat(domain.ChatTypeGroup, &name, nil, uuid.New())

	assert.False(t, chat.IsDeleted())

	chat.SoftDelete()

	assert.True(t, chat.IsDeleted())
	assert.NotNil(t, chat.DeletedAt)
}

func TestChat_Update(t *testing.T) {
	name := "Original"
	chat, _ := domain.NewChat(domain.ChatTypeGroup, &name, nil, uuid.New())

	newName := "Updated"
	chat.Update(&newName, nil, nil)

	assert.Equal(t, "Updated", *chat.Name)
	assert.True(t, chat.UpdatedAt.After(chat.CreatedAt))
}
