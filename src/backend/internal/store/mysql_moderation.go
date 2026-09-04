package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MySQLModerationStore contains the privileged moderation boundary. It is kept
// separate from the public Prompt/User stores so ordinary handlers cannot call
// status-changing SQL accidentally.
type MySQLModerationStore struct {
	db *sql.DB
}

func NewMySQLModerationStore(db *sql.DB) *MySQLModerationStore {
	return &MySQLModerationStore{db: db}
}

func (s *MySQLModerationStore) IsAdmin(userID int) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE ur.user_id = ? AND ur.role = 'admin' AND u.status = 1
	`, userID).Scan(&count)
	return count > 0, err
}

func (s *MySQLModerationStore) ListReports(status string, page, pageSize int) ([]Report, int, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" && status != "pending" && status != "reviewed" && status != "rejected" {
		return nil, 0, ErrInvalidModeration
	}
	page, pageSize = normalizeModerationPage(page, pageSize)

	where := ""
	args := []any{}
	if status != "" {
		where = " WHERE status = ?"
		args = append(args, status)
	}
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM reports"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, target_type, target_id, reason, detail, status,
		       created_at, updated_at, reviewed_by, review_note
		FROM reports` + where + `
		ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, created_at ASC, id ASC
		LIMIT ? OFFSET ?`
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]Report, 0, pageSize)
	for rows.Next() {
		report, found, err := scanModerationReport(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		if found {
			result = append(result, report)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (s *MySQLModerationStore) ReviewReport(input ReviewReportInput) (Report, error) {
	if input.ReportID <= 0 || input.ActorID <= 0 {
		return Report{}, ErrInvalidModeration
	}
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Action = strings.TrimSpace(strings.ToLower(input.Action))
	if input.Status != "reviewed" && input.Status != "rejected" {
		return Report{}, ErrInvalidModeration
	}
	if input.Action == "" {
		input.Action = "none"
	}
	if input.Action != "none" && input.Action != "remove" {
		return Report{}, ErrInvalidModeration
	}
	if len([]rune(strings.TrimSpace(input.Note))) > MaxReportDetailRunes {
		return Report{}, ErrReportDetailTooLong
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Report{}, err
	}
	defer tx.Rollback()
	if ok, err := isAdminTx(tx, input.ActorID); err != nil {
		return Report{}, err
	} else if !ok {
		return Report{}, ErrAdminRequired
	}

	report, found, err := scanModerationReport(tx.QueryRow(`
		SELECT id, user_id, target_type, target_id, reason, detail, status,
		       created_at, updated_at, reviewed_by, review_note
		FROM reports WHERE id = ? FOR UPDATE
	`, input.ReportID).Scan)
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, ErrModerationNotFound
	}

	if input.Action == "remove" {
		if report.TargetType != "prompt" && report.TargetType != "skill" {
			// Comments intentionally have no hard-delete moderation path yet;
			// deleting one would leave polymorphic interaction rows orphaned.
			return Report{}, ErrInvalidModeration
		}
		if _, err := tx.Exec(`UPDATE `+report.TargetType+` SET status = -1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status <> -1`, report.TargetID); err != nil {
			return Report{}, err
		}
	}

	if _, err := tx.Exec(`
		UPDATE reports
		SET status = ?, reviewed_by = ?, review_note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, input.Status, input.ActorID, strings.TrimSpace(input.Note), input.ReportID); err != nil {
		return Report{}, err
	}

	metadata := map[string]any{
		"status": input.Status,
		"action": input.Action,
		"reason": report.Reason,
	}
	if strings.TrimSpace(input.Note) != "" {
		metadata["note"] = strings.TrimSpace(input.Note)
	}
	if err := appendAuditTx(tx, input.ActorID, "report.review", report.TargetType, int64(report.TargetID), metadata, input.RequestID); err != nil {
		return Report{}, err
	}
	if err := tx.Commit(); err != nil {
		return Report{}, err
	}

	updated, found, err := scanModerationReport(s.db.QueryRow(`
		SELECT id, user_id, target_type, target_id, reason, detail, status,
		       created_at, updated_at, reviewed_by, review_note
		FROM reports WHERE id = ?
	`, input.ReportID).Scan)
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, ErrModerationNotFound
	}
	return updated, nil
}

func (s *MySQLModerationStore) SetPromptStatus(promptID, actorID, status int, reason string) error {
	if promptID <= 0 || actorID <= 0 || (status != -1 && status != 1) {
		return ErrInvalidModeration
	}
	return s.setTargetStatus("prompts", promptID, actorID, status, reason)
}

func (s *MySQLModerationStore) setTargetStatus(targetType string, targetID, actorID, status int, reason string) error {
	if len([]rune(strings.TrimSpace(reason))) > MaxReportDetailRunes {
		return ErrReportDetailTooLong
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if ok, err := isAdminTx(tx, actorID); err != nil {
		return err
	} else if !ok {
		return ErrAdminRequired
	}
	result, err := tx.Exec(`UPDATE `+targetType+` SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, targetID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrModerationNotFound
	}
	if err := appendAuditTx(tx, actorID, "content.status", targetType, int64(targetID), map[string]any{
		"status": status,
		"reason": strings.TrimSpace(reason),
	}, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLModerationStore) SetUserStatus(userID, actorID, status int, reason string) error {
	if userID <= 0 || actorID <= 0 || (status != 0 && status != 1) {
		return ErrInvalidModeration
	}
	if userID == actorID {
		return ErrCannotModerateSelf
	}
	if len([]rune(strings.TrimSpace(reason))) > MaxReportDetailRunes {
		return ErrReportDetailTooLong
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if ok, err := isAdminTx(tx, actorID); err != nil {
		return err
	} else if !ok {
		return ErrAdminRequired
	}
	var exists int
	if err := tx.QueryRow(`SELECT 1 FROM users WHERE id = ? FOR UPDATE`, userID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return ErrModerationNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE users
		SET status = ?, session_version = session_version + IF(? = 0, 1, 0)
		WHERE id = ?
	`, status, status, userID); err != nil {
		return err
	}
	if err := appendAuditTx(tx, actorID, "user.status", "user", int64(userID), map[string]any{
		"status": status,
		"reason": strings.TrimSpace(reason),
	}, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLModerationStore) ListAuditEvents(page, pageSize int) ([]AuditEvent, int, error) {
	page, pageSize = normalizeModerationPage(page, pageSize)
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(`
		SELECT id, actor_user_id, action, target_type, target_id, metadata,
		       request_id, prev_hash, event_hash, created_at
		FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := make([]AuditEvent, 0, pageSize)
	for rows.Next() {
		var event AuditEvent
		var metadata []byte
		var createdAt time.Time
		if err := rows.Scan(&event.ID, &event.ActorID, &event.Action, &event.TargetType, &event.TargetID, &metadata, &event.RequestID, &event.PrevHash, &event.EventHash, &createdAt); err != nil {
			return nil, 0, err
		}
		event.Metadata = string(metadata)
		event.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func normalizeModerationPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func isAdminTx(tx *sql.Tx, userID int) (bool, error) {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE ur.user_id = ? AND ur.role = 'admin' AND u.status = 1
	`, userID).Scan(&count)
	return count > 0, err
}

func appendAuditTx(tx *sql.Tx, actorID int, action, targetType string, targetID int64, metadata map[string]any, requestID string) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	previousHash := ""
	if err := tx.QueryRow(`SELECT event_hash FROM audit_logs ORDER BY id DESC LIMIT 1 FOR UPDATE`).Scan(&previousHash); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	// MySQL rejects RFC3339's `T`/`Z` layout for TIMESTAMP columns, so the
	// stored timestamp uses the driver-native layout. The canonical hash is
	// computed over exactly this string, keeping the chain self-consistent.
	createdAt := time.Now().UTC().Format("2006-01-02 15:04:05.999999")
	canonical := strings.Join([]string{
		previousHash,
		strconvFormatInt(int64(actorID)),
		action,
		targetType,
		strconvFormatInt(targetID),
		string(metadataJSON),
		requestID,
		createdAt,
	}, "\n")
	digest := sha256.Sum256([]byte(canonical))
	eventHash := hex.EncodeToString(digest[:])
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_user_id, action, target_type, target_id, metadata, request_id, prev_hash, event_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, actorID, action, targetType, targetID, string(metadataJSON), strings.TrimSpace(requestID), previousHash, eventHash, createdAt)
	return err
}

func strconvFormatInt(value int64) string {
	return fmt.Sprintf("%d", value)
}

func scanModerationReport(scan func(dest ...any) error) (Report, bool, error) {
	var (
		report     Report
		createdAt  time.Time
		updatedAt  time.Time
		reviewedBy sql.NullInt64
		reviewNote sql.NullString
	)
	err := scan(&report.ID, &report.UserID, &report.TargetType, &report.TargetID, &report.Reason, &report.Detail, &report.Status, &createdAt, &updatedAt, &reviewedBy, &reviewNote)
	if errors.Is(err, sql.ErrNoRows) {
		return Report{}, false, nil
	}
	if err != nil {
		return Report{}, false, err
	}
	report.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	report.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if reviewedBy.Valid {
		report.ReviewedBy = int(reviewedBy.Int64)
	}
	if reviewNote.Valid {
		report.ReviewNote = reviewNote.String
	}
	return report, true, nil
}
