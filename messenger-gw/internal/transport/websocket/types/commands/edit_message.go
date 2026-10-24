package commands

import "fmt"

type EditMessagePayload struct {
	MessageID  string `json:"message_id"`
	NewContent string `json:"content"`
}

func (p *EditMessagePayload) IsValid() error {
	if p.MessageID == "" || p.NewContent == "" {
		return fmt.Errorf("message_id and content are required")
	}
	return nil
}
