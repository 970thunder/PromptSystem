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

// newSetupTestHandler builds a server with the development memory backend and
// the settings-backed setup stores wired the same way NewServer does.
func newSetupTestHandler(t *testing.T) (*server, http.Handler, *store.MemorySettingsStore) {
	t.Helper()
	cfg := config.Config{AppEnv: "test", JWTSecret: "setup-test-secret", JWTExpireHours: 1, EmailAuthEnabled: true}
	users := store.NewUserStore()
	memorySettings := store.NewMemorySettingsStore()
	memorySettings.SetUserStore(users)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, time.Hour)
	h := newServerWithDeps(serverDeps{
		config:        cfg,
		tokenManager:  tokenManager,
		captcha:       newCaptchaManager(),
		userStore:     users,
		promptStore:   store.NewMemoryPromptStore(),
		commentStore:  store.NewMemoryCommentStore(),
		settingsStore: memorySettings,
		setupStore:    memorySettings,
		storageMode:   "memory",
	})
	s := &server{
		config:        cfg,
		tokenManager:  tokenManager,
		userStore:     users,
		settingsStore: memorySettings,
		setupStore:    memorySettings,
		storageMode:   "memory",
	}
	return s, h, memorySettings
}

func decodeSetupStatus(t *testing.T, rec *httptest.ResponseRecorder) setupStatusResponse {
	t.Helper()
	var envelope struct {
		Code int                 `json:"code"`
		Data setupStatusResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode status response: %v body=%s", err, rec.Body.String())
	}
	if envelope.Code != 200 {
		t.Fatalf("expected code 200, got %d body=%s", envelope.Code, rec.Body.String())
	}
	return envelope.Data
}

func TestSetupStatusStartsUninitialized(t *testing.T) {
	_, h, _ := newSetupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code=%d body=%s", rec.Code, rec.Body.String())
	}
	status := decodeSetupStatus(t, rec)
	if status.Initialized || status.StorageMode != "memory" || !status.RegistrationOpen {
		t.Fatalf("unexpected initial status: %+v", status)
	}
}

