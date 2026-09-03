package service

import (
	"context"
	"errors"
	"testing"

	"promptos-backend/internal/store"
)

func TestPromptServiceInvalidatesOnlyAppliedInteractions(t *testing.T) {
	prompts := store.NewMemoryPromptStore()
	invalidations := 0
	s := NewPromptService(prompts, nil, func(context.Context) { invalidations++ })

	if _, applied, err := s.Like(context.Background(), 101, 1); err != nil || !applied {
		t.Fatalf("first like = applied %v err %v, want applied", applied, err)
	}
	if invalidations != 1 {
		t.Fatalf("invalidations after first like = %d, want 1", invalidations)
	}
	if _, applied, err := s.Like(context.Background(), 101, 1); err != nil || applied {
		t.Fatalf("duplicate like = applied %v err %v, want no-op", applied, err)
	}
	if invalidations != 1 {
		t.Fatalf("invalidations after duplicate like = %d, want 1", invalidations)
	}
}

func TestPromptServiceValidatesAndMarksUploadOwnership(t *testing.T) {
	uploads := store.NewMemoryUploadStore()
	if _, err := uploads.RecordUpload(store.UploadRecord{
		OwnerID: 1, ObjectKey: "prompt_image/1/cover.png", Status: store.UploadStatusPending,
	}); err != nil {
		t.Fatalf("record upload: %v", err)
	}
	s := NewPromptService(nil, uploads, nil)

	if err := s.ValidateAndMarkUploadOwnership(1, "/uploads/prompt_image/1/cover.png", nil); err != nil {
		t.Fatalf("owner validation error = %v", err)
	}
	record, found, err := uploads.FindUpload("prompt_image/1/cover.png")
	if err != nil || !found || record.Status != store.UploadStatusReferenced {
		t.Fatalf("upload after validation = %#v found=%v err=%v, want referenced", record, found, err)
	}
	if err := s.ValidateAndMarkUploadOwnership(2, "/uploads/prompt_image/1/cover.png", nil); !errors.Is(err, ErrInvalidUploadOwnership) {
		t.Fatalf("other owner error = %v, want ErrInvalidUploadOwnership", err)
	}
}
