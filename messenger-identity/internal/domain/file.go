package domain

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	OwnerType  string // 'user', 'chat', 'server', 'message'
	BucketName string
	ObjectKey  string
	FileName   string
	MimeType   string
	FileSize   int64
	Status     string // "pending", "active", "deleted"
	CreatedAt  time.Time
	DeletedAt  *time.Time
}
