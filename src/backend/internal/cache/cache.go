package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"promptos-backend/internal/config"
)

// Cache is a small abstraction over Redis used by auth/session logic.
type Cache interface {
	// Set stores value with a TTL.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Get returns the stored value as a string.
	Get(ctx context.Context, key string) (string, error)
	// Exists reports whether key exists.
	Exists(ctx context.Context, key string) (bool, error)
	// Delete removes key.
	Delete(ctx context.Context, key string) error
	// Ping verifies connectivity.
	Ping(ctx context.Context) error
	// Close releases the client.
	Close() error
}

type redisCache struct {
	client *redis.Client
}

// New returns a Redis-backed Cache, or nil if Redis is not configured.
func New(cfg config.Config) Cache {
	addr := cfg.RedisHost + ":" + cfg.RedisPort
	if addr == ":" {
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.RedisPass,
		DB:           0,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
	return &redisCache{client: client}
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return errors.New("cache not available")
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("cache not available")
	}
	return c.client.Get(ctx, key).Result()
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	if c == nil || c.client == nil {
		return false, errors.New("cache not available")
	}
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return errors.New("cache not available")
	}
	return c.client.Del(ctx, key).Err()
}

func (c *redisCache) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("cache not available")
	}
	return c.client.Ping(ctx).Err()
}

func (c *redisCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}
