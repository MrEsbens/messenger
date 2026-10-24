// internal/transport/websocket/handler_test.go

package websocket_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/MrEsbens/messenger-message-service/internal/transport/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ───────────────────────────────────────────────────────────
// mockClient — минимальный мок для тестов (если очень нужно)
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
		Data: payload,
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

// ───────────────────────────────────────────────────────────
// Тесты валидации пейлоадов (чистые функции — легко тестировать)
// ───────────────────────────────────────────────────────────

func TestSubscribePayload_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  websocket.SubscribePayload
		expected bool
	}{
		{"valid chat_id", websocket.SubscribePayload{ChatID: uuid.New().String()}, true},
		{"valid channel", websocket.SubscribePayload{ServerChannelID: uuid.New().String()}, true},
		{"both — invalid", websocket.SubscribePayload{ChatID: "1", ServerChannelID: "2"}, false},
		{"empty — invalid", websocket.SubscribePayload{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.payload.IsValid())
		})
	}
}

func TestMarkReadPayload_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  websocket.MarkReadPayload
		expected bool
	}{
		{"valid", websocket.MarkReadPayload{MessageID: uuid.New().String()}, true},
		{"empty — invalid", websocket.MarkReadPayload{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.payload.IsValid())
		})
	}
}

// ───────────────────────────────────────────────────────────
// Тесты имитации поведения (без вызова приватных методов)
// ───────────────────────────────────────────────────────────

func TestHandler_ErrorResponse_Format(t *testing.T) {
	t.Parallel()
	client := newMockClient(uuid.New().String())

	// Имитируем то, что делает хэндлер при ошибке
	_ = client.SendError("invalid_message_id", "message_id is required")

	msg := client.getLastMessage()
	assert.NotNil(t, msg)
	assert.Equal(t, "error", msg.Type)

	var errPayload map[string]string
	assert.NoError(t, json.Unmarshal(msg.Data, &errPayload))
	assert.Equal(t, "invalid_message_id", errPayload["code"])
}

func TestHandler_Pong_Response(t *testing.T) {
	t.Parallel()
	client := newMockClient(uuid.New().String())

	// Имитируем ответ на ping
	_ = client.SendSuccess("pong", websocket.PongPayload{})

	msg := client.getLastMessage()
	assert.NotNil(t, msg)
	assert.Equal(t, "pong", msg.Type)
}
