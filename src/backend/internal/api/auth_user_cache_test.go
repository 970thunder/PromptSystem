package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

// countingUserStore wraps the in-memory user store and counts FindByID calls
// so tests can observe whether the auth user cache absorbed the lookup.
type countingUserStore struct {
	store.UserManager
	findByIDCalls int
}

func (c *countingUserStore) FindByID(id int) (store.AuthUser, bool) {
	c.findByIDCalls++
	return c.UserManager.FindByID(id)
}

// newAuthCacheTestHandler wires a handler whose auth user cache is explicitly
// provided (clock-controllable via the returned cache). Seed user 2's token
// is issued without touching FindByID (SessionVer 0) to keep the counter clean.
func newAuthCacheTestHandler(t *testing.T, ttlSeconds int) (http.Handler, *authUserCache, *countingUserStore, string) {
	t.Helper()
	cfg := config.Config{
		AppEnv:                  "test",
		JWTSecret:               "auth-cache-secret",
		JWTExpireHours:          1,
		AuthUserCacheTTLSeconds: ttlSeconds,
	}
	users := &countingUserStore{UserManager: store.NewUserStore()}
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, time.Hour)
	memorySettings := store.NewMemorySettingsStore()
	memorySettings.SetUserStore(users.UserManager.(*store.UserStore))
	moderation := &fakeModerationStore{admins: map[int]bool{1: true}}
	// Apply the admin decision to the real user store: disabling deletes the
	// account row state (status=0) exactly so the next request fails closed.
	moderation.onSetUserStatus = func(userID, status int) {
		if status == 0 {
			_ = users.DeleteAccount(userID)
		}
	}

	var cache *authUserCache
	if ttlSeconds > 0 {
		cache = newAuthUserCache(time.Duration(ttlSeconds) * time.Second)
	}
	h := newServerWithDeps(serverDeps{
		config:          cfg,
		tokenManager:    tokenManager,
		captcha:         newCaptchaManager(),
		userStore:       users,
		promptStore:     store.NewMemoryPromptStore(),
		commentStore:    store.NewMemoryCommentStore(),
		moderationStore: moderation,
		settingsStore:   memorySettings,
		setupStore:      memorySettings,
		authUserCache:   cache,
		storageMode:     "memory",
	})

	token, err := tokenManager.Generate(2, "nora@example.com", 0)
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := tokenManager.Generate(1, "astra@example.com", 0)
	if err != nil {
		t.Fatal(err)
	}
	return h, cache, users, token + "\n" + adminToken
}

func splitTokens(pair string) (userToken, adminToken string) {
	parts := strings.SplitN(pair, "\n", 2)
	return parts[0], parts[1]
}

func authedGet(h http.Handler, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuthUserCacheAbsorbsRepeatLookups(t *testing.T) {
	h, cache, users, tokens := newAuthCacheTestHandler(t, 30)
	if cache == nil {
		t.Fatal("expected cache for ttl > 0")
	}
	userToken, _ := splitTokens(tokens)

	// Request 1: cache miss -> withAuth FindByID + handler FindByID.
	if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
		t.Fatalf("first request status=%d body=%s", rec.Code, rec.Body.String())
	}
	// Request 2: withAuth served from cache -> only the handler's FindByID.
	if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
		t.Fatalf("second request status=%d body=%s", rec.Code, rec.Body.String())
	}
	if users.findByIDCalls != 3 {
		t.Fatalf("expected 3 FindByID calls with warm cache, got %d", users.findByIDCalls)
	}
}

func TestAuthUserCacheDisabledWithZeroTTL(t *testing.T) {
	h, cache, users, tokens := newAuthCacheTestHandler(t, 0)
	if cache != nil {
		t.Fatal("zero ttl must not construct a cache")
	}
	userToken, _ := splitTokens(tokens)

	for i := 0; i < 2; i++ {
		if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
			t.Fatalf("request %d status=%d", i+1, rec.Code)
		}
	}
	// Uncached: withAuth + handler hit the store on every request.
	if users.findByIDCalls != 4 {
		t.Fatalf("expected 4 FindByID calls without cache, got %d", users.findByIDCalls)
	}
}

