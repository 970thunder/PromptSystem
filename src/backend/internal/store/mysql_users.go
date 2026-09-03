package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type MySQLUserStore struct {
	db *sql.DB
}

func NewMySQLUserStore(db *sql.DB) *MySQLUserStore {
	return &MySQLUserStore{db: db}
}

const userSelectColumns = `
	id, username, avatar, email, github_id, password, bio, level, experience, session_version, status, created_at
`

const qualifiedUserSelectColumns = `
	users.id, users.username, users.avatar, users.email, users.github_id, users.password, users.bio,
	users.level, users.experience, users.session_version, users.status, users.created_at
`

func (s *MySQLUserStore) Register(username, email, password string) (AuthUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)

	if !IsValidEmail(email) {
		return AuthUser{}, ErrInvalidEmail
	}

	if len(password) < 8 {
		return AuthUser{}, ErrWeakPassword
	}
	if len(password) > maxPasswordBytes {
		return AuthUser{}, ErrPasswordTooLong
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthUser{}, err
	}

	result, err := s.db.Exec(`
		INSERT INTO users (username, avatar, email, password, bio, level, experience, status)
		VALUES (?, '', ?, ?, 'New PromptOS creator', 1, 0, 1)
	`, username, email, string(passwordHash))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return AuthUser{}, ErrUserExists
		}
		return AuthUser{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return AuthUser{}, err
	}

	user, ok := s.FindByID(int(id))
	if !ok {
		return AuthUser{}, ErrUserNotFound
	}

	return user, nil
}

