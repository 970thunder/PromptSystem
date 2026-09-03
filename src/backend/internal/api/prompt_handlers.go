package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"promptos-backend/internal/store"
)

type promptPayload struct {
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	Cover        string             `json:"cover"`
	Images       []string           `json:"images"`
	Content      string             `json:"content"`
	SystemPrompt string             `json:"systemPrompt"`
	Model        string             `json:"model"`
	Params       store.PromptParams `json:"params"`
	CategoryID   int                `json:"categoryId"`
	Tags         []string           `json:"tags"`
	Status       *int               `json:"status"`
}

type promptActionResponse struct {
	Prompt  store.Prompt `json:"prompt"`
	Applied bool         `json:"applied"`
}

func (s *server) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	var categories []store.Category
	if hit, _ := s.cachedJSON(r.Context(), cacheKeyCategories, &categories); hit {
		writeJSON(w, http.StatusOK, apiResponse[[]store.Category]{
			Code:    200,
			Message: "Success",
			Data:    categories,
		})
		return
	}

	categories, err := s.promptStore.ListCategories()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:      500,
			Message:   "Failed to load categories",
			ErrorCode: "CATEGORIES_LOAD_FAILED",
			Data:      nil,
		})
		return
	}
	if len(categories) == 0 {
		categories = store.Categories()
	}

	s.setCached(r.Context(), cacheKeyCategories, categories, categoriesTTL)
	writeJSON(w, http.StatusOK, apiResponse[[]store.Category]{
		Code:    200,
		Message: "Success",
		Data:    categories,
	})
}

