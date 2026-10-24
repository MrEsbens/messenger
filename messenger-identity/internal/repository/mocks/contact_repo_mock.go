package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-identity/internal/domain"
	"github.com/MrEsbens/messenger-identity/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockContactRepository struct {
	mock.Mock
}

var _ repository.ContactRepository = (*MockContactRepository)(nil)

func NewMockContactRepository(t *testing.T) *MockContactRepository {
	m := &MockContactRepository{}
	m.Test(t)
	return m
}

func (m *MockContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockContactRepository) GetByPair(ctx context.Context, userID, contactID uuid.UUID) (*domain.Contact, error) {
	args := m.Called(ctx, userID, contactID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Contact), args.Error(1)
}

func (m *MockContactRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ContactStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockContactRepository) ListAccepted(ctx context.Context, userID uuid.UUID) ([]*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockContactRepository) ListIncomingPending(ctx context.Context, userID uuid.UUID) ([]*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockContactRepository) SoftDelete(ctx context.Context, userID, contactID uuid.UUID) error {
	args := m.Called(ctx, userID, contactID)
	return args.Error(0)
}
