package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	chatsv1 "github.com/MrEsbens/messenger-chats-service/api/chats/v1"
	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"github.com/MrEsbens/messenger-message-service/internal/cache/redis"
	rediscache_mocks "github.com/MrEsbens/messenger-message-service/internal/cache/redis/mocks"
	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/repository"
	repoMocks "github.com/MrEsbens/messenger-message-service/internal/repository/mocks"
	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	chatMocks "github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient/mocks"
	fileMocks "github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient/mocks"
	grpcclient_mocks "github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient/mocks"
	redisMocks "github.com/MrEsbens/messenger-message-service/pkg/redis/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(
	t *testing.T,
	msgRepo *repoMocks.MockMessageRepository,
	statusRepo *repoMocks.MockStatusRepository,
	redisClient *redisMocks.MockRedisClient,
	fileClient *fileMocks.MockFileClient,
	chatClient *chatMocks.MockChatClient,
	serversClient grpcclient.ServersClientInterface,
	moderationClient grpcclient.ModerationClientInterface,
	modConfigCache redis.ModerationConfigCacheInterface,
) service.MessageServiceInterface {
	return service.NewMessageService(
		msgRepo,
		statusRepo,
		redisClient,
		fileClient,
		chatClient,
		serversClient,
		moderationClient,
		nil,
		modConfigCache,
	)
}

func newTestMessage(chatID, senderID uuid.UUID) *domain.Message {
	return &domain.Message{
		ID:          uuid.New(),
		ChatID:      chatID,
		SenderID:    senderID,
		Content:     "Test message",
		ContentType: domain.ContentTypeText,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestMessageService_SendMessage_Success(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.Nil
	content := "Hello, World!"

	createdMsg := newTestMessage(chatID, senderID)
	createdMsg.Content = content

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{
			IsMember: true,
			Role:     chatsv1.MembershipRole_ROLE_MEMBER,
		}, nil)

	mockMsgRepo.On("Create", mock.Anything, mock.MatchedBy(func(msg *domain.Message) bool {
		return msg.ChatID == chatID &&
			msg.SenderID == senderID &&
			msg.Content == content &&
			msg.ContentType == domain.ContentTypeText
	})).Run(func(args mock.Arguments) {
		msg := args.Get(1).(*domain.Message)
		msg.ID = createdMsg.ID
		msg.CreatedAt = createdMsg.CreatedAt
		msg.UpdatedAt = createdMsg.UpdatedAt
	}).Return(nil)

	mockStatusRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(s *repository.MessageStatus) bool {
		return s.MessageID == createdMsg.ID &&
			s.UserID == senderID &&
			s.Status == domain.StatusSent
	})).Return(nil)

	mockFileClient.On("GetFilesByOwner",
		mock.Anything,
		mock.MatchedBy(func(req *filev1.GetFilesByOwnerRequest) bool {
			return req.OwnerType == domain.FileOwnerTypeMessage &&
				req.BucketType == domain.FileBucketAttachments
		}),
	).Return(&filev1.GetFilesByOwnerResponse{
		Files: []*filev1.FileInfo{},
	}, nil).Maybe()

	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		content,
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, createdMsg.ID, result.ID)

	mockMsgRepo.AssertExpectations(t)
	mockStatusRepo.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
	mockFileClient.AssertExpectations(t)
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_SendMessage_NotMember(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.Nil

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{
			IsMember: false,
			Role:     chatsv1.MembershipRole_ROLE_MEMBER,
		}, nil)

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		"content",
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)

	mockMsgRepo.AssertNotCalled(t, "Create")
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_SendMessage_ChatClientError(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.Nil

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(nil, errors.New("chats service unavailable"))

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		"content",
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check membership")

	mockChatClient.AssertExpectations(t)
}

