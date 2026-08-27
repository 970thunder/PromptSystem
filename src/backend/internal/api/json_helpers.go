package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// maxJSONBodyBytes caps the size of decoded JSON request bodies.
const maxJSONBodyBytes = 1 << 20 // 1 MiB

// decodeJSON reads a single JSON value from the request body with a size
// limit, rejects unknown fields, and disallows trailing data. It returns an
// apiError ready for writeAPIError.
func decodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return &apiError{Status: http.StatusRequestEntityTooLarge, Code: "BODY_TOO_LARGE", Message: "Request body too large"}
		}
		return &apiError{Status: http.StatusBadRequest, Code: "INVALID_JSON", Message: "Invalid request body"}
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return &apiError{Status: http.StatusBadRequest, Code: "INVALID_JSON", Message: "Request body must contain a single JSON value"}
	}

	return nil
}

// parsePageParams parses and validates page/pageSize. Invalid values produce
// a 400 apiError instead of silently falling back.
func parsePageParams(query stringPageParams) (page, pageSize int, err error) {
	page, err = parseBoundedInt(query.Page, 1, 1, 1<<30)
	if err != nil {
		return 0, 0, &apiError{Status: http.StatusBadRequest, Code: "INVALID_PAGE", Message: "page must be >= 1"}
	}
	pageSize, err = parseBoundedInt(query.PageSize, 12, 1, 100)
	if err != nil {
		return 0, 0, &apiError{Status: http.StatusBadRequest, Code: "INVALID_PAGE_SIZE", Message: "pageSize must be between 1 and 100"}
	}
	return page, pageSize, nil
}

type stringPageParams struct {
	Page     string
	PageSize string
}

func parseBoundedInt(raw string, fallback, min, max int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("out of range")
	}
	return value, nil
}

// apiError carries a stable machine-readable code alongside the HTTP status.
type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string {
	return e.Message
}

func writeAPIError(w http.ResponseWriter, err error) {
	if apiErr, ok := err.(*apiError); ok {
		writeJSON(w, apiErr.Status, apiResponse[any]{
			Code:      apiErr.Status,
			Message:   apiErr.Message,
			ErrorCode: apiErr.Code,
			Data:      nil,
		})
		return
	}

	writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
		Code:      http.StatusInternalServerError,
		Message:   "Internal server error",
		ErrorCode: "INTERNAL_ERROR",
		Data:      nil,
	})
}
