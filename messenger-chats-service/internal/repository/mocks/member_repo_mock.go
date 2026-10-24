package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMemberRepository struct {
	mock.Mock
}

var _ repository.MemberRepository = (*MockMemberRepository)(nil)

func NewMockMemberRepository(t *testing.T) *MockMemberRepository {
	return &MockMemberRepository{}
}

func (m *MockMemberRepository) Create(ctx context.Context, member *domain.ChatMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChatMember, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatMember), args.Error(1)
}

func (m *MockMemberRepository) GetByChatAndUser(ctx context.Context, chatID, userID uuid.UUID) (*domain.ChatMember, error) {
	args := m.Called(ctx, chatID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatMember), args.Error(1)
}

func (m *MockMemberRepository) GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.ChatMember, error) {
	args := m.Called(ctx, chatID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ChatMember), args.Error(1)
}

func (m *MockMemberRepository) GetByUser(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*domain.ChatMember, error) {
	args := m.Called(ctx, userID, activeOnly, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ChatMember), args.Error(1)
}

func (m *MockMemberRepository) UpdateRole(ctx context.Context, id uuid.UUID, role domain.MembershipRole) error {
	args := m.Called(ctx, id, role)
	return args.Error(0)
}

func (m *MockMemberRepository) UpdateNotificationMode(ctx context.Context, id uuid.UUID, mode domain.NotificationMode) error {
	args := m.Called(ctx, id, mode)
	return args.Error(0)
}

func (m *MockMemberRepository) UpdateLastRead(ctx context.Context, id uuid.UUID, messageID uuid.UUID) error {
	args := m.Called(ctx, id, messageID)
	return args.Error(0)
}

func (m *MockMemberRepository) Leave(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemberRepository) Exists(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, chatID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMemberRepository) CountByChat(ctx context.Context, chatID uuid.UUID) (int, error) {
	args := m.Called(ctx, chatID)
	return args.Int(0), args.Error(1)
}
