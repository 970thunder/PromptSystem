package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const rateLimitPrefix = "promptos:rl:"

// rateLimitKey keeps potentially sensitive identifiers out of Redis keys while
// retaining separate buckets for each action and dimension.
func rateLimitKey(action, bucket string) string {
	return rateLimitPrefix + action + ":" + bucket
}

func rateLimitEmail(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(normalized))
	return "email:" + hex.EncodeToString(sum[:])
}

func rateLimitIP(r *http.Request) string {
	return "ip:" + clientIP(r)
}

// allowRateLimit atomically increments a Redis counter. A cache error is
// returned separately so production handlers can fail closed with 503 while
// development/test environments can continue with an explicit degradation.
func (s *server) allowRateLimit(ctx context.Context, action, bucket string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return false, 0, fmt.Errorf("invalid rate limit configuration")
	}
	if s.cache == nil {
		return s.rateLimitUnavailable(action)
	}
	count, retryAfter, err := s.cache.Increment(ctx, rateLimitKey(action, bucket), window)
	if err != nil {
		return s.rateLimitUnavailable(action)
	}
	if count > int64(limit) {
		if retryAfter <= 0 {
			retryAfter = window
		}
		return false, retryAfter, nil
	}
	return true, 0, nil
}

func (s *server) rateLimitUnavailable(action string) (bool, time.Duration, error) {
	if isDevelopmentEnvironment(s.config.AppEnv) {
		log.Printf("rate limit degraded: action=%s reason=redis_unavailable", action)
		return true, 0, nil
	}
	return false, 0, errRateLimitUnavailable
}

func isDevelopmentEnvironment(env string) bool {
	return env == "development" || env == "docker" || env == "test"
}

var errRateLimitUnavailable = fmt.Errorf("rate limit dependency unavailable")

type rateLimitRule struct {
	bucket string
	limit  int
	window time.Duration
}

func (s *server) enforceRateLimits(ctx context.Context, w http.ResponseWriter, action string, rules ...rateLimitRule) bool {
	for _, rule := range rules {
		ok, retryAfter, err := s.allowRateLimit(ctx, action, rule.bucket, rule.limit, rule.window)
		if writeRateLimitResult(w, ok, retryAfter, err) {
			return false
		}
	}
	return true
}

func writeRateLimitResult(w http.ResponseWriter, ok bool, retryAfter time.Duration, err error) bool {
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{
			Code: http.StatusServiceUnavailable, Message: "Request protection unavailable",
			ErrorCode: "RATE_LIMIT_UNAVAILABLE", Data: nil,
		})
		return true
	}
	if !ok {
		seconds := int(retryAfter / time.Second)
		if retryAfter%time.Second != 0 {
			seconds++
		}
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
		writeJSON(w, http.StatusTooManyRequests, apiResponse[any]{
			Code: http.StatusTooManyRequests, Message: "Too many requests",
			ErrorCode: "RATE_LIMITED", Data: nil,
		})
		return true
	}
	return false
}

// clientIP extracts the best-effort client IP, preferring X-Forwarded-For when
// present (single first hop) and falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
