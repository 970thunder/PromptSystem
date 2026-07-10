// Package api tests the HTTP response envelope contract.
package api

import (
	"encoding/json"
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
