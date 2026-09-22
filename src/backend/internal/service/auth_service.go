package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"promptos-backend/internal/cache"
	"promptos-backend/internal/store"
)

const jwtDenylistPrefix = "promptos:jwt:denylist:"

// ErrHistoryClear identifies the first phase of account deletion so the API
// can preserve its dedicated error code without exposing store details.
var ErrHistoryClear = errors.New("history clear failed")

// AuthService is the business boundary for account and session operations.
// It deliberately returns store values and errors so the API layer remains
// responsible only for HTTP validation and stable error translation.
type AuthService struct {
	users   store.UserManager
	prompts store.PromptManager
	cache   cache.Cache
}

func NewAuthService(users store.UserManager, prompts store.PromptManager, runtimeCache cache.Cache) *AuthService {
	return &AuthService{users: users, prompts: prompts, cache: runtimeCache}
}

func (s *AuthService) Authenticate(email, password string) (store.AuthUser, error) {
	return s.users.Authenticate(email, password)
}

func (s *AuthService) Register(username, email, password string) (store.AuthUser, error) {
	return s.users.Register(username, email, password)
}

func (s *AuthService) ResetPassword(email, password string) error {
	return s.users.ResetPassword(email, password)
}

func (s *AuthService) FindByID(id int) (store.AuthUser, bool) {
	return s.users.FindByID(id)
}

func (s *AuthService) UpdateProfile(id int, username, bio, avatar string) (store.AuthUser, error) {
	return s.users.UpdateProfile(id, username, bio, avatar)
}

func (s *AuthService) Follow(followerID, followingID int) (store.FollowStatus, bool, error) {
	return s.users.Follow(followerID, followingID)
}

func (s *AuthService) Unfollow(followerID, followingID int) (store.FollowStatus, bool, error) {
	return s.users.Unfollow(followerID, followingID)
}

func (s *AuthService) FollowStatus(userID, viewerID int) (store.FollowStatus, error) {
	return s.users.FollowStatus(userID, viewerID)
}

func (s *AuthService) ListFollowing(userID int) ([]store.PublicUser, error) {
	return s.users.ListFollowing(userID)
}

func (s *AuthService) ListFollowers(userID int) ([]store.PublicUser, error) {
	return s.users.ListFollowers(userID)
}

func (s *AuthService) ListFavorites(userID int) ([]store.Prompt, error) {
	return s.prompts.ListUserFavorites(userID)
}

func (s *AuthService) ListLikes(userID int) ([]store.Prompt, error) {
	return s.prompts.ListUserLikes(userID)
}

func (s *AuthService) ListHistory(userID int) ([]store.Prompt, error) {
	return s.prompts.ListUserHistory(userID)
}

func (s *AuthService) ClearHistory(userID int) error {
	return s.prompts.ClearUserHistory(userID)
}

// AccountExport bundles the store reads required by the personal data export
// endpoint so handlers cannot accidentally omit one of the owned datasets.
type AccountExport struct {
	User      store.AuthUser
	Prompts   []store.Prompt
	Favorites []store.Prompt
	Likes     []store.Prompt
	History   []store.Prompt
}

func (s *AuthService) ExportAccount(userID int) (AccountExport, error) {
	user, found := s.users.FindByID(userID)
	if !found || user.Status != 1 {
		return AccountExport{}, store.ErrUserNotFound
	}
	prompts, err := s.prompts.ListUserPrompts(userID)
	if err != nil {
		return AccountExport{}, err
	}
	favorites, err := s.prompts.ListUserFavorites(userID)
	if err != nil {
		return AccountExport{}, err
	}
	likes, err := s.prompts.ListUserLikes(userID)
	if err != nil {
		return AccountExport{}, err
	}
	history, err := s.prompts.ListUserHistory(userID)
	if err != nil {
		return AccountExport{}, err
	}
	return AccountExport{User: user, Prompts: prompts, Favorites: favorites, Likes: likes, History: history}, nil
}

// DeleteAccount applies the account-retention workflow in a single business
// operation. The prompt store clears personal history before the user store
// disables and anonymizes the account, matching the established retry-safe
// behavior of the endpoint.
func (s *AuthService) DeleteAccount(userID int) error {
	if err := s.prompts.ClearUserHistory(userID); err != nil {
		return fmt.Errorf("%w: %v", ErrHistoryClear, err)
	}
	return s.users.DeleteAccount(userID)
}

func (s *AuthService) RevokeToken(ctx context.Context, jti string, ttl time.Duration) error {
	if s.cache == nil || jti == "" {
		return nil
	}
	return s.cache.Set(ctx, jwtDenylistPrefix+jti, "1", ttl)
}

func (s *AuthService) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	if s.cache == nil || jti == "" {
		return false, nil
	}
	return s.cache.Exists(ctx, jwtDenylistPrefix+jti)
}

// SessionVersionMatches keeps the token/session comparison in one place for
// middleware and optional-auth endpoints.
func (s *AuthService) SessionVersionMatches(claimsVersion, userID int) bool {
	user, found := s.users.FindByID(userID)
	return found && user.Status == 1 && user.SessionVer == claimsVersion
}

func UserIDFromSubject(subject string) (int, error) {
	return strconv.Atoi(subject)
}