func (s *server) handleHomeSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	var summary store.HomeSummary
	if hit, _ := s.cachedJSON(r.Context(), cacheKeyHomeSummary, &summary); hit {
		writeJSON(w, http.StatusOK, apiResponse[store.HomeSummary]{
			Code:    200,
			Message: "Success",
			Data:    summary,
		})
		return
	}

	summary, err := s.promptStore.HomeSummary()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:      500,
			Message:   "Failed to load home summary",
			ErrorCode: "HOME_SUMMARY_FAILED",
			Data:      nil,
		})
		return
	}

	s.setCached(r.Context(), cacheKeyHomeSummary, summary, homeSummaryTTL)
	// Hot tags are part of the summary but cached under their own key so other
	// consumers can fetch just the tags without the full aggregate.
	s.setCached(r.Context(), cacheKeyHotTags, summary.HotTags, hotTagsTTL)
	writeJSON(w, http.StatusOK, apiResponse[store.HomeSummary]{
		Code:    200,
		Message: "Success",
		Data:    summary,
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
	if !s.enforceRateLimits(r.Context(), w, "search", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}) {
		return
	}
	query := r.URL.Query()
	page, pageSize, err := parsePageParams(stringPageParams{Page: query.Get("page"), PageSize: query.Get("pageSize")})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	categoryID := parseInt(query.Get("categoryId"), 0)
	userID := parseInt(query.Get("userId"), 0)
	sortBy := query.Get("sort")
	keyword := query.Get("keyword")
	model := query.Get("model")
	tag := query.Get("tag")

	list, total, err := s.promptStore.QueryPage(store.PromptFilter{
		CategoryID: categoryID,
		SortBy:     sortBy,
		UserID:     userID,
		Keyword:    keyword,
		Model:      model,
		Tag:        tag,
	}, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load prompts"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Prompt]]{
		Code:    200,
		Message: "Success",
		Data: pageResponse[store.Prompt]{
			List:     list,
			Total:    total,
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
	if !s.enforceRateLimits(r.Context(), w, "search", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}) {
		return
	}

	query := r.URL.Query()
	page, pageSize, err := parsePageParams(stringPageParams{Page: query.Get("page"), PageSize: query.Get("pageSize")})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	categoryID := parseInt(query.Get("categoryId"), 0)
	userID := parseInt(query.Get("userId"), 0)
	sortBy := query.Get("sort")
	keyword := query.Get("keyword")
	model := query.Get("model")
	tag := query.Get("tag")

	list, total, err := s.promptStore.QueryPage(store.PromptFilter{
		CategoryID: categoryID,
		SortBy:     sortBy,
		UserID:     userID,
		Keyword:    keyword,
		Model:      model,
		Tag:        tag,
	}, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to search prompts"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Prompt]]{
		Code:    200,
		Message: "Success",
		Data: pageResponse[store.Prompt]{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
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

	userRecord, found := s.getAuthService().FindByID(userID)
	if !found {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	if err := s.validateUploadOwnership(userID, payload.Cover, payload.Images); err != nil {
		if errors.Is(err, errInvalidUploadOwnership) {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: http.StatusBadRequest, ErrorCode: "INVALID_UPLOAD_OWNERSHIP", Message: "Upload reference is invalid"})
		} else {
			writeStoreError(w, err)
		}
		return
	}

	prompt, err := s.getPromptService().Create(r.Context(), store.CreatePromptInput{
		Title:        payload.Title,
		Description:  payload.Description,
		Cover:        payload.Cover,
		Images:       payload.Images,
		Content:      payload.Content,
		SystemPrompt: payload.SystemPrompt,
		Model:        payload.Model,
		Params:       payload.Params,
		CategoryID:   payload.CategoryID,
		Tags:         payload.Tags,
		Status:       promptPayloadStatus(payload),
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
		writeStoreError(w, err)
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
		case "unlike":
			if r.Method != http.MethodDelete {
				writeMethodNotAllowed(w)
				return
			}
			s.withAuth(func(w http.ResponseWriter, r *http.Request) {
				s.handlePromptUnlike(w, r, id)
			}).ServeHTTP(w, r)
			return
		case "unfavorite":
			if r.Method != http.MethodDelete {
				writeMethodNotAllowed(w)
				return
			}
			s.withAuth(func(w http.ResponseWriter, r *http.Request) {
				s.handlePromptUnfavorite(w, r, id)
			}).ServeHTTP(w, r)
			return
		case "interaction":
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w)
				return
			}
			s.withAuth(func(w http.ResponseWriter, r *http.Request) {
				s.handlePromptInteractionStatus(w, r, id)
			}).ServeHTTP(w, r)
			return
		case "view":
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w)
				return
			}
			// Views are auth-optional (anonymous callers count a view without a
			// history row); handlePromptView resolves the optional user itself.
			s.handlePromptView(w, r, id)
			return
		case "report":
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w)
				return
			}
			s.withAuth(func(w http.ResponseWriter, r *http.Request) {
				s.handlePromptReport(w, r, id)
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
		case "related":
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w)
				return
			}
			s.handleRelatedPrompts(w, r, id)
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

// handleRelatedPrompts performs a fresh backend query instead of relying on
// whatever page happens to be cached in the browser store.
func (s *server) handleRelatedPrompts(w http.ResponseWriter, r *http.Request, id int) {
	current, found, err := s.promptStore.FindByID(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load related prompts", ErrorCode: "RELATED_LOAD_FAILED"})
		return
	}
	if !found {
		writeAPIError(w, &apiError{Status: http.StatusNotFound, Code: "PROMPT_NOT_FOUND", Message: "Prompt not found"})
		return
	}
	list, _, err := s.promptStore.QueryPage(store.PromptFilter{CategoryID: current.CategoryID, SortBy: "popular"}, 1, 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load related prompts", ErrorCode: "RELATED_LOAD_FAILED"})
		return
	}
	filtered := make([]store.Prompt, 0, 3)
	for _, item := range list {
		if item.ID == id {
			continue
		}
		filtered = append(filtered, item)
		if len(filtered) == 3 {
			break
		}
	}
	writeJSON(w, http.StatusOK, apiResponse[[]store.Prompt]{Code: 200, Message: "Success", Data: filtered})
}

