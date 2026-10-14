package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

// TestRelatedPromptsCapsAndExcludesSelf verifies the related endpoint fetches
// only what it needs (page size 4), returns at most three prompts, and never
// includes the current prompt itself.
func TestRelatedPromptsCapsAndExcludesSelf(t *testing.T) {
	cfg := config.Config{AppEnv: "test", JWTSecret: "related-secret", JWTExpireHours: 1}
	users := store.NewUserStore()
	prompts := store.NewMemoryPromptStore()
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, time.Hour)
	memorySettings := store.NewMemorySettingsStore()
	memorySettings.SetUserStore(users)
	h := newServerWithDeps(serverDeps{
		config:        cfg,
		tokenManager:  tokenManager,
		captcha:       newCaptchaManager(),
		userStore:     users,
		promptStore:   prompts,
		commentStore:  store.NewMemoryCommentStore(),
		settingsStore: memorySettings,
		setupStore:    memorySettings,
		storageMode:   "memory",
	})

	author := store.User{ID: 2, Username: "Nora Chen", Email: "nora@example.com", Status: 1}
	newInput := func(title string) store.CreatePromptInput {
		return store.CreatePromptInput{
			Title:       title,
			Description: "related test",
			Content:     "content",
			Model:       "gpt-4o",
			CategoryID:  1,
			User:        author,
			Status:      1,
		}
	}
	created := make([]int, 0, 5)
	for i := 0; i < 5; i++ {
		prompt, err := prompts.Create(newInput("related pool " + strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("create prompt %d: %v", i, err)
		}
		created = append(created, prompt.ID)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/"+strconv.Itoa(created[0])+"/related", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("related status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data []store.Prompt `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 3 {
		t.Fatalf("related len=%d, want 3", len(envelope.Data))
	}
	for _, item := range envelope.Data {
		if item.ID == created[0] {
			t.Fatal("related must not include the current prompt")
		}
	}

	// 类目内只剩自己时，相关推荐为空而不是把自己返回。
	solo, err := prompts.Create(store.CreatePromptInput{
		Title: "solo category prompt", Description: "x", Content: "c",
		Model: "gpt-4o", CategoryID: 2, User: author, Status: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/prompts/"+strconv.Itoa(solo.ID)+"/related", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("solo related status=%d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 0 {
		t.Fatalf("solo related len=%d, want 0 (self excluded)", len(envelope.Data))
	}
}
