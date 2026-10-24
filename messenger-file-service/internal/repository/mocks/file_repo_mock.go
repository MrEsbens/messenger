package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/domain"
	"github.com/MrEsbens/messenger-file-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockFileRepository struct {
	mock.Mock
}

var _ repository.FileRepository = (*MockFileRepository)(nil)

func NewMockFileRepository(t *testing.T) *MockFileRepository {
	m := &MockFileRepository{}
	m.Test(t)
	return m
}

func (m *MockFileRepository) Create(ctx context.Context, file *domain.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.File), args.Error(1)
}

func (m *MockFileRepository) GetActiveByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.File), args.Error(1)
}

func (m *MockFileRepository) GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) (*domain.File, error) {
	args := m.Called(ctx, ownerType, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.File), args.Error(1)
}

func (m *MockFileRepository) GetActiveByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) (*domain.File, error) {
	args := m.Called(ctx, ownerID, ownerType, bucketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.File), args.Error(1)
}

func (m *MockFileRepository) GetAllActiveByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) ([]*domain.File, error) {
	args := m.Called(ctx, ownerID, ownerType, bucketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.File), args.Error(1)
}

func (m *MockFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) ConfirmUpload(ctx context.Context, fileID uuid.UUID) error {
	args := m.Called(ctx, fileID)
	return args.Error(0)
}

func (m *MockFileRepository) GetPendingOlderThan(ctx context.Context, olderThan time.Time) ([]uuid.UUID, error) {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockFileRepository) GetByOwnerAndType(ctx context.Context, ownerID uuid.UUID, ownerType, bucketType string) (*domain.File, error) {
	args := m.Called(ctx, ownerID, ownerType, bucketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.File), args.Error(1)
}
