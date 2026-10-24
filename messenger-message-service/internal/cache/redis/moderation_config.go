package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	redislib "github.com/go-redis/redis/v8"
)

const (
	ModerationConfigKeyPattern = "srv:moderation:config:%s"
	ModerationConfigTTL        = 10 * time.Minute
)

type ModerationConfigCacheInterface interface {
	Get(ctx context.Context, serverID string) (*domain.ModerationConfig, bool, error)
	Set(ctx context.Context, serverID string, cfg *domain.ModerationConfig) error
}

type ModerationConfigCache struct {
	client *redislib.Client
}

var _ ModerationConfigCacheInterface = (*ModerationConfigCache)(nil)

func NewModerationConfigCache(client *redislib.Client) *ModerationConfigCache {
	return &ModerationConfigCache{
		client: client,
	}
}

func (c *ModerationConfigCache) Get(ctx context.Context, serverID string) (*domain.ModerationConfig, bool, error) {
	if serverID == "" {
		return nil, false, nil
	}

	key := fmt.Sprintf(ModerationConfigKeyPattern, serverID)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redislib.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get moderation config from cache: %w", err)
	}

	var cfg domain.ModerationConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, false, nil
	}

	return &cfg, true, nil
}

func (c *ModerationConfigCache) Set(ctx context.Context, serverID string, cfg *domain.ModerationConfig) error {
	if serverID == "" || cfg == nil {
		return nil
	}

	key := fmt.Sprintf(ModerationConfigKeyPattern, serverID)

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal moderation config: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ModerationConfigTTL).Err(); err != nil {
		return fmt.Errorf("failed to set moderation config in cache: %w", err)
	}

	return nil
}
