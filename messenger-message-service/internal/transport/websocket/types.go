package websocket

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/google/uuid"
)

// ───────────────────────────────────────────────────────────
// WebSocket сообщения (клиент ↔ сервер)
// ───────────────────────────────────────────────────────────

// WSMessage — базовая структура любого сообщения через WebSocket
// Используется как для входящих, так и для исходящих сообщений
type WSMessage struct {
	// Type определяет тип события и способ обработки
	// Входящие: "subscribe", "unsubscribe", "send_message", "mark_read"
	// Исходящие: "new_message", "message_updated", "message_deleted", "status_updated", "error", "pong"
	Type string `json:"type"`

	// Data содержит полезные данные события
	// Структура зависит от типа — см. соответствующие *Payload типы ниже
	// Используем json.RawMessage для отложенной десериализации
	Data json.RawMessage `json:"data"`

	// RequestID опционально: корреляционный идентификатор для отслеживания запрос-ответ
	// Клиент может отправить его в запросе, сервер вернёт в ответе
	RequestID string `json:"request_id,omitempty"`
}

// ───────────────────────────────────────────────────────────
// Входящие сообщения (клиент → сервер)
// ───────────────────────────────────────────────────────────

// SubscribePayload — данные для подписки на чат
type SubscribePayload struct {
	// ChatID идентификатор чата, на который нужно подписаться
	// Если указан, ServerChannelID должен быть пустым
	ChatID string `json:"chat_id,omitempty"`

	// ServerChannelID идентификатор канала сервера
	// Если указан, ChatID должен быть пустым
	ServerChannelID string `json:"server_channel_id,omitempty"`
}

// IsValid проверяет, что указан ровно один идентификатор цели
func (p *SubscribePayload) IsValid() bool {
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	return (hasChat || hasChannel) && !(hasChat && hasChannel)
}

// UnsubscribePayload — данные для отписки от чата
type UnsubscribePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
}

func (p *UnsubscribePayload) IsValid() bool {
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	return (hasChat || hasChannel) && !(hasChat && hasChannel)
}

// SendMessagePayload — данные для отправки сообщения через WebSocket
// ⚠️ Важно: в нашей архитектуре отправка идёт через gRPC, а WS — только для пушей
// Но оставляем на случай, если захотим поддерживать отправку через WS
type SendMessagePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
	Content         string `json:"content"`
	ContentType     string `json:"content_type,omitempty"`
	ReplyToID       string `json:"reply_to_id,omitempty"`
	ServerID        string `json:"server_id,omitempty"`
}

func (p *SendMessagePayload) IsValid() bool {
	if p.Content == "" {
		return false
	}
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	return (hasChat || hasChannel) && !(hasChat && hasChannel)
}

// MarkReadPayload — данные для отметки сообщения как прочитанного
type MarkReadPayload struct {
	// MessageID идентификатор сообщения
	MessageID string `json:"message_id"`

	// ChatID опционально: для оптимизации можно указать чат, чтобы сервер не делал лишний запрос в БД
	ChatID string `json:"chat_id,omitempty"`
}

func (p *MarkReadPayload) IsValid() bool {
	return p.MessageID != ""
}

// ───────────────────────────────────────────────────────────
// Исходящие сообщения (сервер → клиент)
// ───────────────────────────────────────────────────────────

