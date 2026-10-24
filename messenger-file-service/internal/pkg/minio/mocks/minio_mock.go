package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/MrEsbens/messenger-file-service/internal/pkg/minio"
	"github.com/MrEsbens/messenger-file-service/internal/pkg/storage"
	MinioSDK "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

type MockMinIOClient struct {
	mock.Mock
}

var _ minio.MinIOClient = (*MockMinIOClient)(nil)

func NewMockMinIOClient(t *testing.T) *MockMinIOClient {
	m := &MockMinIOClient{}
	m.Test(t)
	return m
}

func (m *MockMinIOClient) GetPresignedPutURL(ctx context.Context, bucketType storage.BucketType, objectKey string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucketType, objectKey, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockMinIOClient) GetPresignedGetURL(ctx context.Context, bucketType storage.BucketType, objectKey string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucketType, objectKey, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockMinIOClient) GetPublicURL(bucketType storage.BucketType, objectKey string) string {
	args := m.Called(bucketType, objectKey)
	return args.String(0)
}

func (m *MockMinIOClient) StatObject(ctx context.Context, bucketName, objectKey string) error {
	args := m.Called(ctx, bucketName, objectKey)
	return args.Error(0)
}

func (m *MockMinIOClient) RemoveObject(ctx context.Context, bucket, key string, opts MinioSDK.RemoveObjectOptions) error {
	args := m.Called(ctx, bucket, key, opts)
	return args.Error(0)
}

func (m *MockMinIOClient) Endpoint() string {
	args := m.Called()
	return args.String(0)
}
