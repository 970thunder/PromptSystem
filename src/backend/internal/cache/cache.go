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
	// GetAndDelete atomically returns the value and removes the key in one
	// operation, guaranteeing single consumption of one-time codes.
	GetAndDelete(ctx context.Context, key string) (string, error)
	// Increment atomically increments a counter and gives it a TTL on its first
	// use. retryAfter is the remaining TTL for the counter.
	Increment(ctx context.Context, key string, window time.Duration) (count int64, retryAfter time.Duration, err error)
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

func (c *redisCache) GetAndDelete(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("cache not available")
	}
	value, err := c.client.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

func (c *redisCache) Increment(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if c == nil || c.client == nil {
		return 0, 0, errors.New("cache not available")
	}
	if window <= 0 {
		return 0, 0, errors.New("cache counter window must be positive")
	}

	// INCR and EXPIRE must be one operation: separate commands allow concurrent
	// requests to create a counter without a TTL or to race while setting it.
	result, err := c.client.Eval(ctx, `
		local count = redis.call('INCR', KEYS[1])
		if count == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
		local ttl = redis.call('TTL', KEYS[1])
		return {count, ttl}
	`, []string{key}, int64(window/time.Second)).Result()
	if err != nil {
		return 0, 0, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return 0, 0, errors.New("invalid rate limit response")
	}
	count, ok := values[0].(int64)
	if !ok {
		return 0, 0, errors.New("invalid rate limit count")
	}
	ttlSeconds, ok := values[1].(int64)
	if !ok {
		return 0, 0, errors.New("invalid rate limit ttl")
	}
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}
	return count, time.Duration(ttlSeconds) * time.Second, nil
}

// IncrementBy atomically increments a quota counter by delta and sets its TTL
// on first use. It is intentionally an optional capability so lightweight
// test/development cache implementations can continue using the Cache API.
func (c *redisCache) IncrementBy(ctx context.Context, key string, delta int64, window time.Duration) (int64, time.Duration, error) {
	if c == nil || c.client == nil {
		return 0, 0, errors.New("cache not available")
	}
	if delta <= 0 || window <= 0 {
		return 0, 0, errors.New("cache quota increment must be positive")
	}
	result, err := c.client.Eval(ctx, `
		local count = redis.call('INCRBY', KEYS[1], ARGV[1])
		if count == tonumber(ARGV[1]) then redis.call('EXPIRE', KEYS[1], ARGV[2]) end
		local ttl = redis.call('TTL', KEYS[1])
		return {count, ttl}
	`, []string{key}, delta, int64(window/time.Second)).Result()
	if err != nil {
		return 0, 0, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return 0, 0, errors.New("invalid quota response")
	}
	count, ok := values[0].(int64)
	if !ok {
		return 0, 0, errors.New("invalid quota count")
	}
	ttlSeconds, ok := values[1].(int64)
	if !ok {
		return 0, 0, errors.New("invalid quota ttl")
	}
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}
	return count, time.Duration(ttlSeconds) * time.Second, nil
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
