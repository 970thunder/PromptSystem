package api

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const requestIDKey = "requestID"

// chain builds a middleware stack that runs in the given order.
func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// withRecovery converts panics into a stable 500 response and logs the stack.
func withRecovery(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID := requestIDFrom(r)
				logger.Error("panic recovered",
					slog.String("requestId", requestID),
					slog.Any("panic", recovered),
				)
				writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
					Code:      500,
					Message:   "Internal server error",
					ErrorCode: "INTERNAL_ERROR",
					Data:      nil,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withRequestID accepts a client X-Request-ID or generates one and writes it back.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(withRequestIDCtx(r.Context(), id)))
	})
}

// withAccessLog logs requestId, method, path, status and duration.
func withAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.Info("request",
			slog.String("requestId", requestIDFrom(r)),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// withSecurityHeaders sets basic API response security headers.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// withCORS validates Origin against the configured allowlist.
func (s *server) withCORS(next http.Handler) http.Handler {
	allowed := s.config.AllowedOrigins()
	wildcard := len(allowed) == 0 || containsOrigin(allowed, "*")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Vary", "Origin")
		allowOrigin := "*"
		if !wildcard {
			if !containsOrigin(allowed, origin) {
				writeJSON(w, http.StatusForbidden, apiResponse[any]{
					Code:      403,
					Message:   "Origin not allowed",
					ErrorCode: "ORIGIN_NOT_ALLOWED",
					Data:      nil,
				})
				return
			}
			allowOrigin = origin
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func containsOrigin(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func newRequestID() string {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buffer[:])
}
