package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"testing"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

// newIntegrationServer builds a fully injectable API server backed by the
// in-memory stores and a real local image storage rooted at a temp directory.
// It returns the *server too, so tests can seed upload records whose ownership
// the create/update handlers validate. It never touches MySQL, Redis, or the
// network, so tests are parallel-safe and repeatable.
func newIntegrationServer(t *testing.T) (*server, http.Handler) {
	t.Helper()

	cfg := config.Config{
		AppEnv:         "test",
		JWTSecret:      "test-secret-with-sufficient-length-for-signing",
		JWTExpireHours: 72,
		UploadProvider: "local",
		UploadDir:      t.TempDir(),
		UploadBaseURL:  "http://test.local",
		UploadMaxMB:    2,
		AllowGif:       false,
	}

	imageStorage, err := storage.NewImageStorage(cfg)
	if err != nil {
		t.Fatalf("NewImageStorage() error = %v", err)
	}

	uploadStore := store.NewMemoryUploadStore()
	s := &server{
		config:       cfg,
		tokenManager: auth.NewTokenManager(cfg.JWTSecret, time.Duration(cfg.JWTExpireHours)*time.Hour),
		captcha:      newCaptchaManager(),
		githubClient: &http.Client{Timeout: 2 * time.Second},
		cache:        nil,
		userStore:    store.UserManager(store.NewUserStore()),
		promptStore:  store.PromptManager(store.NewMemoryPromptStore()),
		commentStore: store.CommentManager(store.NewMemoryCommentStore()),
		uploadStore:  store.UploadManager(uploadStore),
		imageStorage: imageStorage,
		storageMode:  "memory",
	}
	return s, newServerWithDeps(serverDeps{
		config:       cfg,
		tokenManager: s.tokenManager,
		captcha:      s.captcha,
		githubClient: s.githubClient,
		cache:        s.cache,
		userStore:    s.userStore,
		promptStore:  s.promptStore,
		commentStore: s.commentStore,
		uploadStore:  s.uploadStore,
		imageStorage: s.imageStorage,
		storageMode:  s.storageMode,
	})
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, token string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var envelope map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &envelope)
	return rec, envelope
}

