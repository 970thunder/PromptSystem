package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/store"
)

type contextKey string

const userContextKey contextKey = "userID"

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

type updateUserRequest struct {
	Username string `json:"username"`
	Bio      string `json:"bio"`
	Avatar   string `json:"avatar"`
}

type authResponse struct {
	Token string           `json:"token"`
	User  store.PublicUser `json:"user"`
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload loginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
		return
	}

	user, err := s.userStore.Authenticate(payload.Email, payload.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "邮箱或密码错误"})
		return
	}

	token, err := s.tokenManager.Generate(user.ID, user.Email)
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

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var payload registerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
		return
	}

	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "用户名、邮箱和密码不能为空"})
		return
	}

	user, err := s.userStore.Register(payload.Username, payload.Email, payload.Password)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, apiResponse[any]{Code: 409, Message: "该邮箱已注册"})
		case errors.Is(err, store.ErrWeakPassword):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "密码至少需要 8 位"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Register failed"})
		}
		return
	}

	token, err := s.tokenManager.Generate(user.ID, user.Email)
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

	writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: "Success"})
}

func (s *server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		claims, err := s.tokenManager.Verify(token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "Unauthorized"
			if errors.Is(err, auth.ErrExpiredToken) {
				message = "Token expired"
			}

			writeJSON(w, status, apiResponse[any]{Code: 401, Message: message})
			return
		}

		userID, err := strconv.Atoi(claims.Subject)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
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
