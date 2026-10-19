package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"promptos-backend/internal/auth"
)

// Cookie 会话在最后 1/3 寿命内被访问时应透明换发新 JWT，且 CSRF 值不轮换。
func TestCookieSessionSlidingRefresh(t *testing.T) {
	s, fc := newTestServer(t)
	s.config.AuthCookieEnabled = true
	s.config.JWTExpireHours = 72
	full := auth.NewTokenManager(s.config.JWTSecret, 72*time.Hour)
	h := newServerWithDeps(serverDeps{
		config:       s.config,
		tokenManager: full,
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

	call := func(token string) *http.Response {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/info", nil)
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "csrf-kept"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("user info status = %d: %s", rec.Code, rec.Body.String())
		}
		return rec.Result()
	}
	cookieNamed := func(resp *http.Response, name string) *http.Cookie {
		for _, c := range resp.Cookies() {
			if c.Name == name {
				return c
			}
		}
		return nil
	}

	// 新令牌（剩 72h）：不续期。
	fresh, _ := full.Generate(user.ID, user.Email, user.SessionVer)
	if c := cookieNamed(call(fresh), authCookieName); c != nil {
		t.Fatalf("fresh token should not be refreshed, got cookie %q", c.Name)
	}

	// 剩 20h 的令牌（< 24h 窗口）：应换发新令牌，CSRF 值保持。
	aging, _ := auth.NewTokenManager(s.config.JWTSecret, 20*time.Hour).Generate(user.ID, user.Email, user.SessionVer)
	resp := call(aging)
	session := cookieNamed(resp, authCookieName)
	if session == nil || session.Value == "" || session.Value == aging {
		t.Fatalf("aging token should be refreshed with a new token, got %+v", session)
	}
	if session.MaxAge != 72*3600 || !session.HttpOnly {
		t.Fatalf("refreshed cookie MaxAge=%d HttpOnly=%v", session.MaxAge, session.HttpOnly)
	}
	claims, err := full.Verify(session.Value)
	if err != nil || time.Until(time.Unix(claims.Expiry, 0)) < 71*time.Hour {
		t.Fatalf("refreshed token should be valid for ~72h: %v %+v", err, claims)
	}
	if csrf := cookieNamed(resp, csrfCookieName); csrf == nil || csrf.Value != "csrf-kept" {
		t.Fatalf("csrf cookie should be extended without rotation, got %+v", csrf)
	}

	// 续期出来的新令牌可直接使用。
	call(session.Value)
}
