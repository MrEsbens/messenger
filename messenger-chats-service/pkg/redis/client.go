package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisClientInterface interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
	PSubscribe(ctx context.Context, channels ...string) *redis.PubSub
	Close() error
}

type Client struct {
	*redis.Client
	ctx context.Context
}

var _ RedisClientInterface = (*Client)(nil)

func NewClient(ctx context.Context, url string) *Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("❌ Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	log.Printf("✅ Connected to Redis at %s", opt.Addr)

	return &Client{
		Client: client,
		ctx:    ctx,
	}
}

func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := c.Client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish to channel %s: %w", channel, err)
	}

	return nil
}

func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.Client.Subscribe(ctx, channels...)
}

func (c *Client) PSubscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.Client.PSubscribe(ctx, channels...)
}

func (c *Client) Close() error {
	if c.Client != nil {
		log.Println("🔌 Closing Redis connection...")
		return c.Client.Close()
	}
	return nil
}
