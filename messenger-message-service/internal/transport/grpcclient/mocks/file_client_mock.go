package mocks

import (
	"context"
	"testing"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"github.com/MrEsbens/messenger-message-service/internal/transport/grpcclient"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MockFileClient struct {
	mock.Mock
}

var _ grpcclient.FileClientInterface = (*MockFileClient)(nil)

func NewMockFileClient(t *testing.T) *MockFileClient {
	m := &MockFileClient{}
	return m
}

func (m *MockFileClient) GetUploadURL(ctx context.Context, req *filev1.GetUploadURLRequest) (*filev1.GetUploadURLResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetUploadURLResponse), args.Error(1)
}

func (m *MockFileClient) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*emptypb.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *MockFileClient) GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetDownloadURLResponse), args.Error(1)
}

func (m *MockFileClient) GetFileByOwner(ctx context.Context, req *filev1.GetFileByOwnerRequest) (*filev1.GetFileByOwnerResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetFileByOwnerResponse), args.Error(1)
}

func (m *MockFileClient) GetFilesByOwner(ctx context.Context, req *filev1.GetFilesByOwnerRequest) (*filev1.GetFilesByOwnerResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetFilesByOwnerResponse), args.Error(1)
}

func (m *MockFileClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
