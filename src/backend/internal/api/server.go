package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"promptos-backend/internal/auth"
	"promptos-backend/internal/cache"
	"promptos-backend/internal/config"
	"promptos-backend/internal/database"
	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

type server struct {
	config       config.Config
	tokenManager *auth.TokenManager
	captcha      *captchaManager
	githubClient *http.Client
	cache        cache.Cache
	userStore    store.UserManager
	promptStore  store.PromptManager
	commentStore store.CommentManager
	uploadStore  store.UploadManager
	imageStorage storage.ImageStorage
	storageMode  string
}

// serverDeps carries the pluggable dependencies of the API server. NewServer
// resolves them from configuration; tests construct them directly so HTTP
// integration tests do not depend on real MySQL, Redis, or the network.
type serverDeps struct {
	config       config.Config
	tokenManager *auth.TokenManager
	captcha      *captchaManager
	githubClient *http.Client
	cache        cache.Cache
	userStore    store.UserManager
	promptStore  store.PromptManager
	commentStore store.CommentManager
	uploadStore  store.UploadManager
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
	uploadStore := store.UploadManager(store.NewMemoryUploadStore())
	storageMode := "memory"

	db, err := database.OpenMySQL(cfg)
	if err == nil {
		if migrateErr := database.RunMigrations(db, ""); migrateErr != nil {
			log.Printf("failed to run MySQL migrations, falling back to memory store: %v", migrateErr)
		} else if seedErr := store.SeedMySQLData(db); seedErr == nil {
			userStore = store.NewMySQLUserStore(db)
			promptStore = store.NewMySQLPromptStore(db)
			if s, ok := promptStore.(interface{ SetAllowedImageDomains([]string) }); ok {
				s.SetAllowedImageDomains(cfg.AllowedImageDomains)
			}
			commentStore = store.NewMySQLCommentStore(db)
			uploadStore = store.NewMySQLUploadStore(db)
			storageMode = "mysql"
			log.Printf("using MySQL-backed user and prompt stores at %s:%s/%s", cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDB)
		} else {
			log.Printf("failed to seed MySQL data, falling back to memory store: %v", seedErr)
		}
	} else {
		log.Printf("failed to connect to MySQL, falling back to memory store: %v", err)
	}

	return newServerWithDeps(serverDeps{
		config:       cfg,
		tokenManager: auth.NewTokenManager(cfg.JWTSecret, time.Duration(cfg.JWTExpireHours)*time.Hour),
		captcha:      newCaptchaManager(),
		githubClient: newGitHubClient(),
		cache:        cache.New(cfg),
		userStore:    userStore,
		promptStore:  promptStore,
		commentStore: commentStore,
		uploadStore:  uploadStore,
		imageStorage: imageStorage,
		storageMode:  storageMode,
	})
}

// newServerWithDeps assembles the HTTP handler from explicit dependencies. It is
// the single wiring point for both production (NewServer) and tests.
func newServerWithDeps(deps serverDeps) http.Handler {
	s := &server{
		config:       deps.config,
		tokenManager: deps.tokenManager,
		captcha:      deps.captcha,
		githubClient: deps.githubClient,
		cache:        deps.cache,
		userStore:    deps.userStore,
		promptStore:  deps.promptStore,
		commentStore: deps.commentStore,
		uploadStore:  deps.uploadStore,
		imageStorage: deps.imageStorage,
		storageMode:  deps.storageMode,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/health/live", s.handleHealthLive)
	mux.HandleFunc("/api/v1/health/ready", s.handleHealthReady)
	mux.HandleFunc("/api/v1/categories", s.handleCategories)
	mux.HandleFunc("/api/v1/home/summary", s.handleHomeSummary)
	mux.HandleFunc("/api/v1/prompts", s.handlePrompts)
	mux.HandleFunc("/api/v1/prompts/search", s.handlePromptSearch)
	mux.HandleFunc("/api/v1/prompts/", s.handlePromptDetail)
	mux.HandleFunc("/api/v1/comments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}

		s.withAuth(s.handleCommentAction).ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/v1/uploads/images", s.withAuth(s.handleImageUpload))
	mux.HandleFunc("/api/v1/auth/github", s.handleGitHubAuthStart)
	mux.HandleFunc("/api/v1/auth/github/callback", s.handleGitHubAuthCallback)
	mux.HandleFunc("/api/v1/auth/exchange", s.handleAuthExchange)
	mux.HandleFunc("/api/v1/user/login", s.handleLogin)
	mux.HandleFunc("/api/v1/user/captcha", s.handleCaptcha)
	mux.HandleFunc("/api/v1/user/password/reset", s.handleResetPassword)
	mux.HandleFunc("/api/v1/user/register", s.handleRegister)
	mux.HandleFunc("/api/v1/user/info", s.withAuth(s.handleCurrentUser))
	mux.HandleFunc("/api/v1/user/favorites", s.withAuth(s.handleUserFavorites))
	mux.HandleFunc("/api/v1/user/likes", s.withAuth(s.handleUserLikes))
	// History list is served by the paginated handler in prompt_handlers.go
	// (handleUserHistoryList) which returns a pageResponse consistent with
	// parsePageParams and excludes soft-deleted prompts.
	mux.HandleFunc("/api/v1/user/history", s.withAuth(s.handleUserHistoryList))
	mux.HandleFunc("/api/v1/user/drafts", s.withAuth(s.handleUserDrafts))
	mux.HandleFunc("/api/v1/user/prompts/", s.withAuth(s.handleUserPromptDetail))
	mux.HandleFunc("/api/v1/user/following", s.withAuth(s.handleUserFollowing))
	mux.HandleFunc("/api/v1/user/followers", s.withAuth(s.handleUserFollowers))
	mux.HandleFunc("/api/v1/user/logout", s.withAuth(s.handleLogout))
	mux.HandleFunc("/api/v1/users/", s.handleUserAction)
	mux.HandleFunc("/uploads/", s.handleStaticUpload)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return chain(mux,
		func(next http.Handler) http.Handler { return withRecovery(logger, next) },
		withRequestID,
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), envContextKey, s.config.AppEnv)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		},
		func(next http.Handler) http.Handler { return withAccessLog(logger, next) },
		withSecurityHeaders,
		s.withCORS,
	)
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

type reportCommentPayload struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type reportActionResponse struct {
	Report  store.Report `json:"report"`
	Applied bool         `json:"applied"`
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, apiResponse[any]{
		Code:    405,
		Message: "Method not allowed",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Surface the machine-readable errorCode to the structured access log via
	// the (possibly wrapped) ResponseWriter, without changing the client
	// contract. Never includes tokens, passwords, or secrets.
	if wc, ok := payload.(errorCoded); ok {
		if code := wc.errorCodeValue(); code != "" {
			if er, ok := w.(errorCodeRecorder); ok {
				er.setErrorCode(code)
			}
		}
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// errorCoded lets writeJSON extract the stable errorCode from any apiResponse
// payload for structured logging.
type errorCoded interface {
	errorCodeValue() string
}

type apiResponse[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode,omitempty"`
	Data      T      `json:"data"`
}

// errorCodeValue implements errorCoded for apiResponse.
func (a apiResponse[T]) errorCodeValue() string {
	return a.ErrorCode
}

type pageResponse[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}
