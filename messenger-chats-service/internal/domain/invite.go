package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInviteExpired     = errors.New("invite has expired")
	ErrInviteMaxUses     = errors.New("invite has reached maximum uses")
	ErrInviteAlreadyUsed = errors.New("invite has already been used")
	ErrInvalidInvite     = errors.New("invalid invite")
	ErrInviteNotFound    = errors.New("invite not found")
)

type ChatInvite struct {
	ID        uuid.UUID
	ChatID    uuid.UUID
	CreatedBy uuid.UUID
	Token     string
	ExpiresAt *time.Time
	MaxUses   *int
	UsesCount int
	CreatedAt time.Time
	UsedAt    *time.Time
}

func (i *ChatInvite) Validate() error {
	if i.Token == "" {
		return ErrInvalidInvite
	}
	if len(i.Token) > MaxInviteTokenLength {
		return ErrInvalidInvite
	}
	return nil
}

func (i *ChatInvite) IsActive() bool {
	if i.ExpiresAt != nil && time.Now().After(*i.ExpiresAt) {
		return false
	}

	if i.MaxUses != nil && i.UsesCount >= *i.MaxUses {
		return false
	}

	if i.MaxUses != nil && *i.MaxUses == 1 && i.UsedAt != nil {
		return false
	}

	return true
}

func (i *ChatInvite) Use() error {
	if !i.IsActive() {
		if i.ExpiresAt != nil && time.Now().After(*i.ExpiresAt) {
			return ErrInviteExpired
		}
		if i.MaxUses != nil && i.UsesCount >= *i.MaxUses {
			return ErrInviteMaxUses
		}
		return ErrInvalidInvite
	}

	i.UsesCount++

	if i.MaxUses != nil && *i.MaxUses == 1 {
		now := time.Now()
		i.UsedAt = &now
	}

	return nil
}

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func NewChatInvite(chatID, createdBy uuid.UUID, expiresAt *time.Time, maxUses *int) (*ChatInvite, error) {
	token, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	invite := &ChatInvite{
		ID:        uuid.New(),
		ChatID:    chatID,
		CreatedBy: createdBy,
		Token:     token,
		ExpiresAt: expiresAt,
		MaxUses:   maxUses,
		UsesCount: 0,
		CreatedAt: time.Now(),
	}

	if err := invite.Validate(); err != nil {
		return nil, err
	}

	return invite, nil
}
