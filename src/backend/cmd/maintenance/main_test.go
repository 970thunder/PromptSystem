package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

func TestSameStorageProvider(t *testing.T) {
	tests := []struct {
		recorded, configured string
		want                 bool
	}{
		{"local", "local", true},
		{"", "local", true},
		{"rustfs", "s3", true},
		{"r2", "local", false},
	}
	for _, tt := range tests {
		if got := sameStorageProvider(tt.recorded, tt.configured); got != tt.want {
			t.Errorf("sameStorageProvider(%q, %q) = %v, want %v", tt.recorded, tt.configured, got, tt.want)
		}
	}
}

type fakeImageStorage struct {
	deleted []string
	err     error
}

func (f *fakeImageStorage) Save(context.Context, string, string, []byte) (string, error) {
	return "", nil
}
func (f *fakeImageStorage) Delete(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, key)
	return nil
}

var _ storage.ImageStorage = (*fakeImageStorage)(nil)

func TestCleanupUploadRecordsDeletesOnlySafePendingObjects(t *testing.T) {
	uploads := store.NewMemoryUploadStore()
	oldKey := "prompt_image/1/old.png"
	keepKey := "prompt_image/1/keep.png"
	for _, record := range []store.UploadRecord{
		{OwnerID: 1, ObjectKey: oldKey, Provider: "local", Size: 10, Status: store.UploadStatusPending, CreatedAt: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)},
		{OwnerID: 1, ObjectKey: keepKey, Provider: "local", Size: 10, Status: store.UploadStatusPending, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	} {
		if _, err := uploads.RecordUpload(record); err != nil {
			t.Fatalf("record upload: %v", err)
		}
	}
	storage := &fakeImageStorage{}
	trashed, failures, err := cleanupUploadRecords("local", uploads, storage, 24*time.Hour)
	if err != nil || trashed != 1 || failures != 0 {
		t.Fatalf("cleanup = trashed %d failures %d err %v, want 1/0/nil", trashed, failures, err)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != oldKey {
		t.Fatalf("deleted keys = %#v, want [%q]", storage.deleted, oldKey)
	}
	old, found, err := uploads.FindUpload(oldKey)
	if err != nil || !found || old.Status != store.UploadStatusTrashed {
		t.Fatalf("old record = %#v found=%v err=%v, want trashed", old, found, err)
	}
	if _, found, err := uploads.FindUpload(keepKey); err != nil || !found {
		t.Fatalf("recent pending record unexpectedly missing: found=%v err=%v", found, err)
	}
}

func TestCleanupUploadRecordsRetainsFailedOrMismatchedObjects(t *testing.T) {
	uploads := store.NewMemoryUploadStore()
	failedKey := "prompt_image/1/failed.png"
	mismatchedKey := "prompt_image/1/rustfs.png"
	for _, record := range []store.UploadRecord{
		{OwnerID: 1, ObjectKey: failedKey, Provider: "local", Status: store.UploadStatusPending, CreatedAt: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)},
		{OwnerID: 1, ObjectKey: mismatchedKey, Provider: "rustfs", Status: store.UploadStatusPending, CreatedAt: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)},
	} {
		if _, err := uploads.RecordUpload(record); err != nil {
			t.Fatalf("record upload: %v", err)
		}
	}
	storage := &fakeImageStorage{err: errors.New("temporary object store failure")}
	trashed, failures, err := cleanupUploadRecords("local", uploads, storage, 24*time.Hour)
	if err != nil || trashed != 0 || failures != 2 {
		t.Fatalf("cleanup = trashed %d failures %d err %v, want 0/2/nil", trashed, failures, err)
	}
	for _, key := range []string{failedKey, mismatchedKey} {
		record, found, findErr := uploads.FindUpload(key)
		if findErr != nil || !found || record.Status != store.UploadStatusPending {
			t.Fatalf("record %s = %#v found=%v err=%v, want pending", key, record, found, findErr)
		}
	}
}
