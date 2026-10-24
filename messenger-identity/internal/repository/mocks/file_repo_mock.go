package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/MrEsbens/messenger-identity/internal/repository"
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

func (m *MockFileRepository) GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) (*domain.File, error) {
	args := m.Called(ctx, ownerType, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.File), args.Error(1)
}

func (m *MockFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) ConfirmUpload(ctx context.Context, fileID uuid.UUID) error {
	args := m.Called(ctx, fileID)
	return args.Error(0)
}
