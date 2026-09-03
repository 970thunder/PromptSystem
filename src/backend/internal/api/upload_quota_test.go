package api

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

func TestUploadQuotaReservesConcurrentAndPersistedCapacity(t *testing.T) {
	uploads := store.NewMemoryUploadStore()
	s := &server{
		config:      config.Config{UploadTotalQuotaMB: 1, UploadDailyQuotaMB: 10},
		uploadStore: uploads,
	}

	release, err := s.reserveUploadQuota(context.Background(), 7, 700*1024)
	if err != nil {
		t.Fatalf("first reservation error = %v", err)
	}
	if _, err := s.reserveUploadQuota(context.Background(), 7, 400*1024); !errors.Is(err, errUploadTotalQuota) {
		t.Fatalf("concurrent reservation error = %v, want total quota error", err)
	}

	release()
	if _, err := uploads.RecordUpload(store.UploadRecord{
		OwnerID: 7, ObjectKey: "prompt_image/7/existing.png", Size: 700 * 1024, Status: store.UploadStatusPending,
	}); err != nil {
		t.Fatalf("record existing upload: %v", err)
	}
	if _, err := s.reserveUploadQuota(context.Background(), 7, 400*1024); !errors.Is(err, errUploadTotalQuota) {
		t.Fatalf("persisted reservation error = %v, want total quota error", err)
	}
}

func TestUploadQuotaTracksDailyUsagePerUser(t *testing.T) {
	s := &server{
		config:           config.Config{UploadTotalQuotaMB: 10, UploadDailyQuotaMB: 1},
		uploadStore:      store.NewMemoryUploadStore(),
		uploadDailyUsage: make(map[string]int64),
	}

	release, err := s.reserveUploadQuota(context.Background(), 7, 700*1024)
	if err != nil {
		t.Fatalf("first daily reservation error = %v", err)
	}
	release()
	if _, err := s.reserveUploadQuota(context.Background(), 7, 400*1024); !errors.Is(err, errUploadDailyQuota) {
		t.Fatalf("second daily reservation error = %v, want daily quota error", err)
	}
	release, err = s.reserveUploadQuota(context.Background(), 8, 400*1024)
	if err != nil {
		t.Fatalf("different user should have an independent quota: %v", err)
	}
	release()
}

func TestUploadConcurrencyLimitReturnsRetryableResponse(t *testing.T) {
	s := &server{config: config.Config{UploadMaxConcurrent: 1}}
	first := httptest.NewRecorder()
	if !s.acquireUploadSlot(first) {
		t.Fatal("first upload should acquire the only slot")
	}
	defer s.releaseUploadSlot()

	second := httptest.NewRecorder()
	if s.acquireUploadSlot(second) {
		t.Fatal("second upload should be rejected while the slot is busy")
	}
	if second.Code != 429 {
		t.Fatalf("concurrency status = %d, want 429", second.Code)
	}
	if second.Header().Get("Retry-After") != "1" {
		t.Fatalf("Retry-After = %q, want 1", second.Header().Get("Retry-After"))
	}
}
