package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidChatType    = errors.New("invalid chat type")
	ErrInvalidChatName    = errors.New("invalid chat name")
	ErrInvalidDescription = errors.New("invalid description")
	ErrChatNotFound       = errors.New("chat not found")
	ErrChatDeleted        = errors.New("chat is deleted")
)

type Chat struct {
	ID          uuid.UUID
	Type        ChatType
	Name        *string
	Description *string
	ConfigID    *uuid.UUID
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func (c *Chat) Validate() error {
	if !IsValidChatType(c.Type) {
		return ErrInvalidChatType
	}

	if c.Type != ChatTypePrivate {
		if c.Name == nil || *c.Name == "" {
			return ErrInvalidChatName
		}
		if len(*c.Name) > MaxChatNameLength {
			return ErrInvalidChatName
		}
	}

	if c.Description != nil && len(*c.Description) > MaxChatDescriptionLength {
		return ErrInvalidDescription
	}

	if c.DeletedAt != nil {
		return ErrChatDeleted
	}

	return nil
}

func (c *Chat) IsPrivate() bool {
	return c.Type == ChatTypePrivate
}

func (c *Chat) IsGroup() bool {
	return c.Type == ChatTypeGroup
}

func (c *Chat) IsServer() bool {
	return c.Type == ChatTypeServer
}

func (c *Chat) IsDeleted() bool {
	return c.DeletedAt != nil
}

func (c *Chat) SoftDelete() {
	now := time.Now()
	c.DeletedAt = &now
}

func (c *Chat) Update(name *string, description *string, configID *uuid.UUID) {
	if name != nil {
		c.Name = name
	}
	if description != nil {
		c.Description = description
	}
	if configID != nil {
		c.ConfigID = configID
	}
	c.UpdatedAt = time.Now()
}

func NewChat(chatType ChatType, name *string, description *string, createdBy uuid.UUID) (*Chat, error) {
	chat := &Chat{
		ID:          uuid.New(),
		Type:        chatType,
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := chat.Validate(); err != nil {
		return nil, err
	}

	return chat, nil
}
