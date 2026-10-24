package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/repository"
	"github.com/MrEsbens/messenger-message-service/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMessageService struct {
	mock.Mock
}

var _ service.MessageServiceInterface = (*MockMessageService)(nil)

func NewMockMessageService(t *testing.T) *MockMessageService {
	mock := &MockMessageService{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

func (m *MockMessageService) SendMessage(
	ctx context.Context,
	chatID, senderID, serverID uuid.UUID,
	content, contentType string,
	replyToID *uuid.UUID,
	hasAttachment bool,
) (*domain.Message, error) {
	args := m.Called(ctx, chatID, senderID, serverID, content, contentType, replyToID, hasAttachment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Message), args.Error(1)
}

func (m *MockMessageService) GetMessage(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Message, []*service.AttachmentPreview, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*domain.Message), args.Get(1).([]*service.AttachmentPreview), args.Error(2)
}

func (m *MockMessageService) GetChatHistory(
	ctx context.Context,
	chatID uuid.UUID,
	limit, offset int,
) ([]*domain.Message, error) {
	args := m.Called(ctx, chatID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Message), args.Error(1)
}

func (m *MockMessageService) GetChatHistoryWithAttachments(
	ctx context.Context,
	chatID uuid.UUID,
	limit, offset int,
) ([]*domain.Message, map[uuid.UUID][]*service.AttachmentPreview, error) {
	args := m.Called(ctx, chatID, limit, offset)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*domain.Message), args.Get(1).(map[uuid.UUID][]*service.AttachmentPreview), args.Error(2)
}

func (m *MockMessageService) GetChatHistoryAfter(
	ctx context.Context,
	chatID uuid.UUID,
	afterTime time.Time,
	limit int,
) ([]*domain.Message, error) {
	args := m.Called(ctx, chatID, afterTime, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Message), args.Error(1)
}

func (m *MockMessageService) EditMessage(
	ctx context.Context,
	messageID, editorID uuid.UUID,
	newContent string,
) error {
	args := m.Called(ctx, messageID, editorID, newContent)
	return args.Error(0)
}

func (m *MockMessageService) DeleteMessage(
	ctx context.Context,
	messageID, deleterID uuid.UUID,
) error {
	args := m.Called(ctx, messageID, deleterID)
	return args.Error(0)
}

func (m *MockMessageService) UpdateMessageStatus(
	ctx context.Context,
	messageID, userID uuid.UUID,
	newStatus string,
) error {
	args := m.Called(ctx, messageID, userID, newStatus)
	return args.Error(0)
}

func (m *MockMessageService) GetMessageStatuses(
	ctx context.Context,
	messageID uuid.UUID,
) ([]*repository.MessageStatus, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.MessageStatus), args.Error(1)
}

func (m *MockMessageService) MarkChatAsRead(
	ctx context.Context,
	chatID, userID uuid.UUID,
) error {
	args := m.Called(ctx, chatID, userID)
	return args.Error(0)
}
