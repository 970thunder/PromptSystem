package api

import (
	"net/http"
	"strconv"
	"strings"

	"promptos-backend/internal/store"
)

type reviewReportRequest struct {
	Status string `json:"status"`
	Action string `json:"action"`
	Note   string `json:"note"`
}

type moderationStatusRequest struct {
	Status int    `json:"status"`
	Reason string `json:"reason"`
}

// withAdmin layers role authorization over the normal session checks. The
// role is read from the database on every request, so disabling or removing a
// role takes effect without waiting for JWT expiry.
func (s *server) withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.withAuth(func(w http.ResponseWriter, r *http.Request) {
		if s.moderationStore == nil {
			writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{
				Code:      http.StatusServiceUnavailable,
				ErrorCode: "MODERATION_UNAVAILABLE",
				Message:   "Moderation is unavailable",
			})
			return
		}
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: http.StatusUnauthorized, ErrorCode: "AUTH_TOKEN_MISSING", Message: "Unauthorized"})
			return
		}
		admin, err := s.moderationStore.IsAdmin(userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, ErrorCode: "INTERNAL_ERROR", Message: "Internal server error"})
			return
		}
		if !admin {
			writeJSON(w, http.StatusForbidden, apiResponse[any]{Code: http.StatusForbidden, ErrorCode: "ADMIN_REQUIRED", Message: "Administrator role required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/"), "/")
	parts := strings.Split(path, "/")
	if path == "reports" && r.Method == http.MethodGet {
		s.handleAdminReports(w, r)
		return
	}
	if path == "audit" && r.Method == http.MethodGet {
		s.handleAdminAudit(w, r)
		return
	}
	if len(parts) == 2 {
		id, err := strconv.Atoi(parts[1])
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_MODERATION_ID", Message: "Invalid moderation id"})
			return
		}
		switch {
		case parts[0] == "reports" && r.Method == http.MethodPatch:
			s.handleAdminReportReview(w, r, id)
		case parts[0] == "prompts" && r.Method == http.MethodPatch:
			s.handleAdminPromptStatus(w, r, id)
		case parts[0] == "users" && r.Method == http.MethodPatch:
			s.handleAdminUserStatus(w, r, id)
		default:
			writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, ErrorCode: "NOT_FOUND", Message: "Not found"})
		}
		return
	}
	writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, ErrorCode: "NOT_FOUND", Message: "Not found"})
}

func (s *server) handleAdminReports(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePageParams(stringPageParams{Page: r.URL.Query().Get("page"), PageSize: r.URL.Query().Get("pageSize")})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	reports, total, err := s.moderationStore.ListReports(r.URL.Query().Get("status"), page, pageSize)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.Report]]{
		Code: 200, Message: "Success",
		Data: pageResponse[store.Report]{List: reports, Total: total, Page: page, PageSize: pageSize},
	})
}

func (s *server) handleAdminReportReview(w http.ResponseWriter, r *http.Request, reportID int) {
	var payload reviewReportRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	actorID, _ := userIDFromContext(r.Context())
	report, err := s.moderationStore.ReviewReport(store.ReviewReportInput{
		ReportID:  reportID,
		ActorID:   actorID,
		Status:    payload.Status,
		Action:    payload.Action,
		Note:      payload.Note,
		RequestID: requestIDFrom(r),
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[store.Report]{Code: 200, Message: "Success", Data: report})
}

func (s *server) handleAdminPromptStatus(w http.ResponseWriter, r *http.Request, promptID int) {
	var payload moderationStatusRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	actorID, _ := userIDFromContext(r.Context())
	if err := s.moderationStore.SetPromptStatus(promptID, actorID, payload.Status, payload.Reason); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[map[string]bool]{Code: 200, Message: "Success", Data: map[string]bool{"updated": true}})
}

func (s *server) handleAdminUserStatus(w http.ResponseWriter, r *http.Request, userID int) {
	var payload moderationStatusRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	actorID, _ := userIDFromContext(r.Context())
	if err := s.moderationStore.SetUserStatus(userID, actorID, payload.Status, payload.Reason); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[map[string]bool]{Code: 200, Message: "Success", Data: map[string]bool{"updated": true}})
}

func (s *server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePageParams(stringPageParams{Page: r.URL.Query().Get("page"), PageSize: r.URL.Query().Get("pageSize")})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	events, total, err := s.moderationStore.ListAuditEvents(page, pageSize)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[pageResponse[store.AuditEvent]]{
		Code: 200, Message: "Success",
		Data: pageResponse[store.AuditEvent]{List: events, Total: total, Page: page, PageSize: pageSize},
	})
}
