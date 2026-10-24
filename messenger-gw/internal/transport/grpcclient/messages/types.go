package messages

import (
	"time"
	"github.com/google/uuid"
)

// ─── Доменные модели (ядро) ──────────────────────────────────────────────

// Message — доменная модель сообщения
type Message struct {
	ID            uuid.UUID
	SenderID      uuid.UUID
	ChatID        uuid.UUID
	Content       string
	ContentType   string // "text", "markdown", "code"
	HasAttachment bool
	Attachments   []Attachment
	ReplyToID     *uuid.UUID
	ServerID      uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	IsDeleted     bool
}

// Attachment — доменная модель вложения
type Attachment struct {
	FileID    uuid.UUID
	PublicURL string
	MimeType  string
	FileSize  int64
	FileName  string
}

// MessageStatus — статус прочтения сообщения
type MessageStatus struct {
	UserID    uuid.UUID
	Status    string // "sent", "delivered", "read"
	UpdatedAt time.Time
}

// ─── Запросы/ответы: SendMessage ─────────────────────────────────────────

type SendMessageRequest struct {
	SenderID    uuid.UUID
	ChatID      uuid.UUID
	ServerID    uuid.UUID
	Content     string
	ContentType string
	ReplyToID   *uuid.UUID
}

type SendMessageResponse struct {
	MessageID uuid.UUID
	CreatedAt time.Time
}

// ─── Запросы/ответы: UpdateMessageStatus ─────────────────────────────────

type UpdateStatusRequest struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Status    string
}

// ─── Запросы/ответы: EditMessage ─────────────────────────────────────────

type EditMessageRequest struct {
	MessageID  uuid.UUID
	EditorID   uuid.UUID
	NewContent string
}

// ─── Запросы/ответы: DeleteMessage ───────────────────────────────────────

type DeleteMessageRequest struct {
	MessageID uuid.UUID
	DeleterID uuid.UUID
}

// ─── Запросы/ответы: GetMessages ─────────────────────────────────────────

type GetMessagesRequest struct {
	ChatID uuid.UUID
	Limit  int
	Offset int
	After  *time.Time
}

type GetMessagesResponse struct {
	Messages []Message
	HasMore  bool
}

// ─── Запросы/ответы: GetMessageStatuses ──────────────────────────────────

type GetMessageStatusesRequest struct {
	MessageID uuid.UUID
}

type GetMessageStatusesResponse struct {
	Statuses []MessageStatus
}

// ─── Запросы/ответы: GetMessageWithAttachments ───────────────────────────

type GetMessageWithAttachmentsResponse struct {
	Message     Message
	Attachments []Attachment
}