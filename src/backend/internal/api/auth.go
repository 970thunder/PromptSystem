package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/service"
	"promptos-backend/internal/store"
)

type contextKey string

const userContextKey contextKey = "userID"

type authResponse struct {
	Token string            `json:"token,omitempty"`
	User  store.PrivateUser `json:"user"`
}

func (s *server) authResponseToken(token string) string {
	if s.authCookieEnabled() {
		return ""
	}
	return token
}

type followActionResponse struct {
	Status  store.FollowStatus `json:"status"`
	Applied bool               `json:"applied"`
}

type userDataExport struct {
	ExportedAt string            `json:"exportedAt"`
	User       store.PrivateUser `json:"user"`
	Prompts    []store.Prompt    `json:"prompts"`
	Favorites  []store.Prompt    `json:"favorites"`
	Likes      []store.Prompt    `json:"likes"`
	History    []store.Prompt    `json:"history"`
}

func timeUntil(target time.Time) time.Duration {
	remaining := time.Until(target)
	if remaining < 0 {
		return 0
	}

	return remaining
}

func (s *server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, found := s.getAuthService().FindByID(userID)
		if !found {
			writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "User not found"})
			return
		}
		// 昵称与头像由 isoumao 身份中心权威维护：读取时按秒级节流同步展示副本。
		user = s.syncUnifiedProfile(r.Context(), user)

		writeJSON(w, http.StatusOK, apiResponse[store.PrivateUser]{
			Code:    200,
			Message: "Success",
			Data:    store.ToPrivateUser(user),
		})
	case http.MethodPut:
		// 资料（昵称、头像）已统一到身份中心，本站不再提供任何资料编辑入口。
		writeJSON(w, http.StatusForbidden, apiResponse[any]{
			Code:      403,
			ErrorCode: "PROFILE_MANAGED_BY_IDENTITY",
			Message:   "昵称与头像由 isoumao 统一账号维护，请在 https://id.isoumao.cn/profile/ 修改。",
		})
	default:
		writeMethodNotAllowed(w)
	}
}

// handleUserDataExport returns the authenticated user's account and retained
// Prompt data. Password hashes and OAuth identifiers are excluded; the account
// email is included because it is part of the user's personal export.
func (s *server) handleUserDataExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, ErrorCode: "AUTH_TOKEN_MISSING", Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "data_export", rateLimitRule{bucket: rateLimitUser(userID), limit: 3, window: time.Hour}) {
		return
	}

	export, err := s.getAuthService().ExportAccount(userID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, ErrorCode: "AUTH_USER_DISABLED", Message: "Unauthorized"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, ErrorCode: "DATA_EXPORT_FAILED", Message: "Failed to export account data"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[userDataExport]{
		Code:    200,
		Message: "Success",
		Data: userDataExport{
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			User:       store.ToPrivateUser(export.User),
			Prompts:    nonNilPrompts(export.Prompts),
			Favorites:  nonNilPrompts(export.Favorites),
			Likes:      nonNilPrompts(export.Likes),
			History:    nonNilPrompts(export.History),
		},
	})
}

func nonNilPrompts(prompts []store.Prompt) []store.Prompt {
	if prompts == nil {
		return []store.Prompt{}
	}
	return prompts
}

// handleDeleteAccount performs the authenticated account-retention transition.
// Browsing history is cleared before disabling the account; if the second step
// fails the account remains usable and the caller can retry safely.
func (s *server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, ErrorCode: "AUTH_TOKEN_MISSING", Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "account_delete", rateLimitRule{bucket: rateLimitUser(userID), limit: 2, window: time.Hour}) {
		return
	}
	if err := s.getAuthService().DeleteAccount(userID); err != nil {
		if errors.Is(err, service.ErrHistoryClear) {
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, ErrorCode: "HISTORY_CLEAR_FAILED", Message: "Failed to delete account"})
			return
		}
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[map[string]bool]{
		Code:    200,
		Message: "Account deleted",
		Data:    map[string]bool{"deleted": true},
	})
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	token, _ := sessionTokenFromRequest(r)
	if token != "" {
		claims, err := s.tokenManager.Verify(token)
		if err == nil && claims.JTI != "" {
			// Denylist the token until it would have expired naturally.
			ttl := time.Until(time.Unix(claims.Expiry, 0))
			if ttl < 0 {
				ttl = 0
			}
			_ = s.getAuthService().RevokeToken(r.Context(), claims.JTI, ttl)
		}
	}

	// 单点登出：清本站会话，同时给出身份中心的结束会话地址。
	logoutURL := s.oidcEndSessionURL(r, idTokenHintFromRequest(r))
	http.SetCookie(w, &http.Cookie{Name: "promptos_id_token_hint", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.authCookieSecure(), SameSite: http.SameSiteLaxMode})
	clearSessionCookies(w, s.authCookieSecure())
	writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: "Success", Data: map[string]any{"logoutUrl": logoutURL}})
}

