package servers

import (
	"time"

	"github.com/google/uuid"
)

type ServerDTO struct {
	ID          uuid.UUID
	Name        string
	Description *string
	OwnerID     uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsDeleted   bool
}

type CreateServerRequest struct {
	Name        string
	Description *string
	OwnerID     uuid.UUID
}

type CreateServerResponse struct {
	ServerID  uuid.UUID
	CreatedAt time.Time
}

type GetServerRequest struct {
	ServerID    uuid.UUID
	RequesterID uuid.UUID
}

type UpdateServerRequest struct {
	ServerID    uuid.UUID
	UpdaterID   uuid.UUID
	Name        *string
	Description *string
}

type DeleteServerRequest struct {
	ServerID  uuid.UUID
	DeleterID uuid.UUID
}

type ListUserServersRequest struct {
	UserID uuid.UUID
	Limit  int
	Offset int
}

type ListUserServersResponse struct {
	Servers []ServerDTO
}

type ServerConfigDTO struct {
	ServerID                uuid.UUID
	MaxMembers              int32
	MaxChannels             int32
	DefaultNotificationMode string
	ModerationEnabled       bool
}

type GetServerConfigRequest struct {
	ServerID uuid.UUID
}

type UpdateServerConfigRequest struct {
	ServerID                uuid.UUID
	UpdaterID               uuid.UUID
	MaxMembers              *int32
	MaxChannels             *int32
	DefaultNotificationMode *string
	ModerationEnabled       *bool
}

type ModerationConfigDTO struct {
	ServerID               uuid.UUID
	ProfanityFilterAction  string
	ToxicityFilterAction   string
	NsfwTextFilterAction   string
	HateSpeechFilterAction string
}

type GetModerationConfigRequest struct {
	ServerID uuid.UUID
}

type UpdateModerationConfigRequest struct {
	ServerID               uuid.UUID
	UpdaterID              uuid.UUID
	ProfanityFilterAction  *string
	ToxicityFilterAction   *string
	NsfwTextFilterAction   *string
	HateSpeechFilterAction *string
}

type MemberDTO struct {
	UserID     uuid.UUID
	ServerID   uuid.UUID
	RoleIDs    []uuid.UUID
	JoinedAt   time.Time
	IsMuted    bool
	IsBanned   bool
	MutedUntil *time.Time
}

type AddMemberRequest struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	AddedBy  uuid.UUID
}

type RemoveMemberRequest struct {
	ServerID  uuid.UUID
	UserID    uuid.UUID
	RemovedBy uuid.UUID
}

type GetMemberRequest struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
}

type ListMembersRequest struct {
	ServerID uuid.UUID
	Limit    int
	Offset   int
}

type ListMembersResponse struct {
	Members []MemberDTO
}

type BanMemberRequest struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	BannedBy uuid.UUID
}

type UnbanMemberRequest struct {
	ServerID   uuid.UUID
	UserID     uuid.UUID
	UnbannedBy uuid.UUID
}

type MuteMemberRequest struct {
	ServerID      uuid.UUID
	UserID        uuid.UUID
	MutedBy       uuid.UUID
	DurationHours int64
}

type UnmuteMemberRequest struct {
	ServerID  uuid.UUID
	UserID    uuid.UUID
	UnmutedBy uuid.UUID
}

type RoleDTO struct {
	ID          uuid.UUID
	ServerID    uuid.UUID
	Name        string
	Color       string
	Permissions []string
	Position    int32
	IsDefault   bool
}

type CreateRoleRequest struct {
	ServerID  uuid.UUID
	Name      string
	CreatorID uuid.UUID
}

type UpdateRoleRequest struct {
	ServerID    uuid.UUID
	RoleID      uuid.UUID
	UpdaterID   uuid.UUID
	Name        *string
	Color       *string
	Permissions []string
}

type DeleteRoleRequest struct {
	ServerID  uuid.UUID
	RoleID    uuid.UUID
	DeleterID uuid.UUID
}

type AssignRoleRequest struct {
	ServerID   uuid.UUID
	MemberID   uuid.UUID
	RoleID     uuid.UUID
	AssignedBy uuid.UUID
}

type RemoveRoleRequest struct {
	ServerID  uuid.UUID
	MemberID  uuid.UUID
	RoleID    uuid.UUID
	RemovedBy uuid.UUID
}

type GetMemberRolesRequest struct {
	MemberID uuid.UUID
}

type GetMemberRolesResponse struct {
	Roles []RoleDTO
}

type ServerChatDTO struct {
	ChatID    uuid.UUID
	ServerID  uuid.UUID
	Name      string
	Position  int32
	CreatedAt time.Time
}

type CreateServerChatRequest struct {
	ServerID  uuid.UUID
	Name      string
	CreatedBy uuid.UUID
}

type ListServerChatsRequest struct {
	ServerID    uuid.UUID
	RequesterID uuid.UUID
}

type ListServerChatsResponse struct {
	Chats []ServerChatDTO
}

type DeleteServerChatRequest struct {
	ServerID  uuid.UUID
	ChatID    uuid.UUID
	DeleterID uuid.UUID
}