func (s *server) handleUserPromptDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/user/prompts/")
	id, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid prompt id"})
		return
	}

	prompt, found, err := s.promptStore.FindOwnedByID(id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load prompt"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Prompt not found"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.Prompt]{
		Code:    200,
		Message: "Success",
		Data:    prompt,
	})
}

func (s *server) handleUserDrafts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, err := s.promptStore.ListUserDrafts(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Failed to load drafts"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[[]store.Prompt]{
		Code:    200,
		Message: "Success",
		Data:    list,
	})
}

// handleUserHistoryList returns the requesting user's browsing history as a
// paginated page, aligned with parsePageParams. Only the caller's own history
// is returned (userID comes from the auth context) and soft-deleted prompts are
// excluded by the store.
func (s *server) handleUserHistoryList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, ErrorCode: "AUTH_TOKEN_MISSING", Message: "Unauthorized"})
			return
		}
		if !s.enforceRateLimits(r.Context(), w, "history_clear", rateLimitRule{bucket: rateLimitUser(userID), limit: 6, window: time.Hour}) {
			return
		}
		if err := s.getAuthService().ClearHistory(userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, ErrorCode: "HISTORY_CLEAR_FAILED", Message: "Failed to clear history"})
			return
		}
		writeJSON(w, http.StatusOK, apiResponse[map[string]bool]{Code: 200, Message: "Success", Data: map[string]bool{"cleared": true}})
		return
	}
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	query := r.URL.Query()
	page, pageSize, err := parsePageParams(stringPageParams{Page: query.Get("page"), PageSize: query.Get("pageSize")})
	if err != nil {
		writeAPIError(w, err)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	list, total, err := s.promptStore.ListUserHistoryPage(userID, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:      500,
			Message:   "Failed to load history",
			ErrorCode: "HISTORY_LOAD_FAILED",
			Data:      nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Prompt]]{
		Code:    200,
		Message: "Success",
		Data: pageResponse[store.Prompt]{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// writeStoreError maps store sentinel errors to stable HTTP status codes and
// machine-readable errorCodes. It deliberately never forwards internal error
// text to the client, so database/wiring detail cannot leak.
// Any unrecognized error is normalized to a 500 INTERNAL_ERROR. Treating an
// unknown store failure as a client mistake would hide outages and encourage
// retries that cannot succeed.
func writeStoreError(w http.ResponseWriter, err error) {
	var apiErr *apiError
	switch {
	case errors.Is(err, store.ErrUserNotFound):
		apiErr = &apiError{Status: http.StatusNotFound, Code: "USER_NOT_FOUND", Message: "User not found"}
	case errors.Is(err, store.ErrCannotFollowSelf):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "CANNOT_FOLLOW_SELF", Message: "You cannot follow yourself"}
	case errors.Is(err, store.ErrInvalidUser):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_USER", Message: "Invalid user"}
	case errors.Is(err, store.ErrPromptNotFound):
		apiErr = &apiError{Status: http.StatusNotFound, Code: "PROMPT_NOT_FOUND", Message: "Prompt not found"}
	case errors.Is(err, store.ErrPromptForbidden):
		apiErr = &apiError{Status: http.StatusForbidden, Code: "PROMPT_FORBIDDEN", Message: "Forbidden"}
	case errors.Is(err, store.ErrCommentNotFound):
		apiErr = &apiError{Status: http.StatusNotFound, Code: "COMMENT_NOT_FOUND", Message: "Comment not found"}
	case errors.Is(err, store.ErrInvalidCommentID):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_COMMENT_ID", Message: "Invalid comment id"}
	case errors.Is(err, store.ErrInvalidCommentTarget):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_COMMENT_TARGET", Message: "Invalid comment target"}
	case errors.Is(err, store.ErrInvalidCommentContent):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_COMMENT_CONTENT", Message: "Invalid comment content"}
	case errors.Is(err, store.ErrInvalidCommentUser):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_USER", Message: "Invalid user"}
	case errors.Is(err, store.ErrCommentParentNotFound):
		apiErr = &apiError{Status: http.StatusNotFound, Code: "COMMENT_PARENT_NOT_FOUND", Message: "Parent comment not found"}
	case errors.Is(err, store.ErrCommentParentMismatch):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "COMMENT_PARENT_MISMATCH", Message: "Parent comment does not match prompt"}
	case errors.Is(err, store.ErrReportNotFound):
		apiErr = &apiError{Status: http.StatusNotFound, Code: "REPORT_NOT_FOUND", Message: "Report not found"}
	case errors.Is(err, store.ErrReportDetailTooLong):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "REPORT_DETAIL_TOO_LONG", Message: "Report detail is too long"}
	case errors.Is(err, store.ErrInvalidCategory):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_CATEGORY", Message: "Category does not exist"}
	case errors.Is(err, store.ErrInvalidTag):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_TAG", Message: "Invalid tag"}
	case errors.Is(err, store.ErrInvalidImageURL):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_IMAGE_URL", Message: "Invalid image URL"}
	case errors.Is(err, store.ErrInvalidReportReason):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_REPORT_REASON", Message: "Invalid report reason"}
	case errors.Is(err, store.ErrInvalidContent):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "INVALID_CONTENT", Message: "Content contains invalid characters"}
	case errors.Is(err, store.ErrContentTooLong):
		apiErr = &apiError{Status: http.StatusRequestEntityTooLarge, Code: "CONTENT_TOO_LONG", Message: "Content exceeds the allowed length"}
	case errors.Is(err, store.ErrUnsafeContent):
		apiErr = &apiError{Status: http.StatusBadRequest, Code: "UNSAFE_CONTENT", Message: "Content does not meet platform safety rules"}
	default:
		apiErr = &apiError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "Internal server error"}
	}
	writeAPIError(w, apiErr)
}

