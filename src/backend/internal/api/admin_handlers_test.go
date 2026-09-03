package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

type fakeModerationStore struct {
	admins      map[int]bool
	listCalls   int
	report      store.Report
	reviewInput store.ReviewReportInput
	statusCalls int
}

func (f *fakeModerationStore) IsAdmin(userID int) (bool, error) {
	return f.admins[userID], nil
}

func (f *fakeModerationStore) ListReports(string, int, int) ([]store.Report, int, error) {
	f.listCalls++
	return []store.Report{f.report}, 1, nil
}

func (f *fakeModerationStore) ReviewReport(input store.ReviewReportInput) (store.Report, error) {
	f.reviewInput = input
	f.report.Status = input.Status
	return f.report, nil
}

func (f *fakeModerationStore) SetPromptStatus(int, int, int, string) error {
	f.statusCalls++
	return nil
}

func (f *fakeModerationStore) SetUserStatus(int, int, int, string) error {
	f.statusCalls++
	return nil
}

func (f *fakeModerationStore) ListAuditEvents(int, int) ([]store.AuditEvent, int, error) {
	return []store.AuditEvent{}, 0, nil
}

func newAdminTestHandler(t *testing.T, moderation store.ModerationManager) (*server, http.Handler) {
	t.Helper()
	cfg := config.Config{AppEnv: "test", JWTSecret: "admin-test-secret", JWTExpireHours: 1}
	users := store.NewUserStore()
	s := &server{
		config:       cfg,
		tokenManager: auth.NewTokenManager(cfg.JWTSecret, time.Hour),
		captcha:      newCaptchaManager(),
		userStore:    users,
		promptStore:  store.NewMemoryPromptStore(),
		commentStore: store.NewMemoryCommentStore(),
	}
	h := newServerWithDeps(serverDeps{
		config:          cfg,
		tokenManager:    s.tokenManager,
		captcha:         s.captcha,
		userStore:       s.userStore,
		promptStore:     s.promptStore,
		commentStore:    s.commentStore,
		moderationStore: moderation,
		storageMode:     "memory",
	})
	return s, h
}

func TestAdminEndpointsRequireRoleAndUseSessionActor(t *testing.T) {
	moderation := &fakeModerationStore{
		admins: map[int]bool{1: true},
		report: store.Report{ID: 7, TargetType: "prompt", TargetID: 101, Status: "pending"},
	}
	s, h := newAdminTestHandler(t, moderation)

	regular, found := s.userStore.FindByID(2)
	if !found {
		t.Fatal("expected seeded regular user")
	}
	regularToken, err := s.tokenManager.Generate(regular.ID, regular.Email, regular.SessionVer)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || moderation.listCalls != 0 {
		t.Fatalf("regular admin request status=%d calls=%d body=%s", rec.Code, moderation.listCalls, rec.Body.String())
	}

	admin, found := s.userStore.FindByID(1)
	if !found {
		t.Fatal("expected seeded admin candidate")
	}
	adminToken, err := s.tokenManager.Generate(admin.ID, admin.Email, admin.SessionVer)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports?status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || moderation.listCalls != 1 {
		t.Fatalf("admin list status=%d calls=%d body=%s", rec.Code, moderation.listCalls, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/reports/7", strings.NewReader(`{"status":"reviewed","action":"none","note":"checked"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin review status=%d body=%s", rec.Code, rec.Body.String())
	}
	if moderation.reviewInput.ActorID != admin.ID || moderation.reviewInput.RequestID == "" {
		t.Fatalf("review actor/request not populated: %+v", moderation.reviewInput)
	}
}
