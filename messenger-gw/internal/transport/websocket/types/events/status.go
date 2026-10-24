package events

type StatusUpdated struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}
