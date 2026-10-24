package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	"github.com/stretchr/testify/mock"
)

type MockServersClient struct {
	mock.Mock
}

var _ grpcclient.ServersClientInterface = (*MockServersClient)(nil)

func NewMockServersClient(t *testing.T) *MockServersClient {
	mock := &MockServersClient{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

func (m *MockServersClient) GetModerationConfig(ctx context.Context, serverID string) (*domain.ModerationConfig, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ModerationConfig), args.Error(1)
}

func (m *MockServersClient) IsMember(ctx context.Context, serverID, userID string) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockServersClient) BanMember(ctx context.Context, serverID, userID, executorID string) error {
	args := m.Called(ctx, serverID, userID, executorID)
	return args.Error(0)
}

func (m *MockServersClient) MuteMember(ctx context.Context, serverID, userID, executorID string, duration time.Duration) error {
	args := m.Called(ctx, serverID, userID, executorID, duration)
	return args.Error(0)
}

func (m *MockServersClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
