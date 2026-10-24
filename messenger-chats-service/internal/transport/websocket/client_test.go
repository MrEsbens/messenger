package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/transport/websocket"
	"github.com/stretchr/testify/assert"
)

func TestClient_SendJSON_Success(t *testing.T) {
	hub, _ := newTestHub(t)
	client := websocket.NewClientForTest("test-client", nil, hub, 256)

	data := map[string]string{"key": "value"}
	err := client.SendJSON("test_type", data)

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case msg := <-sendCh:
		var wsMsg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(msg, &wsMsg))
		assert.Equal(t, "test_type", wsMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message in send channel")
	}
}

func TestClient_SendError(t *testing.T) {
	hub, _ := newTestHub(t)
	client := websocket.NewClientForTest("test-client", nil, hub, 256)

	err := client.SendError("test_code", "test message")

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case msg := <-sendCh:
		var wsMsg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(msg, &wsMsg))
		assert.Equal(t, "error", wsMsg.Type)

		var errPayload map[string]string
		assert.NoError(t, json.Unmarshal(wsMsg.Data, &errPayload))
		assert.Equal(t, "test_code", errPayload["code"])
		assert.Equal(t, "test message", errPayload["message"])
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestClient_SendSuccess(t *testing.T) {
	hub, _ := newTestHub(t)
	client := websocket.NewClientForTest("test-client", nil, hub, 256)

	data := map[string]string{"result": "ok"}
	err := client.SendSuccess("success_type", data)

	assert.NoError(t, err)

	sendCh := client.GetSendChannelForTest()
	select {
	case msg := <-sendCh:
		var wsMsg websocket.WSMessage
		assert.NoError(t, json.Unmarshal(msg, &wsMsg))
		assert.Equal(t, "success_type", wsMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestClient_SendJSON_BufferFull(t *testing.T) {
	hub, _ := newTestHub(t)
	client := websocket.NewClientForTest("test-client", nil, hub, 1)

	_ = client.SendJSON("filler", map[string]bool{"fill": true})

	err := client.SendJSON("test", map[string]bool{"test": true})

	assert.Error(t, err)
}
