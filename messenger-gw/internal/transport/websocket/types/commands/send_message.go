package commands

import "fmt"

type SendMessagePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
	Content         string `json:"content"`
	ContentType     string `json:"content_type,omitempty"`
	ReplyToID       string `json:"reply_to_id,omitempty"`
	ServerID        string `json:"server_id,omitempty"`
}

func (p *SendMessagePayload) IsValid() error {
	if p.Content == "" {
		return fmt.Errorf("content is required")
	}
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	if !hasChat && !hasChannel {
		return fmt.Errorf("specify chat_id or server_channel_id")
	}
	if hasChat && hasChannel {
		return fmt.Errorf("specify exactly one of chat_id or server_channel_id")
	}
	if p.ContentType == "" {
		p.ContentType = "text"
	}
	if p.ContentType != "text" && p.ContentType != "markdown" && p.ContentType != "code" {
		return fmt.Errorf("invalid content_type: allowed text, markdown, code")
	}
	return nil
}
