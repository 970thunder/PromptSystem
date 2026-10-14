package api

import (
	"sync"
	"time"

	"promptos-backend/internal/store"
)

// authUserCache is a small in-process TTL cache for the user row that
// withAuth re-validates on every authenticated request. It removes one
// primary-key SELECT per request; correctness is preserved by invalidating
// on every API-level write to a user and by the short TTL bounding anything
// the API layer does not see (e.g. out-of-band SQL).
//
// Deliberately NOT cached here: the administrator role check (withAdmin)
// reads the role per request so grants/revocations stay immediate, and the
// Redis token denylist stays the authoritative revocation path.
type authUserCache struct {
	mu      sync.RWMutex
	entries map[int]authUserCacheEntry
	ttl     time.Duration
	now     func() time.Time
}

type authUserCacheEntry struct {
	user    store.AuthUser
	expires time.Time
}

func newAuthUserCache(ttl time.Duration) *authUserCache {
	return &authUserCache{
		entries: make(map[int]authUserCacheEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

func (c *authUserCache) get(id int) (store.AuthUser, bool) {
	c.mu.RLock()
	entry, ok := c.entries[id]
	c.mu.RUnlock()
	if !ok || !c.now().Before(entry.expires) {
		return store.AuthUser{}, false
	}
	return entry.user, true
}

func (c *authUserCache) put(user store.AuthUser) {
	if user.ID <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[user.ID] = authUserCacheEntry{user: user, expires: c.now().Add(c.ttl)}
}

func (c *authUserCache) invalidate(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, id)
}

func (c *authUserCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[int]authUserCacheEntry)
}

// primeAuthUserCache refreshes the cached row after a write that returned the
// fresh user (login, register, profile update, GitHub upsert).
func (s *server) primeAuthUserCache(user store.AuthUser) {
	if s.authUserCache != nil {
		s.authUserCache.put(user)
	}
}

// dropAuthUserCache clears every cached row. Used when writes are keyed by
// email or affect an unknown set of ids (password reset).
func (s *server) dropAuthUserCache() {
	if s.authUserCache != nil {
		s.authUserCache.invalidateAll()
	}
}
