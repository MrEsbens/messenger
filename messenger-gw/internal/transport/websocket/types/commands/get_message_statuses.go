package commands

import "fmt"

type GetMessageStatusesPayload struct {
	MessageID string `json:"message_id"`
}

func (p *GetMessageStatusesPayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}
