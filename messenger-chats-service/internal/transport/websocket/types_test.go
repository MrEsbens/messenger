package websocket_test

import (
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/transport/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSubscribePayload_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		payload  websocket.SubscribePayload
		expected bool
	}{
		{
			name:     "valid chat_id only",
			payload:  websocket.SubscribePayload{ChatID: uuid.New().String()},
			expected: true,
		},
		{
			name:     "valid server_channel_id only",
			payload:  websocket.SubscribePayload{ServerChannelID: uuid.New().String()},
			expected: true,
		},
		{
			name:     "both IDs — invalid",
			payload:  websocket.SubscribePayload{ChatID: uuid.New().String(), ServerChannelID: uuid.New().String()},
			expected: false,
		},
		{
			name:     "empty — invalid",
			payload:  websocket.SubscribePayload{},
			expected: false,
		},
		{
			name:     "empty strings — invalid",
			payload:  websocket.SubscribePayload{ChatID: "", ServerChannelID: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.payload.IsValid())
		})
	}
}

func TestUnsubscribePayload_AcceptsSameFormat(t *testing.T) {
	chatID := uuid.New().String()

	payload := websocket.UnsubscribePayload{ChatID: chatID}
	assert.Equal(t, chatID, payload.ChatID)
}