func TestMessageService_SendMessage_InvalidContentType(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	result, err := svc.SendMessage(
		context.Background(),
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		"content",
		"invalid_type",
		nil,
		false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid content type")

	mockMsgRepo.AssertNotCalled(t, "Create")
	mockChatClient.AssertNotCalled(t, "IsMember")
}

func TestMessageService_SendMessage_EmptyContent(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	result, err := svc.SendMessage(
		context.Background(),
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		"",
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.ErrorIs(t, err, repository.ErrInvalidInput)
	assert.Nil(t, result)

	mockMsgRepo.AssertNotCalled(t, "Create")
	mockChatClient.AssertNotCalled(t, "IsMember")
}

func TestMessageService_SendMessage_RepoError(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.Nil

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockMsgRepo.On("Create", mock.Anything, mock.Anything).
		Return(errors.New("database connection lost"))

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		"content",
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create message")

	mockStatusRepo.AssertNotCalled(t, "Upsert")
	mockRedis.AssertNotCalled(t, "Publish")
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_SendMessage_WithAttachment(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.Nil
	messageID := uuid.New()

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	createdMsg := newTestMessage(chatID, senderID)
	createdMsg.ID = messageID
	createdMsg.HasAttachment = true

	mockMsgRepo.On("Create", mock.Anything, mock.MatchedBy(func(msg *domain.Message) bool {
		return msg.HasAttachment
	})).Run(func(args mock.Arguments) {
		msg := args.Get(1).(*domain.Message)
		msg.ID = messageID
	}).Return(nil)

	mockStatusRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)

	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.MatchedBy(func(req *filev1.GetFilesByOwnerRequest) bool {
		return req.OwnerId == messageID.String() &&
			req.OwnerType == domain.FileOwnerTypeMessage &&
			req.BucketType == domain.FileBucketAttachments
	})).Return(&filev1.GetFilesByOwnerResponse{
		Files: []*filev1.FileInfo{
			{
				FileId:    uuid.New().String(),
				PublicUrl: "http://minio/bucket/file.png?sig=abc",
				MimeType:  "image/png",
				FileSize:  102400,
				FileName:  "image.png",
			},
		},
	}, nil)

	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		"Message with file",
		domain.ContentTypeText,
		nil,
		true,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasAttachment)

	mockFileClient.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_SendMessage_WithModeration_Allowed(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)
	mockServersClient := grpcclient_mocks.NewMockServersClient(t)
	mockModerationClient := grpcclient_mocks.NewMockModerationClient(t)
	mockCache := rediscache_mocks.NewMockModerationConfigCache(t)

	svc := service.NewMessageService(
		mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient,
		mockServersClient, mockModerationClient, nil, mockCache,
	)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.New()
	content := "Hello, World!"

	mockServersClient.On("IsMember", mock.Anything, serverID.String(), senderID.String()).Return(true, nil)

	mockCache.On("Get", mock.Anything, serverID.String()).Return(&domain.ModerationConfig{
		ServerID:               serverID.String(),
		ProfanityFilterAction:  "none",
		ToxicityFilterAction:   "none",
		NsfwTextFilterAction:   "none",
		HateSpeechFilterAction: "none",
	}, true, nil)

	createdMsg := newTestMessage(chatID, senderID)
	createdMsg.Content = content
	mockMsgRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			msg := args.Get(1).(*domain.Message)
			msg.ID = createdMsg.ID
		}).Return(nil)
	mockStatusRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.Anything).Return(&filev1.GetFilesByOwnerResponse{}, nil)
	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		content,
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, createdMsg.ID, result.ID)

	mockServersClient.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockModerationClient.AssertNotCalled(t, "ModerateText")
}

func TestMessageService_SendMessage_WithModeration_Blocked(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)
	mockServersClient := grpcclient_mocks.NewMockServersClient(t)
	mockModerationClient := grpcclient_mocks.NewMockModerationClient(t)
	mockCache := rediscache_mocks.NewMockModerationConfigCache(t)

	svc := service.NewMessageService(
		mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient,
		mockServersClient, mockModerationClient, nil, mockCache,
	)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.New()
	content := "bad word"

	mockServersClient.On("IsMember", mock.Anything, serverID.String(), senderID.String()).Return(true, nil)

	mockCache.On("Get", mock.Anything, serverID.String()).Return(&domain.ModerationConfig{
		ServerID:              serverID.String(),
		ProfanityFilterAction: "delete",
	}, true, nil)

	mockModerationClient.On("ModerateText", mock.Anything, content, senderID.String(), serverID.String(), mock.Anything).
		Return(&domain.ModerationResult{
			Allowed: false,
			Violations: []domain.Violation{
				{Type: "profanity", Message: "detected", Score: 0.9},
			},
		}, nil)

	mockMsgRepo.AssertNotCalled(t, "Create")

	result, err := svc.SendMessage(
		context.Background(),
		chatID, senderID, serverID, content, domain.ContentTypeText, nil, false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "blocked by moderation")

	mockServersClient.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockModerationClient.AssertExpectations(t)
}

