package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler реализует сгенерированный интерфейс ChatServiceServer
type GRPCHandler struct {
	chatsv1.UnimplementedChatServiceServer
	service service.ChatServiceInterface
}

// NewGRPCHandler создаёт новый хэндлер
func NewGRPCHandler(svc service.ChatServiceInterface) *GRPCHandler {
	return &GRPCHandler{service: svc}
}

// ───────────────────────────────────────────────────────────
// CRUD чатов
// ───────────────────────────────────────────────────────────

func (h *GRPCHandler) CreateChat(ctx context.Context, req *chatsv1.CreateChatRequest) (*chatsv1.CreateChatResponse, error) {
	chatType, err := protoToDomainChatType(req.Type)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	initialMemberIDs, err := parseUUIDs(req.InitialMemberIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid initial_member_ids")
	}

	response, err := h.service.CreateChat(ctx, service.CreateChatRequest{
		Type:             chatType,
		Name:             optionalStringFromValue(req.Name),
		Description:      optionalStringFromValue(req.Description),
		ConfigID:         optionalUUIDFromValue(req.ConfigId),
		CreatedBy:        parseUUIDStrict(req.CreatedBy),
		InitialMemberIDs: initialMemberIDs,
		CreateConfig:     req.Type != chatsv1.ChatType_CHAT_TYPE_PRIVATE,
	})
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &chatsv1.CreateChatResponse{
		ChatId:    response.Chat.ID.String(),
		CreatedAt: timestamppb.New(response.Chat.CreatedAt),
	}, nil
}

func (h *GRPCHandler) GetChat(ctx context.Context, req *chatsv1.GetChatRequest) (*chatsv1.ChatDTO, error) {
	chatID := parseUUIDStrict(req.ChatId)
	requesterID := parseUUIDStrict(req.RequesterId)

	response, err := h.service.GetChat(ctx, chatID, requesterID)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return chatResponseToProto(response), nil
}

func (h *GRPCHandler) UpdateChat(ctx context.Context, req *chatsv1.UpdateChatRequest) (*emptypb.Empty, error) {
	err := h.service.UpdateChat(ctx, service.UpdateChatRequest{
		ChatID:      parseUUIDStrict(req.ChatId),
		UpdaterID:   parseUUIDStrict(req.UpdaterId),
		Name:        optionalStringFromPtr(req.Name),
		Description: optionalStringFromPtr(req.Description),
		ConfigID:    optionalUUIDFromPtr(req.ConfigId),
	})
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) DeleteChat(ctx context.Context, req *chatsv1.DeleteChatRequest) (*emptypb.Empty, error) {
	err := h.service.DeleteChat(ctx, parseUUIDStrict(req.ChatId), parseUUIDStrict(req.DeleterId))
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) ListUserChats(ctx context.Context, req *chatsv1.ListUserChatsRequest) (*chatsv1.ListUserChatsResponse, error) {
	var typeFilter *domain.ChatType
	if req.TypeFilter != nil && *req.TypeFilter != chatsv1.ChatType_CHAT_TYPE_UNSPECIFIED {
		t, err := protoToDomainChatType(*req.TypeFilter)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		typeFilter = &t
	}

	response, err := h.service.ListUserChats(ctx, parseUUIDStrict(req.UserId), service.ListChatsFilter{
		TypeFilter: typeFilter,
		Limit:      int(req.Limit),
		Offset:     int(req.Offset),
		AfterID:    optionalUUIDFromPtr(req.AfterId),
	})
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	dtos := make([]*chatsv1.ChatDTO, 0, len(response.Chats))
	for _, chatResp := range response.Chats {
		dtos = append(dtos, chatResponseToProto(chatResp))
	}

	return &chatsv1.ListUserChatsResponse{
		Chats:      dtos,
		HasMore:    response.HasMore,
		NextCursor: response.NextCursor,
	}, nil
}

// ───────────────────────────────────────────────────────────
// Управление участниками
// ───────────────────────────────────────────────────────────

