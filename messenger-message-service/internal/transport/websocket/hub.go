package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/MrEsbens/messenger-message-service/pkg/redis"
	redisv8 "github.com/go-redis/redis/v8"
)

// Hub управляет всеми подключёнными клиентами и маршрутизирует сообщения
type Hub struct {
	// Канал для регистрации новых клиентов
	Register chan *Client
	// Канал для отписки клиентов
	Unregister chan *Client

	// Все активные клиенты: user_id → Client
	clients map[string]*Client

	// Подписки по чатам: chat_id → set of user_id
	chatSubscriptions map[string]map[string]bool
	// Подписки по каналам сервера: server_channel_id → set of user_id
	serverChannelSubscriptions map[string]map[string]bool

	mu sync.RWMutex

	redis       redis.RedisClientInterface
	redisPrefix string
	Handler     MessageHandler
	ctx         context.Context
	cancel      context.CancelFunc

	running bool
}

func NewHub(redisClient redis.RedisClientInterface, redisPrefix string, handler MessageHandler) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		Register:                   make(chan *Client),
		Unregister:                 make(chan *Client),
		clients:                    make(map[string]*Client),
		chatSubscriptions:          make(map[string]map[string]bool),
		serverChannelSubscriptions: make(map[string]map[string]bool),
		redis:                      redisClient,
		redisPrefix:                redisPrefix,
		Handler:                    handler,
		ctx:                        ctx,
		cancel:                     cancel,
		running:                    false,
	}
}

func (h *Hub) Run() {
	if h.running {
		log.Println("⚠️ Hub is already running")
		return
	}
	h.running = true

	log.Println("🚀 WebSocket Hub started")

	go h.subscribeToRedis()

	for {
		select {
		case client := <-h.Register:
			h.addClient(client)
			log.Printf("📡 Client connected: %s (total: %d)", client.ID, len(h.clients))

		case client := <-h.Unregister:
			h.removeClient(client)
			log.Printf("📡 Client disconnected: %s (total: %d)", client.ID, len(h.clients))

		case <-h.ctx.Done():
			log.Println("🛑 Hub shutting down...")
			h.shutdown()
			return
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ID]; exists {
		log.Printf("⚠️ Client %s already registered, closing duplicate", client.ID)
		client.Close()
		return
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
}

func (h *Hub) removeAllSubscriptions(clientID string) {
	for chatID, subscribers := range h.chatSubscriptions {
		if subscribers[clientID] {
			delete(subscribers, clientID)
			if len(subscribers) == 0 {
				delete(h.chatSubscriptions, chatID)
			}
		}
	}

	for channelID, subscribers := range h.serverChannelSubscriptions {
		if subscribers[clientID] {
			delete(subscribers, clientID)
			if len(subscribers) == 0 {
				delete(h.serverChannelSubscriptions, channelID)
			}
		}
	}
}

func (h *Hub) SubscribeToChat(clientID, chatID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	if h.chatSubscriptions[chatID] == nil {
		h.chatSubscriptions[chatID] = make(map[string]bool)
	}

	h.chatSubscriptions[chatID][clientID] = true

	client.Subscribe(chatID)

	log.Printf("📌 Client %s subscribed to chat %s", clientID, chatID)
	return nil
}

func (h *Hub) UnsubscribeFromChat(clientID, chatID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, exists := h.chatSubscriptions[chatID]; exists {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(h.chatSubscriptions, chatID)
		}
	}

	if client, exists := h.clients[clientID]; exists {
		client.Unsubscribe(chatID)
	}

	log.Printf("📌 Client %s unsubscribed from chat %s", clientID, chatID)
}

func (h *Hub) SubscribeToServerChannel(clientID, channelID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	if h.serverChannelSubscriptions[channelID] == nil {
		h.serverChannelSubscriptions[channelID] = make(map[string]bool)
	}

	h.serverChannelSubscriptions[channelID][clientID] = true
	client.Subscribe(channelID)

	log.Printf("📌 Client %s subscribed to server channel %s", clientID, channelID)
	return nil
}

func (h *Hub) UnsubscribeFromServerChannel(clientID, channelID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, exists := h.serverChannelSubscriptions[channelID]; exists {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(h.serverChannelSubscriptions, channelID)
		}
	}

	if client, exists := h.clients[clientID]; exists {
		client.Unsubscribe(channelID)
	}

	log.Printf("📌 Client %s unsubscribed from server channel %s", clientID, channelID)
}

