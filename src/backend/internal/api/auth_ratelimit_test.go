package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

// fakeCache is a deterministic in-memory Cache used to exercise rate limiting
// without a live Redis. Increment returns a fixed window as the retry-after so
// tests can assert the Retry-After header value. A mutex keeps concurrent
// handler tests race-free while preserving Redis-like atomic counters.
type fakeCache struct {
	mu       sync.Mutex
	counters map[string]int64
	store    map[string]string
	err      error
}

type fakeEmailSender struct {
	err  error
	to   string
	body string
}

func (f *fakeEmailSender) Send(_ context.Context, to, _ string, body string) error {
	f.to, f.body = to, body
	return f.err
}

func newFakeCache() *fakeCache {
	return &fakeCache{
		counters: map[string]int64{},
		store:    map[string]string{},
	}
}

func (f *fakeCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[key] = value.(string)
	return nil
}

func (f *fakeCache) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.store[key]
	if !ok {
		return "", nil
	}
	return value, nil
}

func (f *fakeCache) Exists(_ context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return false, f.err
	}
	_, ok := f.store[key]
	return ok, nil
}

func (f *fakeCache) GetAndDelete(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	value, ok := f.store[key]
	if ok {
		delete(f.store, key)
	}
	return value, nil
}

func (f *fakeCache) Increment(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counters[key]++
	return f.counters[key], window, nil
}

func (f *fakeCache) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.store, key)
	delete(f.counters, key)
	return nil
}

func (f *fakeCache) Ping(context.Context) error { return nil }
func (f *fakeCache) Close() error               { return nil }

func newTestServer(t *testing.T) (*server, *fakeCache) {
	t.Helper()
	cfg := config.Config{AppEnv: "test", JWTSecret: "test-secret", JWTExpireHours: 1}
	fc := newFakeCache()
	s := &server{
		config:       cfg,
		tokenManager: auth.NewTokenManager(cfg.JWTSecret, time.Hour),
		cache:        fc,
		userStore:    store.NewUserStore(),
	}
	return s, fc
}

func TestLogoutRevokesTokenImmediately(t *testing.T) {
	s, fc := newTestServer(t)
	user, found := s.userStore.FindByID(1)
	if !found {
		t.Fatal("expected seeded test user")
	}
	token, err := s.tokenManager.Generate(user.ID, user.Email, user.SessionVer)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.handleLogout(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status = %d", rec.Code)
	}
	claims, err := s.tokenManager.Verify(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if _, ok := fc.store["promptos:jwt:denylist:"+claims.JTI]; !ok {
		t.Fatal("logout must add token to denylist")
	}
	wrapped := s.withAuth(func(w http.ResponseWriter, _ *http.Request) {})
	check := httptest.NewRecorder()
	checkReq := httptest.NewRequest(http.MethodGet, "/api/v1/user/info", nil)
	checkReq.Header.Set("Authorization", "Bearer "+token)
	wrapped(check, checkReq)
	if check.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d, want 401", check.Code)
	}
}

func decodeTestResponse(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}
