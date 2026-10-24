package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/repository"
	grpcclient "github.com/MrEsbens/messenger-chats-service/internal/transport/grpcclient"
	"github.com/MrEsbens/messenger-chats-service/pkg/redis"
	"github.com/google/uuid"
)

type ChatServiceInterface interface {
	CreateChat(ctx context.Context, req CreateChatRequest) (*ChatResponse, error)
	GetChat(ctx context.Context, chatID, requesterID uuid.UUID) (*ChatResponse, error)
	UpdateChat(ctx context.Context, req UpdateChatRequest) error
	DeleteChat(ctx context.Context, chatID, deleterID uuid.UUID) error
	ListUserChats(ctx context.Context, userID uuid.UUID, filter ListChatsFilter) (*ChatListResponse, error)

	AddMember(ctx context.Context, chatID, userID, addedBy uuid.UUID, role domain.MembershipRole) error
	RemoveMember(ctx context.Context, chatID, userID, removedBy uuid.UUID) error
	UpdateMemberRole(ctx context.Context, chatID, targetUserID, updatedBy uuid.UUID, newRole domain.MembershipRole) error
	ListChatMembers(ctx context.Context, chatID uuid.UUID, filter ListMembersFilter) ([]*domain.ChatMember, error)

	CreateInvite(ctx context.Context, chatID, createdBy uuid.UUID, opts InviteOptions) (*domain.ChatInvite, error)
	JoinByInvite(ctx context.Context, token string, userID uuid.UUID) error

	IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, domain.MembershipRole, error)

	CreateChatConfig(ctx context.Context, config *domain.ChatConfig) error
	GetChatConfig(ctx context.Context, configID uuid.UUID) (*domain.ChatConfig, error)
	UpdateChatConfig(ctx context.Context, config *domain.ChatConfig) error
}

type ChatService struct {
	chatRepo       repository.ChatRepository
	configRepo     repository.ChatConfigRepository
	memberRepo     repository.MemberRepository
	inviteRepo     repository.InviteRepository
	redis          redis.RedisClientInterface
	redisPrefix    string
	fileClient     grpcclient.FileClientInterface
	identityClient grpcclient.IdentityClientInterface
}

func NewChatService(
	chatRepo repository.ChatRepository,
	configRepo repository.ChatConfigRepository,
	memberRepo repository.MemberRepository,
	inviteRepo repository.InviteRepository,
	redisClient redis.RedisClientInterface,
	redisPrefix string,
	fileClient grpcclient.FileClientInterface,
	identityClient grpcclient.IdentityClientInterface,
) ChatServiceInterface {
	return &ChatService{
		chatRepo:       chatRepo,
		configRepo:     configRepo,
		memberRepo:     memberRepo,
		inviteRepo:     inviteRepo,
		redis:          redisClient,
		redisPrefix:    redisPrefix,
		fileClient:     fileClient,
		identityClient: identityClient,
	}
}

type CreateChatRequest struct {
	Type             domain.ChatType
	Name             *string
	Description      *string
	ConfigID         *uuid.UUID
	CreatedBy        uuid.UUID
	InitialMemberIDs []uuid.UUID
	CreateConfig     bool
}

type UpdateChatRequest struct {
	ChatID      uuid.UUID
	UpdaterID   uuid.UUID
	Name        *string
	Description *string
	ConfigID    *uuid.UUID
}

type ListChatsFilter struct {
	TypeFilter *domain.ChatType
	Limit      int
	Offset     int
	AfterID    *uuid.UUID
}

type ListMembersFilter struct {
	RoleFilter *domain.MembershipRole
	Limit      int
	Offset     int
}

type InviteOptions struct {
	ExpiresAt *time.Time
	MaxUses   *int
}

type ChatResponse struct {
	Chat         *domain.Chat
	AvatarFileID string
	AvatarURL    string
	HasAvatar    bool
}

type ChatListResponse struct {
	Chats      []*ChatResponse
	HasMore    bool
	NextCursor string
}

