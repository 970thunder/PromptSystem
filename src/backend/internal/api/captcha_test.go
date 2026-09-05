package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRedisCaptchaIsHashedAndSingleUse(t *testing.T) {
	s, fc := newTestServer(t)
	code, _, _, err := s.issueRedisCaptcha(context.Background(), "USER@example.com")
	if err != nil {
		t.Fatalf("issueRedisCaptcha() error = %v", err)
	}
	stored := fc.store["promptos:captcha:email:user@example.com"]
	if stored == "" || stored == code {
		t.Fatalf("captcha must be stored as a digest, got %q", stored)
	}
	if !s.verifyRedisCaptcha(context.Background(), "user@example.com", code) {
		t.Fatal("expected captcha to verify")
	}
	if s.verifyRedisCaptcha(context.Background(), "user@example.com", code) {
		t.Fatal("expected captcha to be consumed atomically")
	}
}

// TestRedisCaptchaWrongAttemptPreservesCode is a regression test for the S-14
// era production incident: a wrong code used to delete the stored captcha, so
// a typo (or a code from an older email) locked the user out until they
// requested another one.
func TestRedisCaptchaWrongAttemptPreservesCode(t *testing.T) {
	s, fc := newTestServer(t)
	code, _, _, err := s.issueRedisCaptcha(context.Background(), "USER@example.com")
	if err != nil {
		t.Fatalf("issueRedisCaptcha() error = %v", err)
	}

	wrongCode := "000000"
	if wrongCode == code {
		wrongCode = "000001"
	}
	if s.verifyRedisCaptcha(context.Background(), "user@example.com", wrongCode) {
		t.Fatal("expected wrong code to be rejected")
	}
	if fc.store["promptos:captcha:email:user@example.com"] == "" {
		t.Fatal("wrong attempt must not destroy the stored captcha")
	}

	if !s.verifyRedisCaptcha(context.Background(), "user@example.com", code) {
		t.Fatal("expected the correct code to still verify after a wrong attempt")
	}
}

func TestCaptchaManagerIssueAndVerify(t *testing.T) {
	base := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	manager := newCaptchaManager()
	manager.now = func() time.Time { return base }

	code, expiresAt, retryAfter, err := manager.issue("USER@example.com")
	if err != nil {
		t.Fatalf("issue() error = %v", err)
	}
	if retryAfter != 0 {
		t.Fatalf("expected no retry delay, got %s", retryAfter)
	}
	if code == "" || len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %q", code)
	}
	if !expiresAt.Equal(base.Add(captchaTTL)) {
		t.Fatalf("expected captcha TTL %s, got %s", captchaTTL, expiresAt.Sub(base))
	}
	if !manager.verify("user@example.com", code) {
		t.Fatal("expected captcha to verify")
	}
	if manager.verify("user@example.com", code) {
		t.Fatal("expected captcha to be consumed after first use")
	}
}

func TestCaptchaManagerCooldown(t *testing.T) {
	base := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	manager := newCaptchaManager()
	manager.now = func() time.Time { return base }

	if _, _, _, err := manager.issue("user@example.com"); err != nil {
		t.Fatalf("issue() error = %v", err)
	}

	_, _, retryAfter, err := manager.issue("user@example.com")
	if err != nil {
		t.Fatalf("issue() during cooldown error = %v", err)
	}
	if retryAfter <= 0 {
		t.Fatalf("expected retry delay, got %s", retryAfter)
	}
}

func TestCaptchaManagerExpires(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	manager := newCaptchaManager()
	manager.now = func() time.Time { return now }

	code, _, _, err := manager.issue("user@example.com")
	if err != nil {
		t.Fatalf("issue() error = %v", err)
	}

	now = now.Add(captchaTTL + time.Second)
	if manager.verify("user@example.com", code) {
		t.Fatal("expected expired captcha to fail")
	}
}

func TestCaptchaFallsBackToMemoryWhenDevelopmentRedisIsUnavailable(t *testing.T) {
	s, fc := newTestServer(t)
	s.config.AppEnv = "development"
	fc.err = errors.New("redis unavailable")

	code, _, _, err := s.issueRedisCaptcha(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("issueRedisCaptcha() error = %v", err)
	}
	if code == "" {
		t.Fatal("expected development fallback code")
	}
	if !s.verifyRedisCaptcha(context.Background(), "user@example.com", code) {
		t.Fatal("expected development fallback code to verify")
	}
}

// TestCaptchaConcurrentIssueSingleWinner fires parallel captcha sends for one
// email and asserts the rate limit (1 per cooldown window) admits exactly one
// request, so concurrent callers cannot receive multiple live codes.
func TestCaptchaConcurrentIssueSingleWinner(t *testing.T) {
	s, _ := newTestServer(t)

	const concurrency = 8
	results := make(chan int, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/user/captcha", strings.NewReader(`{"email":"user@example.com"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "10.0.0.1")
			rec := httptest.NewRecorder()
			s.handleCaptcha(rec, req)
			results <- rec.Code
		}()
	}
	wg.Wait()
	close(results)

	okCount, limitedCount := 0, 0
	for status := range results {
		switch status {
		case http.StatusOK:
			okCount++
		case http.StatusTooManyRequests:
			limitedCount++
		default:
			t.Fatalf("unexpected status %d during concurrent captcha issue", status)
		}
	}
	if okCount != 1 {
		t.Fatalf("expected exactly 1 accepted captcha send, got %d", okCount)
	}
	if limitedCount != concurrency-1 {
		t.Fatalf("expected %d rate-limited sends, got %d", concurrency-1, limitedCount)
	}
}
