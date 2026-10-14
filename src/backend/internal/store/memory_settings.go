package store

import (
	"strings"
	"sync"
	"time"
)

// MemorySettingsStore backs development/test runs. The bootstrap gate mirrors
// the MySQL semantics: the completed marker plus the granted-admin list are
// updated under one lock so concurrent setups produce exactly one winner.
type MemorySettingsStore struct {
	mu        sync.Mutex
	values    map[string]string
	adminIDs  map[int]struct{}
	nextAdmin int
	cipher    *SettingsCipher
	// users, when wired, makes the wizard-created administrator a real
	// login-able account in the development memory backend (parity with the
	// MySQL implementation, which inserts the user in the same transaction).
	users *UserStore
}

func NewMemorySettingsStore() *MemorySettingsStore {
	return &MemorySettingsStore{
		values:    map[string]string{},
		adminIDs:  map[int]struct{}{},
		nextAdmin: 100000,
	}
}

// SetCipher enables at-rest encryption for sensitive settings (smtp.password).
func (s *MemorySettingsStore) SetCipher(cipher *SettingsCipher) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cipher = cipher
}

func (s *MemorySettingsStore) encodeValue(key, value string) (string, error) {
	if s.cipher == nil || key != SettingSMTPPassword || value == "" {
		return value, nil
	}
	return s.cipher.Encrypt(value)
}

func (s *MemorySettingsStore) decodeValue(key, value string) string {
	if key != SettingSMTPPassword {
		return value
	}
	return s.cipher.Decrypt(value)
}

// SetUserStore wires the development user store so bootstrap creates a real
// account. It must be called before CompleteSetup.
func (s *MemorySettingsStore) SetUserStore(users *UserStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = users
}

func (s *MemorySettingsStore) Get(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return "", false, nil
	}
	return s.decodeValue(key, value), true, nil
}

func (s *MemorySettingsStore) GetAll() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = s.decodeValue(key, value)
	}
	return result, nil
}

func (s *MemorySettingsStore) SetMany(values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, value := range values {
		if key == SettingSetupCompletedAt {
			continue
		}
		encoded, err := s.encodeValue(key, value)
		if err != nil {
			return err
		}
		if encoded == "" {
			delete(s.values, key)
			continue
		}
		s.values[key] = encoded
	}
	return nil
}

func (s *MemorySettingsStore) HasAdmin() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.adminIDs) > 0, nil
}

// GrantAdmin records an out-of-band administrator for tests that need to
// simulate an already-provisioned deployment.
func (s *MemorySettingsStore) GrantAdmin(userID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adminIDs[userID] = struct{}{}
}

func (s *MemorySettingsStore) Completed() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.values[SettingSetupCompletedAt]
	return ok, nil
}

func (s *MemorySettingsStore) CompleteSetup(input SetupInput) (int, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, done := s.values[SettingSetupCompletedAt]; done {
		return 0, ErrSetupCompleted
	}
	if len(s.adminIDs) > 0 {
		return 0, ErrSetupCompleted
	}

	completedAt := input.Now
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	adminID := s.nextAdmin
	if s.users != nil {
		user, err := s.users.Register(input.Username, input.Email, input.Password)
		if err != nil {
			return 0, err
		}
		adminID = user.ID
	}
	s.nextAdmin++
	s.adminIDs[adminID] = struct{}{}
	s.values[SettingSetupCompletedAt] = completedAt.Format(time.RFC3339)
	for key, value := range settings {
		encoded, err := s.encodeValue(key, value)
		if err != nil {
			return 0, err
		}
		s.values[key] = encoded
	}
	return adminID, nil
}