func TestMessageService_SendMessage_WithModeration_FailOpen(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)
	mockServersClient := grpcclient_mocks.NewMockServersClient(t)
	mockModerationClient := grpcclient_mocks.NewMockModerationClient(t)
	mockCache := rediscache_mocks.NewMockModerationConfigCache(t)

	svc := service.NewMessageService(
		mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient,
		mockServersClient, mockModerationClient, nil, mockCache,
	)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.New()
	content := "test"

	mockServersClient.On("IsMember", mock.Anything, serverID.String(), senderID.String()).Return(true, nil)

	mockCache.On("Get", mock.Anything, serverID.String()).Return(nil, false, errors.New("redis down"))

	mockServersClient.On("GetModerationConfig", mock.Anything, serverID.String()).
		Return(nil, errors.New("servers down"))

	createdMsg := newTestMessage(chatID, senderID)
	mockMsgRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			msg := args.Get(1).(*domain.Message)
			msg.ID = createdMsg.ID
		}).Return(nil)
	mockStatusRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.Anything).Return(&filev1.GetFilesByOwnerResponse{}, nil)
	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.SendMessage(
		context.Background(),
		chatID, senderID, serverID, content, domain.ContentTypeText, nil, false,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	mockCache.AssertExpectations(t)
	mockServersClient.AssertExpectations(t)
	mockModerationClient.AssertNotCalled(t, "ModerateText")
}

func TestMessageService_SendMessage_WithModeration_AutoBan(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)
	mockServersClient := grpcclient_mocks.NewMockServersClient(t)
	mockModerationClient := grpcclient_mocks.NewMockModerationClient(t)
	mockCache := rediscache_mocks.NewMockModerationConfigCache(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, mockServersClient, mockModerationClient, mockCache)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.New()
	content := "very bad word"

	mockServersClient.On("IsMember", mock.Anything, serverID.String(), senderID.String()).Return(true, nil)
	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockCache.On("Get", mock.Anything, serverID.String()).
		Return(&domain.ModerationConfig{
			ProfanityFilterAction: "ban",
		}, true, nil)

	mockModerationClient.On("ModerateText", mock.Anything, content, senderID.String(), serverID.String(), mock.Anything).
		Return(&domain.ModerationResult{
			Allowed: false,
			Violations: []domain.Violation{
				{Type: "profanity", Message: "profanity detected", Score: 0.95},
			},
		}, nil)

	systemModID := domain.SystemModeratorID.String()
	mockServersClient.On("BanMember", mock.Anything, serverID.String(), senderID.String(), systemModID).Return(nil)

	mockMsgRepo.AssertNotCalled(t, "Create")

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		content,
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "blocked by moderation")

	mockServersClient.AssertExpectations(t)
	mockModerationClient.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestMessageService_SendMessage_WithModeration_AutoMute(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)
	mockServersClient := grpcclient_mocks.NewMockServersClient(t)
	mockModerationClient := grpcclient_mocks.NewMockModerationClient(t)
	mockCache := rediscache_mocks.NewMockModerationConfigCache(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, mockServersClient, mockModerationClient, mockCache)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.New()
	content := "toxic message"

	mockServersClient.On("IsMember", mock.Anything, serverID.String(), senderID.String()).Return(true, nil)
	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockCache.On("Get", mock.Anything, serverID.String()).
		Return(&domain.ModerationConfig{
			ToxicityFilterAction: "mute",
		}, true, nil)

	mockModerationClient.On("ModerateText", mock.Anything, content, senderID.String(), serverID.String(), mock.Anything).
		Return(&domain.ModerationResult{
			Allowed: false,
			Violations: []domain.Violation{
				{Type: "toxicity", Message: "toxicity detected", Score: 0.85},
			},
		}, nil)

	systemModID := domain.SystemModeratorID.String()
	mockServersClient.On("MuteMember", mock.Anything, serverID.String(), senderID.String(), systemModID, 24*time.Hour).Return(nil)

	mockMsgRepo.AssertNotCalled(t, "Create")

	result, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		content,
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "blocked by moderation")

	mockServersClient.AssertExpectations(t)
	mockModerationClient.AssertExpectations(t)
}