func (s *ChatService) CreateChat(ctx context.Context, req CreateChatRequest) (*ChatResponse, error) {
	if !domain.IsValidChatType(req.Type) {
		return nil, fmt.Errorf("invalid chat type: %s", req.Type)
	}

	if req.Type == domain.ChatTypePrivate && len(req.InitialMemberIDs) != 2 {
		return nil, domain.ErrPrivateChatLimit
	}

	if s.identityClient != nil {
		creatorExists, err := s.identityClient.UserExists(ctx, req.CreatedBy.String())
		if err != nil {
			return nil, fmt.Errorf("⚠️ Identity service error for creator %s: %v", req.CreatedBy, err)
		} else if !creatorExists {
			return nil, fmt.Errorf("creator not found: %s", req.CreatedBy)
		}

		for _, memberID := range req.InitialMemberIDs {
			if memberID == req.CreatedBy {
				continue
			}

			exists, err := s.identityClient.UserExists(ctx, memberID.String())
			if err != nil {
				log.Printf("⚠️ Identity service error for member %s: %v", memberID, err)
				continue // В dev продолжаем
			}
			if !exists {
				return nil, fmt.Errorf("member not found: %s", memberID)
			}
		}
	}

	var configID *uuid.UUID
	if req.CreateConfig && req.Type != domain.ChatTypePrivate {
		config := domain.NewChatConfig()
		if err := s.configRepo.Create(ctx, config); err != nil {
			return nil, fmt.Errorf("failed to create chat config: %w", err)
		}
		configID = &config.ID
	} else if req.ConfigID != nil {
		exists, err := s.configRepo.Exists(ctx, *req.ConfigID)
		if err != nil {
			return nil, fmt.Errorf("failed to check config exists: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("config not found: %s", *req.ConfigID)
		}
		configID = req.ConfigID
	}

	chat, err := domain.NewChat(req.Type, req.Name, req.Description, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat domain object: %w", err)
	}

	chat.ConfigID = configID

	if err := s.chatRepo.Create(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to create chat in repository: %w", err)
	}

	if req.Type == domain.ChatTypePrivate {
		for _, memberID := range req.InitialMemberIDs {
			member, err := domain.NewChatMember(chat.ID, memberID, domain.RoleMember)
			if err != nil {
				return nil, fmt.Errorf("failed to create member: %w", err)
			}
			if err := s.memberRepo.Create(ctx, member); err != nil {
				return nil, fmt.Errorf("failed to add member: %w", err)
			}
		}
	}

	if req.Type == domain.ChatTypeGroup || req.Type == domain.ChatTypeServer {
		owner, err := domain.NewChatMember(chat.ID, req.CreatedBy, domain.RoleOwner)
		if err != nil {
			return nil, fmt.Errorf("failed to create owner member: %w", err)
		}
		if err := s.memberRepo.Create(ctx, owner); err != nil {
			return nil, fmt.Errorf("failed to add owner: %w", err)
		}

		for _, memberID := range req.InitialMemberIDs {
			if memberID == req.CreatedBy {
				continue
			}
			role := domain.RoleMember
			if req.Type == domain.ChatTypeServer && len(req.InitialMemberIDs) > 1 {
				role = domain.RoleMember
			}
			member, err := domain.NewChatMember(chat.ID, memberID, role)
			if err != nil {
				return nil, fmt.Errorf("failed to create member: %w", err)
			}
			if err := s.memberRepo.Create(ctx, member); err != nil {
				return nil, fmt.Errorf("failed to add member: %w", err)
			}
		}
	}

	_ = s.publishChatCreatedEvent(ctx, chat)

	return &ChatResponse{
		Chat: chat,
	}, nil
}

func (s *ChatService) GetChat(ctx context.Context, chatID, requesterID uuid.UUID) (*ChatResponse, error) {
	isMember, _, err := s.IsMember(ctx, chatID, requesterID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, domain.ErrPermissionDenied
	}

	chat, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	response := &ChatResponse{
		Chat:      chat,
		HasAvatar: false,
	}

	response.AvatarFileID, response.AvatarURL = s.loadChatAvatar(ctx, chatID)
	response.HasAvatar = response.AvatarURL != ""

	return response, nil
}

func (s *ChatService) UpdateChat(ctx context.Context, req UpdateChatRequest) error {
	chat, err := s.chatRepo.GetByID(ctx, req.ChatID)
	if err != nil {
		return fmt.Errorf("failed to get chat: %w", err)
	}

	role, isActive, err := s.GetMemberRole(ctx, req.ChatID, req.UpdaterID)
	if err != nil {
		return fmt.Errorf("failed to get member role: %w", err)
	}
	if !isActive {
		return domain.ErrNotMember
	}

	if !domain.CanUpdateChat(chat.Type, role, role == domain.RoleOwner) {
		return domain.ErrPermissionDenied
	}

	chat.Update(req.Name, req.Description, req.ConfigID)

	if err := s.chatRepo.Update(ctx, chat); err != nil {
		return fmt.Errorf("failed to update chat: %w", err)
	}

	_ = s.publishChatUpdatedEvent(ctx, chat)

	return nil
}

func (s *ChatService) DeleteChat(ctx context.Context, chatID, deleterID uuid.UUID) error {
	role, isActive, err := s.GetMemberRole(ctx, chatID, deleterID)
	if err != nil {
		return fmt.Errorf("failed to get member role: %w", err)
	}
	if !isActive || role != domain.RoleOwner {
		return domain.ErrPermissionDenied
	}

	if err := s.chatRepo.Delete(ctx, chatID); err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}

	_ = s.publishChatDeletedEvent(ctx, chatID, deleterID)

	return nil
}

