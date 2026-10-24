package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockChatRepository struct {
	mock.Mock
}

var _ repository.ChatRepository = (*MockChatRepository)(nil)

func NewMockChatRepository(t *testing.T) *MockChatRepository {
	return &MockChatRepository{}
}

func (m *MockChatRepository) Create(ctx context.Context, chat *domain.Chat) error {
	args := m.Called(ctx, chat)
	return args.Error(0)
}

func (m *MockChatRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Chat), args.Error(1)
}

func (m *MockChatRepository) GetByType(ctx context.Context, chatType domain.ChatType, limit, offset int) ([]*domain.Chat, error) {
	args := m.Called(ctx, chatType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Chat), args.Error(1)
}

func (m *MockChatRepository) GetByCreatedBy(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Chat, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Chat), args.Error(1)
}

func (m *MockChatRepository) Update(ctx context.Context, chat *domain.Chat) error {
	args := m.Called(ctx, chat)
	return args.Error(0)
}

func (m *MockChatRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
