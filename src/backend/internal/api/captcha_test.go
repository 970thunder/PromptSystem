package api

import (
	"testing"
	"time"
)

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
