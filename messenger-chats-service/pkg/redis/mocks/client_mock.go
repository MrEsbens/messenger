package mocks

import (
	"context"

	"github.com/MrEsbens/messenger-message-service/pkg/redis"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/mock"
)

type MockRedisClient struct {
	mock.Mock
}

var _ redis.RedisClientInterface = (*MockRedisClient)(nil)

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{}
}

func (m *MockRedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}

func (m *MockRedisClient) Subscribe(ctx context.Context, channels ...string) *redisv8.PubSub {
	args := m.Called(ctx, channels)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*redisv8.PubSub)
}

func (m *MockRedisClient) PSubscribe(ctx context.Context, channels ...string) *redisv8.PubSub {
	args := m.Called(ctx, channels)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*redisv8.PubSub)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRedisClient) Raw() *redisv8.Client {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*redisv8.Client)
}
