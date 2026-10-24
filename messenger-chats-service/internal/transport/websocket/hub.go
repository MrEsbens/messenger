package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/MrEsbens/messenger-chats-service/pkg/redis"
	redisv8 "github.com/go-redis/redis/v8"
)

// MessageHandler — интерфейс для обработки входящих сообщений
type MessageHandler interface {
	HandleMessage(client *Client, msg WSMessage)
}

// Hub управляет подключёнными клиентами и рассылкой событий
type Hub struct {
	Register   chan *Client
	Unregister chan *Client

	mu                         sync.RWMutex
	clients                    map[string]*Client
	chatSubscriptions          map[string]map[string]bool // chat_id -> {client_id: true}
	serverChannelSubscriptions map[string]map[string]bool // channel_id -> {client_id: true}

	redis       redis.RedisClientInterface
	redisPrefix string
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewHub создаёт новый хаб
func NewHub(redisClient redis.RedisClientInterface, redisPrefix string) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		Register:                   make(chan *Client),
		Unregister:                 make(chan *Client),
		clients:                    make(map[string]*Client),
		chatSubscriptions:          make(map[string]map[string]bool),
		serverChannelSubscriptions: make(map[string]map[string]bool),
		redis:                      redisClient,
		redisPrefix:                redisPrefix,
		ctx:                        ctx,
		cancel:                     cancel,
	}

	go h.subscribeToRedis()
	return h
}

// Run запускает основной цикл хаба
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.addClient(client)
		case client := <-h.Unregister:
			h.removeClient(client)
		case <-h.ctx.Done():
			return
		}
	}
}

// Stop останавливает хаб
func (h *Hub) Stop() {
	h.cancel()
	h.mu.Lock()
	for _, client := range h.clients {
		close(client.send)
	}
	h.mu.Unlock()
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ID]; exists {
		// Клиент уже подключён — закрываем старый
		close(h.clients[client.ID].send)
		delete(h.clients, client.ID)
	}

	h.clients[client.ID] = client
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ID]; !exists {
		return
	}

	delete(h.clients, client.ID)
	h.removeAllSubscriptions(client.ID)
	// Канал закрывается в client.WriteLoop
}

func (h *Hub) removeAllSubscriptions(clientID string) {
	// Отписываем от всех чатов
	for chatID, subscribers := range h.chatSubscriptions {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(h.chatSubscriptions, chatID)
		}
	}
	// Отписываем от всех серверных каналов
	for channelID, subscribers := range h.serverChannelSubscriptions {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(h.serverChannelSubscriptions, channelID)
		}
	}
}

// SubscribeToChat подписывает клиента на чат
func (h *Hub) SubscribeToChat(clientID, chatID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[clientID]; !exists {
		return fmt.Errorf("client not found")
	}

	if h.chatSubscriptions[chatID] == nil {
		h.chatSubscriptions[chatID] = make(map[string]bool)
	}
	h.chatSubscriptions[chatID][clientID] = true
	return nil
}

// UnsubscribeFromChat отписывает клиента от чата
func (h *Hub) UnsubscribeFromChat(clientID, chatID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, exists := h.chatSubscriptions[chatID]; exists {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(h.chatSubscriptions, chatID)
		}
	}
}

// SubscribeToServerChannel подписывает на серверный канал
func (h *Hub) SubscribeToServerChannel(clientID, channelID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[clientID]; !exists {
		return fmt.Errorf("client not found")
	}

	if h.serverChannelSubscriptions[channelID] == nil {
		h.serverChannelSubscriptions[channelID] = make(map[string]bool)
	}
	h.serverChannelSubscriptions[channelID][clientID] = true
	return nil
}

// UnsubscribeFromServerChannel отписывает от серверного канала
func (h *Hub) UnsubscribeFromServerChannel(clientID, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, exists := h.serverChannelSubscriptions[channelID]; exists {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(h.serverChannelSubscriptions, channelID)
		}
	}
}