func (s *ChatService) ListUserChats(ctx context.Context, userID uuid.UUID, filter ListChatsFilter) (*ChatListResponse, error) {
	members, err := s.memberRepo.GetByUser(ctx, userID, true, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user members: %w", err)
	}

	chatIDs := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		chatIDs = append(chatIDs, m.ChatID)
	}

	chatResponses := make([]*ChatResponse, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		chat, err := s.chatRepo.GetByID(ctx, chatID)
		if err != nil {
			continue
		}

		if filter.TypeFilter != nil && chat.Type != *filter.TypeFilter {
			continue
		}

		response := &ChatResponse{
			Chat:      chat,
			HasAvatar: false,
		}
		response.AvatarFileID, response.AvatarURL = s.loadChatAvatar(ctx, chatID)
		response.HasAvatar = response.AvatarURL != ""

		chatResponses = append(chatResponses, response)
	}

	return &ChatListResponse{
		Chats:      chatResponses,
		HasMore:    len(chatResponses) == filter.Limit,
		NextCursor: "",
	}, nil
}

func (s *ChatService) AddMember(ctx context.Context, chatID, userID, addedBy uuid.UUID, role domain.MembershipRole) error {
	chat, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get chat: %w", err)
	}

	if !domain.IsRoleAllowedForChatType(chat.Type, role) {
		return fmt.Errorf("role %s is not valid for chat type %s", role, chat.Type)
	}

	if s.identityClient != nil {
		exists, err := s.identityClient.UserExists(ctx, userID.String())
		if err != nil {
			log.Printf("⚠️ Identity service error for user %s: %v", userID, err)
			return fmt.Errorf("identity service error: %w", err)
		} else if !exists {
			return fmt.Errorf("user not found: %s", userID)
		}
	}

	adderRole, isActive, err := s.GetMemberRole(ctx, chatID, addedBy)
	if err != nil {
		return fmt.Errorf("failed to get adder role: %w", err)
	}
	if !isActive {
		return domain.ErrNotMember
	}

	if !domain.CanAddMember(chat.Type, adderRole, role, adderRole == domain.RoleOwner) {
		return domain.ErrPermissionDenied
	}

	exists, err := s.memberRepo.Exists(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check member exists: %w", err)
	}
	if exists {
		return domain.ErrAlreadyMember
	}

	count, err := s.memberRepo.CountByChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}

	if err := domain.ValidateChatMemberLimit(chat.Type, count, count+1); err != nil {
		return err
	}

	member, err := domain.NewChatMember(chatID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	_ = s.publishMemberAddedEvent(ctx, chatID, userID, addedBy, role)

	return nil
}

