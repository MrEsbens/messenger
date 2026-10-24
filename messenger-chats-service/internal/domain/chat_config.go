package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidConfig = errors.New("invalid chat config")
)

type ChatConfig struct {
	ID                      uuid.UUID
	AllowMemberInvite       bool
	AllowMemberKick         bool
	AllowMemberMute         bool
	AutoModerationEnabled   bool
	MaxMessageLength        int
	DefaultNotificationMode NotificationMode
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (c *ChatConfig) Validate() error {
	if c.MaxMessageLength <= 0 || c.MaxMessageLength > 10000 {
		return ErrInvalidConfig
	}
	if !IsValidNotificationMode(c.DefaultNotificationMode) {
		return ErrInvalidConfig
	}
	return nil
}

func NewChatConfig() *ChatConfig {
	return &ChatConfig{
		ID:                      uuid.New(),
		AllowMemberInvite:       true,
		AllowMemberKick:         false,
		AllowMemberMute:         true,
		AutoModerationEnabled:   false,
		MaxMessageLength:        4000,
		DefaultNotificationMode: NotifyAll,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
}

func (c *ChatConfig) Update(
	allowMemberInvite *bool,
	allowMemberKick *bool,
	allowMemberMute *bool,
	autoModerationEnabled *bool,
	maxMessageLength *int,
	defaultNotificationMode *NotificationMode,
) {
	if allowMemberInvite != nil {
		c.AllowMemberInvite = *allowMemberInvite
	}
	if allowMemberKick != nil {
		c.AllowMemberKick = *allowMemberKick
	}
	if allowMemberMute != nil {
		c.AllowMemberMute = *allowMemberMute
	}
	if autoModerationEnabled != nil {
		c.AutoModerationEnabled = *autoModerationEnabled
	}
	if maxMessageLength != nil {
		c.MaxMessageLength = *maxMessageLength
	}
	if defaultNotificationMode != nil {
		c.DefaultNotificationMode = *defaultNotificationMode
	}
	c.UpdatedAt = time.Now()
}
