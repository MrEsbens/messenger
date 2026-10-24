package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/domain"
	"github.com/MrEsbens/messenger-chats-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockInviteRepository struct {
	mock.Mock
}

var _ repository.InviteRepository = (*MockInviteRepository)(nil)

func NewMockInviteRepository(t *testing.T) *MockInviteRepository {
	return &MockInviteRepository{}
}

func (m *MockInviteRepository) Create(ctx context.Context, invite *domain.ChatInvite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockInviteRepository) GetByToken(ctx context.Context, token string) (*domain.ChatInvite, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatInvite), args.Error(1)
}

func (m *MockInviteRepository) GetByChat(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.ChatInvite, error) {
	args := m.Called(ctx, chatID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ChatInvite), args.Error(1)
}

func (m *MockInviteRepository) IncrementUses(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInviteRepository) Exists(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}
