// internal/domain/contact.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

type ContactStatus string

const (
	ContactStatusPending  ContactStatus = "pending"
	ContactStatusAccepted ContactStatus = "accepted"
	ContactStatusRejected ContactStatus = "rejected"
	ContactStatusBlocked  ContactStatus = "blocked"
)

type Contact struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ContactID uuid.UUID
	Status    ContactStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s ContactStatus) IsValid() bool {
	switch s {
	case ContactStatusPending, ContactStatusAccepted, ContactStatusRejected, ContactStatusBlocked:
		return true
	}
	return false
}
