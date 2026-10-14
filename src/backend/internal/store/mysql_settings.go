package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

// MySQLSettingsStore persists wizard-owned site configuration and executes the
// one-shot bootstrap. The setup marker row acts as the atomic gate: concurrent
// CompleteSetup calls serialize on the system_settings primary key.
type MySQLSettingsStore struct {
	db *sql.DB
}

func NewMySQLSettingsStore(db *sql.DB) *MySQLSettingsStore {
	return &MySQLSettingsStore{db: db}
}

func (s *MySQLSettingsStore) Get(key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT setting_value FROM system_settings WHERE setting_key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *MySQLSettingsStore) GetAll() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT setting_key, setting_value FROM system_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

func (s *MySQLSettingsStore) SetMany(values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for key, value := range values {
		if key == SettingSetupCompletedAt {
			continue
		}
		if value == "" {
			if _, err := tx.Exec(`DELETE FROM system_settings WHERE setting_key = ?`, key); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO system_settings (setting_key, setting_value) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)
		`, key, value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// AppendSettingsAudit records a settings.update event on the tamper-evident
// chain. Only key names are recorded; values may contain secrets.
func (s *MySQLSettingsStore) AppendSettingsAudit(actorID int, changedKeys []string, requestID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := appendAuditTx(tx, actorID, "settings.update", "system", 0, map[string]any{
		"changedKeys": changedKeys,
	}, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLSettingsStore) HasAdmin() (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE ur.role = 'admin' AND u.status = 1
	`).Scan(&count)
	return count > 0, err
}

func (s *MySQLSettingsStore) Completed() (bool, error) {
	_, found, err := s.Get(SettingSetupCompletedAt)
	return found, err
}

func (s *MySQLSettingsStore) CompleteSetup(input SetupInput) (int, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if err := ValidateUserProfile(input.Username, ""); err != nil {
		return 0, err
	}
	if !IsValidEmail(input.Email) {
		return 0, ErrInvalidEmail
	}
	if len(input.Password) < 8 {
		return 0, ErrWeakPassword
	}
	if len(input.Password) > maxPasswordBytes {
		return 0, ErrPasswordTooLong
	}
	settings := NormalizeSettingsMap(input.Settings)
	if _, reserved := settings[SettingSetupCompletedAt]; reserved {
		return 0, ErrInvalidSetting
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	completedAt := input.Now
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Atomic gate: the marker insert fails on the primary key if any earlier
	// bootstrap committed, including one that crashed mid-transaction.
	if _, err := tx.Exec(`
		INSERT INTO system_settings (setting_key, setting_value) VALUES (?, ?)
	`, SettingSetupCompletedAt, completedAt.Format(time.RFC3339)); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, ErrSetupCompleted
		}
		return 0, err
	}

	// A deployment that already provisioned an administrator out-of-band
	// (manual SQL) must not be hijackable through the wizard.
	var existingAdmin int
	if err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE ur.role = 'admin' AND u.status = 1
	`).Scan(&existingAdmin); err != nil {
		return 0, err
	}
	if existingAdmin > 0 {
		return 0, ErrSetupCompleted
	}

	result, err := tx.Exec(`
		INSERT INTO users (username, avatar, email, password, bio, level, experience, status)
		VALUES (?, '', ?, ?, 'PromptOS administrator', 1, 0, 1)
	`, input.Username, input.Email, string(passwordHash))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, ErrUserExists
		}
		return 0, err
	}
	adminID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`INSERT INTO user_roles (user_id, role) VALUES (?, 'admin')`, adminID); err != nil {
		return 0, err
	}

	for key, value := range settings {
		if _, err := tx.Exec(`
			INSERT INTO system_settings (setting_key, setting_value) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)
		`, key, value); err != nil {
			return 0, err
		}
	}

	changed := make([]string, 0, len(settings))
	for key := range settings {
		changed = append(changed, key)
	}
	if err := appendAuditTx(tx, int(adminID), "setup.initialize", "system", 0, map[string]any{
		"adminId":       adminID,
		"settingsCount": len(changed),
	}, input.RequestID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(adminID), nil
}
