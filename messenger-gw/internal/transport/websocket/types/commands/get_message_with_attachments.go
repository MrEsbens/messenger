package commands

import "fmt"

type GetMessageWithAttachmentsPayload struct {
	MessageID string `json:"message_id"`
}

func (p *GetMessageWithAttachmentsPayload) IsValid() error {
	if p.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	return nil
}
