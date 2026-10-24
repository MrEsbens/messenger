package websocket

import (
	redisv8 "github.com/go-redis/redis/v8"
)

// HandleRedisEventForTest позволяет тестировать обработку Redis событий без реального Redis
// Используется только в тестах
func (h *Hub) HandleRedisEventForTest(channel, payload string) error {
	// Создаём мок сообщения, совместимый с *redisv8.Message
	msg := &redisv8.Message{
		Channel: channel,
		Payload: payload,
	}
	return h.handleRedisEvent(msg)
}

// GetChatSubscriptionsForTest возвращает подписки на чаты для тестов
// Используется только в тестах
func (h *Hub) GetChatSubscriptionsForTest() map[string]map[string]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Возвращаем копию, чтобы тесты не ломали состояние
	result := make(map[string]map[string]bool, len(h.chatSubscriptions))
	for k, v := range h.chatSubscriptions {
		result[k] = make(map[string]bool, len(v))
		for k2, v2 := range v {
			result[k][k2] = v2
		}
	}
	return result
}

// GetClientsForTest возвращает клиентов для тестов
// Используется только в тестах
func (h *Hub) GetClientsForTest() map[string]*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*Client, len(h.clients))
	for k, v := range h.clients {
		result[k] = v
	}
	return result
}
