package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/MrEsbens/messenger-gw/internal/config"
	"github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/responses"
)

type Hub struct {
	clients       map[string]*Client
	mu            sync.RWMutex
	Register      chan *Client
	Unregister    chan *Client
	Config        *config.Config
	Log           *log.Logger
	subscriptions map[string]map[string]bool
	subMu         sync.RWMutex
	msgHandler    func(*Client, []byte)
}

func NewHub(cfg *config.Config) *Hub {
	return &Hub{
		clients:       make(map[string]*Client),
		subscriptions: make(map[string]map[string]bool),
		Register:      make(chan *Client, 100),
		Unregister:    make(chan *Client, 100),
		Config:        cfg,
		Log:           log.Default(),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.Log.Printf("Client connected: %s (user: %s)", client.ID, client.UserID)
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				client.Close()
				h.Log.Printf("Client disconnected: %s", client.ID)
			}
			h.mu.Unlock()

		case <-ctx.Done():
			h.Log.Println("Hub stopping")
			return
		}
	}
}

func (h *Hub) Subscribe(c *Client, targetID string) {
	h.subMu.Lock()
	defer h.subMu.Unlock()

	if h.subscriptions[targetID] == nil {
		h.subscriptions[targetID] = make(map[string]bool)
	}
	h.subscriptions[targetID][c.ID] = true
}

func (h *Hub) Unsubscribe(c *Client, targetID string) {
	h.subMu.Lock()
	defer h.subMu.Unlock()

	if subs, ok := h.subscriptions[targetID]; ok {
		delete(subs, c.ID)
		if len(subs) == 0 {
			delete(h.subscriptions, targetID)
		}
	}
}

func (h *Hub) SendToSubscribers(targetID string, resp *responses.Response) {
	h.subMu.RLock()
	defer h.subMu.RUnlock()

	data, err := json.Marshal(resp)
	if err != nil {
		h.Log.Printf("Failed to marshal response: %v", err)
		return
	}

	if subs, ok := h.subscriptions[targetID]; ok {
		for clientID := range subs {
			if c, exists := h.clients[clientID]; exists {
				c.Send(data)
			}
		}
	}
}

func (h *Hub) SendToUser(userID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.clients {
		if c.UserID == userID {
			c.Send(data)
		}
	}
}

func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.clients {
		c.Send(data)
	}
}

func (h *Hub) SetMessageHandler(fn func(*Client, []byte)) {
	h.msgHandler = fn
}

func (h *Hub) HandleMessage(c *Client, msg []byte) {
	if h.msgHandler != nil {
		h.msgHandler(c, msg)
	} else {
		c.Send(msg)
	}
}