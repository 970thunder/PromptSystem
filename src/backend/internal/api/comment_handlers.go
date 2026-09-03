package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"promptos-backend/internal/store"
)

type commentPayload struct {
	Content  string `json:"content"`
	ParentID *int   `json:"parentId"`
}

type commentActionResponse struct {
	Comment store.Comment `json:"comment"`
	Applied bool          `json:"applied"`
}

func (s *server) handlePromptComments(w http.ResponseWriter, r *http.Request, id int) {
	query := r.URL.Query()
	page, pageSize, err := parsePageParams(stringPageParams{Page: query.Get("page"), PageSize: query.Get("pageSize")})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	comments, total, err := s.commentStore.ListByTargetPage(store.CommentFilter{
		TargetType: "prompt",
		TargetID:   id,
		SortBy:     r.URL.Query().Get("sort"),
	}, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:    500,
			Message: "Failed to load comments",
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Comment]]{
		Code:    200,
		Message: "Success",
		Data: pageResponse[store.Comment]{
			List:     comments,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
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
	if !s.enforceRateLimits(r.Context(), w, "comment", rateLimitRule{bucket: rateLimitIP(r), limit: 60, window: time.Minute}, rateLimitRule{bucket: rateLimitUser(userID), limit: 120, window: time.Minute}) {
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
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[store.Comment]{
		Code:    200,
		Message: "Success",
		Data:    comment,
	})
}

func (s *server) handleCommentAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/comments/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Invalid comment id"})
		return
	}

	switch parts[1] {
	case "like":
		s.handleCommentLike(w, r, id)
	case "report":
		s.handleCommentReport(w, r, id)
	default:
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
	}
}

func (s *server) handleCommentLike(w http.ResponseWriter, r *http.Request, id int) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "comment_interaction", rateLimitRule{bucket: rateLimitIP(r), limit: 120, window: time.Minute}, rateLimitRule{bucket: rateLimitUser(userID), limit: 240, window: time.Minute}) {
		return
	}

	comment, applied, err := s.commentStore.Like(id, userID)
	if err != nil {
		writeStoreError(w, err)
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

func (s *server) handleCommentReport(w http.ResponseWriter, r *http.Request, id int) {
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
	if !s.enforceRateLimits(r.Context(), w, "comment_report", rateLimitRule{bucket: rateLimitIP(r), limit: 20, window: time.Hour}, rateLimitRule{bucket: rateLimitUser(userID), limit: 10, window: time.Hour}) {
		return
	}

	report, applied, err := s.commentStore.Report(store.ReportCommentInput{
		CommentID: id,
		UserID:    userID,
		Reason:    payload.Reason,
		Detail:    payload.Detail,
	})
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