func (h *GRPCHandler) AddMember(ctx context.Context, req *chatsv1.AddMemberRequest) (*emptypb.Empty, error) {
	role, err := protoToDomainRole(req.Role)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = h.service.AddMember(ctx,
		parseUUIDStrict(req.ChatId),
		parseUUIDStrict(req.UserId),
		parseUUIDStrict(req.AddedBy),
		role,
	)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) RemoveMember(ctx context.Context, req *chatsv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	err := h.service.RemoveMember(ctx,
		parseUUIDStrict(req.ChatId),
		parseUUIDStrict(req.UserId),
		parseUUIDStrict(req.RemovedBy),
	)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) UpdateMemberRole(ctx context.Context, req *chatsv1.UpdateMemberRoleRequest) (*emptypb.Empty, error) {
	newRole, err := protoToDomainRole(req.NewRole)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = h.service.UpdateMemberRole(ctx,
		parseUUIDStrict(req.ChatId),
		parseUUIDStrict(req.TargetUserId),
		parseUUIDStrict(req.UpdatedBy),
		newRole,
	)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) ListChatMembers(ctx context.Context, req *chatsv1.ListChatMembersRequest) (*chatsv1.ListChatMembersResponse, error) {
	var roleFilter *domain.MembershipRole
	if req.RoleFilter != nil && *req.RoleFilter != chatsv1.MembershipRole_ROLE_UNSPECIFIED {
		r, err := protoToDomainRole(*req.RoleFilter)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		roleFilter = &r
	}

	members, err := h.service.ListChatMembers(ctx, parseUUIDStrict(req.ChatId), service.ListMembersFilter{
		RoleFilter: roleFilter,
		Limit:      int(req.Limit),
		Offset:     int(req.Offset),
	})
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	dtos := make([]*chatsv1.ChatMemberDTO, 0, len(members))
	for _, m := range members {
		dtos = append(dtos, domainToProtoMember(m))
	}

	return &chatsv1.ListChatMembersResponse{
		Members: dtos,
		HasMore: len(members) == int(req.Limit),
	}, nil
}

// ───────────────────────────────────────────────────────────
// Приглашения
// ───────────────────────────────────────────────────────────

func (h *GRPCHandler) CreateInvite(ctx context.Context, req *chatsv1.CreateInviteRequest) (*chatsv1.CreateInviteResponse, error) {
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		expiresAt = &t
	}

	var maxUses *int
	if req.MaxUses != nil {
		val := int(*req.MaxUses)
		maxUses = &val
	}

	invite, err := h.service.CreateInvite(ctx,
		parseUUIDStrict(req.ChatId),
		parseUUIDStrict(req.CreatedBy),
		service.InviteOptions{
			ExpiresAt: expiresAt,
			MaxUses:   maxUses,
		},
	)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	resp := &chatsv1.CreateInviteResponse{
		InviteToken: invite.Token,
	}
	if invite.ExpiresAt != nil {
		resp.ExpiresAt = timestamppb.New(*invite.ExpiresAt)
	}

	return resp, nil
}

