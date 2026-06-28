package api

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"promptos-backend/internal/store"
)

const (
	captchaTTL      = 10 * time.Minute
	captchaCooldown = 60 * time.Second
)

type captchaManager struct {
	mu      sync.Mutex
	entries map[string]captchaEntry
	now     func() time.Time
}

type captchaEntry struct {
	code      string
	expiresAt time.Time
	sentAt    time.Time
}

func newCaptchaManager() *captchaManager {
	return &captchaManager{
		entries: map[string]captchaEntry{},
		now:     time.Now,
	}
}

func (m *captchaManager) issue(email string) (string, time.Time, time.Duration, error) {
	normalized, ok := normalizeCaptchaEmail(email)
	if !ok {
		return "", time.Time{}, 0, store.ErrInvalidEmail
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	if entry, exists := m.entries[normalized]; exists && now.Sub(entry.sentAt) < captchaCooldown {
		return "", entry.expiresAt, captchaCooldown - now.Sub(entry.sentAt), nil
	}

	code, err := randomCaptchaCode()
	if err != nil {
		return "", time.Time{}, 0, err
	}

	expiresAt := now.Add(captchaTTL)
	m.entries[normalized] = captchaEntry{
		code:      code,
		expiresAt: expiresAt,
		sentAt:    now,
	}

	return code, expiresAt, 0, nil
}

func (m *captchaManager) verify(email, code string) bool {
	normalized, ok := normalizeCaptchaEmail(email)
	if !ok {
		return false
	}

	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.entries[normalized]
	if !exists || m.now().After(entry.expiresAt) || entry.code != cleanCode {
		return false
	}

	delete(m.entries, normalized)
	return true
}

func normalizeCaptchaEmail(email string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	return normalized, store.IsValidEmail(normalized)
}

func randomCaptchaCode() (string, error) {
	var buf [3]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}

	number := int(buf[0])<<16 | int(buf[1])<<8 | int(buf[2])
	return fmt.Sprintf("%06d", number%1000000), nil
}
