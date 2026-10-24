package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/google/uuid"
)

type MessageHandler interface {
	HandleMessage(client *Client, msg WSMessage)
}

type Handler struct {
	messageService service.MessageServiceInterface
	redisPrefix    string
}

func NewHandler(messageService service.MessageServiceInterface, redisPrefix string) *Handler {
	return &Handler{
		messageService: messageService,
		redisPrefix:    redisPrefix,
	}
}

func (h *Handler) HandleMessage(client *Client, msg WSMessage) {
	client.LastActivity = time.Now()

	switch msg.Type {
	case "subscribe":
		h.handleSubscribe(client, msg.Data)
	case "unsubscribe":
		h.handleUnsubscribe(client, msg.Data)
	case "send_message":
		h.handleSendMessage(client, msg.Data)
	case "mark_read":
		h.handleMarkRead(client, msg.Data)
	case "ping":
		_ = client.SendSuccess("pong", PongPayload{})
	default:
		_ = client.SendError("unknown_message_type", fmt.Sprintf("type '%s' is not supported", msg.Type))
		log.Printf("⚠️ Unknown message type from client %s: %s", client.ID, msg.Type)
	}
}

func (h *Handler) handleSubscribe(client *Client, rawData json.RawMessage) {
	var payload SubscribePayload
	if err := json.Unmarshal(rawData, &payload); err != nil {
		_ = client.SendError("invalid_subscribe_format", "failed to parse subscribe payload")
		return
	}

	if !payload.IsValid() {
		_ = client.SendError("invalid_subscribe_target", "specify exactly one of chat_id or server_channel_id")
		return
	}

	if payload.ChatID != "" {
		if !IsAuthorized(client.ID, payload.ChatID) {
			_ = client.SendError("permission_denied", "not a member of this chat")
			return
		}
		if err := client.Hub.SubscribeToChat(client.ID, payload.ChatID); err != nil {
			_ = client.SendError("subscribe_failed", err.Error())
			return
		}
	} else if payload.ServerChannelID != "" {
		if !IsAuthorized(client.ID, payload.ServerChannelID) {
			_ = client.SendError("permission_denied", "not a member of this server channel")
			return
		}
		if err := client.Hub.SubscribeToServerChannel(client.ID, payload.ServerChannelID); err != nil {
			_ = client.SendError("subscribe_failed", err.Error())
			return
		}
	}

	_ = client.SendSuccess("subscribed", SubscribedPayload{
		ChatID:          payload.ChatID,
		ServerChannelID: payload.ServerChannelID,
		Timestamp:       time.Now(),
	})

	log.Printf("✅ Client %s subscribed: chat=%s, channel=%s",
		client.ID, payload.ChatID, payload.ServerChannelID)
}

func (h *Handler) handleUnsubscribe(client *Client, rawData json.RawMessage) {
	var payload UnsubscribePayload
	if err := json.Unmarshal(rawData, &payload); err != nil {
		_ = client.SendError("invalid_unsubscribe_format", "failed to parse unsubscribe payload")
		return
	}

	if !payload.IsValid() {
		_ = client.SendError("invalid_unsubscribe_target", "specify exactly one of chat_id or server_channel_id")
		return
	}

	if payload.ChatID != "" {
		client.Hub.UnsubscribeFromChat(client.ID, payload.ChatID)
	} else if payload.ServerChannelID != "" {
		client.Hub.UnsubscribeFromServerChannel(client.ID, payload.ServerChannelID)
	}

	_ = client.SendSuccess("unsubscribed", UnsubscribedPayload{
		ChatID:          payload.ChatID,
		ServerChannelID: payload.ServerChannelID,
		Timestamp:       time.Now(),
	})

	log.Printf("✅ Client %s unsubscribed: chat=%s, channel=%s",
		client.ID, payload.ChatID, payload.ServerChannelID)
}

func (h *Handler) handleSendMessage(client *Client, rawData json.RawMessage) {
	var payload SendMessagePayload
	if err := json.Unmarshal(rawData, &payload); err != nil {
		_ = client.SendError("invalid_message_format", "failed to parse message payload")
		return
	}

	if !payload.IsValid() {
		_ = client.SendError("invalid_message_target", "specify exactly one of chat_id or server_channel_id with non-empty content")
		return
	}

	chatID, err := parseUUID(payload.ChatID)
	if err != nil && payload.ChatID != "" {
		_ = client.SendError("invalid_chat_id", err.Error())
		return
	}

	var replyToID *uuid.UUID
	if payload.ReplyToID != "" {
		replyToID, err = parseUUIDPtr(payload.ReplyToID)
		if err != nil {
			_ = client.SendError("invalid_reply_to_id", err.Error())
			return
		}
	}

	var serverID uuid.UUID
	if payload.ServerID != "" {
		serverID, err = uuid.Parse(payload.ServerID)
		if err != nil {
			_ = client.SendError("invalid_server_id", err.Error())
			return
		}
	}

	contentType := payload.ContentType
	if contentType == "" {
		contentType = domain.ContentTypeText
	}
	if !domain.IsValidContentType(contentType) {
		_ = client.SendError("invalid_content_type", fmt.Sprintf("allowed: %s, %s, %s",
			domain.ContentTypeText, domain.ContentTypeMarkdown, domain.ContentTypeCode))
		return
	}

	msg, err := h.messageService.SendMessage(
		context.Background(),
		chatID,
		uuid.MustParse(client.ID),
		serverID,
		payload.Content,
		contentType,
		replyToID,
		false,
	)
	if err != nil {
		_ = client.SendError("send_failed", fmt.Sprintf("failed to send message: %v", err))
		log.Printf("❌ Failed to send message from client %s: %v", client.ID, err)
		return
	}

	_ = client.SendSuccess("message_sent", map[string]string{
		"message_id": msg.ID.String(),
		"created_at": msg.CreatedAt.Format(time.RFC3339),
	})

	log.Printf("📤 Message sent by client %s: id=%s, chat=%s",
		client.ID, msg.ID, chatID)
}

func (h *Handler) handleMarkRead(client *Client, rawData json.RawMessage) {
	var payload MarkReadPayload
	if err := json.Unmarshal(rawData, &payload); err != nil {
		_ = client.SendError("invalid_mark_read_format", "failed to parse mark_read payload")
		return
	}

	if !payload.IsValid() {
		_ = client.SendError("invalid_message_id", "message_id is required")
		return
	}

	messageID, err := parseUUID(payload.MessageID)
	if err != nil {
		_ = client.SendError("invalid_message_id", err.Error())
		return
	}

	userID := uuid.MustParse(client.ID)

	if err := h.messageService.UpdateMessageStatus(
		context.Background(),
		messageID,
		userID,
		domain.StatusRead,
	); err != nil {
		_ = client.SendError("status_update_failed", fmt.Sprintf("failed to update status: %v", err))
		log.Printf("❌ Failed to update status for client %s: %v", client.ID, err)
		return
	}

	_ = client.SendSuccess("status_updated", map[string]string{
		"message_id": payload.MessageID,
		"status":     domain.StatusRead,
		"updated_at": time.Now().Format(time.RFC3339),
	})

	if payload.ChatID != "" {
		_ = h.messageService.MarkChatAsRead(context.Background(), uuid.MustParse(payload.ChatID), userID)
	}

	log.Printf("👁️ Message %s marked as read by client %s", payload.MessageID, client.ID)
}

func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}

func parseUUIDPtr(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
