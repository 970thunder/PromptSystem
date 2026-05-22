package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type oauthStateEntry struct {
	expiresAt time.Time
}

type githubOAuth struct {
	states sync.Map
}

var githubOAuthState = &githubOAuth{}

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

	githubOAuthState.states.Store(state, oauthStateEntry{expiresAt: time.Now().Add(10 * time.Minute)})
	http.SetCookie(w, &http.Cookie{
		Name:     "github_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
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

	entry, ok := githubOAuthState.states.LoadAndDelete(queryState)
	if !ok {
		s.redirectOAuthError(w, r, "OAuth state expired")
		return
	}

	stateEntry, ok := entry.(oauthStateEntry)
	if !ok || time.Now().After(stateEntry.expiresAt) {
		s.redirectOAuthError(w, r, "OAuth state expired")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "github_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

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
		s.redirectOAuthError(w, r, "Failed to create user session")
		return
	}

	token, err := s.tokenManager.Generate(user.ID, user.Email)
	if err != nil {
		s.redirectOAuthError(w, r, "Token generation failed")
		return
	}

	frontend := strings.TrimRight(s.config.FrontendURL, "/")
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontend, url.QueryEscape(token))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *server) redirectOAuthError(w http.ResponseWriter, r *http.Request, message string) {
	frontend := strings.TrimRight(s.config.FrontendURL, "/")
	redirectURL := fmt.Sprintf("%s/auth/callback?error=%s", frontend, url.QueryEscape(message))
	http.Redirect(w, r, redirectURL, http.StatusFound)
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubProfile{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return githubProfile{}, err
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
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