func TestAuthUserCacheExpiryFallsBackToStore(t *testing.T) {
	h, cache, users, tokens := newAuthCacheTestHandler(t, 30)
	userToken, _ := splitTokens(tokens)

	if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
		t.Fatalf("first request status=%d", rec.Code)
	}
	base := time.Now()
	cache.now = func() time.Time { return base.Add(31 * time.Second) }

	if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
		t.Fatalf("post-expiry request status=%d", rec.Code)
	}
	// Expired entry: withAuth misses again (2 misses + 2 handler calls).
	if users.findByIDCalls != 4 {
		t.Fatalf("expected 4 FindByID calls after expiry, got %d", users.findByIDCalls)
	}
}

func TestAdminDisableInvalidatesAuthCache(t *testing.T) {
	h, cache, _, tokens := newAuthCacheTestHandler(t, 30)
	userToken, adminToken := splitTokens(tokens)

	if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
		t.Fatalf("prime request status=%d", rec.Code)
	}

	// Admin disables user 2: handler must invalidate the cached row so the
	// next request fails closed immediately.
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/2", strings.NewReader(`{"status":0,"reason":"cache test"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin disable status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, found := cache.get(2); found {
		t.Fatal("disabled user must not stay cached")
	}

	rec = authedGet(h, userToken, "/api/v1/user/info")
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "AUTH_USER_DISABLED") {
		t.Fatalf("expected immediate disable, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPasswordResetDropsAuthCache(t *testing.T) {
	h, cache, _, tokens := newAuthCacheTestHandler(t, 30)
	userToken, _ := splitTokens(tokens)

	if rec := authedGet(h, userToken, "/api/v1/user/info"); rec.Code != http.StatusOK {
		t.Fatalf("prime request status=%d", rec.Code)
	}

	// Obtain a dev captcha (test env returns devCode) and reset the password.
	captchaReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/captcha", strings.NewReader(`{"email":"nora@example.com"}`))
	captchaReq.Header.Set("Content-Type", "application/json")
	captchaRec := httptest.NewRecorder()
	h.ServeHTTP(captchaRec, captchaReq)
	if captchaRec.Code != http.StatusOK {
		t.Fatalf("captcha status=%d body=%s", captchaRec.Code, captchaRec.Body.String())
	}
	var captchaEnvelope struct {
		Data struct {
			DevCode string `json:"devCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(captchaRec.Body.Bytes(), &captchaEnvelope); err != nil || captchaEnvelope.Data.DevCode == "" {
		t.Fatalf("captcha devCode missing: %s", captchaRec.Body.String())
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/password/reset", strings.NewReader(`{"email":"nora@example.com","captcha":"`+captchaEnvelope.Data.DevCode+`","password":"NewPassword123!"}`))
	resetReq.Header.Set("Content-Type", "application/json")
	resetRec := httptest.NewRecorder()
	h.ServeHTTP(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", resetRec.Code, resetRec.Body.String())
	}
	if len(cache.entries) != 0 {
		t.Fatalf("reset must drop the whole cache, entries=%d", len(cache.entries))
	}

	// The old token must be rejected immediately: the cache drop exposes the
	// bumped session version.
	rec := authedGet(h, userToken, "/api/v1/user/info")
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "AUTH_TOKEN_REVOKED") {
		t.Fatalf("expected token revocation after reset, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthUserCacheBasics(t *testing.T) {
	cache := newAuthUserCache(time.Minute)
	user := store.AuthUser{ID: 7, Username: "u", Status: 1}

	if _, found := cache.get(7); found {
		t.Fatal("empty cache must miss")
	}
	cache.put(user)
	if got, found := cache.get(7); !found || got.Username != "u" {
		t.Fatal("expected cached user")
	}
	cache.put(store.AuthUser{ID: 0}) // ignored
	cache.invalidate(7)
	if _, found := cache.get(7); found {
		t.Fatal("invalidated entry must miss")
	}
	cache.put(user)
	cache.invalidateAll()
	if _, found := cache.get(7); found {
		t.Fatal("invalidateAll must clear entries")
	}
}
