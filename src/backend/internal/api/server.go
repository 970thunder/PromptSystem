package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/config"
	"promptos-backend/internal/database"
	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

type server struct {
	config       config.Config
	tokenManager *auth.TokenManager
	userStore    store.UserManager
	promptStore  store.PromptManager
	commentStore store.CommentManager
	imageStorage storage.ImageStorage
	storageMode  string
}

func NewServer(cfg config.Config) http.Handler {
	imageStorage, err := storage.NewImageStorage(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize image storage: %v", err))
	}

	userStore := store.UserManager(store.NewUserStore())
	promptStore := store.PromptManager(store.NewMemoryPromptStore())
	commentStore := store.CommentManager(store.NewMemoryCommentStore())
	storageMode := "memory"

	db, err := database.OpenMySQL(cfg)
	if err == nil {
		if migrateErr := database.RunMigrations(db, ""); migrateErr != nil {
			log.Printf("failed to run MySQL migrations, falling back to memory store: %v", migrateErr)
		} else if seedErr := store.SeedMySQLData(db); seedErr == nil {
			userStore = store.NewMySQLUserStore(db)
			promptStore = store.NewMySQLPromptStore(db)
			commentStore = store.NewMySQLCommentStore(db)
			storageMode = "mysql"
			log.Printf("using MySQL-backed user and prompt stores at %s:%s/%s", cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDB)
		} else {
			log.Printf("failed to seed MySQL data, falling back to memory store: %v", seedErr)
		}
	} else {
		log.Printf("failed to connect to MySQL, falling back to memory store: %v", err)
	}

	s := &server{
		config:       cfg,
		tokenManager: auth.NewTokenManager(cfg.JWTSecret, time.Duration(cfg.JWTExpireHours)*time.Hour),
		userStore:    userStore,
		promptStore:  promptStore,
		commentStore: commentStore,
		imageStorage: imageStorage,
		storageMode:  storageMode,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/categories", s.handleCategories)
	mux.HandleFunc("/api/v1/prompts", s.handlePrompts)
	mux.HandleFunc("/api/v1/prompts/search", s.handlePromptSearch)
	mux.HandleFunc("/api/v1/prompts/", s.handlePromptDetail)
	mux.HandleFunc("/api/v1/comments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}

		s.withAuth(s.handleCommentLike).ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/v1/uploads/images", s.withAuth(s.handleImageUpload))
	mux.HandleFunc("/api/v1/auth/github", s.handleGitHubAuthStart)
	mux.HandleFunc("/api/v1/auth/github/callback", s.handleGitHubAuthCallback)
	mux.HandleFunc("/api/v1/user/login", s.handleLogin)
	mux.HandleFunc("/api/v1/user/register", s.handleRegister)
	mux.HandleFunc("/api/v1/user/info", s.withAuth(s.handleCurrentUser))
	mux.HandleFunc("/api/v1/user/favorites", s.withAuth(s.handleUserFavorites))
	mux.HandleFunc("/api/v1/user/likes", s.withAuth(s.handleUserLikes))
	mux.HandleFunc("/api/v1/user/logout", s.withAuth(s.handleLogout))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	return s.withCORS(mux)
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[map[string]any]{
		Code:    200,
		Message: "Success",
		Data: map[string]any{
			"status":      "ok",
			"service":     "promptos-backend",
			"runtime":     "golang",
			"environment": s.config.AppEnv,
			"storageMode": s.storageMode,
		},
	})
}

func (s *server) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.Category]{
		Code:    200,
		Message: "Success",
		Data:    store.Categories(),
	})
}

func (s *server) handlePrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handlePromptList(w, r)
	case http.MethodPost:
		s.withAuth(s.handlePromptCreate).ServeHTTP(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *server) handlePromptList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parseInt(query.Get("page"), 1)
	pageSize := parseInt(query.Get("pageSize"), 12)
	categoryID := parseInt(query.Get("categoryId"), 0)
	userID := parseInt(query.Get("userId"), 0)
	sortBy := query.Get("sort")
	keyword := query.Get("keyword")
	model := query.Get("model")

	list, err := s.promptStore.Query(store.PromptFilter{
		CategoryID: categoryID,
		SortBy:     sortBy,
		UserID:     userID,
		Keyword:    keyword,
		Model:      model,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load prompts"})
		return
	}
	start := (page - 1) * pageSize
	if start > len(list) {
		start = len(list)
	}

	end := start + pageSize
	if end > len(list) {
		end = len(list)
	}

	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Prompt]]{
		Code:    200,
		Message: "Success",
		Data: pageResponse[store.Prompt]{
			List:     list[start:end],
			Total:    len(list),
			Page:     page,
			PageSize: pageSize,
		},
	})
}

