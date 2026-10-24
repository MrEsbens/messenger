package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-message-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockStatusRepository struct {
	mock.Mock
}

var _ repository.StatusRepository = (*MockStatusRepository)(nil)

func NewMockStatusRepository(t *testing.T) *MockStatusRepository {
	return &MockStatusRepository{}
}

func (m *MockStatusRepository) Upsert(ctx context.Context, status *repository.MessageStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockStatusRepository) GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*repository.MessageStatus, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.MessageStatus), args.Error(1)
}

func (m *MockStatusRepository) GetByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) (*repository.MessageStatus, error) {
	args := m.Called(ctx, userID, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.MessageStatus), args.Error(1)
}

func (m *MockStatusRepository) BulkUpdateByMessage(ctx context.Context, messageID uuid.UUID, status string, userIDs []uuid.UUID) error {
	args := m.Called(ctx, messageID, status, userIDs)
	return args.Error(0)
}