// registerAndLogin creates a fresh user and returns its bearer token and user
// ID. The test env returns a devCode from the captcha endpoint, which is then
// used to register — the same flow the real frontend follows.
func registerAndLogin(t *testing.T, h http.Handler) (string, int) {
	t.Helper()

	username := "ituser" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	email := username + "@example.com"

	rec, envelope := doJSON(t, h, http.MethodPost, "/api/v1/user/captcha", map[string]any{
		"email": email,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("captcha status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	devCode, _ := data["devCode"].(string)
	if devCode == "" {
		t.Fatalf("captcha response missing devCode in test env: %s", rec.Body.String())
	}

	rec, _ = doJSON(t, h, http.MethodPost, "/api/v1/user/register", map[string]any{
		"username": username,
		"email":    email,
		"password": "StrongPass123!",
		"captcha":  devCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec, envelope = doJSON(t, h, http.MethodPost, "/api/v1/user/login", map[string]any{
		"email":    email,
		"password": "StrongPass123!",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}

	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("login response missing data: %s", rec.Body.String())
	}
	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login response missing token: %s", rec.Body.String())
	}
	userObj, ok := data["user"].(map[string]any)
	if !ok {
		t.Fatalf("login response missing user: %s", rec.Body.String())
	}
	userID, _ := userObj["id"].(float64)
	return token, int(userID)
}

// seedUpload records a local upload owned by userID so prompt create/update
// ownership validation (B6-02) accepts the cover reference.
func seedUpload(t *testing.T, s *server, userID int, objectKey string) string {
	t.Helper()
	if _, err := s.uploadStore.RecordUpload(store.UploadRecord{
		OwnerID:     userID,
		Provider:    "local",
		Purpose:     store.UploadPurposePromptImage,
		ObjectKey:   objectKey,
		ContentType: "image/png",
		Size:        1024,
		Status:      store.UploadStatusPending,
	}); err != nil {
		t.Fatalf("RecordUpload() error = %v", err)
	}
	return "/uploads/" + objectKey
}

func createPrompt(t *testing.T, s *server, h http.Handler, token string, userID int) int {
	t.Helper()

	cover := seedUpload(t, s, userID, "test-cover.png")

	rec, _ := doJSON(t, h, http.MethodPost, "/api/v1/prompts", map[string]any{
		"title":       "Test prompt",
		"description": "integration test prompt",
		"content":     "You are a helpful assistant.",
		"model":       "gpt-4o",
		"params":      map[string]any{"temperature": 0.7},
		"categoryId":  1,
		"cover":       cover,
		"tags":        []string{"测试"},
		"status":      1,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create prompt status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var env struct {
		Data store.Prompt `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode prompt: %v", err)
	}
	return env.Data.ID
}

func TestHealthEndpoints(t *testing.T) {
	_, h := newIntegrationServer(t)

	rec, _ := doJSON(t, h, http.MethodGet, "/api/v1/health/live", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("live status = %d", rec.Code)
	}

	// Test env is non-development, so a memory-backed (degraded) ready probe
	// must return 503 with degraded=true — not pretend everything is fine.
	rec, envelope := doJSON(t, h, http.MethodGet, "/api/v1/health/ready", nil, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want 503 in non-dev env, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["storageMode"] != "memory" {
		t.Fatalf("ready storageMode = %v, want memory", data["storageMode"])
	}
	if data["degraded"] != true {
		t.Fatalf("ready degraded = %v, want true for memory store in tests", data["degraded"])
	}
}

func TestAuthRequiredEndpoints(t *testing.T) {
	_, h := newIntegrationServer(t)

	protected := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/user/info"},
		{http.MethodPost, "/api/v1/prompts"},
		{http.MethodPost, "/api/v1/uploads/images"},
	}
	for _, tc := range protected {
		rec, envelope := doJSON(t, h, tc.method, tc.path, nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want 401", tc.method, tc.path, rec.Code)
		}
		code, _ := envelope["errorCode"].(string)
		if code != "AUTH_TOKEN_MISSING" && code != "AUTH_INVALID_TOKEN" {
			t.Fatalf("%s %s errorCode = %v, want stable auth code", tc.method, tc.path, envelope["errorCode"])
		}
	}
}

func TestRegisterLoginAndProfile(t *testing.T) {
	_, h := newIntegrationServer(t)
	token, _ := registerAndLogin(t, h)

	rec, envelope := doJSON(t, h, http.MethodGet, "/api/v1/user/info", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("user info status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["email"] == nil {
		t.Fatalf("user info missing email: %s", rec.Body.String())
	}

	rec, _ = doJSON(t, h, http.MethodPut, "/api/v1/user/info", map[string]any{
		"username": "updated-name",
		"bio":      "hello",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPromptLifecycleAndInteractions(t *testing.T) {
	s, h := newIntegrationServer(t)
	token, userID := registerAndLogin(t, h)

	promptID := createPrompt(t, s, h, token, userID)
	path := "/api/v1/prompts/" + strconv.Itoa(promptID)

	rec, _ := doJSON(t, h, http.MethodGet, path, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get prompt status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// like -> unlike toggle is idempotent and reversible.
	rec, envelope := doJSON(t, h, http.MethodPost, path+"/like", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("like status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["applied"] != true {
		t.Fatalf("like applied = %v, want true", data["applied"])
	}

	rec, envelope = doJSON(t, h, http.MethodPost, path+"/like", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("second like status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ = envelope["data"].(map[string]any)
	if data["applied"] != false {
		t.Fatalf("second like applied = %v, want false (idempotent)", data["applied"])
	}

	rec, envelope = doJSON(t, h, http.MethodGet, path+"/interaction", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("interaction status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ = envelope["data"].(map[string]any)
	if data["liked"] != true {
		t.Fatalf("interaction liked = %v, want true", data["liked"])
	}

	rec, envelope = doJSON(t, h, http.MethodDelete, path+"/unlike", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("unlike status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ = envelope["data"].(map[string]any)
	if data["applied"] != true {
		t.Fatalf("unlike applied = %v, want true", data["applied"])
	}

	rec, envelope = doJSON(t, h, http.MethodGet, path+"/interaction", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("interaction status after unlike = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ = envelope["data"].(map[string]any)
	if data["liked"] != false {
		t.Fatalf("interaction liked after unlike = %v, want false", data["liked"])
	}
}

func TestSearchPagination(t *testing.T) {
	s, h := newIntegrationServer(t)
	token, userID := registerAndLogin(t, h)

	for i := 0; i < 5; i++ {
		createPrompt(t, s, h, token, userID)
	}

	rec, envelope := doJSON(t, h, http.MethodGet, "/api/v1/prompts?page=1&pageSize=2", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	list, ok := data["list"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("search list length = %v, want 2", len(list))
	}
	if data["total"] == nil {
		t.Fatalf("search missing total: %s", rec.Body.String())
	}

	// Invalid pagination must be rejected, not silently clamped.
	rec, envelope = doJSON(t, h, http.MethodGet, "/api/v1/prompts?page=0&pageSize=500", nil, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination status = %d, want 400", rec.Code)
	}
	code, _ := envelope["errorCode"].(string)
	if code == "" {
		t.Fatalf("invalid pagination missing errorCode: %s", rec.Body.String())
	}
}

func TestCommentsLifecycle(t *testing.T) {
	s, h := newIntegrationServer(t)
	token, userID := registerAndLogin(t, h)
	promptID := createPrompt(t, s, h, token, userID)
	path := "/api/v1/prompts/" + strconv.Itoa(promptID)

	rec, _ := doJSON(t, h, http.MethodPost, path+"/comments", map[string]any{
		"content": "nice prompt",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create comment status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec, envelope := doJSON(t, h, http.MethodGet, path+"/comments?page=1&pageSize=10", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list comments status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	total, _ := data["total"].(float64)
	if int(total) < 1 {
		t.Fatalf("comments total = %v, want >= 1", data["total"])
	}
}

func TestUploadRejectsMissingFile(t *testing.T) {
	_, h := newIntegrationServer(t)
	token, _ := registerAndLogin(t, h)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("upload missing file status = %d, body = %s", rec.Code, rec.Body.String())
	}
	envelope := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &envelope)
	if envelope["errorCode"] != "IMAGE_REQUIRED" {
		t.Fatalf("upload missing file errorCode = %v, want IMAGE_REQUIRED", envelope["errorCode"])
	}
}

// uploadImage builds a multipart request carrying body under field "file".
// The part-level Content-Type is set explicitly so the server's MIME agreement
// check (B6-01) can validate it against the real format.
func uploadImage(t *testing.T, h http.Handler, token, filename, contentType string, body []byte) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	hdr.Set("Content-Type", contentType)
	fw, err := w.CreatePart(hdr)
	if err != nil {
		t.Fatalf("create form part: %v", err)
	}
	if _, err := fw.Write(body); err != nil {
		t.Fatalf("write file body: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	envelope := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &envelope)
	return rec, envelope
}

// tinyPNG returns a valid 1x1 PNG generated by the standard library so the
// real decode path is exercised.
func tinyPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestUploadValidPNG(t *testing.T) {
	_, h := newIntegrationServer(t)
	token, _ := registerAndLogin(t, h)

	rec, envelope := uploadImage(t, h, token, "ok.png", "image/png", tinyPNG(t))
	if rec.Code != http.StatusOK {
		t.Fatalf("valid png status = %d, body = %s", rec.Code, rec.Body.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["url"] == "" {
		t.Fatalf("valid png missing url: %s", rec.Body.String())
	}
}

func TestUploadRejectsCorruptAndSpoofedImages(t *testing.T) {
	_, h := newIntegrationServer(t)
	token, _ := registerAndLogin(t, h)

	// Corrupt bytes that claim to be an image.
	rec, envelope := uploadImage(t, h, token, "corrupt.png", "image/png", []byte("not really a png at all"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("corrupt image status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if envelope["errorCode"] != "INVALID_IMAGE_FORMAT" && envelope["errorCode"] != "IMAGE_DECODE_FAILED" {
		t.Fatalf("corrupt image errorCode = %v", envelope["errorCode"])
	}

	// MIME spoof: text body claiming to be a PNG.
	rec, envelope = uploadImage(t, h, token, "spoof.png", "image/png", []byte("plain text pretending to be an image"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("spoofed mime status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUploadRejectsOversizedBody(t *testing.T) {
	_, h := newIntegrationServer(t)
	token, _ := registerAndLogin(t, h)

	// A single file over the per-file cap is rejected as IMAGE_TOO_LARGE.
	big := make([]byte, 3*1024*1024)
	rec, envelope := uploadImage(t, h, token, "big.png", "image/png", big)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized file status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if envelope["errorCode"] != "IMAGE_TOO_LARGE" {
		t.Fatalf("oversized file errorCode = %v, want IMAGE_TOO_LARGE", envelope["errorCode"])
	}

	// A whole request over the MaxBytesReader cap (file cap + multipart slack)
	// is rejected earlier as REQUEST_TOO_LARGE.
	huge := make([]byte, 12*1024*1024)
	rec, envelope = uploadImage(t, h, token, "huge.png", "image/png", huge)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if envelope["errorCode"] != "REQUEST_TOO_LARGE" {
		t.Fatalf("oversized request errorCode = %v, want REQUEST_TOO_LARGE", envelope["errorCode"])
	}
}