func (s *ChatService) RemoveMember(ctx context.Context, chatID, userID, removedBy uuid.UUID) error {
	chat, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get chat: %w", err)
	}

	if chat.Type == domain.ChatTypePrivate {
		return fmt.Errorf("cannot remove members from private chat")
	}

	if userID == removedBy {
		member, err := s.memberRepo.GetByChatAndUser(ctx, chatID, userID)
		if err != nil {
			return fmt.Errorf("failed to get member: %w", err)
		}

		if member.Role == domain.RoleOwner {
			return domain.ErrOwnerCannotLeave
		}

		if err := s.memberRepo.Leave(ctx, member.ID); err != nil {
			return fmt.Errorf("failed to leave chat: %w", err)
		}

		_ = s.publishMemberLeftEvent(ctx, chatID, userID)
		return nil
	}

	removerRole, isActive, err := s.GetMemberRole(ctx, chatID, removedBy)
	if err != nil {
		return fmt.Errorf("failed to get remover role: %w", err)
	}
	if !isActive {
		return domain.ErrNotMember
	}

	targetRole, _, err := s.GetMemberRole(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to get target role: %w", err)
	}

	if !domain.CanRemoveMember(chat.Type, removerRole, targetRole, removerRole == domain.RoleOwner, targetRole == domain.RoleOwner) {
		return domain.ErrPermissionDenied
	}

	member, err := s.memberRepo.GetByChatAndUser(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}

	if err := s.memberRepo.Leave(ctx, member.ID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	_ = s.publishMemberRemovedEvent(ctx, chatID, userID, removedBy)

	return nil
}

func (s *ChatService) UpdateMemberRole(ctx context.Context, chatID, targetUserID, updatedBy uuid.UUID, newRole domain.MembershipRole) error {
	chat, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get chat: %w", err)
	}

	if !domain.IsRoleAllowedForChatType(chat.Type, newRole) {
		return fmt.Errorf("role %s is not valid for chat type %s", newRole, chat.Type)
	}

	if chat.Type == domain.ChatTypePrivate {
		return fmt.Errorf("cannot change roles in private chat")
	}

	updaterRole, isActive, err := s.GetMemberRole(ctx, chatID, updatedBy)
	if err != nil {
		return fmt.Errorf("failed to get updater role: %w", err)
	}
	if !isActive {
		return domain.ErrNotMember
	}

	targetRole, _, err := s.GetMemberRole(ctx, chatID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get target role: %w", err)
	}

	if !domain.CanChangeRole(chat.Type, updaterRole, targetRole, newRole, updaterRole == domain.RoleOwner) {
		return domain.ErrPermissionDenied
	}

	member, err := s.memberRepo.GetByChatAndUser(ctx, chatID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}

	if err := s.memberRepo.UpdateRole(ctx, member.ID, newRole); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	_ = s.publishMemberRoleChangedEvent(ctx, chatID, targetUserID, updatedBy, targetRole, newRole)

	return nil
}

func (s *ChatService) ListChatMembers(ctx context.Context, chatID uuid.UUID, filter ListMembersFilter) ([]*domain.ChatMember, error) {
	members, err := s.memberRepo.GetByChat(ctx, chatID, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat members: %w", err)
	}

	return members, nil
}

