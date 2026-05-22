package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidGitHubUser  = errors.New("invalid github user")
)

type UserStore struct {
	mu            sync.RWMutex
	nextID        int
	users         map[int]AuthUser
	emailIndex    map[string]int
	githubIDIndex map[int64]int
}

type AuthUser struct {
	ID           int
	Username     string
	Avatar       string
	Email        string
	GitHubID     int64
	PasswordHash string
	Bio          string
	Level        int
	Experience   int
	Status       int
	CreatedAt    string
}

type PublicUser struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Avatar     string `json:"avatar"`
	Email      string `json:"email"`
	Bio        string `json:"bio"`
	Level      int    `json:"level"`
	Experience int    `json:"experience"`
	Status     int    `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

func NewUserStore() *UserStore {
	store := &UserStore{
		nextID:        7,
		users:         map[int]AuthUser{},
		emailIndex:    map[string]int{},
		githubIDIndex: map[int64]int{},
	}

	seedUsers := []AuthUser{
		{
			ID:           1,
			Username:     "Astra Lab",
			Avatar:       "",
			Email:        "astra@example.com",
			PasswordHash: mustHashPassword("PromptOS123!"),
			Bio:          "Visual prompt designer",
			Level:        8,
			Experience:   1320,
			Status:       1,
			CreatedAt:    "2026-05-01",
		},
		{
			ID:           2,
			Username:     "Nora Chen",
			Avatar:       "",
			Email:        "nora@example.com",
			PasswordHash: mustHashPassword("PromptOS123!"),
			Bio:          "Growth copywriter",
			Level:        6,
			Experience:   910,
			Status:       1,
			CreatedAt:    "2026-04-20",
		},
	}

	for _, user := range seedUsers {
		store.users[user.ID] = user
		store.emailIndex[strings.ToLower(user.Email)] = user.ID
	}

	return store
}

func (s *UserStore) Register(username, email, password string) (AuthUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)

	if !IsValidEmail(email) {
		return AuthUser{}, ErrInvalidEmail
	}

	if len(password) < 8 {
		return AuthUser{}, ErrWeakPassword
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.emailIndex[email]; exists {
		return AuthUser{}, ErrUserExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthUser{}, err
	}

	user := AuthUser{
		ID:           s.nextID,
		Username:     username,
		Avatar:       "",
		Email:        email,
		PasswordHash: string(passwordHash),
		Bio:          "New PromptOS creator",
		Level:        1,
		Experience:   0,
		Status:       1,
		CreatedAt:    time.Now().UTC().Format("2006-01-02"),
	}

	s.users[user.ID] = user
	s.emailIndex[email] = user.ID
	s.nextID++

	return user, nil
}

func (s *UserStore) Authenticate(email, password string) (AuthUser, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	s.mu.RLock()
	userID, exists := s.emailIndex[email]
	if !exists {
		s.mu.RUnlock()
		return AuthUser{}, ErrInvalidCredentials
	}

	user := s.users[userID]
	s.mu.RUnlock()

	if user.PasswordHash == "" {
		return AuthUser{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthUser{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *UserStore) UpsertGitHubUser(githubID int64, username, email, avatar string) (AuthUser, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if userID, exists := s.githubIDIndex[githubID]; exists {
		user := s.users[userID]
		resolvedUsername, err := s.resolveUsernameLocked(username, githubID, userID)
		if err != nil {
			return AuthUser{}, err
		}
		if avatar != "" {
			user.Avatar = avatar
		}
		user.Username = resolvedUsername
		user.GitHubID = githubID
		s.users[userID] = user
		return user, nil
	}

	if userID, exists := s.emailIndex[email]; exists {
		resolvedUsername, err := s.resolveUsernameLocked(username, githubID, userID)
		if err != nil {
			return AuthUser{}, err
		}

		user := s.users[userID]
		user.GitHubID = githubID
		if avatar != "" {
			user.Avatar = avatar
		}
		user.Username = resolvedUsername
		s.users[userID] = user
		s.githubIDIndex[githubID] = userID
		return user, nil
	}

	resolvedUsername, err := s.resolveUsernameLocked(username, githubID, 0)
	if err != nil {
		return AuthUser{}, err
	}

	user := AuthUser{
		ID:         s.nextID,
		Username:   resolvedUsername,
		Avatar:     avatar,
		Email:      email,
		GitHubID:   githubID,
		Level:      1,
		Experience: 0,
		Status:     1,
		CreatedAt:  time.Now().UTC().Format("2006-01-02"),
	}

	s.users[user.ID] = user
	s.emailIndex[email] = user.ID
	s.githubIDIndex[githubID] = user.ID
	s.nextID++

	return user, nil
}

func (s *UserStore) resolveUsernameLocked(desired string, githubID int64, excludeUserID int) (string, error) {
	for _, candidate := range githubUsernameCandidates(desired, githubID) {
		if s.isUsernameTakenLocked(candidate, excludeUserID) {
			continue
		}

		return candidate, nil
	}

	return "", ErrUserExists
}

func (s *UserStore) isUsernameTakenLocked(username string, excludeUserID int) bool {
	for id, user := range s.users {
		if excludeUserID > 0 && id == excludeUserID {
			continue
		}
		if user.Username == username {
			return true
		}
	}

	return false
}

func (s *UserStore) FindByID(id int) (AuthUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	return user, ok
}

func (s *UserStore) UpdateProfile(id int, username, bio, avatar string) (AuthUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return AuthUser{}, ErrUserNotFound
	}

	username = strings.TrimSpace(username)
	bio = strings.TrimSpace(bio)
	avatar = strings.TrimSpace(avatar)

	if username != "" {
		user.Username = username
	}
	if bio != "" {
		user.Bio = bio
	}
	if avatar != "" {
		user.Avatar = avatar
	}

	s.users[id] = user
	return user, nil
}

func ToPublicUser(user AuthUser) PublicUser {
	return PublicUser{
		ID:         user.ID,
		Username:   user.Username,
		Avatar:     user.Avatar,
		Email:      user.Email,
		Bio:        user.Bio,
		Level:      user.Level,
		Experience: user.Experience,
		Status:     user.Status,
		CreatedAt:  user.CreatedAt,
	}
}

func mustHashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return string(hash)
}
