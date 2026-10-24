package responses

type SubscribedPayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
	Timestamp       string `json:"timestamp"`
}

type UnsubscribedPayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
	Timestamp       string `json:"timestamp"`
}

type MessageSentPayload struct {
	MessageID string `json:"message_id"`
	CreatedAt string `json:"created_at"`
}

type MessageEditedPayload struct {
	ID         string `json:"id"`
	ChatID     string `json:"chat_id"`
	NewContent string `json:"new_content"`
	UpdatedAt  string `json:"updated_at"`
	EditorID   string `json:"editor_id"`
}

type MessageDeletedPayload struct {
	MessageID string `json:"message_id"`
	DeletedAt string `json:"deleted_at"`
}