func (s *ChatService) CreateInvite(ctx context.Context, chatID, createdBy uuid.UUID, opts InviteOptions) (*domain.ChatInvite, error) {
	role, isActive, err := s.GetMemberRole(ctx, chatID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get member role: %w", err)
	}
	if !isActive {
		return nil, domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	if !domain.CanUpdateChat(chat.Type, role, role == domain.RoleOwner) {
		return nil, domain.ErrPermissionDenied
	}

	if opts.MaxUses != nil && (*opts.MaxUses < domain.MinInviteMaxUses || *opts.MaxUses > domain.MaxInviteMaxUses) {
		return nil, fmt.Errorf("invalid max_uses: must be between %d and %d", domain.MinInviteMaxUses, domain.MaxInviteMaxUses)
	}

	invite, err := domain.NewChatInvite(chatID, createdBy, opts.ExpiresAt, opts.MaxUses)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	if err := s.inviteRepo.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("failed to save invite: %w", err)
	}

	return invite, nil
}

func (s *ChatService) JoinByInvite(ctx context.Context, token string, userID uuid.UUID) error {
	invite, err := s.inviteRepo.GetByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to get invite: %w", err)
	}

	if !invite.IsActive() {
		if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
			return domain.ErrInviteExpired
		}
		if invite.MaxUses != nil && invite.UsesCount >= *invite.MaxUses {
			return domain.ErrInviteMaxUses
		}
		return domain.ErrInviteAlreadyUsed
	}

	if s.identityClient != nil {
		exists, err := s.identityClient.UserExists(ctx, userID.String())
		if err != nil {
			log.Printf("⚠️ Identity service error for user %s: %v", userID, err)
		} else if !exists {
			return fmt.Errorf("user not found: %s", userID)
		}
	}

	exists, err := s.memberRepo.Exists(ctx, invite.ChatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check member exists: %w", err)
	}
	if exists {
		return domain.ErrAlreadyMember
	}

	chat, err := s.chatRepo.GetByID(ctx, invite.ChatID)
	if err != nil {
		return fmt.Errorf("failed to get chat: %w", err)
	}

	count, err := s.memberRepo.CountByChat(ctx, invite.ChatID)
	if err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}

	if err := domain.ValidateChatMemberLimit(chat.Type, count, count+1); err != nil {
		return err
	}

	member, err := domain.NewChatMember(invite.ChatID, userID, domain.RoleMember)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	if err := s.inviteRepo.IncrementUses(ctx, invite.ID); err != nil {
		return fmt.Errorf("failed to increment invite uses: %w", err)
	}

	_ = s.publishInviteAcceptedEvent(ctx, invite.ChatID, userID, invite.CreatedBy)

	return nil
}

func (s *ChatService) IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, domain.MembershipRole, error) {
	exists, err := s.memberRepo.Exists(ctx, chatID, userID)
	if err != nil {
		return false, domain.RoleMember, fmt.Errorf("failed to check member exists: %w", err)
	}

	if !exists {
		return false, domain.RoleMember, nil
	}

	member, err := s.memberRepo.GetByChatAndUser(ctx, chatID, userID)
	if err != nil {
		return false, domain.RoleMember, fmt.Errorf("failed to get member: %w", err)
	}

	return member.IsActive(), member.Role, nil
}

func (s *ChatService) GetMemberRole(ctx context.Context, chatID, userID uuid.UUID) (domain.MembershipRole, bool, error) {
	member, err := s.memberRepo.GetByChatAndUser(ctx, chatID, userID)
	if err != nil {
		return domain.RoleMember, false, fmt.Errorf("failed to get member: %w", err)
	}

	return member.Role, member.IsActive(), nil
}

func (s *ChatService) CreateChatConfig(ctx context.Context, config *domain.ChatConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := s.configRepo.Create(ctx, config); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	return nil
}

func (s *ChatService) GetChatConfig(ctx context.Context, configID uuid.UUID) (*domain.ChatConfig, error) {
	config, err := s.configRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return config, nil
}

func (s *ChatService) UpdateChatConfig(ctx context.Context, config *domain.ChatConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := s.configRepo.Update(ctx, config); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}