// BroadcastToChat отправляет сообщение всем подписчикам чата
// Это основной метод для доставки сообщений в реальном времени
func (h *Hub) BroadcastToChat(chatID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	log.Printf("🔎 BroadcastToChat: looking for chat_id='%s'", chatID)
	log.Printf("🔎 Available chat keys: %v", maps.Keys(h.chatSubscriptions))

	subscribers := h.chatSubscriptions[chatID]
	if len(subscribers) == 0 {
		return
	}

	sentCount := 0
	failedCount := 0

	for clientID := range subscribers {
		if client, exists := h.clients[clientID]; exists {
			select {
			case client.Send <- message:
				sentCount++
			default:
				log.Printf("⚠️ Send buffer full for client %s, skipping message", clientID)
				failedCount++
			}
		}
	}

	if sentCount > 0 {
		log.Printf("📤 Broadcast to chat %s: sent=%d, failed=%d", chatID, sentCount, failedCount)
	}
}

func (h *Hub) BroadcastToServerChannel(channelID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	subscribers := h.serverChannelSubscriptions[channelID]
	if len(subscribers) == 0 {
		return
	}

	for clientID := range subscribers {
		if client, exists := h.clients[clientID]; exists {
			select {
			case client.Send <- message:
			default:
				log.Printf("⚠️ Send buffer full for client %s, skipping message", clientID)
			}
		}
	}
}

func (h *Hub) SendToUser(userID string, message []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.clients[userID]
	if !exists {
		return fmt.Errorf("client %s not found (offline)", userID)
	}

	select {
	case client.Send <- message:
		return nil
	default:
		return fmt.Errorf("send buffer full for client %s", userID)
	}
}

func (h *Hub) subscribeToRedis() {
	channels := []string{
		h.redisPrefix + "new_message:*",
		h.redisPrefix + "message_updated:*",
		h.redisPrefix + "message_deleted:*",
		h.redisPrefix + "status_updated:*",
	}

	pubsub := h.redis.PSubscribe(h.ctx, channels...)
	defer pubsub.Close()

	if _, err := pubsub.Receive(h.ctx); err != nil {
		log.Printf("❌ Failed to subscribe to Redis: %v", err)
		return
	}

	log.Printf("📡 Subscribed to Redis channels: %v", channels)

	for {
		msg, err := pubsub.ReceiveMessage(h.ctx)
		if err != nil {
			if err == context.Canceled {
				log.Println("📡 Redis subscription stopped (context canceled)")
				return
			}
			log.Printf("⚠️ Redis receive error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if err := h.handleRedisEvent(msg); err != nil {
			log.Printf("⚠️ Failed to handle Redis event: %v", err)
		}
	}
}

func (h *Hub) handleRedisEvent(msg *redisv8.Message) error {
	_, eventType, targetID, err := ParseRedisChannel(msg.Channel)
	if err != nil {
		return fmt.Errorf("failed to parse channel %s: %w", msg.Channel, err)
	}

	targetID = stripTargetPrefix(targetID)

	switch eventType {
	case "new_message":
		return h.handleNewMessageEvent(targetID, msg.Payload)

	case "message_updated":
		return h.handleMessageUpdatedEvent(targetID, msg.Payload)

	case "message_deleted":
		return h.handleMessageDeletedEvent(targetID, msg.Payload)

	case "status_updated":
		return h.handleStatusUpdatedEvent(targetID, msg.Payload)

	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}

func (h *Hub) handleNewMessageEvent(chatID, payload string) error {
	var event RedisNewMessageEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("failed to unmarshal new_message event: %w", err)
	}

	wsMessage, err := MarshalWSMessage("new_message", event.Message)
	if err != nil {
		return fmt.Errorf("failed to marshal ws message: %w", err)
	}

	h.BroadcastToChat(chatID, wsMessage)
	return nil
}

func (h *Hub) handleMessageUpdatedEvent(chatID, payload string) error {
	var event RedisMessageUpdatedEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("failed to unmarshal message_updated event: %w", err)
	}

	wsPayload := MessageUpdatedPayload{
		ID:         event.MessageID,
		ChatID:     event.ChatID,
		NewContent: event.NewContent,
		UpdatedAt:  event.UpdatedAt,
		EditorID:   event.EditorID,
	}

	wsMessage, err := MarshalWSMessage("message_updated", wsPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal ws message: %w", err)
	}

	h.BroadcastToChat(chatID, wsMessage)
	return nil
}