func (s *MySQLUserStore) Authenticate(email, password string) (AuthUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, found, err := s.findByEmail(email)
	if err != nil {
		return AuthUser{}, err
	}
	if !found {
		return AuthUser{}, ErrInvalidCredentials
	}

	if user.PasswordHash == "" {
		return AuthUser{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthUser{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *MySQLUserStore) ResetPassword(email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	if !IsValidEmail(email) {
		return ErrInvalidEmail
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if len(password) > maxPasswordBytes {
		return ErrPasswordTooLong
	}

	if _, found, err := s.findByEmail(email); err != nil {
		return err
	} else if !found {
		return ErrUserNotFound
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	result, err := s.db.Exec(`
		UPDATE users
		SET password = ?, session_version = session_version + 1
		WHERE email = ?
	`, string(passwordHash), email)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// BumpSessionVersion increments the user's session version so all previously
// issued tokens are rejected after a password reset.
func (s *MySQLUserStore) BumpSessionVersion(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	result, err := s.db.Exec(`
		UPDATE users SET session_version = session_version + 1 WHERE email = ?
	`, email)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DeleteAccount performs the account-retention transition in one transaction.
// Rows referenced by foreign keys are retained for audit, while personal
// interaction/history rows are removed and denormalized counters adjusted.
func (s *MySQLUserStore) DeleteAccount(id int) error {
	if id <= 0 {
		return ErrUserNotFound
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status int
	err = tx.QueryRow(`SELECT status FROM users WHERE id = ? FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if status == 0 {
		return nil
	}

	// Remove the user's interaction rows without leaving counters inflated.
	// The correlated counts are evaluated inside the same transaction as the
	// deletes, so a rollback preserves both detail rows and counters.
	if _, err := tx.Exec(`
		UPDATE prompts p
		SET p.likes = GREATEST(0, p.likes - (SELECT COUNT(*) FROM likes l WHERE l.user_id = ? AND l.target_type = 'prompt' AND l.target_id = p.id)),
		    p.favorites = GREATEST(0, p.favorites - (SELECT COUNT(*) FROM favorites f WHERE f.user_id = ? AND f.target_type = 'prompt' AND f.target_id = p.id))
		WHERE EXISTS (SELECT 1 FROM likes l WHERE l.user_id = ? AND l.target_type = 'prompt' AND l.target_id = p.id)
		   OR EXISTS (SELECT 1 FROM favorites f WHERE f.user_id = ? AND f.target_type = 'prompt' AND f.target_id = p.id)
	`, id, id, id, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE comments c
		SET c.likes = GREATEST(0, c.likes - (SELECT COUNT(*) FROM likes l WHERE l.user_id = ? AND l.target_type = 'comment' AND l.target_id = c.id))
		WHERE EXISTS (SELECT 1 FROM likes l WHERE l.user_id = ? AND l.target_type = 'comment' AND l.target_id = c.id)
	`, id, id); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM view_histories WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM favorites WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM likes WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM reports WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM follows WHERE follower_id = ? OR following_id = ?`, id, id); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE users
		SET username = ?, email = ?, github_id = NULL, password = NULL,
		    avatar = NULL, bio = NULL, status = 0, session_version = session_version + 1
		WHERE id = ?
	`, fmt.Sprintf("deleted-user-%d", id), fmt.Sprintf("deleted+%d@invalid.promptos.local", id), id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *MySQLUserStore) FindByID(id int) (AuthUser, bool) {
	row := s.db.QueryRow(`SELECT`+userSelectColumns+` FROM users WHERE id = ?`, id)

	user, found, err := scanAuthUser(row.Scan)
	if err != nil || !found {
		return AuthUser{}, false
	}

	return user, true
}

func (s *MySQLUserStore) UpdateProfile(id int, username, bio, avatar string) (AuthUser, error) {
	current, found := s.FindByID(id)
	if !found {
		return AuthUser{}, ErrUserNotFound
	}

	username = strings.TrimSpace(username)
	bio = strings.TrimSpace(bio)
	avatar = strings.TrimSpace(avatar)

	if username == "" {
		username = current.Username
	}
	if bio == "" {
		bio = current.Bio
	}
	if avatar == "" {
		avatar = current.Avatar
	}

	if _, err := s.db.Exec(`
		UPDATE users
		SET username = ?, bio = ?, avatar = ?
		WHERE id = ?
	`, username, bio, avatar, id); err != nil {
		return AuthUser{}, err
	}

	updated, found := s.FindByID(id)
	if !found {
		return AuthUser{}, ErrUserNotFound
	}

	return updated, nil
}

func (s *MySQLUserStore) UpsertGitHubUser(githubID int64, username, email, avatar string) (AuthUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	avatar = strings.TrimSpace(avatar)

	if githubID <= 0 {
		return AuthUser{}, ErrInvalidGitHubUser
	}
	if !IsValidEmail(email) {
		return AuthUser{}, ErrInvalidEmail
	}
	if strings.TrimSpace(username) == "" {
		username = strings.Split(email, "@")[0]
	}

	if user, found, err := s.findByGitHubID(githubID); err != nil {
		return AuthUser{}, err
	} else if found {
		return s.updateGitHubProfile(user.ID, username, avatar, githubID)
	}

	if user, found, err := s.findByEmail(email); err != nil {
		return AuthUser{}, err
	} else if found {
		resolvedUsername, err := s.resolveUsername(username, githubID, user.ID)
		if err != nil {
			return AuthUser{}, err
		}

		if _, err := s.db.Exec(`
			UPDATE users
			SET github_id = ?, username = ?, avatar = CASE WHEN ? = '' THEN avatar ELSE ? END
			WHERE id = ?
		`, githubID, resolvedUsername, avatar, avatar, user.ID); err != nil {
			return AuthUser{}, err
		}

		updated, ok := s.FindByID(user.ID)
		if !ok {
			return AuthUser{}, ErrUserNotFound
		}

		return updated, nil
	}

	resolvedUsername, err := s.resolveUsername(username, githubID, 0)
	if err != nil {
		return AuthUser{}, err
	}

	result, err := s.db.Exec(`
		INSERT INTO users (username, avatar, email, github_id, password, bio, level, experience, status)
		VALUES (?, ?, ?, ?, NULL, NULL, 1, 0, 1)
	`, resolvedUsername, nullIfEmpty(avatar), email, githubID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return AuthUser{}, ErrUserExists
		}
		return AuthUser{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return AuthUser{}, err
	}

	user, ok := s.FindByID(int(id))
	if !ok {
		return AuthUser{}, ErrUserNotFound
	}

	return user, nil
}

func (s *MySQLUserStore) Follow(followerID, followingID int) (FollowStatus, bool, error) {
	if followerID == followingID {
		return FollowStatus{}, false, ErrCannotFollowSelf
	}
	if _, ok := s.FindByID(followerID); !ok {
		return FollowStatus{}, false, ErrUserNotFound
	}
	if _, ok := s.FindByID(followingID); !ok {
		return FollowStatus{}, false, ErrUserNotFound
	}

	result, err := s.db.Exec(`
		INSERT IGNORE INTO follows (follower_id, following_id)
		VALUES (?, ?)
	`, followerID, followingID)
	if err != nil {
		return FollowStatus{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return FollowStatus{}, false, err
	}

	status, err := s.FollowStatus(followingID, followerID)
	if err != nil {
		return FollowStatus{}, false, err
	}

	return status, affected > 0, nil
}

func (s *MySQLUserStore) Unfollow(followerID, followingID int) (FollowStatus, bool, error) {
	if _, ok := s.FindByID(followerID); !ok {
		return FollowStatus{}, false, ErrUserNotFound
	}
	if _, ok := s.FindByID(followingID); !ok {
		return FollowStatus{}, false, ErrUserNotFound
	}

	result, err := s.db.Exec(`
		DELETE FROM follows
		WHERE follower_id = ? AND following_id = ?
	`, followerID, followingID)
	if err != nil {
		return FollowStatus{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return FollowStatus{}, false, err
	}

	status, err := s.FollowStatus(followingID, followerID)
	if err != nil {
		return FollowStatus{}, false, err
	}

	return status, affected > 0, nil
}

func (s *MySQLUserStore) FollowStatus(userID, viewerID int) (FollowStatus, error) {
	if _, ok := s.FindByID(userID); !ok {
		return FollowStatus{}, ErrUserNotFound
	}

	status := FollowStatus{UserID: userID}
	if viewerID > 0 {
		var following int
		if err := s.db.QueryRow(`
			SELECT 1 FROM follows
			WHERE follower_id = ? AND following_id = ?
			LIMIT 1
		`, viewerID, userID).Scan(&following); err == nil {
			status.Following = following == 1
		} else if !errors.Is(err, sql.ErrNoRows) {
			return FollowStatus{}, err
		}
	}

	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM follows WHERE following_id = ?
	`, userID).Scan(&status.FollowerCount); err != nil {
		return FollowStatus{}, err
	}

	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM follows WHERE follower_id = ?
	`, userID).Scan(&status.FollowingCount); err != nil {
		return FollowStatus{}, err
	}

	return status, nil
}

func (s *MySQLUserStore) ListFollowing(userID int) ([]PublicUser, error) {
	if _, ok := s.FindByID(userID); !ok {
		return nil, ErrUserNotFound
	}

	rows, err := s.db.Query(`
		SELECT `+qualifiedUserSelectColumns+`
		FROM users
		JOIN follows ON follows.following_id = users.id
		WHERE follows.follower_id = ?
		ORDER BY follows.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPublicUsers(rows)
}

func (s *MySQLUserStore) ListFollowers(userID int) ([]PublicUser, error) {
	if _, ok := s.FindByID(userID); !ok {
		return nil, ErrUserNotFound
	}

	rows, err := s.db.Query(`
		SELECT `+qualifiedUserSelectColumns+`
		FROM users
		JOIN follows ON follows.follower_id = users.id
		WHERE follows.following_id = ?
		ORDER BY follows.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPublicUsers(rows)
}

func (s *MySQLUserStore) updateGitHubProfile(id int, username, avatar string, githubID int64) (AuthUser, error) {
	resolvedUsername, err := s.resolveUsername(username, githubID, id)
	if err != nil {
		return AuthUser{}, err
	}

	if _, err := s.db.Exec(`
		UPDATE users
		SET username = ?, avatar = CASE WHEN ? = '' THEN avatar ELSE ? END
		WHERE id = ?
	`, resolvedUsername, avatar, avatar, id); err != nil {
		return AuthUser{}, err
	}

	user, ok := s.FindByID(id)
	if !ok {
		return AuthUser{}, ErrUserNotFound
	}

	return user, nil
}

func (s *MySQLUserStore) resolveUsername(desired string, githubID int64, excludeUserID int) (string, error) {
	for _, candidate := range githubUsernameCandidates(desired, githubID) {
		taken, err := s.isUsernameTaken(candidate, excludeUserID)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}

	return "", ErrUserExists
}

func (s *MySQLUserStore) isUsernameTaken(username string, excludeUserID int) (bool, error) {
	var ownerID int
	err := s.db.QueryRow(`SELECT id FROM users WHERE username = ? LIMIT 1`, username).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if excludeUserID > 0 && ownerID == excludeUserID {
		return false, nil
	}

	return true, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}

func (s *MySQLUserStore) findByEmail(email string) (AuthUser, bool, error) {
	row := s.db.QueryRow(`SELECT`+userSelectColumns+` FROM users WHERE email = ?`, email)
	return scanAuthUser(row.Scan)
}

func (s *MySQLUserStore) findByGitHubID(githubID int64) (AuthUser, bool, error) {
	row := s.db.QueryRow(`SELECT`+userSelectColumns+` FROM users WHERE github_id = ?`, githubID)
	return scanAuthUser(row.Scan)
}

func scanPublicUsers(rows *sql.Rows) ([]PublicUser, error) {
	list := make([]PublicUser, 0)
	for rows.Next() {
		user, found, err := scanAuthUser(rows.Scan)
		if err != nil {
			return nil, err
		}
		if found {
			list = append(list, ToPublicUser(user))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func scanAuthUser(scan func(dest ...any) error) (AuthUser, bool, error) {
	var (
		user         AuthUser
		avatar       sql.NullString
		githubID     sql.NullInt64
		passwordHash sql.NullString
		bio          sql.NullString
		createdAt    time.Time
	)

	err := scan(
		&user.ID,
		&user.Username,
		&avatar,
		&user.Email,
		&githubID,
		&passwordHash,
		&bio,
		&user.Level,
		&user.Experience,
		&user.SessionVer,
		&user.Status,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, false, nil
	}
	if err != nil {
		return AuthUser{}, false, err
	}

	if avatar.Valid {
		user.Avatar = avatar.String
	}
	if githubID.Valid {
		user.GitHubID = githubID.Int64
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}

	user.CreatedAt = createdAt.UTC().Format("2006-01-02")
	return user, true, nil
}
