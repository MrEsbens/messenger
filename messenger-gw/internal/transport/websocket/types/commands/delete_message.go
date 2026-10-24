package commands

import "fmt"

type DeleteMessagePayload struct {
	MessageID string `json:"message_id"`
}

func (p *DeleteMessagePayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}