func TestMessageService_EditMessage_Success(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	senderID := uuid.New()
	chatID := uuid.New()
	newContent := "Updated content"

	originalMsg := newTestMessage(chatID, senderID)
	originalMsg.ID = messageID

	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(originalMsg, nil)

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockMsgRepo.On("Update", mock.Anything, messageID, newContent).Return(nil)

	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.Anything).
		Return(&filev1.GetFilesByOwnerResponse{Files: []*filev1.FileInfo{}}, nil)

	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := svc.EditMessage(context.Background(), messageID, senderID, newContent)

	assert.NoError(t, err)

	mockMsgRepo.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_EditMessage_NotMember(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	editorID := uuid.New()
	chatID := uuid.New()

	originalMsg := newTestMessage(chatID, editorID)
	originalMsg.ID = messageID

	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(originalMsg, nil)

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), editorID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: false}, nil)

	err := svc.EditMessage(context.Background(), messageID, editorID, "new content")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)

	mockMsgRepo.AssertNotCalled(t, "Update")
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_EditMessage_PermissionDenied(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	editorID := uuid.New()
	ownerID := uuid.New()
	chatID := uuid.New()

	originalMsg := newTestMessage(chatID, ownerID)
	originalMsg.ID = messageID

	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(originalMsg, nil)
	err := svc.EditMessage(context.Background(), messageID, editorID, "new content")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	mockMsgRepo.AssertNotCalled(t, "Update")
	mockMsgRepo.AssertExpectations(t)
}

func TestMessageService_EditMessage_NotFound(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()

	mockMsgRepo.On("GetByID", mock.Anything, messageID).
		Return(nil, repository.ErrNotFound)

	err := svc.EditMessage(context.Background(), messageID, uuid.New(), "new content")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get message")

	mockMsgRepo.AssertNotCalled(t, "Update")
	mockChatClient.AssertNotCalled(t, "IsMember")
}

func TestMessageService_DeleteMessage_Success(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	senderID := uuid.New()
	chatID := uuid.New()

	originalMsg := newTestMessage(chatID, senderID)
	originalMsg.ID = messageID

	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(originalMsg, nil)

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockMsgRepo.On("Delete", mock.Anything, messageID).Return(nil)

	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.Anything).
		Return(&filev1.GetFilesByOwnerResponse{Files: []*filev1.FileInfo{}}, nil)

	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := svc.DeleteMessage(context.Background(), messageID, senderID)

	assert.NoError(t, err)

	mockMsgRepo.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_DeleteMessage_NotMember(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	deleterID := uuid.New()
	chatID := uuid.New()

	originalMsg := newTestMessage(chatID, deleterID)
	originalMsg.ID = messageID

	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(originalMsg, nil)

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), deleterID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: false}, nil)

	err := svc.DeleteMessage(context.Background(), messageID, deleterID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)

	mockMsgRepo.AssertNotCalled(t, "Delete")
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_UpdateMessageStatus_Success(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	userID := uuid.New()
	chatID := uuid.New()

	mockStatusRepo.On("GetByUserAndMessage", mock.Anything, userID, messageID).
		Return(nil, repository.ErrNotFound)

	mockStatusRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(s *repository.MessageStatus) bool {
		return s.Status == domain.StatusDelivered
	})).Return(nil)

	originalMsg := newTestMessage(chatID, userID)
	originalMsg.ID = messageID
	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(originalMsg, nil)

	mockRedis.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := svc.UpdateMessageStatus(
		context.Background(),
		messageID,
		userID,
		domain.StatusDelivered,
	)

	assert.NoError(t, err)

	mockStatusRepo.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestMessageService_UpdateMessageStatus_InvalidTransition(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	userID := uuid.New()

	mockStatusRepo.On("GetByUserAndMessage", mock.Anything, userID, messageID).
		Return(&repository.MessageStatus{
			MessageID: messageID,
			UserID:    userID,
			Status:    domain.StatusRead,
		}, nil)

	err := svc.UpdateMessageStatus(
		context.Background(),
		messageID,
		userID,
		domain.StatusDelivered,
	)

	assert.NoError(t, err)
	mockStatusRepo.AssertNotCalled(t, "Upsert")
	mockRedis.AssertNotCalled(t, "Publish")
}