func (h *Hub) handleMessageDeletedEvent(chatID, payload string) error {
	var event RedisMessageDeletedEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("failed to unmarshal message_deleted event: %w", err)
	}

	wsPayload := MessageDeletedPayload{
		ID:        event.MessageID,
		ChatID:    event.ChatID,
		DeletedAt: event.DeletedAt,
		DeleterID: event.DeleterID,
	}

	wsMessage, err := MarshalWSMessage("message_deleted", wsPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal ws message: %w", err)
	}

	h.BroadcastToChat(chatID, wsMessage)
	return nil
}

func (h *Hub) handleStatusUpdatedEvent(chatID, payload string) error {
	var event RedisStatusUpdatedEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("failed to unmarshal status_updated event: %w", err)
	}

	wsPayload := StatusUpdatedPayload{
		MessageID: event.MessageID,
		UserID:    event.UserID,
		Status:    event.Status,
		UpdatedAt: event.UpdatedAt,
	}

	wsMessage, err := MarshalWSMessage("status_updated", wsPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal ws message: %w", err)
	}

	return h.SendToUser(event.UserID, wsMessage)
}

func (h *Hub) Stop() {
	log.Println("🛑 Stopping WebSocket Hub...")
	h.cancel() // Отменяем контекст → Run() выйдет из цикла
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	closedCount := 0
	for _, client := range h.clients {
		client.Close()
		closedCount++
	}

	log.Printf("✅ Hub stopped: closed %d client connections", closedCount)

	h.clients = make(map[string]*Client)
	h.chatSubscriptions = make(map[string]map[string]bool)
	h.serverChannelSubscriptions = make(map[string]map[string]bool)
}

func (h *Hub) GetStats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalChatSubs := 0
	for _, subs := range h.chatSubscriptions {
		totalChatSubs += len(subs)
	}

	totalChannelSubs := 0
	for _, subs := range h.serverChannelSubscriptions {
		totalChannelSubs += len(subs)
	}

	return HubStats{
		TotalClients:        len(h.clients),
		TotalChats:          len(h.chatSubscriptions),
		TotalChatSubs:       totalChatSubs,
		TotalServerChannels: len(h.serverChannelSubscriptions),
		TotalChannelSubs:    totalChannelSubs,
		IsRunning:           h.running,
	}
}

type HubStats struct {
	TotalClients        int  `json:"total_clients"`
	TotalChats          int  `json:"total_chats"`
	TotalChatSubs       int  `json:"total_chat_subscriptions"`
	TotalServerChannels int  `json:"total_server_channels"`
	TotalChannelSubs    int  `json:"total_channel_subscriptions"`
	IsRunning           bool `json:"is_running"`
}

func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) IsClientOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[userID]
	return exists
}

func stripTargetPrefix(targetID string) string {
	targetID = strings.TrimPrefix(targetID, "chat_")
	targetID = strings.TrimPrefix(targetID, "channel_")
	return targetID
}
