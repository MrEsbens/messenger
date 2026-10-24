package domain

import "github.com/google/uuid"

var SystemModeratorID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

const (
	ContentTypeText     = "text"
	ContentTypeMarkdown = "markdown"
	ContentTypeCode     = "code"
)

func IsValidContentType(ct string) bool {
	switch ct {
	case ContentTypeText, ContentTypeMarkdown, ContentTypeCode:
		return true
	default:
		return false
	}
}

const (
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusRead      = "read"
)

func IsValidMessageStatus(status string) bool {
	switch status {
	case StatusSent, StatusDelivered, StatusRead:
		return true
	default:
		return false
	}
}

func StatusOrder(status string) int {
	switch status {
	case StatusSent:
		return 1
	case StatusDelivered:
		return 2
	case StatusRead:
		return 3
	default:
		return 0
	}
}

func CanTransitionTo(current, target string) bool {
	return StatusOrder(target) >= StatusOrder(current)
}

const (
	FileOwnerTypeMessage  = "message"
	FileBucketAttachments = "attachments"
)

func IsValidFileOwnerType(ot string) bool {
	return ot == FileOwnerTypeMessage
}
