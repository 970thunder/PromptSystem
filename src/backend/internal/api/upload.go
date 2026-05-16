package api

import (
	"fmt"
	"net/http"
	"strings"

	"promptos-backend/internal/storage"
)

type uploadImageResponse struct {
	URL string `json:"url"`
}

var allowedImageMimeTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

func (s *server) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	maxBytes := int64(s.config.UploadMaxMB) * 1024 * 1024
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: fmt.Sprintf("Invalid upload payload. Max file size is %d MB", s.config.UploadMaxMB),
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: "Image file is required",
		})
		return
	}
	defer file.Close()

	data, err := storage.ReadAll(maxBytes, file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: fmt.Sprintf("Image exceeds %d MB limit", s.config.UploadMaxMB),
		})
		return
	}

	contentType := strings.ToLower(http.DetectContentType(data))
	if _, ok := allowedImageMimeTypes[contentType]; !ok {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:    400,
			Message: "Only jpg, png, webp, and gif images are supported",
		})
		return
	}

	objectKey := storage.BuildObjectKey(header.Filename, contentType)
	url, err := s.imageStorage.Save(r.Context(), objectKey, contentType, data)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:    500,
			Message: "Failed to store image",
		})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[uploadImageResponse]{
		Code:    200,
		Message: "Success",
		Data: uploadImageResponse{
			URL: url,
		},
	})
}
