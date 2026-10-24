package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/responses"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 5 << 20
)

type Client struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	send   chan []byte
	Hub    *Hub
	mu     sync.Mutex
	closed bool
}

func NewClient(conn *websocket.Conn, userID string, hub *Hub) *Client {
	return &Client{
		ID:     userID + ":" + time.Now().Format(time.RFC3339Nano),
		UserID: userID,
		Conn:   conn,
		send:   make(chan []byte, 256),
		Hub:    hub,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.Hub.Log.Printf("WS read error [%s]: %v", c.UserID, err)
			}
			break
		}
		c.Hub.HandleMessage(c, message)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.mu.Unlock()
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

		case <-ticker.C:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}
}

func (c *Client) Send(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		select {
		case c.send <- data:
		default:
			close(c.send)
			c.closed = true
		}
	}
}

func (c *Client) SendJSON(resp *responses.Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	c.Send(data)
	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.send)
	c.Conn.Close()
}