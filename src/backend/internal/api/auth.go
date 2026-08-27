package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/store"
)

type contextKey string

const userContextKey contextKey = "userID"

// maxPasswordBytes is the maximum password length accepted by the API. bcrypt
// silently ignores input beyond its first 72 bytes, so we reject longer
// passwords up front to avoid account enumeration via timing and to keep the
// rule consistent between the API boundary and the store.
const maxPasswordBytes = 72

// resetGenericMessage is returned whether or not the target account exists so
// the password-reset endpoint cannot be used to enumerate registered emails.
const resetGenericMessage = "If the account exists, the password has been reset"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Captcha  string `json:"captcha"`
}

type captchaRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Email    string `json:"email"`
	Captcha  string `json:"captcha"`
	Password string `json:"password"`
}

type captchaResponse struct {
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	DevCode          string `json:"devCode,omitempty"`
}

type updateUserRequest struct {
	Username string `json:"username"`
	Bio      string `json:"bio"`
	Avatar   string `json:"avatar"`
}

type authResponse struct {
	Token string           `json:"token"`
	User  store.PublicUser `json:"user"`
}

type followActionResponse struct {
	Status  store.FollowStatus `json:"status"`
	Applied bool               `json:"applied"`
}

func (s *server) handleCaptcha(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload captchaRequest
	if !s.enforceRateLimits(r.Context(), w, "captcha", rateLimitRule{bucket: rateLimitIP(r), limit: 5, window: 10 * time.Minute}) {
		return
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if store.IsValidEmail(payload.Email) && !s.enforceRateLimits(r.Context(), w, "captcha", rateLimitRule{bucket: rateLimitEmail(payload.Email), limit: 1, window: captchaCooldown}) {
		return
	}

	code, expiresAt, retryAfter, err := s.issueRedisCaptcha(r.Context(), payload.Email)
	if err != nil {
		if errors.Is(err, store.ErrInvalidEmail) {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid email address"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to generate captcha"})
		return
	}

	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		writeJSON(w, http.StatusTooManyRequests, apiResponse[any]{Code: 429, Message: "Captcha was sent too frequently"})
		return
	}

	log.Printf("dev email captcha for %s: %s", strings.TrimSpace(payload.Email), code)
	response := captchaResponse{
		ExpiresInSeconds: int(timeUntil(expiresAt).Seconds()),
	}
	if s.config.AppEnv != "production" {
		response.DevCode = code
	}

	writeJSON(w, http.StatusOK, apiResponse[captchaResponse]{
		Code:    200,
		Message: "Success",
		Data:    response,
	})
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload loginRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "login",
		rateLimitRule{bucket: rateLimitIP(r), limit: 10, window: time.Minute},
		rateLimitRule{bucket: rateLimitEmail(payload.Email), limit: 5, window: 15 * time.Minute}) {
		return
	}
	if len(payload.Password) > maxPasswordBytes {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password is too long", ErrorCode: "INVALID_PASSWORD"})
		return
	}

	user, err := s.userStore.Authenticate(payload.Email, payload.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Invalid email or password"})
		return
	}

	token, err := s.tokenManager.Generate(user.ID, user.Email, user.SessionVer)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Token generation failed"})
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

