package chats

import (
	"context"
	"fmt"
	"strings"
	"time"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatsClientInterface interface {
	CreateChat(ctx context.Context, req *CreateChatRequest) (*CreateChatResponse, error)
	GetChat(ctx context.Context, req *GetChatRequest) (*ChatDTO, error)
	UpdateChat(ctx context.Context, req *UpdateChatRequest) error
	DeleteChat(ctx context.Context, req *DeleteChatRequest) error
	ListUserChats(ctx context.Context, req *ListUserChatsRequest) (*ListUserChatsResponse, error)
	AddMember(ctx context.Context, req *AddMemberRequest) error
	RemoveMember(ctx context.Context, req *RemoveMemberRequest) error
	UpdateMemberRole(ctx context.Context, req *UpdateMemberRoleRequest) error
	ListChatMembers(ctx context.Context, req *ListChatMembersRequest) (*ListChatMembersResponse, error)
	CreateInvite(ctx context.Context, req *CreateInviteRequest) (*CreateInviteResponse, error)
	JoinByInvite(ctx context.Context, req *JoinByInviteRequest) error
	IsMember(ctx context.Context, req *IsMemberRequest) (*IsMemberResponse, error)
	Close() error
}

type chatsClient struct {
	client  chatsv1.ChatServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewChatsClient(endpoint string, timeout time.Duration) (ChatsClientInterface, error) {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "grpc://")

	if endpoint == "" {
		return nil, fmt.Errorf("empty chats service endpoint")
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithReturnConnectionError(),
	}

	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chats service: %w", err)
	}

	return &chatsClient{
		client:  chatsv1.NewChatServiceClient(conn),
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *chatsClient) CreateChat(ctx context.Context, req *CreateChatRequest) (*CreateChatResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	initialMembers := make([]string, 0, len(req.InitialMemberIDs))
	for _, id := range req.InitialMemberIDs {
		initialMembers = append(initialMembers, id.String())
	}

	grpcReq := &chatsv1.CreateChatRequest{
		Type:             chatTypeToProto(req.Type),
		Name:             req.Name,
		Description:      req.Description,
		CreatedBy:        req.CreatedBy.String(),
		InitialMemberIds: initialMembers,
	}
	// 🔥 optional string → *string: присваиваем указатель напрямую
	if req.AvatarFileID != nil {
		grpcReq.AvatarFileId = req.AvatarFileID.String()
	}
	if req.ConfigID != nil {
		grpcReq.ConfigId = req.ConfigID.String()
	}

	resp, err := c.client.CreateChat(callCtx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC CreateChat failed: %w", err)
	}

	return &CreateChatResponse{
		ChatID:    uuid.MustParse(resp.ChatId),
		CreatedAt: resp.CreatedAt.AsTime(),
	}, nil
}

func (c *chatsClient) GetChat(ctx context.Context, req *GetChatRequest) (*ChatDTO, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetChat(callCtx, &chatsv1.GetChatRequest{
		ChatId:      req.ChatID.String(),
		RequesterId: req.RequesterID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetChat failed: %w", err)
	}

	dto := protoToChatDTO(resp)
	return &dto, nil
}

func (c *chatsClient) UpdateChat(ctx context.Context, req *UpdateChatRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &chatsv1.UpdateChatRequest{
		ChatId:    req.ChatID.String(),
		UpdaterId: req.UpdaterID.String(),
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.AvatarFileID != nil {
		val := req.AvatarFileID.String()
		grpcReq.AvatarFileId = &val
	}
	if req.ConfigID != nil {
		val := req.ConfigID.String()
		grpcReq.ConfigId = &val
	}

	_, err := c.client.UpdateChat(callCtx, grpcReq)
	return err
}

func (c *chatsClient) DeleteChat(ctx context.Context, req *DeleteChatRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.DeleteChat(callCtx, &chatsv1.DeleteChatRequest{
		ChatId:    req.ChatID.String(),
		DeleterId: req.DeleterID.String(),
	})
	return err
}

func (c *chatsClient) ListUserChats(ctx context.Context, req *ListUserChatsRequest) (*ListUserChatsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &chatsv1.ListUserChatsRequest{
		UserId: req.UserID.String(),
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
	}
	if req.TypeFilter != nil {
		t := chatTypeToProto(*req.TypeFilter)
		grpcReq.TypeFilter = &t
	}
	if req.AfterID != nil {
		grpcReq.AfterId = req.AfterID
	}

	resp, err := c.client.ListUserChats(callCtx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC ListUserChats failed: %w", err)
	}

	chats := make([]ChatDTO, 0, len(resp.Chats))
	for _, chat := range resp.Chats {
		// 🔥 chat уже *ChatDTO (из []*ChatDTO)
		dto := protoToChatDTO(chat)
		chats = append(chats, dto)
	}

	return &ListUserChatsResponse{
		Chats:      chats,
		HasMore:    resp.HasMore,
		NextCursor: resp.NextCursor,
	}, nil
}

func (c *chatsClient) AddMember(ctx context.Context, req *AddMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.AddMember(callCtx, &chatsv1.AddMemberRequest{
		ChatId:  req.ChatID.String(),
		UserId:  req.UserID.String(),
		AddedBy: req.AddedBy.String(),
		Role:    membershipRoleToProto(req.Role),
	})
	return err
}

func (c *chatsClient) RemoveMember(ctx context.Context, req *RemoveMemberRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.RemoveMember(callCtx, &chatsv1.RemoveMemberRequest{
		ChatId:    req.ChatID.String(),
		UserId:    req.UserID.String(),
		RemovedBy: req.RemovedBy.String(),
	})
	return err
}

func (c *chatsClient) UpdateMemberRole(ctx context.Context, req *UpdateMemberRoleRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.UpdateMemberRole(callCtx, &chatsv1.UpdateMemberRoleRequest{
		ChatId:       req.ChatID.String(),
		TargetUserId: req.TargetUserID.String(),
		UpdatedBy:    req.UpdatedBy.String(),
		NewRole:      membershipRoleToProto(req.NewRole),
	})
	return err
}

func (c *chatsClient) ListChatMembers(ctx context.Context, req *ListChatMembersRequest) (*ListChatMembersResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &chatsv1.ListChatMembersRequest{
		ChatId:      req.ChatID.String(),
		RequesterId: req.RequesterID.String(),
		Limit:       int32(req.Limit),
		Offset:      int32(req.Offset),
	}
	if req.RoleFilter != nil {
		r := membershipRoleToProto(*req.RoleFilter)
		grpcReq.RoleFilter = &r
	}

	resp, err := c.client.ListChatMembers(callCtx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC ListChatMembers failed: %w", err)
	}

	members := make([]ChatMemberDTO, 0, len(resp.Members))
	for _, m := range resp.Members {
		dto := protoToChatMemberDTO(m)
		members = append(members, dto)
	}

	return &ListChatMembersResponse{
		Members: members,
		HasMore: resp.HasMore,
	}, nil
}

func (c *chatsClient) CreateInvite(ctx context.Context, req *CreateInviteRequest) (*CreateInviteResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	grpcReq := &chatsv1.CreateInviteRequest{
		ChatId:    req.ChatID.String(),
		CreatedBy: req.CreatedBy.String(),
	}
	if req.ExpiresAt != nil {
		grpcReq.ExpiresAt = timestamppb.New(*req.ExpiresAt)
	}
	if req.MaxUses != nil {
		grpcReq.MaxUses = req.MaxUses
	}

	resp, err := c.client.CreateInvite(callCtx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC CreateInvite failed: %w", err)
	}

	return &CreateInviteResponse{
		InviteToken: resp.InviteToken,
		ExpiresAt:   resp.ExpiresAt.AsTime(),
	}, nil
}

func (c *chatsClient) JoinByInvite(ctx context.Context, req *JoinByInviteRequest) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.JoinByInvite(callCtx, &chatsv1.JoinByInviteRequest{
		InviteToken: req.InviteToken,
		UserId:      req.UserID.String(),
	})
	return err
}

func (c *chatsClient) IsMember(ctx context.Context, req *IsMemberRequest) (*IsMemberResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.IsMember(callCtx, &chatsv1.IsMemberRequest{
		ChatId: req.ChatID.String(),
		UserId: req.UserID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC IsMember failed: %w", err)
	}

	return &IsMemberResponse{
		IsMember: resp.IsMember,
		Role:     membershipRoleFromProto(resp.Role),
	}, nil
}

func (c *chatsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ─── Вспомогательные функции конвертации ─────────────────────

func chatTypeToProto(t ChatType) chatsv1.ChatType {
	switch t {
	case ChatTypePrivate:
		return chatsv1.ChatType_CHAT_TYPE_PRIVATE
	case ChatTypeGroup:
		return chatsv1.ChatType_CHAT_TYPE_GROUP
	case ChatTypeServer:
		return chatsv1.ChatType_CHAT_TYPE_SERVER
	default:
		return chatsv1.ChatType_CHAT_TYPE_UNSPECIFIED
	}
}

func chatTypeFromProto(t chatsv1.ChatType) ChatType {
	switch t {
	case chatsv1.ChatType_CHAT_TYPE_PRIVATE:
		return ChatTypePrivate
	case chatsv1.ChatType_CHAT_TYPE_GROUP:
		return ChatTypeGroup
	case chatsv1.ChatType_CHAT_TYPE_SERVER:
		return ChatTypeServer
	default:
		return ChatTypeUnspecified
	}
}

func membershipRoleToProto(r MembershipRole) chatsv1.MembershipRole {
	switch r {
	case RoleOwner:
		return chatsv1.MembershipRole_ROLE_OWNER
	case RoleAdmin:
		return chatsv1.MembershipRole_ROLE_ADMIN
	case RoleMember:
		return chatsv1.MembershipRole_ROLE_MEMBER
	case RoleMuted:
		return chatsv1.MembershipRole_ROLE_MUTED
	case RoleBanned:
		return chatsv1.MembershipRole_ROLE_BANNED
	default:
		return chatsv1.MembershipRole_ROLE_UNSPECIFIED
	}
}

func membershipRoleFromProto(r chatsv1.MembershipRole) MembershipRole {
	switch r {
	case chatsv1.MembershipRole_ROLE_OWNER:
		return RoleOwner
	case chatsv1.MembershipRole_ROLE_ADMIN:
		return RoleAdmin
	case chatsv1.MembershipRole_ROLE_MEMBER:
		return RoleMember
	case chatsv1.MembershipRole_ROLE_MUTED:
		return RoleMuted
	case chatsv1.MembershipRole_ROLE_BANNED:
		return RoleBanned
	default:
		return RoleUnspecified
	}
}

func protoToChatDTO(c *chatsv1.ChatDTO) ChatDTO {
	var avatarFileID *uuid.UUID
	// 🔥 optional string → *string: проверяем nil, потом разыменовываем
	if c.AvatarFileId != nil && *c.AvatarFileId != "" {
		id := uuid.MustParse(*c.AvatarFileId)
		avatarFileID = &id
	}
	var avatarURL *string
	if c.AvatarUrl != nil && *c.AvatarUrl != "" {
		avatarURL = c.AvatarUrl
	}
	var hasAvatar *bool
	if c.HasAvatar != nil {
		hasAvatar = c.HasAvatar
	}
	var configID *uuid.UUID
	if c.ConfigId != nil && *c.ConfigId != "" {
		id := uuid.MustParse(*c.ConfigId)
		configID = &id
	}

	return ChatDTO{
		ID:                 uuid.MustParse(c.Id),
		Type:               chatTypeFromProto(c.Type),
		Name:               c.Name,
		Description:        c.Description,
		AvatarFileID:       avatarFileID,
		AvatarURL:          avatarURL,
		HasAvatar:          hasAvatar,
		CreatedBy:          uuid.MustParse(c.CreatedBy),
		MemberCount:        c.MemberCount,
		LastMessagePreview: c.LastMessagePreview,
		CreatedAt:          c.CreatedAt.AsTime(),
		UpdatedAt:          c.UpdatedAt.AsTime(),
		IsDeleted:          c.IsDeleted,
		ConfigID:           configID,
	}
}

func protoToChatMemberDTO(m *chatsv1.ChatMemberDTO) ChatMemberDTO {
	var leftAt *time.Time
	if m.LeftAt != nil {
		t := m.LeftAt.AsTime()
		leftAt = &t
	}
	var isMutedUntil *time.Time
	if m.IsMutedUntil != nil {
		t := m.IsMutedUntil.AsTime()
		isMutedUntil = &t
	}

	return ChatMemberDTO{
		UserID:            uuid.MustParse(m.UserId),
		ChatID:            uuid.MustParse(m.ChatId),
		Role:              membershipRoleFromProto(m.Role),
		JoinedAt:          m.JoinedAt.AsTime(),
		LeftAt:            leftAt,
		NotificationMode:  m.NotificationMode,
		IsMutedUntil:      isMutedUntil,
		LastReadMessageID: m.LastReadMessageId,
		LastActivityAt:    m.LastActivityAt.AsTime(),
	}
}