func (s *ChatService) loadChatAvatar(ctx context.Context, chatID uuid.UUID) (string, string) {
	if s.fileClient == nil {
		return "", ""
	}

	fileCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	file, err := s.fileClient.GetFileByOwner(
		fileCtx,
		chatID.String(),
		"chat",
		"avatars",
	)

	if err == nil && file != nil {
		return file.FileID, file.PublicURL
	}

	return "", ""
}

// ───────────────────────────────────────────────────────────
// Redis Pub/Sub: публикация событий для WebSocket уведомлений
// ───────────────────────────────────────────────────────────

func (s *ChatService) publishChatCreatedEvent(ctx context.Context, chat *domain.Chat) error {
	event := ChatCreatedEvent{
		Type:      "chat_created",
		ChatID:    chat.ID.String(),
		CreatedBy: chat.CreatedBy.String(),
		CreatedAt: time.Now(),
	}
	channel := fmt.Sprintf("%screated:%s", s.redisPrefix, chat.ID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishChatUpdatedEvent(ctx context.Context, chat *domain.Chat) error {
	event := ChatUpdatedEvent{
		Type:      "chat_updated",
		ChatID:    chat.ID.String(),
		UpdatedAt: time.Now(),
	}
	channel := fmt.Sprintf("%supdated:%s", s.redisPrefix, chat.ID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishChatDeletedEvent(ctx context.Context, chatID, deleterID uuid.UUID) error {
	event := ChatDeletedEvent{
		Type:      "chat_deleted",
		ChatID:    chatID.String(),
		DeleterID: deleterID.String(),
		DeletedAt: time.Now(),
	}
	channel := fmt.Sprintf("%sdeleted:%s", s.redisPrefix, chatID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishMemberAddedEvent(ctx context.Context, chatID, userID, addedBy uuid.UUID, role domain.MembershipRole) error {
	event := MemberAddedEvent{
		Type:    "member_added",
		ChatID:  chatID.String(),
		UserID:  userID.String(),
		Role:    string(role),
		AddedBy: addedBy.String(),
	}
	channel := fmt.Sprintf("%smember_added:%s", s.redisPrefix, chatID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishMemberRemovedEvent(ctx context.Context, chatID, userID, removedBy uuid.UUID) error {
	event := MemberRemovedEvent{
		Type:      "member_removed",
		ChatID:    chatID.String(),
		UserID:    userID.String(),
		RemovedBy: removedBy.String(),
	}
	channel := fmt.Sprintf("%smember_removed:%s", s.redisPrefix, chatID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishMemberLeftEvent(ctx context.Context, chatID, userID uuid.UUID) error {
	event := MemberLeftEvent{
		Type:   "member_left",
		ChatID: chatID.String(),
		UserID: userID.String(),
	}
	channel := fmt.Sprintf("%smember_left:%s", s.redisPrefix, chatID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishMemberRoleChangedEvent(ctx context.Context, chatID, userID, updatedBy uuid.UUID, oldRole, newRole domain.MembershipRole) error {
	event := MemberRoleChangedEvent{
		Type:      "member_role_changed",
		ChatID:    chatID.String(),
		UserID:    userID.String(),
		OldRole:   string(oldRole),
		NewRole:   string(newRole),
		UpdatedBy: updatedBy.String(),
	}
	channel := fmt.Sprintf("%smember_role_changed:%s", s.redisPrefix, chatID.String())
	return s.redis.Publish(ctx, channel, event)
}

func (s *ChatService) publishInviteAcceptedEvent(ctx context.Context, chatID, userID, invitedBy uuid.UUID) error {
	event := InviteAcceptedEvent{
		Type:      "invite_accepted",
		ChatID:    chatID.String(),
		UserID:    userID.String(),
		InvitedBy: invitedBy.String(),
	}
	channel := fmt.Sprintf("%sinvite_accepted:%s", s.redisPrefix, chatID.String())
	return s.redis.Publish(ctx, channel, event)
}
