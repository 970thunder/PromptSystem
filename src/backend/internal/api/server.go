package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

type server struct {
	config config.Config
}

func NewServer(cfg config.Config) http.Handler {
	s := &server{config: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/categories", s.handleCategories)
	mux.HandleFunc("/api/v1/prompts", s.handlePrompts)
	mux.HandleFunc("/api/v1/prompts/", s.handlePromptDetail)

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
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	query := r.URL.Query()
	page := parseInt(query.Get("page"), 1)
	pageSize := parseInt(query.Get("pageSize"), 12)
	categoryID := parseInt(query.Get("categoryId"), 0)
	sortBy := query.Get("sort")

	list := store.FilterPrompts(categoryID, sortBy)
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

func (s *server) handlePromptDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/api/v1/prompts/")
	id, err := strconv.Atoi(idText)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: "Invalid prompt id",
		})
		return
	}

	prompt, ok := store.FindPromptByID(id)
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

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

type pageResponse[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}