func (s *server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload resetPasswordRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "password_reset",
		rateLimitRule{bucket: rateLimitIP(r), limit: 5, window: 10 * time.Minute},
		rateLimitRule{bucket: rateLimitEmail(payload.Email), limit: 5, window: 15 * time.Minute}) {
		return
	}
	if len(payload.Password) > maxPasswordBytes {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password is too long", ErrorCode: "INVALID_PASSWORD"})
		return
	}

	if strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Email and password are required"})
		return
	}

	if !store.IsValidEmail(payload.Email) {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid email address"})
		return
	}

	if !s.verifyRedisCaptcha(r.Context(), payload.Email, payload.Captcha) {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid or expired captcha"})
		return
	}

	if err := s.userStore.ResetPassword(payload.Email, payload.Password); err != nil {
		switch {
		case errors.Is(err, store.ErrUserNotFound):
			// Deliberately return the same non-revealing success whether the
			// account exists or not so the endpoint cannot be used to enumerate
			// registered emails via the reset flow.
			writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: resetGenericMessage})
		case errors.Is(err, store.ErrInvalidEmail):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid email address"})
		case errors.Is(err, store.ErrWeakPassword):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password must be at least 8 characters"})
		case errors.Is(err, store.ErrPasswordTooLong):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password must be 72 bytes or fewer"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Reset password failed"})
		}
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: resetGenericMessage})
}

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload registerRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "register",
		rateLimitRule{bucket: rateLimitIP(r), limit: 5, window: 10 * time.Minute},
		rateLimitRule{bucket: rateLimitEmail(payload.Email), limit: 3, window: time.Hour}) {
		return
	}
	if len(payload.Password) > maxPasswordBytes {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password is too long", ErrorCode: "INVALID_PASSWORD"})
		return
	}

	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Username, email, and password are required"})
		return
	}

	if !store.IsValidEmail(payload.Email) {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid email address"})
		return
	}

	if !s.verifyRedisCaptcha(r.Context(), payload.Email, payload.Captcha) {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid or expired captcha"})
		return
	}

	user, err := s.userStore.Register(payload.Username, payload.Email, payload.Password)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, apiResponse[any]{Code: 409, Message: "Email already registered"})
		case errors.Is(err, store.ErrInvalidEmail):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid email address"})
		case errors.Is(err, store.ErrWeakPassword):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password must be at least 8 characters"})
		case errors.Is(err, store.ErrPasswordTooLong):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password must be 72 bytes or fewer"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Register failed"})
		}
		return
	}

	token, err := s.tokenManager.Generate(user.ID, user.Email, user.SessionVer)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Token generation failed"})
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
		user, found := s.userStore.FindByID(userID)
		if !found {
			writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "User not found"})
			return
		}

		writeJSON(w, http.StatusOK, apiResponse[store.PublicUser]{
			Code:    200,
			Message: "Success",
			Data:    store.ToPublicUser(user),
		})
	case http.MethodPut:
		var payload updateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
			return
		}

		user, err := s.userStore.UpdateProfile(userID, payload.Username, payload.Bio, payload.Avatar)
		if err != nil {
			writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "User not found"})
			return
		}

		writeJSON(w, http.StatusOK, apiResponse[store.PublicUser]{
			Code:    200,
			Message: "Success",
			Data:    store.ToPublicUser(user),
		})
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token != "" {
		claims, err := s.tokenManager.Verify(token)
		if err == nil && claims.JTI != "" {
			// Denylist the token until it would have expired naturally.
			ttl := time.Until(time.Unix(claims.Expiry, 0))
			if ttl < 0 {
				ttl = 0
			}
			if s.cache != nil {
				_ = s.cache.Set(r.Context(), "promptos:jwt:denylist:"+claims.JTI, "1", ttl)
			}
		}
	}

	writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: "Success"})
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

	list, err := s.promptStore.ListUserFavorites(userID)
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

	list, err := s.promptStore.ListUserLikes(userID)
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

	list, err := s.promptStore.ListUserHistory(userID)
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

	list, err := s.userStore.ListFollowing(userID)
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

	list, err := s.userStore.ListFollowers(userID)
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
		status, applied, err = s.userStore.Follow(userID, targetID)
	case http.MethodDelete:
		status, applied, err = s.userStore.Unfollow(userID, targetID)
	default:
		writeMethodNotAllowed(w)
		return
	}

	if err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, store.ErrUserNotFound) {
			statusCode = http.StatusNotFound
		}
		writeJSON(w, statusCode, apiResponse[any]{Code: statusCode, Message: err.Error()})
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

	status, err := s.userStore.FollowStatus(targetID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, store.ErrUserNotFound) {
			statusCode = http.StatusNotFound
		}
		writeJSON(w, statusCode, apiResponse[any]{Code: statusCode, Message: err.Error()})
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
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
				Code:      401,
				Message:   "Unauthorized",
				ErrorCode: "AUTH_TOKEN_MISSING",
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
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

		if s.cache != nil && claims.JTI != "" {
			denied, err := s.cache.Exists(r.Context(), "promptos:jwt:denylist:"+claims.JTI)
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
		userRecord, found := s.userStore.FindByID(userID)
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

		ctx := context.WithValue(r.Context(), userContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func userIDFromContext(ctx context.Context) (int, bool) {
	value := ctx.Value(userContextKey)
	userID, ok := value.(int)
	return userID, ok
}
