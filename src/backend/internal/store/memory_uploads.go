package store

import (
	"sort"
	"sync"
	"time"
)

// MemoryUploadStore is an in-memory UploadManager used for tests and the
// development memory fallback. It keeps the same ownership/lifecycle semantics
// as the MySQL implementation.
type MemoryUploadStore struct {
	mu      sync.RWMutex
	uploads map[string]UploadRecord
	nextID  int64
}

func NewMemoryUploadStore() *MemoryUploadStore {
	return &MemoryUploadStore{uploads: make(map[string]UploadRecord)}
}

func (s *MemoryUploadStore) RecordUpload(rec UploadRecord) (UploadRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider := rec.Provider
	if provider == "" {
		provider = "local"
	}
	status := rec.Status
	if status == "" {
		status = UploadStatusPending
	}

	existing, ok := s.uploads[rec.ObjectKey]
	if ok {
		existing.OwnerID = rec.OwnerID
		existing.Provider = provider
		existing.Purpose = rec.Purpose
		existing.ContentType = rec.ContentType
		existing.Size = rec.Size
		existing.Status = status
		s.uploads[rec.ObjectKey] = existing
		return existing, nil
	}

	s.nextID++
	rec.ID = s.nextID
	rec.Provider = provider
	rec.Status = status
	rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	s.uploads[rec.ObjectKey] = rec
	return rec, nil
}

func (s *MemoryUploadStore) MarkUploadsReferenced(objectKeys []string, ownerID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range objectKeys {
		if rec, ok := s.uploads[key]; ok && rec.OwnerID == ownerID && rec.Status != UploadStatusTrashed {
			rec.Status = UploadStatusReferenced
			s.uploads[key] = rec
		}
	}
	return nil
}

func (s *MemoryUploadStore) FindUpload(objectKey string) (UploadRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.uploads[objectKey]
	if !ok {
		return UploadRecord{}, false, nil
	}
	return rec, true, nil
}

func (s *MemoryUploadStore) SoftDeleteUpload(objectKey string, ownerID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.uploads[objectKey]; ok && rec.OwnerID == ownerID {
		rec.Status = UploadStatusTrashed
		s.uploads[objectKey] = rec
	}
	return nil
}

func (s *MemoryUploadStore) ListUnreferencedUploads(olderThan time.Time) ([]UploadRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	threshold := olderThan.UTC()
	list := make([]UploadRecord, 0)
	for _, rec := range s.uploads {
		if rec.Status != UploadStatusPending {
			continue
		}
		created, err := time.Parse(time.RFC3339, rec.CreatedAt)
		if err != nil {
			// If the timestamp is unparseable, treat the record as old enough.
			list = append(list, rec)
			continue
		}
		if created.UTC().Before(threshold) {
			list = append(list, rec)
		}
	}
	return list, nil
}

func (s *MemoryUploadStore) TrashUnreferenced(olderThan time.Time) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threshold := olderThan.UTC()
	var keys []string
	for key, rec := range s.uploads {
		if rec.Status != UploadStatusPending {
			continue
		}
		created, err := time.Parse(time.RFC3339, rec.CreatedAt)
		oldEnough := err != nil || created.UTC().Before(threshold)
		if oldEnough {
			rec.Status = UploadStatusTrashed
			s.uploads[key] = rec
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *MemoryUploadStore) UnreferenceUploadsByOwner(ownerID int, referencedKeys []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := make(map[string]struct{}, len(referencedKeys))
	for _, key := range referencedKeys {
		keep[key] = struct{}{}
	}
	var changed []string
	for key, rec := range s.uploads {
		if rec.OwnerID != ownerID || rec.Status != UploadStatusReferenced {
			continue
		}
		if _, ok := keep[key]; ok {
			continue
		}
		rec.Status = UploadStatusPending
		s.uploads[key] = rec
		changed = append(changed, key)
	}
	sort.Strings(changed)
	return changed, nil
}

// ActiveUploadBytes reports the size of uploads that still occupy storage.
func (s *MemoryUploadStore) ActiveUploadBytes() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total int64
	for _, rec := range s.uploads {
		if rec.Status != UploadStatusTrashed {
			total += rec.Size
		}
	}
	return total, nil
}