// optionalUserID resolves the requesting user from an optional Bearer token,
// returning (id, true) when the token is valid and the user is active, and
// (0, false) otherwise. It is used only by the anonymous-friendly view endpoint
// so a guest can bump a prompt's view counter without a history row, while a
// signed-in user's view is still attributed to their history.
func (s *server) optionalUserID(r *http.Request) (int, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	claims, err := s.tokenManager.Verify(token)
	if err != nil {
		return 0, false
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, false
	}

	userRecord, found := s.getAuthService().FindByID(userID)
	if !found || userRecord.Status != 1 || claims.SessionVersion != userRecord.SessionVer {
		return 0, false
	}

	return userID, true
}

func (s *server) handlePromptLike(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "interaction", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}, rateLimitRule{bucket: rateLimitUser(userID), limit: 240, window: time.Minute}) {
		return
	}

	prompt, applied, err := s.getPromptService().Like(r.Context(), id, userID)
	if err != nil {
		writeStoreError(w, err)
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
	if !s.enforceRateLimits(r.Context(), w, "interaction", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}, rateLimitRule{bucket: rateLimitUser(userID), limit: 240, window: time.Minute}) {
		return
	}

	prompt, applied, err := s.getPromptService().Favorite(r.Context(), id, userID)
	if err != nil {
		writeStoreError(w, err)
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

func (s *server) handlePromptUnlike(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "interaction", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}, rateLimitRule{bucket: rateLimitUser(userID), limit: 240, window: time.Minute}) {
		return
	}

	prompt, applied, err := s.getPromptService().Unlike(r.Context(), id, userID)
	if err != nil {
		writeStoreError(w, err)
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

func (s *server) handlePromptUnfavorite(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "interaction", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}, rateLimitRule{bucket: rateLimitUser(userID), limit: 240, window: time.Minute}) {
		return
	}

	prompt, applied, err := s.getPromptService().Unfavorite(r.Context(), id, userID)
	if err != nil {
		writeStoreError(w, err)
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

// handlePromptInteractionStatus returns whether the current user has liked or
// favorited the prompt, so the frontend can render the correct toggle state
// without inferring it from the denormalized counters.
func (s *server) handlePromptInteractionStatus(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	status, err := s.getPromptService().GetInteractionStatus(id, userID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.InteractionStatus]{
		Code:    200,
		Message: "Success",
		Data:    status,
	})
}

