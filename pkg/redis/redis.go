package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/richxcame/ride-hailing/pkg/config"
)

// Client wraps the Redis client
type Client struct {
	*redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// SetWithExpiration sets a key-value pair with expiration
func (c *Client) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.Set(ctx, key, value, expiration).Err()
}

// GetString gets a string value by key
func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	return c.Get(ctx, key).Result()
}

// Delete deletes a key
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Close closes the Redis client
func (c *Client) Close() error {
	return c.Client.Close()
}
