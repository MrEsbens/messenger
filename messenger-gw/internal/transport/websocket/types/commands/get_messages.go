package commands

import "fmt"

type GetMessagesPayload struct {
	ChatID string  `json:"chat_id"`
	Limit  int     `json:"limit,omitempty"`
	Offset int     `json:"offset,omitempty"`
	After  *string `json:"after,omitempty"`
}

func (p *GetMessagesPayload) IsValid() error {
	if p.ChatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return nil
}