func TestSetupInitializeCreatesAdminAndPersistsSettings(t *testing.T) {
	s, h, _ := newSetupTestHandler(t)

	body := `{
		"username": "founder",
		"email": "founder@example.com",
		"password": "SetupPass123!",
		"siteName": "My Prompt Hub",
		"siteDescription": "A private hub",
		"smtp": {"host": "smtp.example.com", "port": "465", "user": "noreply@example.com", "password": "smtp-secret", "from": "noreply@example.com"},
		"registrationEnabled": false
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/initialize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	status := decodeSetupStatus(t, rec)
	if !status.Initialized {
		t.Fatal("expected initialized status after bootstrap")
	}
	if status.SiteName != "My Prompt Hub" || status.RegistrationOpen {
		t.Fatalf("settings not persisted: %+v", status)
	}

	// The wizard-created administrator must be able to sign in.
	login := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", strings.NewReader(`{"email":"founder@example.com","password":"SetupPass123!"}`))
	login.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, login)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login status=%d body=%s", rec.Code, rec.Body.String())
	}

	// The stored SMTP secret must never appear in the public status payload.
	if strings.Contains(rec.Body.String(), "smtp-secret") {
		t.Fatal("smtp secret leaked via public endpoints")
	}
	_ = s
}

func TestSetupInitializeIsOneShot(t *testing.T) {
	_, h, _ := newSetupTestHandler(t)

	body := `{"username":"founder","email":"founder@example.com","password":"SetupPass123!"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/initialize", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first initialize status=%d body=%s", rec.Code, rec.Body.String())
		}
		if i == 1 && (rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "SETUP_COMPLETED")) {
			t.Fatalf("second initialize status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func TestSetupInitializeRejectedWhenAdminProvisionedOutOfBand(t *testing.T) {
	_, h, memorySettings := newSetupTestHandler(t)
	memorySettings.GrantAdmin(424242) // simulates a manually provisioned admin

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/initialize", strings.NewReader(`{"username":"hijack","email":"hijack@example.com","password":"SetupPass123!"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "SETUP_COMPLETED") {
		t.Fatalf("hijack initialize status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSetupInitializeValidationFailures(t *testing.T) {
	_, h, _ := newSetupTestHandler(t)
	cases := []struct {
		name string
		body string
		code int
	}{
		{"missing fields", `{"username":"","email":"","password":""}`, http.StatusBadRequest},
		{"weak password", `{"username":"a","email":"a@example.com","password":"short"}`, http.StatusBadRequest},
		{"invalid email", `{"username":"a","email":"not-an-email","password":"SetupPass123!"}`, http.StatusBadRequest},
		{"bad smtp port", `{"username":"a","email":"a@example.com","password":"SetupPass123!","smtp":{"host":"smtp.example.com","port":"99999","from":"noreply@example.com"}}`, http.StatusBadRequest},
		{"smtp without from", `{"username":"a","email":"a@example.com","password":"SetupPass123!","smtp":{"host":"smtp.example.com"}}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/initialize", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.code {
			t.Fatalf("%s: status=%d want=%d body=%s", tc.name, rec.Code, tc.code, rec.Body.String())
		}
	}
}

func TestSetupTestSMTPGatesAndValidation(t *testing.T) {
	_, h, _ := newSetupTestHandler(t)

	// Invalid recipient is rejected before any network dial.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-smtp", strings.NewReader(`{"smtp":{"host":"smtp.example.com","from":"noreply@example.com"},"recipient":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "INVALID_EMAIL") {
		t.Fatalf("invalid recipient status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Missing host/from is a client error.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-smtp", strings.NewReader(`{"smtp":{},"recipient":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "INVALID_SMTP") {
		t.Fatalf("missing host status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Unreachable server surfaces as EMAIL_SEND_FAILED (no hang).
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-smtp", strings.NewReader(`{"smtp":{"host":"127.0.0.1","port":"1","from":"noreply@example.com"},"recipient":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "EMAIL_SEND_FAILED") {
		t.Fatalf("unreachable smtp status=%d body=%s", rec.Code, rec.Body.String())
	}

	// After initialization the anonymous probe is closed.
	init := httptest.NewRequest(http.MethodPost, "/api/v1/setup/initialize", strings.NewReader(`{"username":"founder","email":"founder@example.com","password":"SetupPass123!"}`))
	init.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, init)
	if rec.Code != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-smtp", strings.NewReader(`{"smtp":{"host":"smtp.example.com","from":"noreply@example.com"},"recipient":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("post-init probe status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSiteInfoReflectsSettings(t *testing.T) {
	_, h, _ := newSetupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/site/info", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("site info status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data siteInfoResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode site info: %v", err)
	}
	if envelope.Data.Name != store.DefaultSiteName || !envelope.Data.RegistrationOpen {
		t.Fatalf("unexpected defaults: %+v", envelope.Data)
	}
}

func TestRegistrationToggleBlocksRegister(t *testing.T) {
	s, h, _ := newSetupTestHandler(t)
	if err := s.settingsStore.SetMany(map[string]string{store.SettingRegistrationOn: "false"}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", strings.NewReader(`{"username":"u1","email":"u1@example.com","password":"SetupPass123!","captcha":"000000"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "REGISTRATION_DISABLED") {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// --- settings sender resolution ---

func TestSettingsEmailSenderPrefersDatabaseOverEnv(t *testing.T) {
	settings := store.NewMemorySettingsStore()
	envCfg := config.Config{SMTPHost: "env.example.com", SMTPPort: "587", SMTPFrom: "env@example.com"}
	sender := newSettingsEmailSender(settings, envCfg)

	// No database settings: the env fallback applies.
	resolved := sender.(*settingsEmailSender).resolve()
	if smtp, ok := resolved.(*smtpEmailSender); !ok || smtp.host != "env.example.com" {
		t.Fatalf("expected env fallback, got %#v", resolved)
	}

	if err := settings.SetMany(map[string]string{
		store.SettingSMTPHost: "db.example.com",
		store.SettingSMTPPort: "465",
		store.SettingSMTPUser: "noreply@example.com",
		store.SettingSMTPFrom: "noreply@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	resolved = sender.(*settingsEmailSender).resolve()
	smtp, ok := resolved.(*smtpEmailSender)
	if !ok || smtp.host != "db.example.com" || smtp.port != "465" {
		t.Fatalf("expected db sender, got %#v", resolved)
	}

	// Clearing the database host drops back to env again.
	if err := settings.SetMany(map[string]string{store.SettingSMTPHost: ""}); err != nil {
		t.Fatal(err)
	}
	resolved = sender.(*settingsEmailSender).resolve()
	if smtp, ok := resolved.(*smtpEmailSender); !ok || smtp.host != "env.example.com" {
		t.Fatalf("expected env fallback after clear, got %#v", resolved)
	}

	bare := newSettingsEmailSender(store.NewMemorySettingsStore(), config.Config{}).(*settingsEmailSender)
	if bare.Configured() {
		t.Fatal("no sources must report unconfigured")
	}
}

// --- store-level settings semantics ---

func TestSettingsNormalizeAndValidate(t *testing.T) {
	normalized := store.NormalizeSettingsMap(map[string]string{
		store.SettingSiteName:         "  Hub  ",
		store.SettingSMTPHost:         "",
		store.SettingSetupCompletedAt: "2026-01-01T00:00:00Z",
		"unknown.key":                 "value",
	})
	if len(normalized) != 1 || normalized[store.SettingSiteName] != "Hub" {
		t.Fatalf("unexpected normalization: %#v", normalized)
	}

	if err := store.ValidateSiteSettings(store.SiteSettings{SMTPHost: "smtp.example.com", SMTPPort: "587", SMTPFrom: "noreply@example.com", SMTPUser: "user"}); err == nil {
		t.Fatal("user without password must be rejected")
	}
	if err := store.ValidateSiteSettings(store.SiteSettings{SiteName: strings.Repeat("长", 61)}); err == nil {
		t.Fatal("overlong site name must be rejected")
	}
	if err := store.ValidateSiteSettings(store.SiteSettings{SMTPHost: "smtp.example.com", SMTPPort: "587", SMTPFrom: "noreply@example.com"}); err != nil {
		t.Fatalf("valid smtp rejected: %v", err)
	}

	all, err := store.NewMemorySettingsStore().GetAll()
	if err != nil || len(all) != 0 {
		t.Fatalf("empty store read: %v %#v", err, all)
	}
}

func TestMemorySettingsSetManyDeletesOnEmpty(t *testing.T) {
	settings := store.NewMemorySettingsStore()
	if err := settings.SetMany(map[string]string{store.SettingSiteName: "Hub"}); err != nil {
		t.Fatal(err)
	}
	if err := settings.SetMany(map[string]string{store.SettingSiteName: ""}); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := settings.Get(store.SettingSiteName); ok {
		t.Fatal("empty value must delete the key")
	}
}

// --- admin settings console ---

func newAdminSettingsTestHandler(t *testing.T) (*server, http.Handler, string) {
	t.Helper()
	cfg := config.Config{AppEnv: "test", JWTSecret: "admin-settings-secret", JWTExpireHours: 1}
	users := store.NewUserStore()
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, time.Hour)
	memorySettings := store.NewMemorySettingsStore()
	memorySettings.SetUserStore(users)
	moderation := &fakeModerationStore{admins: map[int]bool{1: true}}
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
		storageMode:     "memory",
	})
	s := &server{
		config:        cfg,
		tokenManager:  tokenManager,
		userStore:     users,
		settingsStore: memorySettings,
		setupStore:    memorySettings,
		storageMode:   "memory",
	}
	admin, found := users.FindByID(1)
	if !found {
		t.Fatal("expected seeded user 1")
	}
	token, err := tokenManager.Generate(admin.ID, admin.Email, admin.SessionVer)
	if err != nil {
		t.Fatal(err)
	}
	return s, h, "Bearer " + token
}

func TestAdminSettingsRoundTrip(t *testing.T) {
	_, h, authHeader := newAdminSettingsTestHandler(t)

	// Non-admin is rejected by the role middleware.
	_, h2, _ := newAdminSettingsTestHandler(t)
	regular := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	rec := httptest.NewRecorder()
	h2.ServeHTTP(rec, regular)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("anonymous settings read status=%d", rec.Code)
	}

	put := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(`{
		"siteName": "Renamed Hub",
		"siteDescription": "Updated",
		"smtpHost": "smtp.example.com",
		"smtpPort": "465",
		"smtpUser": "noreply@example.com",
		"smtpPassword": "smtp-secret",
		"smtpFrom": "noreply@example.com",
		"registrationEnabled": false
	}`))
	put.Header.Set("Content-Type", "application/json")
	put.Header.Set("Authorization", authHeader)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, put)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings update status=%d body=%s", rec.Code, rec.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	get.Header.Set("Authorization", authHeader)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, get)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings read status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data adminSettingsResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	data := envelope.Data
	if data.SiteName != "Renamed Hub" || data.RegistrationEnabled || !data.SMTPPasswordSet || data.SMTPHost != "smtp.example.com" {
		t.Fatalf("unexpected settings after update: %+v", data)
	}
	if strings.Contains(rec.Body.String(), "smtp-secret") {
		t.Fatal("smtp password leaked in admin settings response")
	}

	// Re-PUT without the password keeps the stored secret.
	put = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(`{
		"siteName": "Renamed Hub",
		"smtpHost": "smtp.example.com",
		"smtpPort": "465",
		"smtpUser": "noreply@example.com",
		"smtpFrom": "noreply@example.com"
	}`))
	put.Header.Set("Content-Type", "application/json")
	put.Header.Set("Authorization", authHeader)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, put)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings re-update status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Data.SMTPPasswordSet {
		t.Fatal("empty password must keep the stored secret")
	}

	// Invalid SMTP port is rejected without touching stored values.
	put = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(`{"siteName":"Renamed Hub","smtpHost":"smtp.example.com","smtpPort":"not-a-port","smtpFrom":"noreply@example.com"}`))
	put.Header.Set("Content-Type", "application/json")
	put.Header.Set("Authorization", authHeader)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, put)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "INVALID_SETTING") {
		t.Fatalf("invalid port status=%d body=%s", rec.Code, rec.Body.String())
	}

	// The updated site name surfaces on the public site info endpoint.
	info := httptest.NewRequest(http.MethodGet, "/api/v1/site/info", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, info)
	if !strings.Contains(rec.Body.String(), "Renamed Hub") || !strings.Contains(rec.Body.String(), `"registrationOpen":false`) {
		t.Fatalf("site info did not reflect settings: %s", rec.Body.String())
	}
}
