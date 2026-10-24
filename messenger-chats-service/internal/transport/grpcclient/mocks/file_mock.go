package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-chats-service/internal/transport/grpcclient"
	"github.com/stretchr/testify/mock"
)

type MockFileClient struct {
	mock.Mock
}

var _ grpcclient.FileClientInterface = (*MockFileClient)(nil)

func NewMockFileClient(t *testing.T) *MockFileClient {
	return &MockFileClient{}
}

func (m *MockFileClient) GetFileByOwner(ctx context.Context, ownerID, ownerType, bucketType string) (*grpcclient.FileMetadata, error) {
	args := m.Called(ctx, ownerID, ownerType, bucketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcclient.FileMetadata), args.Error(1)
}

func (m *MockFileClient) GetFilesByOwner(ctx context.Context, ownerID, ownerType, bucketType string) ([]*grpcclient.FileMetadata, error) {
	args := m.Called(ctx, ownerID, ownerType, bucketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*grpcclient.FileMetadata), args.Error(1)
}

func (m *MockFileClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
