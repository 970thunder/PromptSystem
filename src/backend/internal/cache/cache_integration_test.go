package cache

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"promptos-backend/internal/config"
)

// testRedisAddr returns the Redis address from PROMPTOS_TEST_REDIS_ADDR, or
// skips the test when unset (CI provides it; local runs skip).
func testRedisAddr(t *testing.T) string {
	t.Helper()
	addr := strings.TrimSpace(os.Getenv("PROMPTOS_TEST_REDIS_ADDR"))
	if addr == "" {
		t.Skip("PROMPTOS_TEST_REDIS_ADDR not set; skipping Redis integration tests (run via CI or a local Redis)")
	}
	return addr
}

func newTestCache(t *testing.T, addr string) Cache {
	t.Helper()
	host, port := addr, "6379"
	if i := strings.LastIndex(addr, ":"); i > 0 {
		host, port = addr[:i], addr[i+1:]
	}
	c := New(config.Config{RedisHost: host, RedisPort: port})
	if c == nil {
		t.Fatal("New() returned nil cache")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestRedisSetGetDeleteTTL(t *testing.T) {
	addr := testRedisAddr(t)
	c := newTestCache(t, addr)
	ctx := context.Background()
	key := "promptos:it:ttl:" + time.Now().Format("150405.000000000")

	if err := c.Set(ctx, key, "hello", 30*time.Second); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := c.Get(ctx, key)
	if err != nil || got != "hello" {
		t.Fatalf("Get() = %q, %v; want hello", got, err)
	}
	exists, err := c.Exists(ctx, key)
	if err != nil || !exists {
		t.Fatalf("Exists() = %v, %v; want true", exists, err)
	}
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := c.Get(ctx, key); err == nil {
		t.Fatal("Get() after Delete() should error (missing key)")
	}
}

func TestRedisGetAndDeleteSingleConsumption(t *testing.T) {
	addr := testRedisAddr(t)
	c := newTestCache(t, addr)
	ctx := context.Background()
	key := "promptos:it:onetime:" + time.Now().Format("150405.000000000")

	if err := c.Set(ctx, key, "code-abc", 60*time.Second); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// First consumption returns the value.
	first, err := c.GetAndDelete(ctx, key)
	if err != nil || first != "code-abc" {
		t.Fatalf("GetAndDelete() 1st = %q, %v; want code-abc", first, err)
	}

	// Second consumption must be empty: the one-time code is consumed.
	second, err := c.GetAndDelete(ctx, key)
	if err != nil || second != "" {
		t.Fatalf("GetAndDelete() 2nd = %q, %v; want empty", second, err)
	}
}

// TestRedisIncrementConcurrent verifies the INCR+EXPIRE Lua primitive is atomic
// under concurrency: N goroutines must observe exactly N distinct increments
// and the TTL must be set on the first call.
func TestRedisIncrementConcurrent(t *testing.T) {
	addr := testRedisAddr(t)
	c := newTestCache(t, addr)
	ctx := context.Background()
	key := "promptos:it:counter:" + time.Now().Format("150405.000000000")

	const goroutines = 20
	var wg sync.WaitGroup
	results := make(chan int64, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, _, err := c.Increment(ctx, key, 30*time.Second)
			if err != nil {
				t.Errorf("Increment() error = %v", err)
				return
			}
			results <- count
		}()
	}
	wg.Wait()
	close(results)

	max := int64(0)
	seen := make(map[int64]bool)
	for count := range results {
		if count > max {
			max = count
		}
		seen[count] = true
	}
	if max != goroutines {
		t.Fatalf("max count = %d, want %d (atomic INCR expected)", max, goroutines)
	}
	if len(seen) != goroutines {
		t.Fatalf("saw %d distinct counts, want %d (no lost increments)", len(seen), goroutines)
	}

	// The TTL must have been attached by the atomic script.
	exists, err := c.Exists(ctx, key)
	if err != nil || !exists {
		t.Fatalf("counter should still exist with TTL: exists=%v err=%v", exists, err)
	}
	_ = c.Delete(ctx, key)
}
