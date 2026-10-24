package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID            uuid.UUID
	ChatID        uuid.UUID
	SenderID      uuid.UUID
	Content       string
	ContentType   string
	ReplyToID     *uuid.UUID
	HasAttachment bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}