// idTokenHintFromRequest 读取登录时保存的 id_token，用于 RP 发起的单点登出。
func idTokenHintFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("promptos_id_token_hint")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// oidcEndSessionURL 组装身份中心结束会话地址；未配置或不可达时返回空串，仅结束本站会话。
func (s *server) oidcEndSessionURL(r *http.Request, idTokenHint string) string {
	if !s.oidcConfigured() {
		return ""
	}
	disc, err := s.loadOIDCDiscovery(r.Context())
	if err != nil || disc.EndSessionEndpoint == "" {
		if err != nil {
			log.Printf("oidc discovery for logout failed: %v", err)
		}
		return ""
	}
	params := url.Values{}
	params.Set("client_id", s.config.OIDCClientID)
	if idTokenHint != "" {
		params.Set("id_token_hint", idTokenHint)
	}
	if frontend := strings.TrimRight(s.config.FrontendURL, "/"); frontend != "" {
		params.Set("post_logout_redirect_uri", frontend)
	}
	return disc.EndSessionEndpoint + "?" + params.Encode()
}

func (s *server) handleUserFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, err := s.getAuthService().ListFavorites(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load favorites"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.Prompt]{
		Code:    200,
		Message: "Success",
		Data:    list,
	})
}

func (s *server) handleUserLikes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, err := s.getAuthService().ListLikes(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load likes"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.Prompt]{
		Code:    200,
		Message: "Success",
		Data:    list,
	})
}

func (s *server) handleUserHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, err := s.getAuthService().ListHistory(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load history"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.Prompt]{
		Code:    200,
		Message: "Success",
		Data:    list,
	})
}

func (s *server) handleUserFollowing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, err := s.getAuthService().ListFollowing(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load following"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.PublicUser]{
		Code:    200,
		Message: "Success",
		Data:    list,
	})
}

func (s *server) handleUserFollowers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, err := s.getAuthService().ListFollowers(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load followers"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.PublicUser]{
		Code:    200,
		Message: "Success",
		Data:    list,
	})
}

func (s *server) handleUserAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	targetID, err := strconv.Atoi(parts[0])
	if err != nil || targetID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid user id"})
		return
	}

	switch parts[1] {
	case "follow":
		s.withAuth(func(w http.ResponseWriter, r *http.Request) {
			s.handleUserFollow(w, r, targetID)
		}).ServeHTTP(w, r)
	case "follow-status":
		s.withAuth(func(w http.ResponseWriter, r *http.Request) {
			s.handleUserFollowStatus(w, r, targetID)
		}).ServeHTTP(w, r)
	default:
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
	}
}

func (s *server) handleUserFollow(w http.ResponseWriter, r *http.Request, targetID int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	var (
		status  store.FollowStatus
		applied bool
		err     error
	)
	switch r.Method {
	case http.MethodPost:
		status, applied, err = s.getAuthService().Follow(userID, targetID)
	case http.MethodDelete:
		status, applied, err = s.getAuthService().Unfollow(userID, targetID)
	default:
		writeMethodNotAllowed(w)
		return
	}

	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[followActionResponse]{
		Code:    200,
		Message: "Success",
		Data: followActionResponse{
			Status:  status,
			Applied: applied,
		},
	})
}

func (s *server) handleUserFollowStatus(w http.ResponseWriter, r *http.Request, targetID int) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	status, err := s.getAuthService().FollowStatus(targetID, userID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.FollowStatus]{
		Code:    200,
		Message: "Success",
		Data:    status,
	})
}

func (s *server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := sessionTokenFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
				Code:      401,
				Message:   "Unauthorized",
				ErrorCode: "AUTH_TOKEN_MISSING",
			})
			return
		}

		claims, err := s.tokenManager.Verify(token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "Unauthorized"
			errorCode := "AUTH_INVALID_TOKEN"
			if errors.Is(err, auth.ErrExpiredToken) {
				message = "Token expired"
				errorCode = "AUTH_TOKEN_EXPIRED"
			}

			writeJSON(w, status, apiResponse[any]{
				Code:      status,
				Message:   message,
				ErrorCode: errorCode,
			})
			return
		}

		if claims.JTI != "" {
			denied, err := s.getAuthService().IsTokenRevoked(r.Context(), claims.JTI)
			if err == nil && denied {
				writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
					Code:      401,
					Message:   "Token has been revoked",
					ErrorCode: "AUTH_TOKEN_REVOKED",
				})
				return
			}
		}

		userID, err := strconv.Atoi(claims.Subject)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
			return
		}

		// Confirm the user still exists and is active; disabled users' old
		// tokens must not keep working.
		userRecord, found := s.getAuthService().FindByID(userID)
		if !found || userRecord.Status != 1 {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
				Code:      401,
				Message:   "Unauthorized",
				ErrorCode: "AUTH_USER_DISABLED",
			})
			return
		}

		// Reject tokens issued before the user's last password reset: the
		// session version is incremented on reset to revoke every old token.
		if claims.SessionVersion != userRecord.SessionVer {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
				Code:      401,
				Message:   "Token has been revoked",
				ErrorCode: "AUTH_TOKEN_REVOKED",
			})
			return
		}

		// 登出会把 JTI 写入吊销名单；这里必须回源检查，否则登出只清了浏览器
		// Cookie，被复制的令牌在自然过期前仍然可用。
		if revoked, err := s.getAuthService().IsTokenRevoked(r.Context(), claims.JTI); err == nil && revoked {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
				Code:      401,
				Message:   "Token has been revoked",
				ErrorCode: "AUTH_TOKEN_REVOKED",
			})
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func userIDFromContext(ctx context.Context) (int, bool) {
	value := ctx.Value(userContextKey)
	userID, ok := value.(int)
	return userID, ok
}
