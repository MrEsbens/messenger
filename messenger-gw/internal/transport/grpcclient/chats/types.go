package chats

import (
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypeUnspecified ChatType = "unspecified"
	ChatTypePrivate     ChatType = "private"
	ChatTypeGroup       ChatType = "group"
	ChatTypeServer      ChatType = "server"
)

type MembershipRole string

const (
	RoleUnspecified MembershipRole = "unspecified"
	RoleOwner       MembershipRole = "owner"
	RoleAdmin       MembershipRole = "admin"
	RoleMember      MembershipRole = "member"
	RoleMuted       MembershipRole = "muted"
	RoleBanned      MembershipRole = "banned"
)

type ChatDTO struct {
	ID                 uuid.UUID
	Type               ChatType
	Name               string
	Description        string
	AvatarFileID       *uuid.UUID
	AvatarURL          *string
	HasAvatar          *bool
	CreatedBy          uuid.UUID
	MemberCount        int32
	LastMessagePreview string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	IsDeleted          bool
	ConfigID           *uuid.UUID
}

type ChatMemberDTO struct {
	UserID            uuid.UUID
	ChatID            uuid.UUID
	Role              MembershipRole
	JoinedAt          time.Time
	LeftAt            *time.Time
	NotificationMode  string
	IsMutedUntil      *time.Time
	LastReadMessageID string
	LastActivityAt    time.Time
}

type CreateChatRequest struct {
	Type             ChatType
	Name             string
	Description      string
	AvatarFileID     *uuid.UUID
	ConfigID         *uuid.UUID
	InitialMemberIDs []uuid.UUID
	CreatedBy        uuid.UUID
}

type CreateChatResponse struct {
	ChatID    uuid.UUID
	CreatedAt time.Time
}

type GetChatRequest struct {
	ChatID      uuid.UUID
	RequesterID uuid.UUID
}

type UpdateChatRequest struct {
	ChatID       uuid.UUID
	UpdaterID    uuid.UUID
	Name         *string
	Description  *string
	AvatarFileID *uuid.UUID
	ConfigID     *uuid.UUID
}

type DeleteChatRequest struct {
	ChatID    uuid.UUID
	DeleterID uuid.UUID
}

type ListUserChatsRequest struct {
	UserID     uuid.UUID
	TypeFilter *ChatType
	Limit      int
	Offset     int
	AfterID    *string
}

type ListUserChatsResponse struct {
	Chats      []ChatDTO
	HasMore    bool
	NextCursor string
}

type AddMemberRequest struct {
	ChatID  uuid.UUID
	UserID  uuid.UUID
	AddedBy uuid.UUID
	Role    MembershipRole
}

type RemoveMemberRequest struct {
	ChatID    uuid.UUID
	UserID    uuid.UUID
	RemovedBy uuid.UUID
}

type UpdateMemberRoleRequest struct {
	ChatID       uuid.UUID
	TargetUserID uuid.UUID
	UpdatedBy    uuid.UUID
	NewRole      MembershipRole
}

type ListChatMembersRequest struct {
	ChatID      uuid.UUID
	RequesterID uuid.UUID
	RoleFilter  *MembershipRole
	Limit       int
	Offset      int
}

type ListChatMembersResponse struct {
	Members []ChatMemberDTO
	HasMore bool
}

type CreateInviteRequest struct {
	ChatID    uuid.UUID
	CreatedBy uuid.UUID
	ExpiresAt *time.Time
	MaxUses   *int32
}

type CreateInviteResponse struct {
	InviteToken string
	ExpiresAt   time.Time
}

type JoinByInviteRequest struct {
	InviteToken string
	UserID      uuid.UUID
}

type IsMemberRequest struct {
	ChatID uuid.UUID
	UserID uuid.UUID
}

type IsMemberResponse struct {
	IsMember bool
	Role     MembershipRole
}

type ChatConfigDTO struct {
	ID                      uuid.UUID
	AllowMemberInvite       bool
	AllowMemberKick         bool
	AllowMemberMute         bool
	AutoModerationEnabled   bool
	MaxMessageLength        int32
	DefaultNotificationMode string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type CreateChatConfigRequest struct {
	AllowMemberInvite       bool
	AllowMemberKick         bool
	AllowMemberMute         bool
	AutoModerationEnabled   bool
	MaxMessageLength        int32
	DefaultNotificationMode string
}

type GetChatConfigRequest struct {
	ConfigID uuid.UUID
}

type UpdateChatConfigRequest struct {
	ConfigID                uuid.UUID
	AllowMemberInvite       *bool
	AllowMemberKick         *bool
	AllowMemberMute         *bool
	AutoModerationEnabled   *bool
	MaxMessageLength        *int32
	DefaultNotificationMode *string
}
