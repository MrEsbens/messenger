package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	"github.com/stretchr/testify/mock"
)

type MockModerationClient struct {
	mock.Mock
}

var _ grpcclient.ModerationClientInterface = (*MockModerationClient)(nil)

func NewMockModerationClient(t *testing.T) *MockModerationClient {
	mock := &MockModerationClient{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

func (m *MockModerationClient) ModerateText(
	ctx context.Context,
	text, userID, serverID string,
	filters *domain.ModerationFilters,
) (*domain.ModerationResult, error) {
	args := m.Called(ctx, text, userID, serverID, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ModerationResult), args.Error(1)
}

func (m *MockModerationClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
