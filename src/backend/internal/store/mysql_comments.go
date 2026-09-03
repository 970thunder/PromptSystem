package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type MySQLCommentStore struct {
	db *sql.DB
}

func NewMySQLCommentStore(db *sql.DB) *MySQLCommentStore {
	return &MySQLCommentStore{db: db}
}

func (s *MySQLCommentStore) ListByTarget(filter CommentFilter) ([]Comment, error) {
	comments, _, err := s.listByTargetPage(filter, 1, 100)
	return comments, err
}

func (s *MySQLCommentStore) ListByTargetPage(filter CommentFilter, page, pageSize int) ([]Comment, int, error) {
	return s.listByTargetPage(filter, page, pageSize)
}

func (s *MySQLCommentStore) listByTargetPage(filter CommentFilter, page, pageSize int) ([]Comment, int, error) {
	targetType := strings.TrimSpace(strings.ToLower(filter.TargetType))
	if targetType != "prompt" {
		return []Comment{}, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	sortBy := normalizeCommentSort(filter.SortBy)
	orderBy := "c.created_at DESC, c.id DESC"
	switch sortBy {
	case "oldest":
		orderBy = "c.created_at ASC, c.id ASC"
	case "popular":
		orderBy = "c.likes DESC, c.created_at DESC, c.id DESC"
	}

	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM comments WHERE target_type = ? AND target_id = ? AND parent_id IS NULL`,
		targetType, filter.TargetID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
		SELECT
			c.id, c.target_type, c.target_id, c.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			c.content, c.likes, c.parent_id, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.target_type = ? AND c.target_id = ? AND c.parent_id IS NULL
		ORDER BY `+orderBy+`
		LIMIT ? OFFSET ?
	`, targetType, filter.TargetID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	flat := make([]Comment, 0)
	for rows.Next() {
		comment, err := scanComment(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		flat = append(flat, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return buildCommentTree(flat, sortBy), total, nil
}

func (s *MySQLCommentStore) Create(input CreateCommentInput) (Comment, error) {
	if err := validateCommentInput(input); err != nil {
		return Comment{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Comment{}, err
	}
	defer tx.Rollback()

	if !s.promptExists(tx, input.TargetID) {
		return Comment{}, ErrPromptNotFound
	}

	var parentID any
	if input.ParentID != nil {
		parent, found, err := s.findCommentByIDTx(tx, *input.ParentID)
		if err != nil {
			return Comment{}, err
		}
		if !found {
			return Comment{}, ErrCommentParentNotFound
		}
		if parent.TargetType != "prompt" || parent.TargetID != input.TargetID {
			return Comment{}, ErrCommentParentMismatch
		}
		parentID = parent.ID
	}

	result, err := tx.Exec(`
		INSERT INTO comments (target_type, target_id, user_id, content, parent_id, likes)
		VALUES (?, ?, ?, ?, ?, 0)
	`, "prompt", input.TargetID, input.User.ID, strings.TrimSpace(input.Content), parentID)
	if err != nil {
		return Comment{}, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return Comment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Comment{}, err
	}

	comment, found, err := s.findCommentByID(int(commentID))
	if err != nil {
		return Comment{}, err
	}
	if !found {
		return Comment{}, ErrCommentNotFound
	}

	return comment, nil
}

func (s *MySQLCommentStore) Like(id int, userID int) (Comment, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Comment{}, false, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"INSERT IGNORE INTO likes (user_id, target_type, target_id) VALUES (?, 'comment', ?)",
		userID,
		id,
	)
	if err != nil {
		return Comment{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Comment{}, false, err
	}
	applied := affected > 0

	if applied {
		updateResult, err := tx.Exec(
			"UPDATE comments SET likes = likes + 1 WHERE id = ?",
			id,
		)
		if err != nil {
			return Comment{}, false, err
		}

		updatedRows, err := updateResult.RowsAffected()
		if err != nil {
			return Comment{}, false, err
		}
		if updatedRows == 0 {
			return Comment{}, false, ErrCommentNotFound
		}
	}

	if err := tx.Commit(); err != nil {
		return Comment{}, false, err
	}

	comment, found, err := s.findCommentByID(id)
	if err != nil {
		return Comment{}, false, err
	}
	if !found {
		return Comment{}, false, ErrCommentNotFound
	}

	return comment, applied, nil
}

func (s *MySQLCommentStore) Report(input ReportCommentInput) (Report, bool, error) {
	if err := validateReportCommentInput(input); err != nil {
		return Report{}, false, err
	}

	if _, found, err := s.findCommentByID(input.CommentID); err != nil {
		return Report{}, false, err
	} else if !found {
		return Report{}, false, ErrCommentNotFound
	}

	result, err := s.db.Exec(`
		INSERT IGNORE INTO reports (user_id, target_type, target_id, reason, detail, status)
		VALUES (?, 'comment', ?, ?, ?, 'pending')
	`, input.UserID, input.CommentID, strings.TrimSpace(input.Reason), strings.TrimSpace(input.Detail))
	if err != nil {
		return Report{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Report{}, false, err
	}

	report, found, err := s.findReport(input.UserID, "comment", input.CommentID)
	if err != nil {
		return Report{}, false, err
	}
	if !found {
		return Report{}, false, ErrReportNotFound
	}

	return report, affected > 0, nil
}

func (s *MySQLCommentStore) promptExists(tx *sql.Tx, id int) bool {
	var exists int
	if err := tx.QueryRow(`SELECT 1 FROM prompts WHERE id = ? AND status = 1 LIMIT 1`, id).Scan(&exists); err != nil {
		return false
	}

	return exists == 1
}

func (s *MySQLCommentStore) findCommentByID(id int) (Comment, bool, error) {
	row := s.db.QueryRow(`
		SELECT
			c.id, c.target_type, c.target_id, c.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			c.content, c.likes, c.parent_id, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = ?
	`, id)

	comment, err := scanComment(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, false, nil
	}
	if err != nil {
		return Comment{}, false, err
	}

	return comment, true, nil
}

func (s *MySQLCommentStore) findCommentByIDTx(tx *sql.Tx, id int) (Comment, bool, error) {
	row := tx.QueryRow(`
		SELECT
			c.id, c.target_type, c.target_id, c.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			c.content, c.likes, c.parent_id, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = ?
	`, id)

	comment, err := scanComment(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, false, nil
	}
	if err != nil {
		return Comment{}, false, err
	}

	return comment, true, nil
}

func (s *MySQLCommentStore) findReport(userID int, targetType string, targetID int) (Report, bool, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, target_type, target_id, reason, detail, status, created_at
		FROM reports
		WHERE user_id = ? AND target_type = ? AND target_id = ?
	`, userID, targetType, targetID)

	var (
		report    Report
		createdAt time.Time
	)
	if err := row.Scan(
		&report.ID,
		&report.UserID,
		&report.TargetType,
		&report.TargetID,
		&report.Reason,
		&report.Detail,
		&report.Status,
		&createdAt,
	); errors.Is(err, sql.ErrNoRows) {
		return Report{}, false, nil
	} else if err != nil {
		return Report{}, false, err
	}

	report.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return report, true, nil
}

func scanComment(scan func(dest ...any) error) (Comment, error) {
	var (
		comment       Comment
		userAvatar    sql.NullString
		userBio       sql.NullString
		parentID      sql.NullInt64
		userCreatedAt time.Time
		commentTime   time.Time
	)

	err := scan(
		&comment.ID,
		&comment.TargetType,
		&comment.TargetID,
		&comment.UserID,
		&comment.User.Username,
		&userAvatar,
		&comment.User.Email,
		&userBio,
		&comment.User.Level,
		&comment.User.Experience,
		&comment.User.Status,
		&userCreatedAt,
		&comment.Content,
		&comment.Likes,
		&parentID,
		&commentTime,
	)
	if err != nil {
		return Comment{}, err
	}

	comment.User.ID = comment.UserID
	comment.User.Avatar = userAvatar.String
	comment.User.Bio = userBio.String
	comment.User.CreatedAt = userCreatedAt.UTC().Format(time.DateOnly)
	if parentID.Valid {
		value := int(parentID.Int64)
		comment.ParentID = &value
	}
	comment.Replies = []Comment{}
	comment.CreatedAt = commentTime.UTC().Format(time.RFC3339)

	return comment, nil
}
