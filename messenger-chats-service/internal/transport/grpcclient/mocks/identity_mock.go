package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/transport/grpcclient"
	identityv1 "github.com/MrEsbens/messenger-identity/api/identity/v1"
	"github.com/stretchr/testify/mock"
)

type MockIdentityClient struct {
	mock.Mock
}

var _ grpcclient.IdentityClientInterface = (*MockIdentityClient)(nil)

func NewMockIdentityClient(t *testing.T) *MockIdentityClient {
	return &MockIdentityClient{}
}

func (m *MockIdentityClient) UserExists(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockIdentityClient) GetUser(ctx context.Context, userID string) (*identityv1.UserDTO, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identityv1.UserDTO), args.Error(1)
}

func (m *MockIdentityClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