func (s *server) handlePromptView(w http.ResponseWriter, r *http.Request, id int) {
	// The view endpoint is authenticated-optional: a signed-in user is resolved
	// from an optional Bearer token (so the view is written to their history),
	// while anonymous callers are treated as userID 0, which only bumps the
	// total views counter. See store.RecordView for the contract.
	userID, _ := s.optionalUserID(r)

	prompt, applied, err := s.getPromptService().RecordView(id, userID)
	if err != nil {
		writeStoreError(w, err)
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

func (s *server) handlePromptReport(w http.ResponseWriter, r *http.Request, id int) {
	var payload reportCommentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid request body"})
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "report", rateLimitRule{bucket: rateLimitIP(r), limit: 20, window: time.Hour}, rateLimitRule{bucket: rateLimitUser(userID), limit: 10, window: time.Hour}) {
		return
	}

	report, applied, err := s.getPromptService().Report(r.Context(), id, userID, payload.Reason, payload.Detail)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[reportActionResponse]{
		Code:    200,
		Message: "Success",
		Data: reportActionResponse{
			Report:  report,
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

	categoryOK, categoryErr := s.promptStore.CategoryExists(payload.CategoryID)
	if categoryErr == nil && !categoryOK {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      400,
			Message:   "Category does not exist",
			ErrorCode: "INVALID_CATEGORY",
			Data:      nil,
		})
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	userRecord, found := s.getAuthService().FindByID(userID)
	if !found {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	if err := s.validateUploadOwnership(userID, payload.Cover, payload.Images); err != nil {
		if errors.Is(err, errInvalidUploadOwnership) {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: http.StatusBadRequest, ErrorCode: "INVALID_UPLOAD_OWNERSHIP", Message: "Upload reference is invalid"})
		} else {
			writeStoreError(w, err)
		}
		return
	}

	prompt, err := s.getPromptService().Update(r.Context(), id, userID, store.CreatePromptInput{
		Title:        payload.Title,
		Description:  payload.Description,
		Cover:        payload.Cover,
		Images:       payload.Images,
		Content:      payload.Content,
		SystemPrompt: payload.SystemPrompt,
		Model:        payload.Model,
		Params:       payload.Params,
		CategoryID:   payload.CategoryID,
		Tags:         payload.Tags,
		Status:       promptPayloadStatus(payload),
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
		writeStoreError(w, err)
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

	if err := s.getPromptService().Delete(r.Context(), id, userID); err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[any]{Code: 200, Message: "Success", Data: nil})
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

func validatePromptPayload(payload promptPayload) string {
	if promptPayloadStatus(payload) == 0 {
		return validatePromptDraftPayload(payload)
	}
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

func validatePromptDraftPayload(payload promptPayload) string {
	if uniqueTagCount(payload.Tags) > store.MaxPromptTags {
		return fmt.Sprintf("At most %d tags are allowed", store.MaxPromptTags)
	}

	if payload.CategoryID <= 0 {
		return "Category is required"
	}

	hasContent := strings.TrimSpace(payload.Title) != "" ||
		strings.TrimSpace(payload.Description) != "" ||
		strings.TrimSpace(payload.Cover) != "" ||
		strings.TrimSpace(payload.Content) != "" ||
		strings.TrimSpace(payload.SystemPrompt) != "" ||
		strings.TrimSpace(payload.Model) != "" ||
		len(payload.Tags) > 0
	if !hasContent {
		return "Draft needs at least one field"
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

func promptPayloadStatus(payload promptPayload) int {
	if payload.Status != nil && *payload.Status == 0 {
		return 0
	}

	return 1
}
