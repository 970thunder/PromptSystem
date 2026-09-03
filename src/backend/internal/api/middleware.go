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

// envContextKey carries the deployment environment into request-scoped logs.
const envContextKey = "env"

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

// withAccessLog logs requestId, method, route, status, duration and errorCode
// using structured slog JSON. It deliberately omits request bodies, tokens,
// passwords, captcha values, OAuth codes, and any secret material.
func (s *server) withAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if s.metrics != nil {
			s.metrics.observeRequest(recorder.status, time.Since(start))
		}
		attrs := []any{
			slog.String("service", "promptos-backend"),
			slog.String("env", AppEnvFromRequest(r)),
			slog.String("requestId", requestIDFrom(r)),
			slog.String("method", r.Method),
			slog.String("route", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(start)),
		}
		if recorder.errorCode != "" {
			attrs = append(attrs, slog.String("errorCode", recorder.errorCode))
		}
		logger.Info("request", attrs...)
	})
}

// errorCodeRecorder is implemented by statusRecorder so handlers can attach the
// stable errorCode to the access log without leaking it to clients beyond the
// response body.
type errorCodeRecorder interface {
	setErrorCode(code string)
}

func AppEnvFromRequest(r *http.Request) string {
	// The environment is injected by the server via context to avoid importing
	// config here. Fall back to a static value if unavailable.
	if env, ok := r.Context().Value(envContextKey).(string); ok {
		return env
	}
	return "unknown"
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
	status    int
	errorCode string
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) setErrorCode(code string) {
	r.errorCode = code
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
