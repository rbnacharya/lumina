package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/lumina/gateway/internal/models"
)

const (
	keyConfigPrefix  = "key_config:"
	rateLimitPrefix  = "rate_limit:"
	keyConfigTTL     = 1 * time.Hour
	rateLimitWindow  = 1 * time.Minute
)

// Cache wraps the Redis client
type Cache struct {
	client *redis.Client
}

// New creates a new Redis cache connection
func New(redisURL string) (*Cache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// GetKeyConfig retrieves a key configuration from cache
func (c *Cache) GetKeyConfig(ctx context.Context, keyHash string) (*models.KeyConfig, error) {
	key := keyConfigPrefix + keyHash
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key config: %w", err)
	}

	var config models.KeyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key config: %w", err)
	}

	return &config, nil
}

// SetKeyConfig stores a key configuration in cache
func (c *Cache) SetKeyConfig(ctx context.Context, keyHash string, config *models.KeyConfig) error {
	key := keyConfigPrefix + keyHash
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal key config: %w", err)
	}

	if err := c.client.Set(ctx, key, data, keyConfigTTL).Err(); err != nil {
		return fmt.Errorf("failed to set key config: %w", err)
	}

	return nil
}

// DeleteKeyConfig removes a key configuration from cache
func (c *Cache) DeleteKeyConfig(ctx context.Context, keyHash string) error {
	key := keyConfigPrefix + keyHash
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key config: %w", err)
	}
	return nil
}

// IncrementRateLimit increments the rate limit counter and returns the current count
func (c *Cache) IncrementRateLimit(ctx context.Context, keyHash string) (int64, error) {
	key := rateLimitPrefix + keyHash

	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rateLimitWindow)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	return incr.Val(), nil
}

// GetRateLimitCount returns the current rate limit count
func (c *Cache) GetRateLimitCount(ctx context.Context, keyHash string) (int64, error) {
	key := rateLimitPrefix + keyHash
	count, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit count: %w", err)
	}
	return count, nil
}
