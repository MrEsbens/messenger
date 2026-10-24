package events

import "github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/data"

type NewMessage struct {
	Type        string               `json:"type"`
	ChatID      string               `json:"chat_id"`
	Message     *data.MessagePayload `json:"message"`
	PublishedAt string               `json:"published_at"`
}

type MessageUpdated struct {
	Type        string                    `json:"type"`
	ChatID      string                    `json:"chat_id"`
	MessageID   string                    `json:"message_id"`
	NewContent  string                    `json:"new_content"`
	UpdatedAt   string                    `json:"updated_at"`
	EditorID    string                    `json:"editor_id"`
	Attachments []*data.AttachmentPreview `json:"attachments,omitempty"`
}

type MessageDeleted struct {
	Type        string                    `json:"type"`
	ChatID      string                    `json:"chat_id"`
	MessageID   string                    `json:"message_id"`
	DeletedAt   string                    `json:"deleted_at"`
	DeleterID   string                    `json:"deleter_id"`
	Attachments []*data.AttachmentPreview `json:"attachments,omitempty"`
}
