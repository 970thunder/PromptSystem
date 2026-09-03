package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"promptos-backend/internal/config"
)

func TestLocalStorageDeleteIsBoundToBaseDir(t *testing.T) {
	base := t.TempDir()
	s, err := newLocalStorage(config.Config{UploadDir: base})
	if err != nil {
		t.Fatalf("newLocalStorage: %v", err)
	}
	if _, err := s.Save(context.Background(), "prompt_image/1/cover.png", "image/png", []byte("x")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Delete(context.Background(), "prompt_image/1/cover.png"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "prompt_image", "1", "cover.png")); !os.IsNotExist(err) {
		t.Fatalf("deleted object still exists, stat err=%v", err)
	}

	outside := filepath.Join(filepath.Dir(base), "outside-upload.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write outside fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })
	if err := s.Delete(context.Background(), "../../outside-upload.txt"); err == nil {
		t.Fatal("path traversal delete unexpectedly succeeded")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside fixture was removed: %v", err)
	}
}
