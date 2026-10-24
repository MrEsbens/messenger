package domain

type OwnerType string

const (
	OwnerTypeUser    OwnerType = "user"
	OwnerTypeChat    OwnerType = "chat"
	OwnerTypeServer  OwnerType = "server"
	OwnerTypeMessage OwnerType = "message"
)

func (ot OwnerType) IsValid() bool {
	switch ot {
	case OwnerTypeUser, OwnerTypeChat, OwnerTypeServer, OwnerTypeMessage:
		return true
	default:
		return false
	}
}

func (ot OwnerType) String() string {
	return string(ot)
}

func (ot OwnerType) IsSingleFileOwner() bool {
	return ot == OwnerTypeUser || ot == OwnerTypeChat || ot == OwnerTypeServer
}

func (ot OwnerType) CanHaveAttachments() bool {
	return ot == OwnerTypeMessage
}