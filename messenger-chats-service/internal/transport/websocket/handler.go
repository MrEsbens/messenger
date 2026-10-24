package websocket

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler обрабатывает HTTP-запросы на WebSocket upgrade
type Handler struct {
	upgrader websocket.Upgrader
	hub      *Hub
}

// NewHandler создаёт новый обработчик
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// В продакшене: проверять Origin заголовок
				return true
			},
		},
		hub: hub,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "X-User-Id header required", http.StatusUnauthorized)
		return
	}

	if _, err := uuid.Parse(userID); err != nil {
		http.Error(w, "invalid user ID format", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(userID, conn, h.hub)
	h.hub.Register <- client

	// Запускаем циклы чтения/записи
	ctx := h.hub.ctx
	go client.ReadLoop(ctx)
	go client.WriteLoop(ctx)
}
