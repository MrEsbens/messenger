package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/transport/websocket"
	redisMocks "github.com/MrEsbens/messenger-message-service/pkg/redis/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockMessageHandler — мок для интерфейса MessageHandler
type mockMessageHandler struct {
	mock.Mock
}

func (m *mockMessageHandler) HandleMessage(client *websocket.Client, msg websocket.WSMessage) {
	m.Called(client, msg)
}

// ───────────────────────────────────────────────────────────
// Тесты валидации каналов Redis
// ───────────────────────────────────────────────────────────

func TestFormatRedisChannel(t *testing.T) {
	t.Parallel()

	got := websocket.FormatRedisChannel("msg:", "new_message", "chat_123")
	assert.Equal(t, "msg:new_message:chat_123", got)
}

func TestParseRedisChannel_Success(t *testing.T) {
	t.Parallel()

	prefix, eventType, targetID, err := websocket.ParseRedisChannel("msg:new_message:chat_123")
	assert.NoError(t, err)
	assert.Equal(t, "msg", prefix)
	assert.Equal(t, "new_message", eventType)
	assert.Equal(t, "chat_123", targetID)
}

func TestParseRedisChannel_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, _, _, err := websocket.ParseRedisChannel("invalid:channel")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel format")
}

// ───────────────────────────────────────────────────────────
// Тесты Hub с моками
// ───────────────────────────────────────────────────────────

func TestHub_NewHub_WithMockRedis(t *testing.T) {
	t.Parallel()

	mockRedis := redisMocks.NewMockRedisClient()
	mockHandler := new(mockMessageHandler)

	// Создаём хаб с моками — работает, т.к. принимаем интерфейс
	_ = websocket.NewHub(mockRedis, "msg:", mockHandler)
}

func TestHub_SubscribeToChat_ClientNotFound(t *testing.T) {
	t.Parallel()

	mockRedis := redisMocks.NewMockRedisClient()
	mockHandler := new(mockMessageHandler)
	hub := websocket.NewHub(mockRedis, "msg:", mockHandler)

	// Попытка подписать несуществующего клиента
	err := hub.SubscribeToChat("nonexistent-client", "chat-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ───────────────────────────────────────────────────────────
// Тесты обработки событий Redis (без запуска горутины)
// ───────────────────────────────────────────────────────────

func TestHub_HandleRedisEvent_NewMessage_Valid(t *testing.T) {
	t.Parallel()

	// Проверяем только парсинг и маршалинг — без запуска хаба
	chatID := "chat-1"
	event := websocket.RedisNewMessageEvent{
		Type:   "new_message",
		ChatID: chatID,
		Message: &websocket.MessagePayload{
			ID:      "msg-1",
			Content: "Hello!",
		},
		PublishedAt: time.Now(),
	}

	payload, err := json.Marshal(event)
	assert.NoError(t, err)

	var parsed websocket.RedisNewMessageEvent
	err = json.Unmarshal(payload, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "new_message", parsed.Type)
}

func TestHub_HandleRedisEvent_InvalidChannel(t *testing.T) {
	t.Parallel()

	_, _, _, err := websocket.ParseRedisChannel("invalid:channel")
	assert.Error(t, err)
}
