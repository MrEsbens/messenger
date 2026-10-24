package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB
)

type Client struct {
	ID       string
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub
	mu       sync.Mutex
	lastPing time.Time
}

// NewClient создаёт нового клиента
func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:       id,
		conn:     conn,
		send:     make(chan []byte, 256),
		hub:      hub,
		lastPing: time.Now(),
	}
}

// ReadLoop читает сообщения от клиента
func (c *Client) ReadLoop(ctx context.Context) {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastPing = time.Now()
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// Логируем только неожиданные закрытия
				}
				return
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				_ = c.SendError("invalid_json", "failed to parse message")
				continue
			}

			c.hub.HandleMessage(c, wsMsg)
		}
	}
}

// WriteLoop отправляет сообщения клиенту
func (c *Client) WriteLoop(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if time.Since(c.lastPing) > pongWait {
				// Клиент не отвечал на пинги — закрываем
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON отправляет структурированное сообщение
func (c *Client) SendJSON(msgType string, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	wsMsg := WSMessage{
		Type: msgType,
		Data: json.RawMessage(dataBytes), // ← явное преобразование
	}

	payload, err := json.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal ws message: %w", err)
	}

	select {
	case c.send <- payload:
		return nil
	default:
		// Канал переполнен — клиент не успевает
		return errSendBufferFull
	}
}

// SendError отправляет ошибку
func (c *Client) SendError(code, message string) error {
	return c.SendJSON("error", ErrorResponse{Code: code, Message: message})
}

// SendSuccess отправляет успешный ответ
func (c *Client) SendSuccess(msgType string, data interface{}) error {
	return c.SendJSON(msgType, data)
}

var errSendBufferFull = fmt.Errorf("send buffer full")
