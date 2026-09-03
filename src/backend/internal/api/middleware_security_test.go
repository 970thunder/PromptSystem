package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"promptos-backend/internal/config"
)

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	s, _ := newTestServer(t)
	s.config.AllowedOrigin = "https://promptsystem.isoumao.top"
	h := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unlisted origin status = %d, want 403", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unlisted origin must not receive CORS allow header")
	}
}

func TestCORSPreflightUsesExplicitOriginAndCredentials(t *testing.T) {
	s, _ := newTestServer(t)
	s.config = config.Config{AllowedOrigin: "https://promptsystem.isoumao.top"}
	h := s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("preflight must not reach the handler")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/user/info", nil)
	req.Header.Set("Origin", "https://promptsystem.isoumao.top")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "x-csrf-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://promptsystem.isoumao.top" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q, want true", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("preflight must advertise allowed headers")
	}
}

func TestCSRFDoesNotApplyToBearerClients(t *testing.T) {
	s, _ := newTestServer(t)
	s.config.AuthCookieEnabled = true
	h := s.withCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/logout", nil)
	req.Header.Set("Authorization", "Bearer legacy-client-token")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "ignored-cookie"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Bearer write status = %d, want 204", rec.Code)
	}
}
