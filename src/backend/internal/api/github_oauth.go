package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"promptos-backend/internal/store"
)

// oauthStateEntry is the in-memory fallback record used only when Redis is not
// available (development/test). In production the state lives in Redis.
type oauthStateEntry struct {
	expiresAt time.Time
}

type githubOAuth struct {
	states sync.Map
}

var githubOAuthState = &githubOAuth{}

// oauthStateKey prefixes OAuth state stored in Redis.
const oauthStateKey = "promptos:oauth:state:"

// oauthExchangeKey prefixes the one-time code that frontends exchange for a
// JWT, so the JWT never appears in a URL query.
const oauthExchangeKey = "promptos:oauth:exchange:"

// oauthStateTTL bounds how long a login attempt may remain in progress.
const oauthStateTTL = 10 * time.Minute

// exchangeCodeTTL is the lifetime of a one-time exchange code.
const exchangeCodeTTL = 60 * time.Second

func (s *server) oauthCookieSecure() bool {
	return s.config.AppEnv == "production"
}

func (s *server) githubRedirectURI() string {
	if strings.TrimSpace(s.config.GitHubRedirectURI) != "" {
		return strings.TrimSpace(s.config.GitHubRedirectURI)
	}

	base := strings.TrimRight(s.config.UploadBaseURL, "/")
	if base == "" {
		base = "http://localhost:8080"
	}

	return base + "/api/v1/auth/github/callback"
}

func (s *server) githubConfigured() bool {
	return strings.TrimSpace(s.config.GitHubClientID) != "" &&
		strings.TrimSpace(s.config.GitHubClientSecret) != ""
}

// storeOAuthState records the state for single consumption, preferring Redis
// when available and falling back to in-memory storage (development/test).
func (s *server) storeOAuthState(ctx context.Context, state string) error {
	if s.cache != nil {
		if err := s.cache.Set(ctx, oauthStateKey+state, "1", oauthStateTTL); err != nil {
			log.Printf("oauth: redis store state failed, using memory fallback: %v", err)
		} else {
			return nil
		}
	}
	githubOAuthState.states.Store(state, oauthStateEntry{expiresAt: time.Now().Add(oauthStateTTL)})
	return nil
}

// consumeOAuthState removes the state exactly once, enforcing TTL. It returns
// false when the state is unknown, already used, or expired.
func (s *server) consumeOAuthState(ctx context.Context, state string) bool {
	if s.cache != nil {
		value, err := s.cache.GetAndDelete(ctx, oauthStateKey+state)
		if err != nil {
			log.Printf("oauth: redis consume state failed: %v", err)
		} else if value != "" {
			return true
		}
	}

	entry, ok := githubOAuthState.states.LoadAndDelete(state)
	if !ok {
		return false
	}
	stateEntry, ok := entry.(oauthStateEntry)
	if !ok || time.Now().After(stateEntry.expiresAt) {
		return false
	}
	return true
}

func (s *server) handleGitHubAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	if !s.githubConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{
			Code:    503,
			Message: "GitHub OAuth is not configured",
		})
		return
	}

	state, err := randomOAuthState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:    500,
			Message: "Failed to start GitHub OAuth",
		})
		return
	}

	if err := s.storeOAuthState(r.Context(), state); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:    500,
			Message: "Failed to start GitHub OAuth",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "github_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   int(oauthStateTTL / time.Second),
		HttpOnly: true,
		Secure:   s.oauthCookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})

	params := url.Values{}
	params.Set("client_id", s.config.GitHubClientID)
	params.Set("redirect_uri", s.githubRedirectURI())
	params.Set("scope", "read:user user:email")
	params.Set("state", state)

	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+params.Encode(), http.StatusFound)
}

func (s *server) handleGitHubAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	if !s.githubConfigured() {
		s.redirectOAuthError(w, r, "GitHub OAuth is not configured")
		return
	}

	queryState := strings.TrimSpace(r.URL.Query().Get("state"))
	cookie, err := r.Cookie("github_oauth_state")
	if err != nil || cookie.Value == "" || cookie.Value != queryState {
		s.redirectOAuthError(w, r, "Invalid OAuth state")
		return
	}

	if !s.consumeOAuthState(r.Context(), queryState) {
		s.redirectOAuthError(w, r, "OAuth state expired")
		return
	}

	clearOAuthStateCookie(w, s.oauthCookieSecure())

	if oauthErr := strings.TrimSpace(r.URL.Query().Get("error")); oauthErr != "" {
		s.redirectOAuthError(w, r, oauthErr)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		s.redirectOAuthError(w, r, "Missing authorization code")
		return
	}

	ctx := r.Context()
	accessToken, err := s.exchangeGitHubCode(ctx, code)
	if err != nil {
		s.redirectOAuthError(w, r, "Failed to exchange GitHub code")
		return
	}

	profile, err := s.fetchGitHubProfile(ctx, accessToken)
	if err != nil {
		s.redirectOAuthError(w, r, "Failed to load GitHub profile")
		return
	}

	email := strings.TrimSpace(profile.Email)
	if email == "" {
		email, err = s.fetchGitHubPrimaryEmail(ctx, accessToken)
		if err != nil || email == "" {
			s.redirectOAuthError(w, r, "GitHub account has no public email")
			return
		}
	}

	user, err := s.userStore.UpsertGitHubUser(profile.ID, profile.Login, email, profile.AvatarURL)
	if err != nil {
		log.Printf("github oauth upsert user failed: %v", err)
		s.redirectOAuthError(w, r, "Failed to create user session")
		return
	}

	// Issue a short-lived one-time code instead of placing the JWT in the URL.
	// The frontend calls POST /api/v1/auth/exchange to trade it for a JWT.
	exchangeCode, err := randomOAuthState()
	if err != nil {
		s.redirectOAuthError(w, r, "Token generation failed")
		return
	}

	codeValue := fmt.Sprintf("%d:%d", user.ID, user.SessionVer)
	if s.cache == nil {
		githubExchangeCodes.Store(exchangeCode, exchangeEntry{value: codeValue, expiresAt: time.Now().Add(exchangeCodeTTL)})
	} else if err := s.cache.Set(ctx, oauthExchangeKey+exchangeCode, codeValue, exchangeCodeTTL); err != nil {
		log.Printf("github oauth store exchange code failed: %v", err)
		s.redirectOAuthError(w, r, "Token generation failed")
		return
	}

	frontend := strings.TrimRight(s.config.FrontendURL, "/")
	redirectURL := fmt.Sprintf("%s/auth/callback?code=%s", frontend, url.QueryEscape(exchangeCode))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// exchangeEntry is the in-memory fallback for a one-time exchange code, used
