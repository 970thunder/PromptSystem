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

func (s *MySQLCommentStore) ListByTarget(targetType string, targetID int) ([]Comment, error) {
	targetType = strings.TrimSpace(strings.ToLower(targetType))
	if targetType != "prompt" {
		return []Comment{}, nil
	}

	rows, err := s.db.Query(`
		SELECT
			c.id, c.target_type, c.target_id, c.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			c.content, c.likes, c.parent_id, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.target_type = ? AND c.target_id = ?
		ORDER BY c.created_at ASC, c.id ASC
	`, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flat := make([]Comment, 0)
	for rows.Next() {
		comment, err := scanComment(rows.Scan)
		if err != nil {
			return nil, err
		}
		flat = append(flat, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return buildCommentTree(flat), nil
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
		return Comment{}, errors.New("prompt not found")
	}

	var parentID any
	if input.ParentID != nil {
		parent, found, err := s.findCommentByIDTx(tx, *input.ParentID)
		if err != nil {
			return Comment{}, err
		}
		if !found {
			return Comment{}, errors.New("parent comment not found")
		}
		if parent.TargetType != "prompt" || parent.TargetID != input.TargetID {
			return Comment{}, errors.New("parent comment does not match prompt")
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
		return Comment{}, errors.New("comment not found")
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
			return Comment{}, false, errors.New("comment not found")
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
		return Comment{}, false, errors.New("comment not found")
	}

	return comment, applied, nil
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
