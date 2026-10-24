package domain

import (
	"time"

	"github.com/google/uuid"
)

// ModerationRequest — событие для асинхронной проверки сообщения
type ModerationRequest struct {
	EventID     string    `json:"event_id"`   // uuid для идемпотентности
	EventType   string    `json:"event_type"` // "moderation.request"
	Version     string    `json:"version"`    // "1.0"
	PublishedAt time.Time `json:"published_at"`

	// Данные сообщения
	MessageID   string    `json:"message_id"`
	ChatID      string    `json:"chat_id"`
	ServerID    string    `json:"server_id,omitempty"` // пустой = чат не в сервере
	SenderID    string    `json:"sender_id"`
	Content     string    `json:"content"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`

	// Контекст для логирования
	TraceID string `json:"trace_id,omitempty"`
}

func NewModerationRequest(msg *Message, serverID uuid.UUID) *ModerationRequest {
	var serverIDStr string
	if serverID != uuid.Nil {
		serverIDStr = serverID.String()
	}

	return &ModerationRequest{
		EventID:     uuid.New().String(),
		EventType:   "moderation.request",
		Version:     "1.0",
		PublishedAt: time.Now(),

		MessageID:   msg.ID.String(),
		ChatID:      msg.ChatID.String(),
		ServerID:    serverIDStr,
		SenderID:    msg.SenderID.String(),
		Content:     msg.Content,
		ContentType: msg.ContentType,
		CreatedAt:   msg.CreatedAt,
	}
}
