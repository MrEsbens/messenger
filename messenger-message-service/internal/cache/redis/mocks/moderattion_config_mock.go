package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-message-service/internal/cache/redis"
	"github.com/MrEsbens/messenger-message-service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockModerationConfigCache struct {
	mock.Mock
}

// Проверяем что мок реализует интерфейс
var _ redis.ModerationConfigCacheInterface = (*MockModerationConfigCache)(nil)

func NewMockModerationConfigCache(t *testing.T) *MockModerationConfigCache {
	mock := &MockModerationConfigCache{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

func (m *MockModerationConfigCache) Get(ctx context.Context, serverID string) (*domain.ModerationConfig, bool, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*domain.ModerationConfig), args.Bool(1), args.Error(2)
}

func (m *MockModerationConfigCache) Set(ctx context.Context, serverID string, cfg *domain.ModerationConfig) error {
	args := m.Called(ctx, serverID, cfg)
	return args.Error(0)
}