// BroadcastToChat отправляет сообщение всем подписчикам чата
func (h *Hub) BroadcastToChat(chatID string, message []byte) {
	h.mu.RLock()
	subscribers := h.chatSubscriptions[chatID]
	h.mu.RUnlock()

	for clientID := range subscribers {
		if client, exists := h.clients[clientID]; exists {
			select {
			case client.send <- message:
			default:
				// Буфер переполнен — пропускаем
			}
		}
	}
}

// BroadcastToServerChannel отправляет сообщение подписчикам канала
func (h *Hub) BroadcastToServerChannel(channelID string, message []byte) {
	h.mu.RLock()
	subscribers := h.serverChannelSubscriptions[channelID]
	h.mu.RUnlock()

	for clientID := range subscribers {
		if client, exists := h.clients[clientID]; exists {
			select {
			case client.send <- message:
			default:
				// Буфер переполнен — пропускаем
			}
		}
	}
}

// HandleMessage обрабатывает входящее сообщение от клиента
func (h *Hub) HandleMessage(client *Client, msg WSMessage) {
	switch msg.Type {
	case "subscribe":
		h.handleSubscribe(client, msg.Data)
	case "unsubscribe":
		h.handleUnsubscribe(client, msg.Data)
	case "ping":
		_ = client.SendSuccess("pong", PongPayload{})
	default:
		_ = client.SendError("unknown_message_type", fmt.Sprintf("type '%s' is not supported", msg.Type))
	}
}

func (h *Hub) handleSubscribe(client *Client, data json.RawMessage) {
	var payload SubscribePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		_ = client.SendError("invalid_payload", "failed to parse subscribe payload")
		return
	}

	if !payload.IsValid() {
		_ = client.SendError("invalid_subscribe_target", "specify exactly one of chat_id or server_channel_id")
		return
	}

	if payload.ChatID != "" {
		if err := h.SubscribeToChat(client.ID, payload.ChatID); err != nil {
			_ = client.SendError("subscribe_failed", err.Error())
			return
		}
		_ = client.SendSuccess("subscribed", map[string]string{"chat_id": payload.ChatID})
	}

	if payload.ServerChannelID != "" {
		if err := h.SubscribeToServerChannel(client.ID, payload.ServerChannelID); err != nil {
			_ = client.SendError("subscribe_failed", err.Error())
			return
		}
		_ = client.SendSuccess("subscribed", map[string]string{"server_channel_id": payload.ServerChannelID})
	}
}

func (h *Hub) handleUnsubscribe(client *Client, data json.RawMessage) {
	var payload UnsubscribePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		_ = client.SendError("invalid_payload", "failed to parse unsubscribe payload")
		return
	}

	if payload.ChatID != "" {
		h.UnsubscribeFromChat(client.ID, payload.ChatID)
		_ = client.SendSuccess("unsubscribed", map[string]string{"chat_id": payload.ChatID})
	}

	if payload.ServerChannelID != "" {
		h.UnsubscribeFromServerChannel(client.ID, payload.ServerChannelID)
		_ = client.SendSuccess("unsubscribed", map[string]string{"server_channel_id": payload.ServerChannelID})
	}
}

