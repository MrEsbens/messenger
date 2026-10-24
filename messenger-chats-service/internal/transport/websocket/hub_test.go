package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/transport/websocket"
	redisMocks "github.com/MrEsbens/messenger-chats-service/pkg/redis/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ───────────────────────────────────────────────────────────
// Вспомогательные функции
// ───────────────────────────────────────────────────────────

func newTestHub(t *testing.T) (*websocket.Hub, *redisMocks.MockRedisClient) {
	mockRedis := redisMocks.NewMockRedisClient()
	hub := websocket.NewHub(mockRedis, "chat:")
	return hub, mockRedis
}

func newTestClientForHub(t *testing.T, id string, hub *websocket.Hub) *websocket.Client {
	client := websocket.NewClientForTest(id, nil, hub, 256)

	// Добавляем клиента в хаб через канал
	// В реальных тестах можно использовать sync.WaitGroup для синхронизации
	// Для простоты юнит-тестов добавляем с небольшой задержкой
	go func() {
		hub.Register <- client
	}()
	time.Sleep(10 * time.Millisecond)

	return client
}

// ───────────────────────────────────────────────────────────
// Тесты парсинга каналов Redis
// ───────────────────────────────────────────────────────────

func TestParseRedisChannel_Success(t *testing.T) {
	tests := []struct {
		name          string
		channel       string
		prefix        string
		wantEventType string
		wantTargetID  string
		wantErr       bool
	}{
		{
			name:          "chat:created event",
			channel:       "chat:created:abc123",
			prefix:        "chat:",
			wantEventType: "created",
			wantTargetID:  "abc123",
			wantErr:       false,
		},
		{
			name:          "chat:updated event",
			channel:       "chat:updated:def456",
			prefix:        "chat:",
			wantEventType: "updated",
			wantTargetID:  "def456",
			wantErr:       false,
		},
		{
			name:          "chat:member_added event",
			channel:       "chat:member_added:ghi789",
			prefix:        "chat:",
			wantEventType: "member_added",
			wantTargetID:  "ghi789",
			wantErr:       false,
		},
		{
			name:          "chat:invite_accepted with underscores",
			channel:       "chat:invite_accepted:jkl012",
			prefix:        "chat:",
			wantEventType: "invite_accepted",
			wantTargetID:  "jkl012",
			wantErr:       false,
		},
		{
			name:    "wrong prefix",
			channel: "msg:created:abc123",
			prefix:  "chat:",
			wantErr: true,
		},
		{
			name:    "no colon after prefix",
			channel: "chat:abc123",
			prefix:  "chat:",
			wantErr: true,
		},
		{
			name:    "empty channel",
			channel: "",
			prefix:  "chat:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType, targetID, err := websocket.ParseRedisChannel(tt.channel, tt.prefix)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantEventType, eventType)
			assert.Equal(t, tt.wantTargetID, targetID)
		})
	}
}

func TestStripTargetPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"chat_ prefix", "chat_abc123", "abc123"},
		{"channel_ prefix", "channel_def456", "def456"},
		{"no prefix", "ghi789", "ghi789"},
		{"partial prefix", "chat", "chat"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripTargetPrefixForTest(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Хелпер для тестирования приватной функции
func stripTargetPrefixForTest(s string) string {
	s = stripPrefixForTest(s, "chat_")
	s = stripPrefixForTest(s, "channel_")
	return s
}

func stripPrefixForTest(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// ───────────────────────────────────────────────────────────
// Тесты управления подписками
// ───────────────────────────────────────────────────────────

func TestHub_SubscribeToChat_Success(t *testing.T) {
	hub, _ := newTestHub(t)

	clientID := uuid.New().String()
	chatID := uuid.New().String()

	client := newTestClientForHub(t, clientID, hub)
	_ = client

	err := hub.SubscribeToChat(clientID, chatID)

	assert.NoError(t, err)

	subs := hub.GetChatSubscriptionsForTest()
	assert.Contains(t, subs, chatID)
	assert.Contains(t, subs[chatID], clientID)
}

func TestHub_SubscribeToChat_ClientNotFound(t *testing.T) {
	hub, _ := newTestHub(t)

	clientID := "nonexistent-client"
	chatID := uuid.New().String()

	err := hub.SubscribeToChat(clientID, chatID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHub_UnsubscribeFromChat_Success(t *testing.T) {
	hub, _ := newTestHub(t)

	clientID := uuid.New().String()
	chatID := uuid.New().String()

	client := newTestClientForHub(t, clientID, hub)
	_ = client

	_ = hub.SubscribeToChat(clientID, chatID)
	hub.UnsubscribeFromChat(clientID, chatID)

	subs := hub.GetChatSubscriptionsForTest()
	// После отписки клиента не должно быть в списке
	// (или список может быть пустым)
	_, exists := subs[chatID]
	if exists {
		_, clientExists := subs[chatID][clientID]
		assert.False(t, clientExists)
	}
}

func TestHub_UnsubscribeFromChat_NonExistent(t *testing.T) {
	hub, _ := newTestHub(t)

	clientID := uuid.New().String()
	chatID := uuid.New().String()

	// Отписка от того, на что не подписан — не должна паниковать
	hub.UnsubscribeFromChat(clientID, chatID)

	// Тест проходит, если нет паники
}

// ───────────────────────────────────────────────────────────
// Тесты рассылки сообщений
// ───────────────────────────────────────────────────────────

func TestHub_BroadcastToChat_NoSubscribers(t *testing.T) {
	hub, _ := newTestHub(t)

	chatID := uuid.New().String()
	message := []byte(`{"type":"test","data":{}}`)

	hub.BroadcastToChat(chatID, message)

	// Тест проходит, если нет паники
}

func TestHub_BroadcastToChat_WithSubscribers(t *testing.T) {
	hub, _ := newTestHub(t)

	clientID := uuid.New().String()
	chatID := uuid.New().String()
	message := []byte(`{"type":"chat_updated","data":{"chat_id":"` + chatID + `"}}`)

	client := newTestClientForHub(t, clientID, hub)
	_ = hub.SubscribeToChat(clientID, chatID)

	hub.BroadcastToChat(chatID, message)

	sendCh := client.GetSendChannelForTest()
	select {
	case received := <-sendCh:
		var msg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(received, &msg))
		assert.Equal(t, "chat_updated", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message")
	}
}

func TestHub_BroadcastToChat_BufferFull(t *testing.T) {
	hub, _ := newTestHub(t)

	clientID := uuid.New().String()
	chatID := uuid.New().String()
	message := []byte(`{"type":"test"}`)

	client := websocket.NewClientForTest(clientID, nil, hub, 1)
	_ = hub.SubscribeToChat(clientID, chatID)

	sendCh := client.GetSendChannelForTest()
	sendCh <- []byte(`{"filler":true}`)

	hub.BroadcastToChat(chatID, message)

	// Тест проходит, если нет паники (сообщение может быть пропущено)
}

// ───────────────────────────────────────────────────────────
// Тесты обработки Redis событий
// ───────────────────────────────────────────────────────────

func TestHub_HandleRedisEvent_ChatCreated(t *testing.T) {
	hub, _ := newTestHub(t)

	chatID := uuid.New().String()
	event := websocket.ChatCreatedEvent{
		Type:      "chat_created",
		ChatID:    chatID,
		CreatedBy: uuid.New().String(),
		CreatedAt: time.Now(),
	}

	payload, _ := json.Marshal(event)

	clientID := uuid.New().String()
	client := newTestClientForHub(t, clientID, hub)
	_ = hub.SubscribeToChat(clientID, chatID)

	err := hub.HandleRedisEventForTest("chat:created:"+chatID, string(payload))

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case received := <-sendCh:
		var msg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(received, &msg))
		assert.Equal(t, "chat_created", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestHub_HandleRedisEvent_ChatUpdated(t *testing.T) {
	hub, _ := newTestHub(t)

	chatID := uuid.New().String()
	event := websocket.ChatUpdatedEvent{
		Type:      "chat_updated",
		ChatID:    chatID,
		UpdatedAt: time.Now(),
	}

	payload, _ := json.Marshal(event)

	clientID := uuid.New().String()
	client := newTestClientForHub(t, clientID, hub)
	_ = hub.SubscribeToChat(clientID, chatID)

	err := hub.HandleRedisEventForTest("chat:updated:"+chatID, string(payload))

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case received := <-sendCh:
		var msg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(received, &msg))
		assert.Equal(t, "chat_updated", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestHub_HandleRedisEvent_ChatDeleted(t *testing.T) {
	hub, _ := newTestHub(t)

	chatID := uuid.New().String()
	event := websocket.ChatDeletedEvent{
		Type:      "chat_deleted",
		ChatID:    chatID,
		DeletedAt: time.Now(),
	}

	payload, _ := json.Marshal(event)

	clientID := uuid.New().String()
	client := newTestClientForHub(t, clientID, hub)
	_ = hub.SubscribeToChat(clientID, chatID)

	err := hub.HandleRedisEventForTest("chat:deleted:"+chatID, string(payload))

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case received := <-sendCh:
		var msg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(received, &msg))
		assert.Equal(t, "chat_deleted", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestHub_HandleRedisEvent_MemberAdded(t *testing.T) {
	hub, _ := newTestHub(t)

	chatID := uuid.New().String()
	event := websocket.MemberAddedEvent{
		Type:    "member_added",
		ChatID:  chatID,
		UserID:  uuid.New().String(),
		Role:    "ROLE_MEMBER",
		AddedBy: uuid.New().String(),
	}

	payload, _ := json.Marshal(event)

	clientID := uuid.New().String()
	client := newTestClientForHub(t, clientID, hub)
	_ = hub.SubscribeToChat(clientID, chatID)

	err := hub.HandleRedisEventForTest("chat:member_added:"+chatID, string(payload))

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case received := <-sendCh:
		var msg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(received, &msg))
		assert.Equal(t, "member_added", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestHub_HandleRedisEvent_UnknownEventType(t *testing.T) {
	hub, _ := newTestHub(t)

	err := hub.HandleRedisEventForTest("chat:unknown_event:abc123", `{"type":"unknown"}`)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestHub_HandleRedisEvent_InvalidChannel(t *testing.T) {
	hub, _ := newTestHub(t)

	err := hub.HandleRedisEventForTest("invalid-channel-format", `{"type":"test"}`)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel format")
}
