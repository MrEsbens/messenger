package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockChatConfigRepository struct {
	mock.Mock
}

var _ repository.ChatConfigRepository = (*MockChatConfigRepository)(nil)

func NewMockChatConfigRepository(t *testing.T) *MockChatConfigRepository {
	return &MockChatConfigRepository{}
}

func (m *MockChatConfigRepository) Create(ctx context.Context, config *domain.ChatConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockChatConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChatConfig, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatConfig), args.Error(1)
}

func (m *MockChatConfigRepository) Update(ctx context.Context, config *domain.ChatConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockChatConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatConfigRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