// only when Redis is unavailable (development/test).
type exchangeEntry struct {
	value     string
	expiresAt time.Time
}

// githubExchangeCodes holds one-time exchange codes when Redis is not present.
var githubExchangeCodes sync.Map

func (s *server) handleAuthExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}

	exchangeCode := strings.TrimSpace(payload.Code)
	if exchangeCode == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      400,
			Message:   "Missing code",
			ErrorCode: "INVALID_EXCHANGE_CODE",
		})
		return
	}

	value := ""
	if s.cache != nil {
		v, err := s.cache.GetAndDelete(r.Context(), oauthExchangeKey+exchangeCode)
		if err != nil {
			log.Printf("github oauth exchange read failed: %v", err)
		}
		value = v
	}
	if value == "" {
		if entry, ok := githubExchangeCodes.LoadAndDelete(exchangeCode); ok {
			if codeEntry, valid := entry.(exchangeEntry); valid && time.Now().Before(codeEntry.expiresAt) {
				value = codeEntry.value
			}
		}
	}
	if value == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      400,
			Message:   "Code is invalid or has expired",
			ErrorCode: "INVALID_EXCHANGE_CODE",
		})
		return
	}

	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      400,
			Message:   "Code is invalid or has expired",
			ErrorCode: "INVALID_EXCHANGE_CODE",
		})
		return
	}
	userID, err := strconv.Atoi(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      400,
			Message:   "Code is invalid or has expired",
			ErrorCode: "INVALID_EXCHANGE_CODE",
		})
		return
	}
	sessionVersion := 0
	if len(parts) == 2 {
		if parsed, err := strconv.Atoi(parts[1]); err == nil {
			sessionVersion = parsed
		}
	}

	user, found := s.userStore.FindByID(userID)
	if !found || user.Status != 1 || user.SessionVer != sessionVersion {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
			Code:      401,
			Message:   "Unauthorized",
			ErrorCode: "AUTH_TOKEN_REVOKED",
		})
		return
	}

	token, err := s.tokenManager.Generate(user.ID, user.Email, user.SessionVer)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:    500,
			Message: "Token generation failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[authResponse]{
		Code:    200,
		Message: "Success",
		Data: authResponse{
			Token: token,
			User:  store.ToPublicUser(user),
		},
	})
}

func (s *server) redirectOAuthError(w http.ResponseWriter, r *http.Request, message string) {
	frontend := strings.TrimRight(s.config.FrontendURL, "/")
	redirectURL := fmt.Sprintf("%s/auth/callback?error=%s", frontend, url.QueryEscape(message))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func clearOAuthStateCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "github_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) exchangeGitHubCode(ctx context.Context, code string) (string, error) {
	payload := url.Values{}
	payload.Set("client_id", s.config.GitHubClientID)
	payload.Set("client_secret", s.config.GitHubClientSecret)
	payload.Set("code", code)
	payload.Set("redirect_uri", s.githubRedirectURI())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(payload.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.githubClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github access token request failed with status %d", resp.StatusCode)
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		if parsed.Error != "" {
			return "", errors.New(parsed.Error)
		}
		return "", errors.New("missing access token")
	}

	return parsed.AccessToken, nil
}

type githubProfile struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (s *server) fetchGitHubProfile(ctx context.Context, accessToken string) (githubProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return githubProfile{}, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.githubClient.Do(req)
	if err != nil {
		return githubProfile{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return githubProfile{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubProfile{}, fmt.Errorf("github profile request failed with status %d", resp.StatusCode)
	}

	var profile githubProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return githubProfile{}, err
	}
	if profile.ID <= 0 {
		return githubProfile{}, fmt.Errorf("invalid github profile")
	}

	return profile, nil
}

func (s *server) fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.githubClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github email request failed with status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	for _, item := range emails {
		if item.Primary && item.Verified && strings.TrimSpace(item.Email) != "" {
			return strings.TrimSpace(strings.ToLower(item.Email)), nil
		}
	}

	for _, item := range emails {
		if item.Verified && strings.TrimSpace(item.Email) != "" {
			return strings.TrimSpace(strings.ToLower(item.Email)), nil
		}
	}

	return "", fmt.Errorf("no verified github email")
}

func randomOAuthState() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}
