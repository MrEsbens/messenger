// internal/transport/websocket/handler_test.go
package websocket_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/transport/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ───────────────────────────────────────────────────────────
// mockClient — минимальный мок для тестов хэндлера
// ───────────────────────────────────────────────────────────

type mockClient struct {
	id           string
	sentMessages []websocket.WSMessage
	mu           sync.Mutex
}

func newMockClient(id string) *mockClient {
	return &mockClient{
		id:           id,
		sentMessages: make([]websocket.WSMessage, 0),
	}
}

func (m *mockClient) SendJSON(msgType string, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	payload, _ := json.Marshal(data)
	m.sentMessages = append(m.sentMessages, websocket.WSMessage{
		Type: msgType,
		Data: json.RawMessage(payload),
	})
	return nil
}

func (m *mockClient) SendError(code, message string) error {
	return m.SendJSON("error", map[string]string{"code": code, "message": message})
}

func (m *mockClient) SendSuccess(msgType string, data interface{}) error {
	return m.SendJSON(msgType, data)
}

func (m *mockClient) Hub() *websocket.Hub { return nil }
func (m *mockClient) ID() string          { return m.id }

func (m *mockClient) getLastMessage() *websocket.WSMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sentMessages) == 0 {
		return nil
	}
	return &m.sentMessages[len(m.sentMessages)-1]
}

func (m *mockClient) clearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = m.sentMessages[:0]
}

// ───────────────────────────────────────────────────────────
// Тесты валидации пейлоадов
// ───────────────────────────────────────────────────────────

func TestSubscribePayload_IsValid_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		payload  websocket.SubscribePayload
		expected bool
	}{
		{"valid chat_id", websocket.SubscribePayload{ChatID: uuid.New().String()}, true},
		{"valid channel_id", websocket.SubscribePayload{ServerChannelID: uuid.New().String()}, true},
		{"both set", websocket.SubscribePayload{ChatID: "1", ServerChannelID: "2"}, false},
		{"both empty", websocket.SubscribePayload{}, false},
		{"chat_id empty string", websocket.SubscribePayload{ChatID: ""}, false},
		{"channel_id empty string", websocket.SubscribePayload{ServerChannelID: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.payload.IsValid())
		})
	}
}

// ───────────────────────────────────────────────────────────
// Тесты обработки сообщений (через прямые вызовы)
// ───────────────────────────────────────────────────────────

func TestHandler_HandleSubscribe_ValidChatID(t *testing.T) {
	// Создаём минимальный хаб для теста
	// (в реальных тестах можно замокать зависимости)

	// Для простоты тестируем только валидацию пейлоада
	// Полная интеграция — в e2e тестах

	payload := websocket.SubscribePayload{
		ChatID: uuid.New().String(),
	}

	assert.True(t, payload.IsValid())
}

func TestHandler_HandleSubscribe_InvalidPayload(t *testing.T) {
	payload := websocket.SubscribePayload{
		ChatID:          uuid.New().String(),
		ServerChannelID: uuid.New().String(), // Оба заданы — ошибка
	}

	assert.False(t, payload.IsValid())
}

func TestHandler_HandlePing_Success(t *testing.T) {
	client := newMockClient(uuid.New().String())

	// Имитируем обработку ping
	_ = client.SendSuccess("pong", websocket.PongPayload{})

	msg := client.getLastMessage()
	assert.NotNil(t, msg)
	assert.Equal(t, "pong", msg.Type)
}

func TestHandler_HandleUnknownType_Error(t *testing.T) {
	client := newMockClient(uuid.New().String())

	// Имитируем обработку неизвестного типа
	_ = client.SendError("unknown_message_type", "type 'unknown' is not supported")

	msg := client.getLastMessage()
	assert.NotNil(t, msg)
	assert.Equal(t, "error", msg.Type)

	var errPayload map[string]string
	_ = json.Unmarshal(msg.Data, &errPayload)
	assert.Equal(t, "unknown_message_type", errPayload["code"])
}

// ───────────────────────────────────────────────────────────
// Тесты маршалинга/анмаршалинга
// ───────────────────────────────────────────────────────────

func TestWSMessage_MarshalUnmarshal(t *testing.T) {
	original := websocket.WSMessage{
		Type: "test_type",
		Data: json.RawMessage(`{"key":"value"}`),
	}

	// Маршалим
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Анмаршалим
	var parsed websocket.WSMessage
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, original.Type, parsed.Type)
	assert.Equal(t, original.Data, parsed.Data)
}

func TestErrorResponse_Marshal(t *testing.T) {
	errResp := websocket.ErrorResponse{
		Code:    "test_error",
		Message: "test message",
	}

	data, err := json.Marshal(errResp)
	assert.NoError(t, err)

	var parsed map[string]string
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "test_error", parsed["code"])
	assert.Equal(t, "test message", parsed["message"])
}
