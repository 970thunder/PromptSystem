package store

import (
	"database/sql"
	"errors"
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
	id, username, avatar, email, github_id, password, bio, level, experience, status, created_at
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
	username = strings.TrimSpace(username)
	avatar = strings.TrimSpace(avatar)

	if githubID <= 0 {
		return AuthUser{}, ErrInvalidGitHubUser
	}
	if !IsValidEmail(email) {
		return AuthUser{}, ErrInvalidEmail
	}
	if username == "" {
		username = strings.Split(email, "@")[0]
	}

	if user, found, err := s.findByGitHubID(githubID); err != nil {
		return AuthUser{}, err
	} else if found {
		return s.updateGitHubProfile(user.ID, username, avatar)
	}

	if user, found, err := s.findByEmail(email); err != nil {
		return AuthUser{}, err
	} else if found {
		if _, err := s.db.Exec(`
			UPDATE users
			SET github_id = ?, username = ?, avatar = CASE WHEN ? = '' THEN avatar ELSE ? END
			WHERE id = ?
		`, githubID, username, avatar, avatar, user.ID); err != nil {
			return AuthUser{}, err
		}

		updated, ok := s.FindByID(user.ID)
		if !ok {
			return AuthUser{}, ErrUserNotFound
		}

		return updated, nil
	}

	result, err := s.db.Exec(`
		INSERT INTO users (username, avatar, email, github_id, password, bio, level, experience, status)
		VALUES (?, ?, ?, ?, NULL, 'Signed in with GitHub', 1, 0, 1)
	`, username, avatar, email, githubID)
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

func (s *MySQLUserStore) updateGitHubProfile(id int, username, avatar string) (AuthUser, error) {
	if _, err := s.db.Exec(`
		UPDATE users
		SET username = ?, avatar = CASE WHEN ? = '' THEN avatar ELSE ? END
		WHERE id = ?
	`, username, avatar, avatar, id); err != nil {
		return AuthUser{}, err
	}

	user, ok := s.FindByID(id)
	if !ok {
		return AuthUser{}, ErrUserNotFound
	}

	return user, nil
}

func (s *MySQLUserStore) findByEmail(email string) (AuthUser, bool, error) {
	row := s.db.QueryRow(`SELECT`+userSelectColumns+` FROM users WHERE email = ?`, email)
	return scanAuthUser(row.Scan)
}

func (s *MySQLUserStore) findByGitHubID(githubID int64) (AuthUser, bool, error) {
	row := s.db.QueryRow(`SELECT`+userSelectColumns+` FROM users WHERE github_id = ?`, githubID)
	return scanAuthUser(row.Scan)
}

func scanAuthUser(scan func(dest ...any) error) (AuthUser, bool, error) {
	var (
		user         AuthUser
		githubID     sql.NullInt64
		passwordHash sql.NullString
		createdAt    time.Time
	)

	err := scan(
		&user.ID,
		&user.Username,
		&user.Avatar,
		&user.Email,
		&githubID,
		&passwordHash,
		&user.Bio,
		&user.Level,
		&user.Experience,
		&user.Status,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, false, nil
	}
	if err != nil {
		return AuthUser{}, false, err
	}

	if githubID.Valid {
		user.GitHubID = githubID.Int64
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}

	user.CreatedAt = createdAt.UTC().Format("2006-01-02")
	return user, true, nil
}