func (h *GRPCHandler) JoinByInvite(ctx context.Context, req *chatsv1.JoinByInviteRequest) (*emptypb.Empty, error) {
	err := h.service.JoinByInvite(ctx, req.InviteToken, parseUUIDStrict(req.UserId))
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// ───────────────────────────────────────────────────────────
// Проверки прав (для других сервисов)
// ───────────────────────────────────────────────────────────

func (h *GRPCHandler) IsMember(ctx context.Context, req *chatsv1.IsMemberRequest) (*chatsv1.IsMemberResponse, error) {
	isMember, role, err := h.service.IsMember(ctx, parseUUIDStrict(req.ChatId), parseUUIDStrict(req.UserId))
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &chatsv1.IsMemberResponse{
		IsMember: isMember,
		Role:     domainRoleToProto(role),
	}, nil
}

// ───────────────────────────────────────────────────────────
// Chat Configs
// ───────────────────────────────────────────────────────────

func (h *GRPCHandler) CreateChatConfig(ctx context.Context, req *chatsv1.CreateChatConfigRequest) (*emptypb.Empty, error) {
	config := &domain.ChatConfig{
		ID:                      uuid.New(),
		AllowMemberInvite:       req.AllowMemberInvite,
		AllowMemberKick:         req.AllowMemberKick,
		AllowMemberMute:         req.AllowMemberMute,
		AutoModerationEnabled:   req.AutoModerationEnabled,
		MaxMessageLength:        int(req.MaxMessageLength),
		DefaultNotificationMode: domain.NotificationMode(req.DefaultNotificationMode),
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := h.service.CreateChatConfig(ctx, config); err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) GetChatConfig(ctx context.Context, req *chatsv1.GetChatConfigRequest) (*chatsv1.ChatConfigDTO, error) {
	config, err := h.service.GetChatConfig(ctx, parseUUIDStrict(req.ConfigId))
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return domainToProtoConfig(config), nil
}

func (h *GRPCHandler) UpdateChatConfig(ctx context.Context, req *chatsv1.UpdateChatConfigRequest) (*emptypb.Empty, error) {
	config := &domain.ChatConfig{
		ID:                      parseUUIDStrict(req.ConfigId),
		AllowMemberInvite:       derefBool(req.AllowMemberInvite, false),
		AllowMemberKick:         derefBool(req.AllowMemberKick, false),
		AllowMemberMute:         derefBool(req.AllowMemberMute, false),
		AutoModerationEnabled:   derefBool(req.AutoModerationEnabled, false),
		MaxMessageLength:        derefInt32(req.MaxMessageLength, 4000),
		DefaultNotificationMode: derefNotificationMode(req.DefaultNotificationMode, domain.NotifyAll),
		UpdatedAt:               time.Now(),
	}

	if err := h.service.UpdateChatConfig(ctx, config); err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// ───────────────────────────────────────────────────────────
// Хелперы: маппинг и утилиты
// ───────────────────────────────────────────────────────────

// Для обычных полей (string → *string)
func optionalStringFromValue(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalUUIDFromValue(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func optionalStringFromPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func optionalUUIDFromPtr(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}

func derefBool(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func derefInt32(ptr *int32, def int32) int {
	if ptr == nil {
		return int(def)
	}
	return int(*ptr)
}

func derefNotificationMode(ptr *string, def domain.NotificationMode) domain.NotificationMode {
	if ptr == nil || *ptr == "" {
		return def
	}
	return domain.NotificationMode(*ptr)
}

func protoToDomainChatType(t chatsv1.ChatType) (domain.ChatType, error) {
	switch t {
	case chatsv1.ChatType_CHAT_TYPE_PRIVATE:
		return domain.ChatTypePrivate, nil
	case chatsv1.ChatType_CHAT_TYPE_GROUP:
		return domain.ChatTypeGroup, nil
	case chatsv1.ChatType_CHAT_TYPE_SERVER:
		return domain.ChatTypeServer, nil
	default:
		return "", fmt.Errorf("invalid chat type: %v", t)
	}
}

func domainToProtoChatType(t domain.ChatType) chatsv1.ChatType {
	switch t {
	case domain.ChatTypePrivate:
		return chatsv1.ChatType_CHAT_TYPE_PRIVATE
	case domain.ChatTypeGroup:
		return chatsv1.ChatType_CHAT_TYPE_GROUP
	case domain.ChatTypeServer:
		return chatsv1.ChatType_CHAT_TYPE_SERVER
	default:
		return chatsv1.ChatType_CHAT_TYPE_UNSPECIFIED
	}
}

func protoToDomainRole(r chatsv1.MembershipRole) (domain.MembershipRole, error) {
	switch r {
	case chatsv1.MembershipRole_ROLE_OWNER:
		return domain.RoleOwner, nil
	case chatsv1.MembershipRole_ROLE_ADMIN:
		return domain.RoleAdmin, nil
	case chatsv1.MembershipRole_ROLE_MEMBER:
		return domain.RoleMember, nil
	case chatsv1.MembershipRole_ROLE_MUTED:
		return domain.RoleMuted, nil
	case chatsv1.MembershipRole_ROLE_BANNED:
		return domain.RoleBanned, nil
	default:
		return "", fmt.Errorf("invalid role: %v", r)
	}
}

func domainRoleToProto(r domain.MembershipRole) chatsv1.MembershipRole {
	switch r {
	case domain.RoleOwner:
		return chatsv1.MembershipRole_ROLE_OWNER
	case domain.RoleAdmin:
		return chatsv1.MembershipRole_ROLE_ADMIN
	case domain.RoleMember:
		return chatsv1.MembershipRole_ROLE_MEMBER
	case domain.RoleMuted:
		return chatsv1.MembershipRole_ROLE_MUTED
	case domain.RoleBanned:
		return chatsv1.MembershipRole_ROLE_BANNED
	default:
		return chatsv1.MembershipRole_ROLE_UNSPECIFIED
	}
}

func chatResponseToProto(response *service.ChatResponse) *chatsv1.ChatDTO {
	chat := response.Chat

	dto := &chatsv1.ChatDTO{
		Id:        chat.ID.String(),
		Type:      domainToProtoChatType(chat.Type),
		CreatedBy: chat.CreatedBy.String(),
		CreatedAt: timestamppb.New(chat.CreatedAt),
		UpdatedAt: timestamppb.New(chat.UpdatedAt),
		IsDeleted: chat.DeletedAt != nil,
	}

	if chat.Name != nil {
		dto.Name = *chat.Name
	}
	if chat.Description != nil {
		dto.Description = *chat.Description
	}
	if chat.ConfigID != nil {
		configIDStr := chat.ConfigID.String()
		dto.ConfigId = &configIDStr
	}

	if response.AvatarFileID != "" {
		dto.AvatarFileId = &response.AvatarFileID
	}
	if response.AvatarURL != "" {
		dto.AvatarUrl = &response.AvatarURL
	}
	dto.HasAvatar = &response.HasAvatar

	return dto
}

func domainToProtoMember(m *domain.ChatMember) *chatsv1.ChatMemberDTO {
	dto := &chatsv1.ChatMemberDTO{
		UserId:           m.UserID.String(),
		ChatId:           m.ChatID.String(),
		Role:             domainRoleToProto(m.Role),
		JoinedAt:         timestamppb.New(m.JoinedAt),
		NotificationMode: string(m.NotificationMode),
	}

	if m.LeftAt != nil {
		dto.LeftAt = timestamppb.New(*m.LeftAt)
	}
	if m.IsMutedUntil != nil {
		dto.IsMutedUntil = timestamppb.New(*m.IsMutedUntil)
	}
	if m.LastReadMessageID != nil {
		dto.LastReadMessageId = m.LastReadMessageID.String()
	}
	if m.LastActivityAt != nil {
		dto.LastActivityAt = timestamppb.New(*m.LastActivityAt)
	}

	return dto
}

func domainToProtoConfig(c *domain.ChatConfig) *chatsv1.ChatConfigDTO {
	return &chatsv1.ChatConfigDTO{
		Id:                      c.ID.String(),
		AllowMemberInvite:       c.AllowMemberInvite,
		AllowMemberKick:         c.AllowMemberKick,
		AllowMemberMute:         c.AllowMemberMute,
		AutoModerationEnabled:   c.AutoModerationEnabled,
		MaxMessageLength:        int32(c.MaxMessageLength),
		DefaultNotificationMode: string(c.DefaultNotificationMode),
		CreatedAt:               timestamppb.New(c.CreatedAt),
		UpdatedAt:               timestamppb.New(c.UpdatedAt),
	}
}

func parseUUIDStrict(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func parseUUIDs(ids []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, nil
}

func domainErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrChatNotFound),
		errors.Is(err, domain.ErrMemberNotFound),
		errors.Is(err, domain.ErrInviteNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrPermissionDenied),
		errors.Is(err, domain.ErrNotMember),
		errors.Is(err, domain.ErrInsufficientRole):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrInvalidChatType),
		errors.Is(err, domain.ErrInvalidChatName),
		errors.Is(err, domain.ErrPrivateChatLimit),
		errors.Is(err, domain.ErrGroupChatLimit),
		errors.Is(err, domain.ErrAlreadyMember):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, domain.ErrInviteExpired),
		errors.Is(err, domain.ErrInviteMaxUses),
		errors.Is(err, domain.ErrInviteAlreadyUsed):
		return status.Error(codes.FailedPrecondition, err.Error())

	default:
		return status.Error(codes.Internal, err.Error())
	}
}
