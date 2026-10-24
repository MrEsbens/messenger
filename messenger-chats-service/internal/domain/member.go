package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidRole        = errors.New("invalid membership role")
	ErrNotMember          = errors.New("user is not a member of this chat")
	ErrAlreadyMember      = errors.New("user is already a member")
	ErrCannotKickYourself = errors.New("cannot kick yourself")
	ErrCannotKickOwner    = errors.New("cannot kick the owner")
	ErrInsufficientRole   = errors.New("insufficient role to perform this action")
	ErrMemberNotFound     = errors.New("member not found")
)

type ChatMember struct {
	ID                uuid.UUID
	ChatID            uuid.UUID
	UserID            uuid.UUID
	Role              MembershipRole
	JoinedAt          time.Time
	LeftAt            *time.Time
	NotificationMode  NotificationMode
	IsMutedUntil      *time.Time
	LastReadMessageID *uuid.UUID
	LastActivityAt    *time.Time
}

func (m *ChatMember) Validate() error {
	if !IsValidMembershipRole(m.Role) {
		return ErrInvalidRole
	}
	if !IsValidNotificationMode(m.NotificationMode) {
		return errors.New("invalid notification mode")
	}
	return nil
}

func (m *ChatMember) IsActive() bool {
	return m.LeftAt == nil
}

func (m *ChatMember) IsBanned() bool {
	return m.Role == RoleBanned
}

func (m *ChatMember) IsMuted() bool {
	if m.Role != RoleMuted || m.IsMutedUntil == nil {
		return false
	}
	return time.Now().Before(*m.IsMutedUntil)
}

func (m *ChatMember) Leave() {
	now := time.Now()
	m.LeftAt = &now
}

func (m *ChatMember) UpdateRole(newRole MembershipRole) {
	m.Role = newRole
	m.IsMutedUntil = nil
}

func (m *ChatMember) UpdateNotificationMode(mode NotificationMode) {
	m.NotificationMode = mode
}

func (m *ChatMember) UpdateLastRead(messageID uuid.UUID) {
	m.LastReadMessageID = &messageID
	now := time.Now()
	m.LastActivityAt = &now
}

func NewChatMember(chatID, userID uuid.UUID, role MembershipRole) (*ChatMember, error) {
	member := &ChatMember{
		ID:               uuid.New(),
		ChatID:           chatID,
		UserID:           userID,
		Role:             role,
		JoinedAt:         time.Now(),
		NotificationMode: NotifyAll,
	}

	if err := member.Validate(); err != nil {
		return nil, err
	}

	return member, nil
}