// 📦 MessagePayload — данные сообщения для отправки клиенту
// Это транспортная модель, оптимизированная под WebSocket/JSON
type MessagePayload struct {
	ID              string `json:"id"`
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
	SenderID        string `json:"sender_id"`
	Content         string `json:"content"`
	ContentType     string `json:"content_type"`

	// 🔥 Флаг: есть ли вложения (клиент может запросить детали отдельно)
	HasAttachment bool `json:"has_attachment"`

	// 🔥 Превью вложений (заполняется только если клиент запросил или это новый пуш)
	Attachments []*service.AttachmentPreview `json:"attachments,omitempty"`

	// Ответ на другое сообщение
	ReplyToID *string `json:"reply_to_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsDeleted bool      `json:"is_deleted"`
}

// NewMessagePayloadFromDomain конвертирует доменную сущность + вложения в payload
func NewMessagePayloadFromDomain(
	msg *domain.Message,
	chatID, serverChannelID string,
	attachments []*service.AttachmentPreview,
) *MessagePayload {
	payload := &MessagePayload{
		ID:            msg.ID.String(),
		SenderID:      msg.SenderID.String(),
		Content:       msg.Content,
		ContentType:   msg.ContentType,
		HasAttachment: msg.HasAttachment,
		Attachments:   attachments,
		CreatedAt:     msg.CreatedAt,
		UpdatedAt:     msg.UpdatedAt,
		IsDeleted:     msg.DeletedAt != nil,
	}

	if chatID != "" {
		payload.ChatID = chatID
	}
	if serverChannelID != "" {
		payload.ServerChannelID = serverChannelID
	}
	if msg.ReplyToID != nil {
		replyID := msg.ReplyToID.String()
		payload.ReplyToID = &replyID
	}

	return payload
}

// MessageUpdatedPayload — данные об обновлении сообщения
type MessageUpdatedPayload struct {
	ID          string                       `json:"id"`
	ChatID      string                       `json:"chat_id,omitempty"`
	NewContent  string                       `json:"new_content"`
	UpdatedAt   time.Time                    `json:"updated_at"`
	EditorID    string                       `json:"editor_id"`
	Attachments []*service.AttachmentPreview `json:"attachments,omitempty"`
}

// MessageDeletedPayload — данные об удалении сообщения
type MessageDeletedPayload struct {
	ID          string                       `json:"id"`
	ChatID      string                       `json:"chat_id,omitempty"`
	DeletedAt   time.Time                    `json:"deleted_at"`
	DeleterID   string                       `json:"deleter_id"`
	Attachments []*service.AttachmentPreview `json:"attachments,omitempty"`
}

// StatusUpdatedPayload — данные об обновлении статуса прочтения
type StatusUpdatedPayload struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"` // "sent", "delivered", "read"
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrorPayload — данные об ошибке
type ErrorPayload struct {
	// Code машинный код ошибки для программной обработки
	// Примеры: "invalid_message_format", "chat_not_found", "permission_denied"
	Code string `json:"code"`

	// Message человекочитаемое описание для отображения в UI
	Message string `json:"message"`

	// Details опционально: дополнительные данные для отладки
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewErrorPayload создаёт payload ошибки с заданным кодом и сообщением
func NewErrorPayload(code, message string) *ErrorPayload {
	return &ErrorPayload{
		Code:    code,
		Message: message,
	}
}

// WithDetails добавляет дополнительные данные к ошибке (для цепочки вызовов)
func (p *ErrorPayload) WithDetails(key string, value interface{}) *ErrorPayload {
	if p.Details == nil {
		p.Details = make(map[string]interface{})
	}
	p.Details[key] = value
	return p
}

// ───────────────────────────────────────────────────────────
// Служебные сообщения
// ───────────────────────────────────────────────────────────

// PongPayload — ответ на Ping (heartbeat)
// Пустой, сам факт получения достаточно
type PongPayload struct{}

// SubscribedPayload — подтверждение успешной подписки на чат
type SubscribedPayload struct {
	ChatID          string    `json:"chat_id,omitempty"`
	ServerChannelID string    `json:"server_channel_id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// UnsubscribedPayload — подтверждение успешной отписки
type UnsubscribedPayload struct {
	ChatID          string    `json:"chat_id,omitempty"`
	ServerChannelID string    `json:"server_channel_id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// ───────────────────────────────────────────────────────────
// Redis Pub/Sub события (межсервисная коммуникация)
// ───────────────────────────────────────────────────────────

// Эти события публикуются в Redis и потребляются WebSocket Hub
// для рассылки клиентам. Структуры сериализуются в JSON.

// RedisNewMessageEvent — событие нового сообщения для публикации в Redis
type RedisNewMessageEvent struct {
	Type        string          `json:"type"` // "new_message"
	ChatID      string          `json:"chat_id,omitempty"`
	Message     *MessagePayload `json:"message"`
	PublishedAt time.Time       `json:"published_at"`
}

// NewRedisNewMessageEvent создаёт новое событие с заполненными метаданными
func NewRedisNewMessageEvent(
	msg *domain.Message,
	chatID, serverChannelID string,
	attachments []*service.AttachmentPreview,
) *RedisNewMessageEvent {
	return &RedisNewMessageEvent{
		Type:        "new_message",
		ChatID:      chatID,
		Message:     NewMessagePayloadFromDomain(msg, chatID, serverChannelID, attachments),
		PublishedAt: time.Now(),
	}
}

// RedisMessageUpdatedEvent — событие обновления сообщения
type RedisMessageUpdatedEvent struct {
	Type        string                       `json:"type"` // "message_updated"
	ChatID      string                       `json:"chat_id,omitempty"`
	MessageID   string                       `json:"message_id"`
	NewContent  string                       `json:"new_content"`
	UpdatedAt   time.Time                    `json:"updated_at"`
	EditorID    string                       `json:"editor_id"`
	Attachments []*service.AttachmentPreview `json:"attachments,omitempty"`
	PublishedAt time.Time                    `json:"published_at"`
}

// RedisMessageDeletedEvent — событие удаления сообщения
type RedisMessageDeletedEvent struct {
	Type        string                       `json:"type"` // "message_deleted"
	ChatID      string                       `json:"chat_id,omitempty"`
	MessageID   string                       `json:"message_id"`
	DeletedAt   time.Time                    `json:"deleted_at"`
	DeleterID   string                       `json:"deleter_id"`
	Attachments []*service.AttachmentPreview `json:"attachments,omitempty"`
	PublishedAt time.Time                    `json:"published_at"`
}

// RedisStatusUpdatedEvent — событие обновления статуса прочтения
type RedisStatusUpdatedEvent struct {
	Type        string    `json:"type"` // "status_updated"
	ChatID      string    `json:"chat_id,omitempty"`
	MessageID   string    `json:"message_id"`
	UserID      string    `json:"user_id"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublishedAt time.Time `json:"published_at"`
}

// ───────────────────────────────────────────────────────────
// Вспомогательные функции
// ───────────────────────────────────────────────────────────

// MarshalWSMessage сериализует тип и данные в единое WebSocket сообщение
func MarshalWSMessage(msgType string, data interface{}) ([]byte, error) {
	// Сериализуем данные в JSON
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal  %w", err)
	}

	// Создаём обёртку
	wsMsg := WSMessage{
		Type: msgType,
		Data: dataBytes,
	}

	// Сериализуем обёртку
	return json.Marshal(wsMsg)
}

// UnmarshalWSMessage десериализует входящее сообщение и извлекает данные в целевую структуру
func UnmarshalWSMessage(raw []byte, target interface{}) error {
	var wsMsg WSMessage
	if err := json.Unmarshal(raw, &wsMsg); err != nil {
		return fmt.Errorf("failed to unmarshal ws message: %w", err)
	}

	// Десериализуем данные в целевую структуру
	if err := json.Unmarshal(wsMsg.Data, target); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}

// FormatRedisChannel формирует имя канала Redis для публикации/подписки
// Формат: {prefix}:{event_type}:{target_id}
// Пример: msg:new_message:chat_abc123
func FormatRedisChannel(prefix, eventType, targetID string) string {
	return fmt.Sprintf("%s%s:%s", prefix, eventType, targetID)
}

// ParseRedisChannel разбирает имя канала Redis на компоненты
// Возвращает: prefix, eventType, targetID, error
func ParseRedisChannel(channel string) (prefix, eventType, targetID string, err error) {
	// Ожидаемый формат: "prefix:eventType:targetID"
	// Разделяем по двоеточию
	// Но targetID может содержать двоеточия (если это составной ключ),
	// поэтому берём первые два элемента как prefix:eventType, остальное — targetID

	parts := strings.SplitN(channel, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid channel format: %s", channel)
	}

	return parts[0], parts[1], parts[2], nil
}

// ParseUUID безопасно парсит строку в uuid.UUID
func ParseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}

// UUIDPtrToString безопасно конвертирует *uuid.UUID в string
func UUIDPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
