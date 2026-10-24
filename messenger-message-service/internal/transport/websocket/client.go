package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Максимальный размер входящего сообщения (512 KB)
	maxMessageSize = 512 * 1024

	// Таймауты для записи/чтения
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // 90% от pongWait

	// Размер канала для исходящих сообщений
	// Если канал переполняется — клиент не успевает обрабатывать, соединение закрываем
	sendBuffer = 256
)

// Client представляет одного подключённого пользователя
// Экземпляр создаётся при новом WebSocket соединении и живёт пока клиент подключён
type Client struct {
	// Уникальный идентификатор клиента (обычно user_id из токена)
	ID string

	// WebSocket соединение
	Conn *websocket.Conn

	// Канал для исходящих сообщений
	// Сервер пишет сюда, WritePump читает и отправляет в сокет
	Send chan []byte

	Hub *Hub

	// Подписки: какие чаты/каналы слушает этот клиент
	// Заполняется при получении "subscribe" сообщения
	Subscriptions map[string]bool

	// Мьютекс для безопасного доступа к Subscriptions
	subMu sync.RWMutex

	// Время последней активности (для обнаружения "зомби" соединений)
	LastActivity time.Time

	// Флаг: закрыто ли соединение
	closed  bool
	closeMu sync.Mutex
}

func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:            id,
		Conn:          conn,
		Send:          make(chan []byte, sendBuffer),
		Hub:           hub,
		Subscriptions: make(map[string]bool),
		LastActivity:  time.Now(),
	}
}

func (c *Client) Start() {
	go c.WritePump()
	c.ReadPump()
}

// WritePump читает сообщения из канала c.Send и отправляет их через WebSocket
// Работает в отдельной горутине, завершается при закрытии соединения
func (c *Client) WritePump() {
	// Таймер для периодической отправки Ping (heartbeat)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			_, err = w.Write(message)
			if err != nil {
				_ = w.Close()
				return
			}

			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, err = w.Write([]byte{'\n'})
				if err != nil {
					_ = w.Close()
					return
				}
				select {
				case nextMsg := <-c.Send:
					_, err = w.Write(nextMsg)
					if err != nil {
						_ = w.Close()
						return
					}
				default:
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump читает входящие сообщения из WebSocket и передаёт их на обработку
// Блокирует горутину пока клиент подключён
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.LastActivity = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
			) {
			}
			break
		}

		c.LastActivity = time.Now()

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			_ = c.SendError("invalid_json", "failed to parse message")
			continue
		}

		c.Hub.Handler.HandleMessage(c, wsMsg)
	}
}

// SendJSON сериализует данные в JSON и отправляет клиенту
// Не блокирует: пишет в канал, который обрабатывается WritePump
func (c *Client) SendJSON(msgType string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal  %w", err)
	}

	wsMsg := WSMessage{
		Type: msgType,
		Data: payload,
	}

	message, err := json.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal ws message: %w", err)
	}

	select {
	case c.Send <- message:
		return nil
	default:
		_ = c.Conn.Close()
		return fmt.Errorf("send buffer full, connection closed")
	}
}

// SendError отправляет ошибку клиенту в стандартном формате
func (c *Client) SendError(code, message string) error {
	errPayload := NewErrorPayload(code, message)
	return c.SendJSON("error", errPayload)
}

// SendSuccess отправляет подтверждение операции
func (c *Client) SendSuccess(msgType string, data interface{}) error {
	return c.SendJSON(msgType, data)
}

// Subscribe добавляет чат/канал в список подписок клиента
func (c *Client) Subscribe(targetID string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	c.Subscriptions[targetID] = true
}

// Unsubscribe удаляет чат/канал из подписок клиента
func (c *Client) Unsubscribe(targetID string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()
	delete(c.Subscriptions, targetID)
}

// IsSubscribed проверяет, подписан ли клиент на указанный чат/канал
func (c *Client) IsSubscribed(targetID string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return c.Subscriptions[targetID]
}

// GetSubscriptions возвращает копию всех активных подписок клиента
func (c *Client) GetSubscriptions() map[string]bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	subs := make(map[string]bool, len(c.Subscriptions))
	for k, v := range c.Subscriptions {
		subs[k] = v
	}
	return subs
}

// close корректно закрывает соединение и освобождает ресурсы
// Вызывается из defer в ReadPump/WritePump, не должен вызываться напрямую
func (c *Client) close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return
	}

	c.closed = true

	select {
	case _, ok := <-c.Send:
		if ok {
			close(c.Send)
		}
	default:
		close(c.Send)
	}

	_ = c.Conn.Close()
}

func (c *Client) Close() {
	c.close()
	c.Hub.Unregister <- c
}

// ExtractUserFromRequest извлекает user_id из заголовка запроса
// Вызывается при установлении WebSocket соединения, до апгрейда
// Ожидает, что Gateway уже проверил JWT и добавил заголовок "X-User-ID"
func ExtractUserFromRequest(r *http.Request) (string, error) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return "", fmt.Errorf("missing X-User-ID header")
	}

	_, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user_id format: %v", err)
	}

	return userID, nil
}

// IsAuthorized проверяет, что клиент имеет право на операцию с целевым чатом
// В полной реализации здесь должна быть проверка через Chats Service
// Для MVP: разрешаем всё, если пользователь аутентифицирован
func IsAuthorized(userID, chatID string) bool {
	// Заглушка: в продакшене здесь должен быть gRPC вызов к Chats Service
	// Например:
	//   resp, err := chatsClient.IsMember(ctx, &chatsv1.IsMemberRequest{
	//       ChatId: chatID,
	//       UserId: userID,
	//   })
	//   return err == nil && resp.IsMember
	return true
}
