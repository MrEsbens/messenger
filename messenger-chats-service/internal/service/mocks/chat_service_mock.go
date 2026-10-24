package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockChatService struct {
	mock.Mock
}

var _ service.ChatServiceInterface = (*MockChatService)(nil)

func NewMockChatService(t *testing.T) *MockChatService {
	return &MockChatService{}
}

func (m *MockChatService) CreateChat(ctx context.Context, req service.CreateChatRequest) (*service.ChatResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ChatResponse), args.Error(1)
}

func (m *MockChatService) GetChat(ctx context.Context, chatID, requesterID uuid.UUID) (*service.ChatResponse, error) {
	args := m.Called(ctx, chatID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ChatResponse), args.Error(1)
}

func (m *MockChatService) UpdateChat(ctx context.Context, req service.UpdateChatRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockChatService) DeleteChat(ctx context.Context, chatID, deleterID uuid.UUID) error {
	args := m.Called(ctx, chatID, deleterID)
	return args.Error(0)
}

func (m *MockChatService) ListUserChats(ctx context.Context, userID uuid.UUID, filter service.ListChatsFilter) (*service.ChatListResponse, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ChatListResponse), args.Error(1)
}

func (m *MockChatService) AddMember(ctx context.Context, chatID, userID, addedBy uuid.UUID, role domain.MembershipRole) error {
	args := m.Called(ctx, chatID, userID, addedBy, role)
	return args.Error(0)
}

func (m *MockChatService) RemoveMember(ctx context.Context, chatID, userID, removedBy uuid.UUID) error {
	args := m.Called(ctx, chatID, userID, removedBy)
	return args.Error(0)
}

func (m *MockChatService) UpdateMemberRole(ctx context.Context, chatID, targetUserID, updatedBy uuid.UUID, newRole domain.MembershipRole) error {
	args := m.Called(ctx, chatID, targetUserID, updatedBy, newRole)
	return args.Error(0)
}

func (m *MockChatService) ListChatMembers(ctx context.Context, chatID uuid.UUID, filter service.ListMembersFilter) ([]*domain.ChatMember, error) {
	args := m.Called(ctx, chatID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ChatMember), args.Error(1)
}

func (m *MockChatService) CreateInvite(ctx context.Context, chatID, createdBy uuid.UUID, opts service.InviteOptions) (*domain.ChatInvite, error) {
	args := m.Called(ctx, chatID, createdBy, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatInvite), args.Error(1)
}

func (m *MockChatService) JoinByInvite(ctx context.Context, token string, userID uuid.UUID) error {
	args := m.Called(ctx, token, userID)
	return args.Error(0)
}

func (m *MockChatService) IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, domain.MembershipRole, error) {
	args := m.Called(ctx, chatID, userID)
	return args.Bool(0), args.Get(1).(domain.MembershipRole), args.Error(2)
}

func (m *MockChatService) CreateChatConfig(ctx context.Context, config *domain.ChatConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockChatService) GetChatConfig(ctx context.Context, configID uuid.UUID) (*domain.ChatConfig, error) {
	args := m.Called(ctx, configID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatConfig), args.Error(1)
}

func (m *MockChatService) UpdateChatConfig(ctx context.Context, config *domain.ChatConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}
