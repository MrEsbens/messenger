package commands

import "fmt"

type MarkReadPayload struct {
	MessageID string `json:"message_id"`
	ChatID    string `json:"chat_id,omitempty"`
}

func (p *MarkReadPayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}
