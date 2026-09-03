package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
)

// MySQLUploadStore persists upload metadata in the uploads table. It is the
// source of truth for ownership and lifecycle, independent of the storage
// backend (local disk or R2).
type MySQLUploadStore struct {
	db *sql.DB
}

func NewMySQLUploadStore(db *sql.DB) *MySQLUploadStore {
	return &MySQLUploadStore{db: db}
}

func (s *MySQLUploadStore) RecordUpload(rec UploadRecord) (UploadRecord, error) {
	provider := rec.Provider
	if provider == "" {
		provider = "local"
	}
	status := rec.Status
	if status == "" {
		status = UploadStatusPending
	}

	if _, err := s.db.Exec(`
		INSERT INTO uploads (owner_id, provider, purpose, object_key, content_type, size, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			owner_id = VALUES(owner_id),
			provider = VALUES(provider),
			purpose = VALUES(purpose),
			content_type = VALUES(content_type),
			size = VALUES(size),
			status = VALUES(status),
			updated_at = CURRENT_TIMESTAMP
	`, rec.OwnerID, provider, string(rec.Purpose), rec.ObjectKey, rec.ContentType, rec.Size, string(status)); err != nil {
		return UploadRecord{}, err
	}

	stored, _, err := s.FindUpload(rec.ObjectKey)
	if err != nil {
		return UploadRecord{}, err
	}
	return stored, nil
}

func (s *MySQLUploadStore) MarkUploadsReferenced(objectKeys []string, ownerID int) error {
	if len(objectKeys) == 0 {
		return nil
	}
	// A prompt may reference several images. Mark the whole set atomically so a
	// transient database error cannot leave only a prefix protected from cleanup.
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, key := range objectKeys {
		if _, err := tx.Exec(`
			UPDATE uploads
			SET status = ?, updated_at = CURRENT_TIMESTAMP
			WHERE object_key = ? AND owner_id = ? AND status <> ?
		`, string(UploadStatusReferenced), key, ownerID, string(UploadStatusTrashed)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLUploadStore) FindUpload(objectKey string) (UploadRecord, bool, error) {
	row := s.db.QueryRow(`
		SELECT id, owner_id, provider, purpose, object_key, content_type, size, status, created_at
		FROM uploads
		WHERE object_key = ?
	`, objectKey)

	var (
		rec       UploadRecord
		purpose   string
		status    string
		createdAt time.Time
	)
	if err := row.Scan(
		&rec.ID, &rec.OwnerID, &rec.Provider, &purpose, &rec.ObjectKey,
		&rec.ContentType, &rec.Size, &status, &createdAt,
	); errors.Is(err, sql.ErrNoRows) {
		return UploadRecord{}, false, nil
	} else if err != nil {
		return UploadRecord{}, false, err
	}

	rec.Purpose = UploadPurpose(purpose)
	rec.Status = UploadStatus(status)
	rec.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return rec, true, nil
}

func (s *MySQLUploadStore) SoftDeleteUpload(objectKey string, ownerID int) error {
	_, err := s.db.Exec(`
		UPDATE uploads
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE object_key = ? AND owner_id = ?
	`, string(UploadStatusTrashed), objectKey, ownerID)
	return err
}

func (s *MySQLUploadStore) ListUnreferencedUploads(olderThan time.Time) ([]UploadRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, owner_id, provider, purpose, object_key, content_type, size, status, created_at
		FROM uploads
		WHERE status = ? AND created_at < ?
	`, string(UploadStatusPending), olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]UploadRecord, 0)
	for rows.Next() {
		rec, err := scanUploadRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *MySQLUploadStore) TrashUnreferenced(olderThan time.Time) ([]string, error) {
	unref, err := s.ListUnreferencedUploads(olderThan)
	if err != nil {
		return nil, err
	}
	if len(unref) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(unref))
	args := make([]any, 0, len(unref))
	placeholders := make([]string, 0, len(unref))
	for _, rec := range unref {
		keys = append(keys, rec.ObjectKey)
		args = append(args, rec.ObjectKey)
		placeholders = append(placeholders, "?")
	}

	if _, err := s.db.Exec(
		`UPDATE uploads SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE status = ? AND object_key IN (`+strings.Join(placeholders, ",")+`)`,
		append([]any{string(UploadStatusTrashed), string(UploadStatusPending)}, args...)...,
	); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *MySQLUploadStore) UnreferenceUploadsByOwner(ownerID int, referencedKeys []string) ([]string, error) {
	query := `SELECT object_key FROM uploads WHERE owner_id = ? AND status = ?`
	args := []any{ownerID, string(UploadStatusReferenced)}
	if len(referencedKeys) > 0 {
		placeholders := make([]string, len(referencedKeys))
		for i, key := range referencedKeys {
			placeholders[i] = "?"
			args = append(args, key)
		}
		query += ` AND object_key NOT IN (` + strings.Join(placeholders, ",") + `)`
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(keys) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(keys))
	updateArgs := []any{string(UploadStatusPending), ownerID, string(UploadStatusReferenced)}
	for i, key := range keys {
		placeholders[i] = "?"
		updateArgs = append(updateArgs, key)
	}
	_, err = s.db.Exec(`UPDATE uploads SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_id = ? AND status = ? AND object_key IN (`+strings.Join(placeholders, ",")+")", updateArgs...)
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	return keys, nil
}

// ActiveUploadBytes reports the size of uploads that still occupy storage.
func (s *MySQLUploadStore) ActiveUploadBytes() (int64, error) {
	var total sql.NullInt64
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(size), 0)
		FROM uploads
		WHERE status <> ?
	`, string(UploadStatusTrashed)).Scan(&total); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

func scanUploadRow(scan func(dest ...any) error) (UploadRecord, error) {
	var (
		rec       UploadRecord
		purpose   string
		status    string
		createdAt time.Time
	)
	if err := scan(
		&rec.ID, &rec.OwnerID, &rec.Provider, &purpose, &rec.ObjectKey,
		&rec.ContentType, &rec.Size, &status, &createdAt,
	); err != nil {
		return UploadRecord{}, err
	}
	rec.Purpose = UploadPurpose(purpose)
	rec.Status = UploadStatus(status)
	rec.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return rec, nil
}