func (s *server) handlePromptSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	query := r.URL.Query()
	page := parseInt(query.Get("page"), 1)
	pageSize := parseInt(query.Get("pageSize"), 12)
	categoryID := parseInt(query.Get("categoryId"), 0)
	userID := parseInt(query.Get("userId"), 0)
	sortBy := query.Get("sort")
	keyword := query.Get("keyword")
	model := query.Get("model")

	list, err := s.promptStore.Query(store.PromptFilter{
		CategoryID: categoryID,
		SortBy:     sortBy,
		UserID:     userID,
		Keyword:    keyword,
		Model:      model,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to search prompts"})
		return
	}

	start := (page - 1) * pageSize
	if start > len(list) {
		start = len(list)
	}

	end := start + pageSize
	if end > len(list) {
		end = len(list)
	}

	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Prompt]]{
		Code:    200,
		Message: "Success",
		Data: pageResponse[store.Prompt]{
			List:     list[start:end],
			Total:    len(list),
			Page:     page,
			PageSize: pageSize,
		},
	})
}

type promptPayload struct {
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	Cover        string             `json:"cover"`
	Content      string             `json:"content"`
	SystemPrompt string             `json:"systemPrompt"`
	Model        string             `json:"model"`
	Params       store.PromptParams `json:"params"`
	CategoryID   int                `json:"categoryId"`
	Tags         []string           `json:"tags"`
}

type commentPayload struct {
	Content  string `json:"content"`
	ParentID *int   `json:"parentId"`
}

type commentActionResponse struct {
	Comment store.Comment `json:"comment"`
	Applied bool          `json:"applied"`
}

func (s *server) handlePromptCreate(w http.ResponseWriter, r *http.Request) {
	var payload promptPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	if message := validatePromptPayload(payload); message != "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: message,
		})
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	userRecord, found := s.userStore.FindByID(userID)
	if !found {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	prompt, err := s.promptStore.Create(store.CreatePromptInput{
		Title:        payload.Title,
		Description:  payload.Description,
		Cover:        payload.Cover,
		Content:      payload.Content,
		SystemPrompt: payload.SystemPrompt,
		Model:        payload.Model,
		Params:       payload.Params,
		CategoryID:   payload.CategoryID,
		Tags:         payload.Tags,
		User: store.User{
			ID:         userRecord.ID,
			Username:   userRecord.Username,
			Avatar:     userRecord.Avatar,
			Email:      userRecord.Email,
			Bio:        userRecord.Bio,
			Level:      userRecord.Level,
			Experience: userRecord.Experience,
			Status:     userRecord.Status,
			CreatedAt:  userRecord.CreatedAt,
		},
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.Prompt]{
		Code:    200,
		Message: "Success",
		Data:    prompt,
	})
}

func (s *server) handlePromptDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/prompts/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: "Invalid prompt id",
		})
		return
	}

	idText := parts[0]
	id, err := strconv.Atoi(idText)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: "Invalid prompt id",
		})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "like":
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w)
				return
			}
			s.withAuth(func(w http.ResponseWriter, r *http.Request) {
				s.handlePromptLike(w, r, id)
			}).ServeHTTP(w, r)
			return
		case "favorite":
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w)
				return
			}
			s.withAuth(func(w http.ResponseWriter, r *http.Request) {
				s.handlePromptFavorite(w, r, id)
			}).ServeHTTP(w, r)
			return
		case "comments":
			switch r.Method {
			case http.MethodGet:
				s.handlePromptComments(w, r, id)
			case http.MethodPost:
				s.withAuth(func(w http.ResponseWriter, r *http.Request) {
					s.handlePromptCommentCreate(w, r, id)
				}).ServeHTTP(w, r)
			default:
				writeMethodNotAllowed(w)
			}
			return
		default:
			writeJSON(w, http.StatusNotFound, apiResponse[any]{
				Code:    404,
				Message: "Not found",
			})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		prompt, ok, err := s.promptStore.FindByID(id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
				Code:    500,
				Message: "Failed to load prompt",
			})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, apiResponse[any]{
				Code:    404,
				Message: "Prompt not found",
			})
			return
		}

		writeJSON(w, http.StatusOK, apiResponse[store.Prompt]{
			Code:    200,
			Message: "Success",
			Data:    prompt,
		})
	case http.MethodPut:
		s.withAuth(func(w http.ResponseWriter, r *http.Request) {
			s.handlePromptUpdate(w, r, id)
		}).ServeHTTP(w, r)
	case http.MethodDelete:
		s.withAuth(func(w http.ResponseWriter, r *http.Request) {
			s.handlePromptDelete(w, r, id)
		}).ServeHTTP(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *server) handlePromptComments(w http.ResponseWriter, r *http.Request, id int) {
	comments, err := s.commentStore.ListByTarget("prompt", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:    500,
			Message: "Failed to load comments",
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.Comment]{
		Code:    200,
		Message: "Success",
		Data:    comments,
	})
}

