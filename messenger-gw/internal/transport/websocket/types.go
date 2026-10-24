package websocket

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ───────────────────────────────────────────────────────────
// Входящие команды (клиент → Gateway)
// ───────────────────────────────────────────────────────────

type Command struct {
	Cmd       string          `json:"cmd"`
	Payload   json.RawMessage `json:"payload"`
	RequestID string          `json:"request_id,omitempty"`
}

// ─── Payloads для команд ─────────────────────────────────

type SendMessagePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
	Content         string `json:"content"`
	ContentType     string `json:"content_type,omitempty"`
	ReplyToID       string `json:"reply_to_id,omitempty"`
	ServerID        string `json:"server_id,omitempty"`
}

func (p *SendMessagePayload) IsValid() error {
	if p.Content == "" {
		return fmt.Errorf("content is required")
	}
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	if !hasChat && !hasChannel {
		return fmt.Errorf("specify chat_id or server_channel_id")
	}
	if hasChat && hasChannel {
		return fmt.Errorf("specify exactly one of chat_id or server_channel_id")
	}
	if p.ContentType == "" {
		p.ContentType = "text"
	}
	if p.ContentType != "text" && p.ContentType != "markdown" && p.ContentType != "code" {
		return fmt.Errorf("invalid content_type: allowed text, markdown, code")
	}
	return nil
}

type SubscribePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
}

func (p *SubscribePayload) IsValid() error {
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	if !hasChat && !hasChannel {
		return fmt.Errorf("specify chat_id or server_channel_id")
	}
	if hasChat && hasChannel {
		return fmt.Errorf("specify exactly one of chat_id or server_channel_id")
	}
	return nil
}

type MarkReadPayload struct {
	MessageID string `json:"message_id"`
	ChatID    string `json:"chat_id,omitempty"`
}

func (p *MarkReadPayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}

// 🔥 Новые команды
type EditMessagePayload struct {
	MessageID  string `json:"message_id"`
	NewContent string `json:"content"`
}

func (p *EditMessagePayload) IsValid() error {
	if p.MessageID == "" || p.NewContent == "" {
		return fmt.Errorf("message_id and content are required")
	}
	return nil
}

type DeleteMessagePayload struct {
	MessageID string `json:"message_id"`
}

func (p *DeleteMessagePayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}

type GetMessagesPayload struct {
	ChatID string  `json:"chat_id"`
	Limit  int     `json:"limit,omitempty"`
	Offset int     `json:"offset,omitempty"`
	After  *string `json:"after,omitempty"`
}

func (p *GetMessagesPayload) IsValid() error {
	if p.ChatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return nil
}

type GetMessageStatusesPayload struct {
	MessageID string `json:"message_id"`
}

func (p *GetMessageStatusesPayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}

type GetMessageWithAttachmentsPayload struct {
	MessageID string `json:"message_id"`
}

func (p *GetMessageWithAttachmentsPayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}

// ───────────────────────────────────────────────────────────
// Исходящие ответы (Gateway → клиент)
// ───────────────────────────────────────────────────────────

type Response struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	RequestID string      `json:"request_id,omitempty"`
}

type MessageSentPayload struct {
	MessageID string    `json:"message_id"`
	CreatedAt time.Time `json:"created_at"`
}

type SubscribedPayload struct {
	ChatID          string    `json:"chat_id,omitempty"`
	ServerChannelID string    `json:"server_channel_id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

type UnsubscribedPayload struct {
	ChatID          string    `json:"chat_id,omitempty"`
	ServerChannelID string    `json:"server_channel_id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

type PongPayload struct{}

// 🔥 AttachmentPreview — ДОЛЖЕН СОВПАДАТЬ с messages-service
type AttachmentPreview struct {
	FileID    string `json:"file_id"`
	PublicURL string `json:"public_url"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	FileName  string `json:"file_name"`
}

type MessagePayload struct {
	ID              string               `json:"id"`
	ChatID          string               `json:"chat_id"`
	SenderID        string               `json:"sender_id"`
	Content         string               `json:"content"`
	ContentType     string               `json:"content_type"`
	HasAttachment   bool                 `json:"has_attachment"`
	Attachments     []*AttachmentPreview `json:"attachments,omitempty"`
	ReplyToID       *string              `json:"reply_to_id,omitempty"`
	ServerID        string               `json:"server_id,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	IsDeleted       bool                 `json:"is_deleted"`
}

type NewMessagePayload struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	Message     *MessagePayload      `json:"message"`
	Attachments []*AttachmentPreview `json:"attachments,omitempty"`
	PublishedAt time.Time            `json:"published_at"`
}

type MessageUpdatedPayload struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	MessageID   string               `json:"message_id"`
	NewContent  string               `json:"new_content"`
	UpdatedAt   time.Time            `json:"updated_at"`
	EditorID    string               `json:"editor_id"`
	Attachments []*AttachmentPreview `json:"attachments,omitempty"`
}

type MessageDeletedPayload struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	MessageID   string               `json:"message_id"`
	DeletedAt   time.Time            `json:"deleted_at"`
	DeleterID   string               `json:"deleter_id"`
	Attachments []*AttachmentPreview `json:"attachments,omitempty"`
}

type StatusUpdatedPayload struct {
	Type      string    `json:"type"`
	ChatID    string    `json:"chat_id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MessageEditedPayload struct {
	ID         string    `json:"id"`
	ChatID     string    `json:"chat_id"`
	NewContent string    `json:"new_content"`
	UpdatedAt  time.Time `json:"updated_at"`
	EditorID   string    `json:"editor_id"`
}

type ErrorPayload struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type GetMessagesResponsePayload struct {
	Messages []*MessagePayload `json:"messages"`
	HasMore  bool              `json:"has_more"`
}

type GetMessageStatusesResponsePayload struct {
	Statuses []MessageStatusPayload `json:"statuses"`
}

type MessageStatusPayload struct {
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetMessageWithAttachmentsResponsePayload struct {
	Message     *MessagePayload      `json:"message"`
	Attachments []*AttachmentPreview `json:"attachments"`
}

// ───────────────────────────────────────────────────────────
// Redis Pub/Sub каналы
// ───────────────────────────────────────────────────────────

func FormatRedisChannel(prefix, eventType, targetID string) string {
	return fmt.Sprintf("%s%s:%s", prefix, eventType, targetID)
}

func ParseRedisChannel(channel string) (prefix, eventType, targetID string, err error) {
	parts := strings.SplitN(channel, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid channel format: %s", channel)
	}
	return parts[0], parts[1], parts[2], nil
}

// ───────────────────────────────────────────────────────────
// Вспомогательные функции
// ───────────────────────────────────────────────────────────

func MarshalResponse(resp *Response) ([]byte, error) {
	return json.Marshal(resp)
}

func UnmarshalCommand(raw []byte) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(raw, &cmd); err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}
	return &cmd, nil
}

func UnmarshalRedisEvent(payload []byte, target interface{}) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("failed to unmarshal redis event: %w", err)
	}
	return nil
}

func ParseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}

func UUIDPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func StringPtr(s string) *string {
	return &s
}