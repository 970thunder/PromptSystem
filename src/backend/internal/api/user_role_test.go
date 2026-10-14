package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

// TestCurrentUserExposesAdminRoleOnlyToSelf verifies that /user/info annotates
// the administrator role on the self DTO, that regular users see false, and
// that the public author profile DTO never carries the flag.
func TestCurrentUserExposesAdminRoleOnlyToSelf(t *testing.T) {
	cfg := config.Config{AppEnv: "test", JWTSecret: "role-test-secret", JWTExpireHours: 1}
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

	getInfo := func(userID int, email string) *httptest.ResponseRecorder {
		token, err := tokenManager.Generate(userID, email, 0)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/info", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	adminRec := getInfo(1, "astra@example.com")
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin info status=%d body=%s", adminRec.Code, adminRec.Body.String())
	}
	var adminEnvelope struct {
		Data store.PrivateUser `json:"data"`
	}
	if err := json.Unmarshal(adminRec.Body.Bytes(), &adminEnvelope); err != nil {
		t.Fatal(err)
	}
	if !adminEnvelope.Data.IsAdmin {
		t.Fatalf("admin role missing on self DTO: %s", adminRec.Body.String())
	}

	regularRec := getInfo(2, "nora@example.com")
	if regularRec.Code != http.StatusOK {
		t.Fatalf("regular info status=%d", regularRec.Code)
	}
	var regularEnvelope struct {
		Data store.PrivateUser `json:"data"`
	}
	if err := json.Unmarshal(regularRec.Body.Bytes(), &regularEnvelope); err != nil {
		t.Fatal(err)
	}
	if regularEnvelope.Data.IsAdmin {
		t.Fatal("regular user must not be flagged admin")
	}

	// Public author DTOs (nested in prompt detail) never carry the admin flag.
	prompts := store.NewMemoryPromptStore()
	probe, err := prompts.Create(store.CreatePromptInput{
		Title:       "public role probe",
		Description: "probe",
		Content:     "content",
		Model:       "gpt-4o",
		CategoryID:  1,
		User:        store.User{ID: 1, Username: "Astra Lab", Email: "astra@example.com", Status: 1},
		Status:      1,
	})
	if err != nil {
		t.Fatalf("create probe prompt: %v", err)
	}
	publicRec := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/"+strconv.Itoa(probe.ID), nil)
	publicWriter := httptest.NewRecorder()
	h.ServeHTTP(publicWriter, publicRec)
	if publicWriter.Code != http.StatusOK {
		t.Fatalf("prompt detail status=%d body=%s", publicWriter.Code, publicWriter.Body.String())
	}
	if strings.Contains(publicWriter.Body.String(), "isAdmin") {
		t.Fatalf("public author DTO leaks admin flag: %s", publicWriter.Body.String())
	}
}
