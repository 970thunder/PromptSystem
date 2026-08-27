package api

import (
	"context"
	"encoding/json"
	"time"
)

// Versioned cache key prefixes. Bumping the version (v1 -> v2) invalidates all
// previously cached content at once, which is safer than relying on TTLs during
// a schema/semantic change.
const (
	cacheKeyHomeSummary = "promptos:v1:home:summary"
	cacheKeyCategories  = "promptos:v1:categories"
	cacheKeyHotTags     = "promptos:v1:hot:tags"
)

// Cache TTLs are intentionally short: the data is cheap to recompute and we
// prefer freshness over hit ratio for community aggregates.
const (
	homeSummaryTTL = 30 * time.Second
	categoriesTTL  = 5 * time.Minute
	hotTagsTTL     = 5 * time.Minute
)

// cachedJSON returns the cached value for key if present and valid, decoding it
// into out. The second return value reports whether a cache hit occurred. When
// the cache is unavailable (nil) or returns an error, it degrades to a miss so
// the caller always falls back to the store. This keeps private responses out
// of the cache because only the explicitly listed public aggregates use it.
func (s *server) cachedJSON(ctx context.Context, key string, out any) (bool, error) {
	if s.cache == nil {
		return false, nil
	}
	raw, err := s.cache.Get(ctx, key)
	if err != nil || raw == "" {
		return false, err
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		// A corrupt entry must not break the request; treat as a miss.
		return false, err
	}
	return true, nil
}

// setCached stores value under key with ttl. Errors are swallowed because a
// failed cache write must never fail the request; the store result is still
// returned to the client.
func (s *server) setCached(ctx context.Context, key string, value any, ttl time.Duration) {
	if s.cache == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, key, string(data), ttl)
}

// invalidateContentCaches drops the public aggregate caches. It is called after
// a prompt is published, updated, deleted, or its like/favorite state changes.
// View counts are deliberately excluded (high frequency) and rely on the short
// TTL for eventual consistency.
func (s *server) invalidateContentCaches(ctx context.Context) {
	if s.cache == nil {
		return
	}
	for _, key := range []string{cacheKeyHomeSummary, cacheKeyCategories, cacheKeyHotTags} {
		if err := s.cache.Delete(ctx, key); err != nil {
			// Logged at debug level elsewhere; do not fail the request.
			_ = err
		}
	}
}
