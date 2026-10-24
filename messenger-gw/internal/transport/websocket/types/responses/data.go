package responses

import "github.com/MrEsbens/messenger-gw/internal/transport/websocket/types/data"

type GetMessagesResponsePayload struct {
	Messages []*data.MessagePayload `json:"messages"`
	HasMore  bool                   `json:"has_more"`
}

type MessageStatusPayload struct {
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type GetMessageStatusesResponsePayload struct {
	Statuses []MessageStatusPayload `json:"statuses"`
}

type GetMessageWithAttachmentsResponsePayload struct {
	Message     *data.MessagePayload      `json:"message"`
	Attachments []*data.AttachmentPreview `json:"attachments"`
}