// subscribeToRedis подписывается на события из Redis и рассылает их клиентам
func (h *Hub) subscribeToRedis() {
	log.Printf("📡 [HUB] Starting Redis subscription...")

	channels := []string{
		h.redisPrefix + "created:*",
		h.redisPrefix + "updated:*",
		h.redisPrefix + "deleted:*",
		h.redisPrefix + "member_added:*",
		h.redisPrefix + "member_removed:*",
		h.redisPrefix + "member_left:*",
		h.redisPrefix + "member_role_changed:*",
		h.redisPrefix + "invite_accepted:*",
	}

	log.Printf("📡 [HUB] Subscribing to channels: %v", channels)

	pubsub := h.redis.PSubscribe(h.ctx, channels...)
	defer pubsub.Close()

	log.Printf("✅ [HUB] Redis subscription established")

	for {
		msg, err := pubsub.ReceiveMessage(h.ctx)
		if err != nil {
			if err == context.Canceled {
				log.Printf("🛑 [HUB] Redis subscription cancelled")
				return
			}
			log.Printf("⚠️ [HUB] Redis receive error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("📥 [HUB] Received Redis message: channel=%s, payload=%s", msg.Channel, msg.Payload)

		if err := h.handleRedisEvent(msg); err != nil {
			log.Printf("❌ [HUB] Error handling Redis event: %v", err)
		}
	}
}

func (h *Hub) handleRedisEvent(msg *redisv8.Message) error {
	eventType, targetID, err := ParseRedisChannel(msg.Channel, h.redisPrefix)
	if err != nil {
		return fmt.Errorf("failed to parse channel: %w", err)
	}

	// Очищаем префиксы
	targetID = stripTargetPrefix(targetID)

	payloadBytes := []byte(msg.Payload)

	switch eventType {
	case "created":
		log.Printf("📤 [HUB] Broadcasting chat_created to %s", targetID)
		return h.handleChatCreated(targetID, payloadBytes)

	case "updated":
		log.Printf("📤 [HUB] Broadcasting chat_updated to %s", targetID)
		return h.handleChatUpdated(targetID, payloadBytes)

	case "deleted":
		log.Printf("📤 [HUB] Broadcasting chat_deleted to %s", targetID)
		return h.handleChatDeleted(targetID, payloadBytes)

	case "member_added":
		log.Printf("📤 [HUB] Broadcasting member_added to %s", targetID)
		return h.handleMemberAdded(targetID, payloadBytes)

	case "member_removed":
		log.Printf("📤 [HUB] Broadcasting member_removed to %s", targetID)
		return h.handleMemberRemoved(targetID, payloadBytes)

	case "member_left":
		log.Printf("📤 [HUB] Broadcasting member_left to %s", targetID)
		return h.handleMemberLeft(targetID, payloadBytes)

	case "member_role_changed":
		log.Printf("📤 [HUB] Broadcasting member_role_changed to %s", targetID)
		return h.handleMemberRoleChanged(targetID, payloadBytes)

	case "invite_accepted":
		log.Printf("📤 [HUB] Broadcasting invite_accepted to %s", targetID)
		return h.handleInviteAccepted(targetID, payloadBytes)

	default:
		log.Printf("❌ [HUB] Unknown event type: '%s'", eventType)
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}

// Обработчики событий (рассылают клиентам)

func (h *Hub) handleChatCreated(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleChatUpdated(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleChatDeleted(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleMemberAdded(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleMemberRemoved(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleMemberLeft(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleMemberRoleChanged(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

func (h *Hub) handleInviteAccepted(chatID string, payload []byte) error {
	h.BroadcastToChat(chatID, payload)
	return nil
}

// ───────────────────────────────────────────────────────────
// Хелперы для парсинга каналов Redis
// ───────────────────────────────────────────────────────────

func ParseRedisChannel(channel, redisPrefix string) (eventType, targetID string, err error) {
	if !strings.HasPrefix(channel, redisPrefix) {
		return "", "", fmt.Errorf("channel does not start with prefix %s: %s", redisPrefix, channel)
	}
	rest := strings.TrimPrefix(channel, redisPrefix)

	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid channel format after prefix: %s", rest)
	}

	return parts[0], parts[1], nil
}

func splitChannel(channel string) []string {
	// Разделяем по ":", но только первые 3 части
	// "chat:chat:created:abc123" → ["chat", "chat:created", "abc123"]
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(channel) && len(parts) < 3; i++ {
		if channel[i] == ':' {
			if len(parts) == 2 {
				// Последняя часть — всё до конца
				parts = append(parts, channel[start:])
				return parts
			}
			parts = append(parts, channel[start:i])
			start = i + 1
		}
	}
	parts = append(parts, channel[start:])
	return parts
}

func stripTargetPrefix(targetID string) string {
	targetID = stripPrefix(targetID, "chat_")
	targetID = stripPrefix(targetID, "channel_")
	return targetID
}

func stripPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
