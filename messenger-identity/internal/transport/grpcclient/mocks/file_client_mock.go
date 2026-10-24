package mocks

import (
	"context"
	"testing"

	filev1 "github.com/MrEsbens/messenger-file-service/api/file/v1"
	"github.com/MrEsbens/messenger-identity/internal/transport/grpcclient"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MockFileClient реализует интерфейс grpcclient.FileClientInterface
type MockFileClient struct {
	mock.Mock
}

// Проверяем на этапе компиляции, что мок реализует интерфейс
var _ grpcclient.FileClientInterface = (*MockFileClient)(nil)

// NewMockFileClient создаёт новый мок и связывает его с тестом
func NewMockFileClient(t *testing.T) *MockFileClient {
	m := &MockFileClient{}
	m.Test(t)
	return m
}

// GetUploadURL мокирует вызов к file-service
func (m *MockFileClient) GetUploadURL(ctx context.Context, req *filev1.GetUploadURLRequest) (*filev1.GetUploadURLResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetUploadURLResponse), args.Error(1)
}

// ConfirmUpload мокирует подтверждение загрузки
func (m *MockFileClient) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*emptypb.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

// GetDownloadURL мокирует получение ссылки на скачивание
func (m *MockFileClient) GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetDownloadURLResponse), args.Error(1)
}

// GetFileByOwner мокирует поиск файла по владельцу
func (m *MockFileClient) GetFileByOwner(ctx context.Context, req *filev1.GetFileByOwnerRequest) (*filev1.GetFileByOwnerResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*filev1.GetFileByOwnerResponse), args.Error(1)
}

// Close мокирует закрытие соединения
func (m *MockFileClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
