package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"promptos-backend/internal/auth"
)

func TestAuthCookieFlagsAndCSRFToken(t *testing.T) {
	s, _ := newTestServer(t)
	s.config.AuthCookieEnabled = true
	s.config.AppEnv = "production"
	rec := httptest.NewRecorder()
	s.setAuthCookie(rec, "signed-session")
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected session and csrf cookies, got %d", len(cookies))
	}
	if cookies[0].Name != authCookieName || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie flags are too weak: %+v", cookies[0])
	}
	if cookies[1].Name != csrfCookieName || cookies[1].HttpOnly || !cookies[1].Secure || cookies[1].Value == "" {
		t.Fatalf("csrf cookie flags are invalid: %+v", cookies[1])
	}
}

func TestCookieAuthenticatedWriteRequiresCSRF(t *testing.T) {
	s, _ := newTestServer(t)
	s.config.AuthCookieEnabled = true
	h := s.withCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/logout", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "signed-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "expected"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing csrf header status = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/user/logout", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "signed-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "expected"})
	req.Header.Set("X-CSRF-Token", "expected")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("valid csrf header status = %d, want 204", rec.Code)
	}
}

func TestCookieAuthenticatedUserInfo(t *testing.T) {
	s, fc := newTestServer(t)
	s.config.AuthCookieEnabled = true
	h := newServerWithDeps(serverDeps{
		config:       s.config,
		tokenManager: auth.NewTokenManager(s.config.JWTSecret, time.Duration(s.config.JWTExpireHours)*time.Hour),
		captcha:      s.captcha,
		cache:        fc,
		userStore:    s.userStore,
		promptStore:  s.promptStore,
		commentStore: s.commentStore,
		uploadStore:  s.uploadStore,
		imageStorage: s.imageStorage,
		storageMode:  s.storageMode,
	})
	user, found := s.userStore.FindByID(1)
	if !found {
		t.Fatal("expected seeded test user")
	}
	token, err := s.tokenManager.Generate(user.ID, user.Email, user.SessionVer)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/info", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cookie-authenticated user info status = %d: %s", rec.Code, rec.Body.String())
	}
}
