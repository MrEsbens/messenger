package commands

import "fmt"

type SubscribePayload struct {
	ChatID          string `json:"chat_id,omitempty"`
	ServerChannelID string `json:"server_channel_id,omitempty"`
}

func (p *SubscribePayload) IsValid() error {
	hasChat := p.ChatID != ""
	hasChannel := p.ServerChannelID != ""
	if !hasChat && !hasChannel {
		return fmt.Errorf("specify chat_id or server_channel_id")
	}
	if hasChat && hasChannel {
		return fmt.Errorf("specify exactly one of chat_id or server_channel_id")
	}
	return nil
}
