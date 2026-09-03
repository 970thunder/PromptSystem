package api

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"promptos-backend/internal/config"
)

type testRateLimitCache struct {
	mu       sync.Mutex
	counts   map[string]int64
	failPing bool
}

func (c *testRateLimitCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (c *testRateLimitCache) Get(context.Context, string) (string, error)           { return "", nil }
func (c *testRateLimitCache) Exists(context.Context, string) (bool, error)          { return false, nil }
func (c *testRateLimitCache) GetAndDelete(context.Context, string) (string, error)  { return "", nil }
func (c *testRateLimitCache) Delete(context.Context, string) error                  { return nil }
func (c *testRateLimitCache) Close() error                                          { return nil }
func (c *testRateLimitCache) Ping(context.Context) error {
	if c.failPing {
		return errors.New("redis unavailable")
	}
	return nil
}
func (c *testRateLimitCache) Increment(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if window <= 0 {
		return 0, 0, errors.New("invalid window")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[key]++
	return c.counts[key], window, nil
}

func TestRateLimitUsesIndependentActionAndDimensionBuckets(t *testing.T) {
	c := &testRateLimitCache{counts: map[string]int64{}}
	s := &server{cache: c, config: configForRateLimitTest("production")}
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		allowed, _, err := s.allowRateLimit(ctx, "login", "ip:1", 2, time.Minute)
		if err != nil || !allowed {
			t.Fatalf("login attempt %d = allowed %v err %v", i, allowed, err)
		}
	}
	allowed, _, err := s.allowRateLimit(ctx, "login", "ip:1", 2, time.Minute)
	if err != nil || allowed {
		t.Fatalf("third login attempt = allowed %v err %v, want blocked", allowed, err)
	}
	if allowed, _, err := s.allowRateLimit(ctx, "register", "ip:1", 1, time.Minute); err != nil || !allowed {
		t.Fatalf("different action shared bucket: allowed %v err %v", allowed, err)
	}
	if allowed, _, err := s.allowRateLimit(ctx, "login", "ip:2", 1, time.Minute); err != nil || !allowed {
		t.Fatalf("different dimension shared bucket: allowed %v err %v", allowed, err)
	}
}

func TestRateLimitFailsClosedOutsideDevelopmentWhenRedisUnavailable(t *testing.T) {
	// Increment is the operation that fails in production; emulate that in a
	// separate server so allowRateLimit exercises the fail-closed branch.
	broken := &server{cache: &failingRateLimitCache{}, config: configForRateLimitTest("production")}
	allowed, _, err := broken.allowRateLimit(context.Background(), "upload", "ip:1", 1, time.Minute)
	if allowed || !errors.Is(err, errRateLimitUnavailable) {
		t.Fatalf("production Redis failure = allowed %v err %v, want fail closed", allowed, err)
	}
}

type failingRateLimitCache struct{ testRateLimitCache }

func (*failingRateLimitCache) Increment(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 0, 0, errors.New("redis unavailable")
}

func configForRateLimitTest(env string) config.Config {
	return config.Config{AppEnv: env}
}
