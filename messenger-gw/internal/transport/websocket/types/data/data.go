package data

type AttachmentPreview struct {
	FileID    string `json:"file_id"`
	PublicURL string `json:"public_url"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	FileName  string `json:"file_name"`
}

type MessagePayload struct {
	ID            string               `json:"id"`
	ChatID        string               `json:"chat_id"`
	SenderID      string               `json:"sender_id"`
	Content       string               `json:"content"`
	ContentType   string               `json:"content_type"`
	HasAttachment bool                 `json:"has_attachment"`
	Attachments   []*AttachmentPreview `json:"attachments,omitempty"`
	ReplyToID     *string              `json:"reply_to_id,omitempty"`
	ServerID      string               `json:"server_id,omitempty"`
	CreatedAt     string               `json:"created_at"`
	UpdatedAt     string               `json:"updated_at"`
	IsDeleted     bool                 `json:"is_deleted"`
}
