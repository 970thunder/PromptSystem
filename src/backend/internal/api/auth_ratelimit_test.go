package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	_, ok := f.store[key]
	return ok, nil
}

func (f *fakeCache) GetAndDelete(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
		captcha:      newCaptchaManager(),
		cache:        fc,
		userStore:    store.NewUserStore(),
	}
	return s, fc
}

func doLogin(t *testing.T, s *server, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"email":"` + email + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", strings.NewReader(body))
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	rec := httptest.NewRecorder()
	s.handleLogin(rec, req)
	return rec
}

func TestLoginUnifiedError(t *testing.T) {
	s, _ := newTestServer(t)

	// Unknown account and wrong password must be indistinguishable so account
	// existence cannot be enumerated through the login endpoint.
	unknown := doLogin(t, s, "nobody@example.com", "Whatever1!")
	wrongPassword := doLogin(t, s, "astra@example.com", "WrongPass1!")

	for i, rec := range []*httptest.ResponseRecorder{unknown, wrongPassword} {
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("case %d: expected 401, got %d", i, rec.Code)
		}
		var payload apiResponse[any]
		decodeTestResponse(t, rec, &payload)
		if payload.Message != "Invalid email or password" {
			t.Fatalf("case %d: expected unified message, got %q", i, payload.Message)
		}
		if payload.ErrorCode != "AUTH_INVALID_CREDENTIALS" {
			t.Fatalf("case %d: expected AUTH_INVALID_CREDENTIALS, got %q", i, payload.ErrorCode)
		}
	}
}

func TestLoginRateLimitEmailBucket(t *testing.T) {
	s, _ := newTestServer(t)

	// The login email bucket allows 5 attempts per 15 minutes. The 6th attempt
	// for the same normalized email must be rejected with 429 + Retry-After.
	const emailLimit = 5
	for i := 0; i < emailLimit; i++ {
		rec := doLogin(t, s, "astra@example.com", "bad-password")
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was unexpectedly rate limited", i+1)
		}
	}

	rec := doLogin(t, s, "astra@example.com", "bad-password")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding email bucket, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}
}

func TestLoginRateLimitIPBucket(t *testing.T) {
	s, _ := newTestServer(t)

	// With distinct emails the email bucket never fills, but a single IP is
	// still throttled on its own (login IP bucket allows 10 per minute).
	const ipLimit = 10
	for i := 0; i < ipLimit; i++ {
		rec := doLogin(t, s, "user"+string(rune('a'+i))+"@example.com", "bad-password")
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was unexpectedly rate limited", i+1)
		}
	}

	rec := doLogin(t, s, "beyond@example.com", "bad-password")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding IP bucket, got %d", rec.Code)
	}
}

func TestLoginRateLimitPerIPNotShared(t *testing.T) {
	s, _ := newTestServer(t)

	// A different client IP must not inherit the first client's bucket, so a
	// legitimate user behind a shared NAT is not blocked by another IP's abuse.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", strings.NewReader(`{"email":"astra@example.com","password":"PromptOS123!"}`))
	req.Header.Set("X-Forwarded-For", "10.0.0.99")
	rec := httptest.NewRecorder()
	s.handleLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected fresh IP to be allowed, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterRateLimited(t *testing.T) {
	s, _ := newTestServer(t)

	// Register IP bucket allows 5 per 10 minutes. All requests fail captcha
	// validation (400) until the rate limit itself is hit, so assert the final
	// 429 with Retry-After.
	const ipLimit = 5
	for i := 0; i < ipLimit; i++ {
		body := `{"username":"u","email":"reg` + string(rune('a'+i)) + `@example.com","password":"PromptOS123!","captcha":"123456"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", strings.NewReader(body))
		req.Header.Set("X-Forwarded-For", "10.0.0.2")
		rec := httptest.NewRecorder()
		s.handleRegister(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was unexpectedly rate limited", i+1)
		}
	}

	body := `{"username":"u","email":"regz@example.com","password":"PromptOS123!","captcha":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", strings.NewReader(body))
	req.Header.Set("X-Forwarded-For", "10.0.0.2")
	rec := httptest.NewRecorder()
	s.handleRegister(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding register IP bucket, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}
	var payload apiResponse[any]
	decodeTestResponse(t, rec, &payload)
	if payload.ErrorCode != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED errorCode, got %q", payload.ErrorCode)
	}
}

func TestResetPasswordDoesNotRevealRegistration(t *testing.T) {
	s, fc := newTestServer(t)

	// A valid captcha is required before ResetPassword runs, so seed one for a
	// registered and an unregistered email. The responses must be identical so
	// the reset flow cannot be used to enumerate registered emails.
	for _, email := range []string{"astra@example.com", "nobody@example.com"} {
		fc.store["promptos:captcha:email:"+email] = captchaDigest(s.config.JWTSecret, email, "123456")
	}

	var messages []string
	for _, email := range []string{"astra@example.com", "nobody@example.com"} {
		body := `{"email":"` + email + `","captcha":"123456","password":"PromptOS123!"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/password/reset", strings.NewReader(body))
		req.Header.Set("X-Forwarded-For", "10.0.0.3")
		rec := httptest.NewRecorder()
		s.handleResetPassword(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", email, rec.Code)
		}
		var payload apiResponse[any]
		decodeTestResponse(t, rec, &payload)
		messages = append(messages, payload.Message)
	}

	if messages[0] != messages[1] {
		t.Fatalf("reset responses differ, revealing registration: %q vs %q", messages[0], messages[1])
	}
}

func TestProductionCaptchaResponseDoesNotExposeCode(t *testing.T) {
	s, _ := newTestServer(t)
	s.config.AppEnv = "production"
	s.emailSender = &fakeEmailSender{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/captcha", strings.NewReader(`{"email":"safe@example.com"}`))
	req.Header.Set("X-Forwarded-For", "10.0.0.50")
	rec := httptest.NewRecorder()
	s.handleCaptcha(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "devCode") {
		t.Fatalf("production captcha response leaked devCode: %s", rec.Body.String())
	}
}

func TestProductionCaptchaFailsClosedWithoutRedis(t *testing.T) {
	s, _ := newTestServer(t)
	s.config.AppEnv = "production"
	s.cache = nil
	if _, _, _, err := s.issueRedisCaptcha(context.Background(), "safe@example.com"); err == nil {
		t.Fatal("production captcha must fail closed when Redis is unavailable")
	}
}

func TestProductionCaptchaSendFailureDeletesPendingCode(t *testing.T) {
	s, fc := newTestServer(t)
	s.config.AppEnv = "production"
	s.emailSender = &fakeEmailSender{err: errors.New("smtp unavailable")}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/captcha", strings.NewReader(`{"email":"cleanup@example.com"}`))
	req.Header.Set("X-Forwarded-For", "10.0.0.51")
	rec := httptest.NewRecorder()
	s.handleCaptcha(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, ok := fc.store["promptos:captcha:email:cleanup@example.com"]; ok {
		t.Fatal("captcha must be deleted after SMTP send failure")
	}
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