func (s *server) handlePromptCommentCreate(w http.ResponseWriter, r *http.Request, id int) {
	var payload commentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	userRecord, found := s.userStore.FindByID(userID)
	if !found {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	comment, err := s.commentStore.Create(store.CreateCommentInput{
		TargetType: "prompt",
		TargetID:   id,
		User: store.User{
			ID:         userRecord.ID,
			Username:   userRecord.Username,
			Avatar:     userRecord.Avatar,
			Email:      userRecord.Email,
			Bio:        userRecord.Bio,
			Level:      userRecord.Level,
			Experience: userRecord.Experience,
			Status:     userRecord.Status,
			CreatedAt:  userRecord.CreatedAt,
		},
		Content:  payload.Content,
		ParentID: payload.ParentID,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "prompt not found" {
			status = http.StatusNotFound
		}

		writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.Comment]{
		Code:    200,
		Message: "Success",
		Data:    comment,
	})
}

func (s *server) handleCommentLike(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/comments/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "like" {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid comment id"})
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	comment, applied, err := s.commentStore.Like(id, userID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "comment not found" {
			status = http.StatusNotFound
		}

		writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[commentActionResponse]{
		Code:    200,
		Message: "Success",
		Data: commentActionResponse{
			Comment: comment,
			Applied: applied,
		},
	})
}

func (s *server) handlePromptLike(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	prompt, applied, err := s.promptStore.Like(id, userID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "prompt not found" {
			status = http.StatusNotFound
		}

		writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[promptActionResponse]{
		Code:    200,
		Message: "Success",
		Data: promptActionResponse{
			Prompt:  prompt,
			Applied: applied,
		},
	})
}

func (s *server) handlePromptFavorite(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	prompt, applied, err := s.promptStore.Favorite(id, userID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "prompt not found" {
			status = http.StatusNotFound
		}

		writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[promptActionResponse]{
		Code:    200,
		Message: "Success",
		Data: promptActionResponse{
			Prompt:  prompt,
			Applied: applied,
		},
	})
}

func (s *server) handlePromptUpdate(w http.ResponseWriter, r *http.Request, id int) {
	var payload promptPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
		return
	}
	if message := validatePromptPayload(payload); message != "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: message})
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	userRecord, found := s.userStore.FindByID(userID)
	if !found {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	prompt, err := s.promptStore.Update(id, userID, store.CreatePromptInput{
		Title:        payload.Title,
		Description:  payload.Description,
		Cover:        payload.Cover,
		Content:      payload.Content,
		SystemPrompt: payload.SystemPrompt,
		Model:        payload.Model,
		Params:       payload.Params,
		CategoryID:   payload.CategoryID,
		Tags:         payload.Tags,
		User: store.User{
			ID:         userRecord.ID,
			Username:   userRecord.Username,
			Avatar:     userRecord.Avatar,
			Email:      userRecord.Email,
			Bio:        userRecord.Bio,
			Level:      userRecord.Level,
			Experience: userRecord.Experience,
			Status:     userRecord.Status,
			CreatedAt:  userRecord.CreatedAt,
		},
	})
	if err != nil {
		status := http.StatusBadRequest
		switch err.Error() {
		case "forbidden":
			status = http.StatusForbidden
		case "prompt not found":
			status = http.StatusNotFound
		}
		writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.Prompt]{Code: 200, Message: "Success", Data: prompt})
}

func (s *server) handlePromptDelete(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	if err := s.promptStore.Delete(id, userID); err != nil {
		status := http.StatusBadRequest
		switch err.Error() {
		case "forbidden":
			status = http.StatusForbidden
		case "prompt not found":
			status = http.StatusNotFound
		}
		writeJSON(w, status, apiResponse[any]{Code: status, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: "Success", Data: nil})
}

func (s *server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.config.AllowedOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return fallback
	}

	return number
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, apiResponse[any]{
		Code:    405,
		Message: "Method not allowed",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func validatePromptPayload(payload promptPayload) string {
	if strings.TrimSpace(payload.Title) == "" {
		return "Title is required"
	}
	if strings.TrimSpace(payload.Description) == "" {
		return "Description is required"
	}
	if strings.TrimSpace(payload.Cover) == "" {
		return "Cover image is required"
	}
	if strings.TrimSpace(payload.Content) == "" {
		return "Prompt content is required"
	}
	if strings.TrimSpace(payload.Model) == "" {
		return "Model is required"
	}
	if payload.CategoryID <= 0 {
		return "Category is required"
	}
	if uniqueTagCount(payload.Tags) > store.MaxPromptTags {
		return fmt.Sprintf("At most %d tags are allowed", store.MaxPromptTags)
	}

	return ""
}

func uniqueTagCount(tags []string) int {
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		seen[tag] = struct{}{}
	}

	return len(seen)
}

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

type promptActionResponse struct {
	Prompt  store.Prompt `json:"prompt"`
	Applied bool         `json:"applied"`
}

type pageResponse[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}
