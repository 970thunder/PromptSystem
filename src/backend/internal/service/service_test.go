package service

import (
	"context"
	"errors"
	"testing"

	"promptos-backend/internal/store"
)

type failOnceReferenceUploads struct {
	*store.MemoryUploadStore
	failNext bool
}

func (s *failOnceReferenceUploads) MarkUploadsReferenced(keys []string, ownerID int) error {
	if s.failNext {
		s.failNext = false
		return errors.New("injected upload reference failure")
	}
	return s.MemoryUploadStore.MarkUploadsReferenced(keys, ownerID)
}

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

	if err := s.ValidateUploadOwnership(1, "/uploads/prompt_image/1/cover.png", nil); err != nil {
		t.Fatalf("owner validation error = %v", err)
	}
	if err := s.MarkUploadsReferenced(1, "/uploads/prompt_image/1/cover.png", nil); err != nil {
		t.Fatalf("mark upload reference error = %v", err)
	}
	record, found, err := uploads.FindUpload("prompt_image/1/cover.png")
	if err != nil || !found || record.Status != store.UploadStatusReferenced {
		t.Fatalf("upload after validation = %#v found=%v err=%v, want referenced", record, found, err)
	}
	if err := s.ValidateUploadOwnership(2, "/uploads/prompt_image/1/cover.png", nil); !errors.Is(err, ErrInvalidUploadOwnership) {
		t.Fatalf("other owner error = %v, want ErrInvalidUploadOwnership", err)
	}
}

func TestPromptServiceReconcilesReplacedUploadReferences(t *testing.T) {
	uploads := store.NewMemoryUploadStore()
	oldKey := "prompt_image/91/old.png"
	newKey := "prompt_image/91/new.png"
	for _, key := range []string{oldKey, newKey} {
		if _, err := uploads.RecordUpload(store.UploadRecord{OwnerID: 91, ObjectKey: key, Status: store.UploadStatusReferenced}); err != nil {
			t.Fatalf("record upload %s: %v", key, err)
		}
	}
	if _, err := uploads.UnreferenceUploadsByOwner(91, []string{newKey}); err != nil {
		t.Fatalf("unreference old upload: %v", err)
	}
	old, found, err := uploads.FindUpload(oldKey)
	if err != nil || !found || old.Status != store.UploadStatusPending {
		t.Fatalf("old upload = %#v found=%v err=%v, want pending", old, found, err)
	}
	newRecord, found, err := uploads.FindUpload(newKey)
	if err != nil || !found || newRecord.Status != store.UploadStatusReferenced {
		t.Fatalf("new upload = %#v found=%v err=%v, want referenced", newRecord, found, err)
	}
}

func TestPromptServiceCompensatesUploadReferenceFailure(t *testing.T) {
	const ownerID = 92
	prompts := store.NewMemoryPromptStore()
	uploads := &failOnceReferenceUploads{
		MemoryUploadStore: store.NewMemoryUploadStore(),
		failNext:          true,
	}
	key := "prompt_image/92/cover.png"
	if _, err := uploads.RecordUpload(store.UploadRecord{
		OwnerID:   ownerID,
		ObjectKey: key,
		Purpose:   store.UploadPurposePromptImage,
		Status:    store.UploadStatusPending,
	}); err != nil {
		t.Fatalf("record upload: %v", err)
	}
	if _, err := prompts.Create(store.CreatePromptInput{
		Title:      "compensation",
		Content:    "content",
		Cover:      "/uploads/" + key,
		Model:      "gpt-4o",
		CategoryID: 1,
		User:       store.User{ID: ownerID, Username: "compensating-owner", Status: 1},
		Status:     0,
	}); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	s := NewPromptService(prompts, uploads, nil)
	if err := s.FinalizeUploadReferences(ownerID, "/uploads/"+key, nil); err != nil {
		t.Fatalf("finalize upload references: %v", err)
	}
	record, found, err := uploads.FindUpload(key)
	if err != nil || !found {
		t.Fatalf("find repaired upload: record=%#v found=%v err=%v", record, found, err)
	}
	if record.Status != store.UploadStatusReferenced {
		t.Fatalf("repaired upload status = %q, want referenced", record.Status)
	}
}
