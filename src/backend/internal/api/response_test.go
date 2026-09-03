// Package api tests the HTTP response envelope contract.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"promptos-backend/internal/store"
)

func TestAPIResponseIncludesEmptySliceData(t *testing.T) {
	payload, err := json.Marshal(apiResponse[[]store.Comment]{
		Code:    200,
		Message: "Success",
		Data:    []store.Comment{},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	body := string(payload)
	if !strings.Contains(body, `"data":[]`) {
		t.Fatalf("expected empty slice data in response envelope, got %s", body)
	}
}

func TestErrorResponseGetsDefaultErrorCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeJSON(recorder, http.StatusNotFound, apiResponse[any]{Code: http.StatusNotFound, Message: "Not found"})

	var payload apiResponse[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ErrorCode != "NOT_FOUND" {
		t.Fatalf("errorCode = %q, want NOT_FOUND", payload.ErrorCode)
	}
}
