package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	"github.com/stretchr/testify/mock"
)

type MockChatClient struct {
	mock.Mock
}

var _ grpcclient.ChatClientInterface = (*MockChatClient)(nil)

func NewMockChatClient(t *testing.T) *MockChatClient {
	return &MockChatClient{}
}

func (m *MockChatClient) IsMember(ctx context.Context, chatID, userID string) (*grpcclient.IsMemberResult, error) {
	args := m.Called(ctx, chatID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcclient.IsMemberResult), args.Error(1)
}

func (m *MockChatClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
