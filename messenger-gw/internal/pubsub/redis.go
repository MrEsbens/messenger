package pubsub

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/MrEsbens/messenger-gw/internal/transport/websocket"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/events"
	"github.com/redis/go-redis/v9"
)

type RedisManager struct {
	client  *redis.Client
	handler *websocket.WSHandler
	log     *log.Logger
}

func NewRedisManager(client *redis.Client, handler *websocket.WSHandler) *RedisManager {
	return &RedisManager{
		client:  client,
		handler: handler,
		log:     log.Default(),
	}
}

func (m *RedisManager) Start(ctx context.Context) {
	pubsub := m.client.PSubscribe(ctx, "msg:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	m.log.Println("Subscribed to Redis channels: msg:*")

	for {
		select {
		case <-ctx.Done():
			m.log.Println("Redis Pub/Sub stopping")
			return

		case msg := <-ch:
			if msg == nil || msg.Payload == "" {
				continue
			}
			m.handleEvent(msg.Channel, msg.Payload)
		}
	}
}

func (m *RedisManager) handleEvent(channel, payload string) {
	parts := strings.SplitN(channel, ":", 3)
	if len(parts) != 3 {
		m.log.Printf("Invalid channel format: %s", channel)
		return
	}

	eventType := parts[1]
	targetID := parts[2]

	switch eventType {
	case "new_message":
		var e events.NewMessage
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			m.log.Printf("Failed to unmarshal new_message: %v", err)
			return
		}
		m.handler.SendToSubscribers(targetID, "new_message", &e)

	case "message_updated":
		var e events.MessageUpdated
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			m.log.Printf("Failed to unmarshal message_updated: %v", err)
			return
		}
		m.handler.SendToSubscribers(targetID, "message_updated", &e)

	case "message_deleted":
		var e events.MessageDeleted
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			m.log.Printf("Failed to unmarshal message_deleted: %v", err)
			return
		}
		m.handler.SendToSubscribers(targetID, "message_deleted", &e)

	case "status_updated":
		var e events.StatusUpdated
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			m.log.Printf("Failed to unmarshal status_updated: %v", err)
			return
		}
		m.handler.SendToSubscribers(targetID, "status_updated", &e)

	default:
		m.log.Printf("Unknown event type: %s", eventType)
	}
}
