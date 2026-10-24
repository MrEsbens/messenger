package storage

import "github.com/MrEsbens/messenger-file-service/internal/domain"

type BucketType string

const (
	BucketAvatars     BucketType = "avatars"     // Аватары: user, chat, server
	BucketAttachments BucketType = "attachments" // Вложения: только message
)

func AllBucketTypes() []BucketType {
	return []BucketType{
		BucketAvatars,
		BucketAttachments,
	}
}

func (b BucketType) IsValid() bool {
	for _, valid := range AllBucketTypes() {
		if b == valid {
			return true
		}
	}
	return false
}

func (b BucketType) Name() string {
	return string(b)
}

func (b BucketType) Prefix(ownerType domain.OwnerType, entityID string) string {
	if entityID == "" || !ownerType.IsValid() {
		return ""
	}

	switch b {
	case BucketAvatars:
		if !ownerType.IsSingleFileOwner() {
			return ""
		}
		return pluralizeOwnerType(ownerType) + "/" + entityID

	case BucketAttachments:
		if !ownerType.CanHaveAttachments() {
			return ""
		}
		return "messages/" + entityID

	default:
		return ""
	}
}

func (b BucketType) AllowedOwnerTypes() []domain.OwnerType {
	switch b {
	case BucketAvatars:
		return []domain.OwnerType{
			domain.OwnerTypeUser,
			domain.OwnerTypeChat,
			domain.OwnerTypeServer,
		}
	case BucketAttachments:
		return []domain.OwnerType{
			domain.OwnerTypeMessage,
		}
	default:
		return nil
	}
}

func (b BucketType) IsValidCombination(ownerType domain.OwnerType) bool {
	for _, allowed := range b.AllowedOwnerTypes() {
		if ownerType == allowed {
			return true
		}
	}
	return false
}

func pluralizeOwnerType(ot domain.OwnerType) string {
	switch ot {
	case domain.OwnerTypeUser:
		return "users"
	case domain.OwnerTypeChat:
		return "chats"
	case domain.OwnerTypeServer:
		return "servers"
	case domain.OwnerTypeMessage:
		return "messages"
	default:
		return ""
	}
}
