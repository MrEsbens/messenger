package websocket

import (
	"encoding/json"
	"time"
)

// ───────────────────────────────────────────────────────────
// Входящие сообщения от клиента
// ───────────────────────────────────────────────────────────

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type SubscribePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
}

func (p *SubscribePayload) IsValid() bool {
	// Ровно один из двух должен быть указан
	chatSet := p.ChatID != ""
	channelSet := p.ServerChannelID != ""
	return (chatSet || channelSet) && !(chatSet && channelSet)
}

type UnsubscribePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
}

type MarkReadPayload struct {
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id"`
}

type PingPayload struct{}
type PongPayload struct{}

// ───────────────────────────────────────────────────────────
// Исходящие сообщения клиенту
// ───────────────────────────────────────────────────────────

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ───────────────────────────────────────────────────────────
// События из Redis Pub/Sub (для рассылки клиентам)
// ───────────────────────────────────────────────────────────

type ChatCreatedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatUpdatedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatDeletedEvent struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type MemberAddedEvent struct {
	Type    string `json:"type"`
	ChatID  string `json:"chat_id"`
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	AddedBy string `json:"added_by"`
}

type MemberRemovedEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	RemovedBy string `json:"removed_by"`
}

type MemberLeftEvent struct {
	Type   string `json:"type"`
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
}

type MemberRoleChangedEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	OldRole   string `json:"old_role"`
	NewRole   string `json:"new_role"`
	UpdatedBy string `json:"updated_by"`
}

type InviteAcceptedEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	InvitedBy string `json:"invited_by"`
}
