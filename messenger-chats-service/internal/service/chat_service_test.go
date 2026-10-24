package service_test

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	repoMocks "github.com/MrEsbens/messenger-chats-service/internal/repository/mocks"
	"github.com/MrEsbens/messenger-chats-service/internal/service"
	grpcclient "github.com/MrEsbens/messenger-chats-service/internal/transport/grpcclient"
	clientMocks "github.com/MrEsbens/messenger-chats-service/internal/transport/grpcclient/mocks"
	redisMocks "github.com/MrEsbens/messenger-chats-service/pkg/redis/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestChatService_CreateChat_Success_Group(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	name := "Test Group"
	createdBy := uuid.New()
	memberID := uuid.New()

	identityClient.On("UserExists", mock.Anything, mock.Anything).Return(true, nil)

	chatRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	configRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	memberRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Twice()
	redisClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	req := service.CreateChatRequest{
		Type:             domain.ChatTypeGroup,
		Name:             &name,
		CreatedBy:        createdBy,
		InitialMemberIDs: []uuid.UUID{memberID},
		CreateConfig:     true,
	}

	response, err := svc.CreateChat(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, domain.ChatTypeGroup, response.Chat.Type)

	identityClient.AssertExpectations(t)
	chatRepo.AssertExpectations(t)
	configRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestChatService_CreateChat_Fail_PrivateWrongMembers(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	req := service.CreateChatRequest{
		Type:             domain.ChatTypePrivate,
		CreatedBy:        uuid.New(),
		InitialMemberIDs: []uuid.UUID{uuid.New()},
	}

	identityClient.On("UserExists", mock.Anything, mock.Anything).Return(true, nil)

	response, err := svc.CreateChat(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, domain.ErrPrivateChatLimit)
}

func TestChatService_AddMember_Success(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	userID := uuid.New()
	addedBy := uuid.New()

	identityClient.On("UserExists", mock.Anything, mock.Anything).Return(true, nil)
	chatRepo.On("GetByID", mock.Anything, chatID).Return(&domain.Chat{ID: chatID, Type: domain.ChatTypeServer}, nil)
	memberRepo.On("GetByChatAndUser", mock.Anything, chatID, addedBy).Return(&domain.ChatMember{ChatID: chatID, UserID: addedBy, Role: domain.RoleOwner}, nil)
	memberRepo.On("Exists", mock.Anything, chatID, userID).Return(false, nil)
	memberRepo.On("CountByChat", mock.Anything, chatID).Return(5, nil)
	memberRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	redisClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := svc.AddMember(context.Background(), chatID, userID, addedBy, domain.RoleMember)

	assert.NoError(t, err)
	identityClient.AssertExpectations(t)
	chatRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestChatService_AddMember_Fail_AlreadyMember(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	userID := uuid.New()
	addedBy := uuid.New()

	identityClient.On("UserExists", mock.Anything, mock.Anything).Return(true, nil)
	chatRepo.On("GetByID", mock.Anything, chatID).Return(&domain.Chat{ID: chatID, Type: domain.ChatTypeServer}, nil)
	memberRepo.On("GetByChatAndUser", mock.Anything, chatID, addedBy).Return(&domain.ChatMember{ChatID: chatID, UserID: addedBy, Role: domain.RoleOwner, LeftAt: nil}, nil)
	memberRepo.On("Exists", mock.Anything, chatID, userID).Return(true, nil)

	err := svc.AddMember(context.Background(), chatID, userID, addedBy, domain.RoleMember)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAlreadyMember)
	chatRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
	identityClient.AssertExpectations(t)
}

func TestChatService_RemoveMember_Success_SelfLeave(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	userID := uuid.New()

	chatRepo.On("GetByID", mock.Anything, chatID).Return(&domain.Chat{ID: chatID, Type: domain.ChatTypeServer}, nil)
	memberRepo.On("GetByChatAndUser", mock.Anything, chatID, userID).Return(&domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil)
	memberRepo.On("Leave", mock.Anything, mock.Anything).Return(nil)
	redisClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := svc.RemoveMember(context.Background(), chatID, userID, userID)

	assert.NoError(t, err)
	chatRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestChatService_RemoveMember_Fail_OwnerCannotLeave(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	userID := uuid.New()

	chatRepo.On("GetByID", mock.Anything, chatID).Return(&domain.Chat{ID: chatID, Type: domain.ChatTypeServer}, nil)
	memberRepo.On("GetByChatAndUser", mock.Anything, chatID, userID).Return(&domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleOwner}, nil)

	err := svc.RemoveMember(context.Background(), chatID, userID, userID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrOwnerCannotLeave)
	chatRepo.AssertExpectations(t)
	memberRepo.AssertExpectations(t)
}

func TestChatService_IsMember_Success(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	userID := uuid.New()

	memberRepo.On("Exists", mock.Anything, chatID, userID).Return(true, nil)
	memberRepo.On("GetByChatAndUser", mock.Anything, chatID, userID).Return(&domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin, LeftAt: nil}, nil)

	isMember, role, err := svc.IsMember(context.Background(), chatID, userID)

	assert.NoError(t, err)
	assert.True(t, isMember)
	assert.Equal(t, domain.RoleAdmin, role)
}

func TestChatService_IsMember_NotMember(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	userID := uuid.New()

	memberRepo.On("Exists", mock.Anything, chatID, userID).Return(false, nil)

	isMember, role, err := svc.IsMember(context.Background(), chatID, userID)

	assert.NoError(t, err)
	assert.False(t, isMember)
	assert.Equal(t, domain.RoleMember, role)
}

func TestChatService_GetChat_Success(t *testing.T) {
	chatRepo := repoMocks.NewMockChatRepository(t)
	configRepo := repoMocks.NewMockChatConfigRepository(t)
	memberRepo := repoMocks.NewMockMemberRepository(t)
	inviteRepo := repoMocks.NewMockInviteRepository(t)
	redisClient := redisMocks.NewMockRedisClient()
	identityClient := clientMocks.NewMockIdentityClient(t)
	fileClient := clientMocks.NewMockFileClient(t)

	svc := service.NewChatService(
		chatRepo, configRepo, memberRepo, inviteRepo,
		redisClient, "chat:",
		fileClient,
		identityClient,
	)

	chatID := uuid.New()
	requesterID := uuid.New()
	chatName := "Test Chat"

	memberRepo.On("Exists", mock.Anything, chatID, requesterID).Return(true, nil)
	memberRepo.On("GetByChatAndUser", mock.Anything, chatID, requesterID).Return(&domain.ChatMember{ChatID: chatID, UserID: requesterID, Role: domain.RoleMember}, nil)
	chatRepo.On("GetByID", mock.Anything, chatID).Return(&domain.Chat{ID: chatID, Type: domain.ChatTypeGroup, Name: &chatName}, nil)

	fileClient.On("GetFileByOwner", mock.Anything, chatID.String(), "chat", "avatars").Return(&grpcclient.FileMetadata{
		FileID:    "avatar-123",
		PublicURL: "https://example.com/avatar.png",
	}, nil)

	response, err := svc.GetChat(context.Background(), chatID, requesterID)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Chat)
	assert.Equal(t, chatID, response.Chat.ID)
	assert.Equal(t, "Test Chat", *response.Chat.Name)

	memberRepo.AssertExpectations(t)
	chatRepo.AssertExpectations(t)
	fileClient.AssertExpectations(t)
}
