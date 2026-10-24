package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMessageRepository struct {
	mock.Mock
}

var _ repository.MessageRepository = (*MockMessageRepository)(nil)

func NewMockMessageRepository(t *testing.T) *MockMessageRepository {
	return &MockMessageRepository{}
}

func (m *MockMessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockMessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Message), args.Error(1)
}

func (m *MockMessageRepository) GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	args := m.Called(ctx, chatID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Message), args.Error(1)
}

func (m *MockMessageRepository) GetByChatAfter(ctx context.Context, chatID uuid.UUID, after time.Time, limit int) ([]*domain.Message, error) {
	args := m.Called(ctx, chatID, after, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Message), args.Error(1)
}

func (m *MockMessageRepository) Update(ctx context.Context, id uuid.UUID, newContent string) error {
	args := m.Called(ctx, id, newContent)
	return args.Error(0)
}

func (m *MockMessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMessageRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
