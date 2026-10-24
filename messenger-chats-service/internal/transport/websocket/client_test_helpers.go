package websocket

import (
	"github.com/gorilla/websocket"
)

func NewClientForTest(id string, conn *websocket.Conn, hub *Hub, sendBufferSize int) *Client {
	return &Client{
		ID:   id,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
		hub:  hub,
	}
}

// GetSendChannel возвращает канал отправки для тестов
// Используется только в тестах
func (c *Client) GetSendChannelForTest() chan []byte {
	return c.send
}

// GetHubForTest возвращает хаб для тестов
// Используется только в тестах
func (c *Client) GetHubForTest() *Hub {
	return c.hub
}
