package storage

type BucketType string

const (
	BucketAvatar     BucketType = "avatars"
	BucketAttachment BucketType = "attachments"
	BucketServerIcon BucketType = "server-icons"
)

func AllBucketTypes() []BucketType {
	return []BucketType{
		BucketAvatar,
		BucketAttachment,
		BucketServerIcon,
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

func (b BucketType) Prefix(entityID string) string {
	switch b {
	case BucketAvatar:
		return "users/" + entityID
	case BucketAttachment:
		return "attachments/" + entityID
	case BucketServerIcon:
		return "servers/" + entityID
	default:
		return ""
	}
}
