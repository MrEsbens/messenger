package websocket

import (
	"net/http"
	"github.com/gorilla/websocket"
)

// NewUpgrader создаёт апгрейдер с поддержкой сжатия.
// EnableCompression: true включает permessage-deflate.
// Это стандарт RFC 7692: сжимает каждый WS-фрейм алгоритмом DEFLATE.
// Экономит ~30-60% трафика на JSON/тексте, почти не грузит CPU.
// Если клиент не поддерживает сжатие → автоматически fallback на обычный режим.
func NewUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}
}