func TestMessageService_UpdateMessageStatus_InvalidStatus(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	err := svc.UpdateMessageStatus(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"invalid_status",
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")

	mockStatusRepo.AssertNotCalled(t, "GetByUserAndMessage")
	mockStatusRepo.AssertNotCalled(t, "Upsert")
}

func TestMessageService_GetMessage_Success(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	expectedMsg := newTestMessage(uuid.New(), uuid.New())
	expectedMsg.ID = messageID

	mockMsgRepo.On("GetByID", mock.Anything, messageID).Return(expectedMsg, nil)

	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.Anything).
		Return(&filev1.GetFilesByOwnerResponse{Files: []*filev1.FileInfo{}}, nil)

	msg, attachments, err := svc.GetMessage(context.Background(), messageID)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, expectedMsg.ID, msg.ID)
	assert.Empty(t, attachments)

	mockMsgRepo.AssertExpectations(t)
	mockFileClient.AssertExpectations(t)
}

func TestMessageService_GetChatHistory_Pagination(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	expectedMsgs := []*domain.Message{
		newTestMessage(chatID, uuid.New()),
		newTestMessage(chatID, uuid.New()),
	}

	mockMsgRepo.On("GetByChat", mock.Anything, chatID, 10, 0).
		Return(expectedMsgs, nil)

	result, err := svc.GetChatHistory(context.Background(), chatID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	mockMsgRepo.AssertExpectations(t)
}

func TestMessageService_SendMessage_PublishesToRedis(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	chatID := uuid.New()
	senderID := uuid.New()
	serverID := uuid.Nil
	messageID := uuid.New()

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockMsgRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			msg := args.Get(1).(*domain.Message)
			msg.ID = messageID
		}).Return(nil)

	mockStatusRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	mockFileClient.On("GetFilesByOwner", mock.Anything, mock.Anything).
		Return(&filev1.GetFilesByOwnerResponse{Files: []*filev1.FileInfo{}}, nil)

	mockRedis.On("Publish",
		mock.Anything,
		mock.MatchedBy(func(channel string) bool {
			return channel == "msg:new_message:"+chatID.String()
		}),
		mock.Anything,
	).Return(nil)

	_, err := svc.SendMessage(
		context.Background(),
		chatID,
		senderID,
		serverID,
		"Test",
		domain.ContentTypeText,
		nil,
		false,
	)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
	mockChatClient.AssertExpectations(t)
}

func TestMessageService_getAttachmentPreviews_FileServiceError(t *testing.T) {
	t.Parallel()

	mockMsgRepo := repoMocks.NewMockMessageRepository(t)
	mockStatusRepo := repoMocks.NewMockStatusRepository(t)
	mockRedis := redisMocks.NewMockRedisClient()
	mockFileClient := fileMocks.NewMockFileClient(t)
	mockChatClient := chatMocks.NewMockChatClient(t)

	svc := newTestService(t, mockMsgRepo, mockStatusRepo, mockRedis, mockFileClient, mockChatClient, nil, nil, nil)

	messageID := uuid.New()
	chatID := uuid.New()
	senderID := uuid.New()

	mockChatClient.On("IsMember", mock.Anything, chatID.String(), senderID.String()).
		Return(&grpcclient.IsMemberResult{IsMember: true}, nil)

	mockMsgRepo.On("GetByID", mock.Anything, messageID).
		Return(&domain.Message{
			ID:            messageID,
			ChatID:        chatID,
			SenderID:      senderID,
			Content:       "Test",
			ContentType:   domain.ContentTypeText,
			HasAttachment: true,
			CreatedAt:     time.Now(),
		}, nil)

	mockFileClient.On("GetFilesByOwner",
		mock.Anything,
		mock.MatchedBy(func(req *filev1.GetFilesByOwnerRequest) bool {
			return req.OwnerId == messageID.String()
		}),
	).Return(nil, errors.New("file service unavailable"))

	msg, attachments, err := svc.GetMessage(context.Background(), messageID)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Empty(t, attachments)

	mockMsgRepo.AssertExpectations(t)
	mockFileClient.AssertExpectations(t)
}
