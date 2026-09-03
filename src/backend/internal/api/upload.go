package api

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path"
	"strings"

	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

var errInvalidUploadOwnership = errors.New("invalid upload ownership")

type uploadImageResponse struct {
	URL       string `json:"url"`
	ObjectKey string `json:"objectKey"`
}

// MaxImagePixels caps the total pixel count of an uploaded image. It guards
// against decompression bombs and runaway memory use during decode.
const MaxImagePixels = 20_000_000 // 20 megapixels

// formOverheadBytes is the slack we allow on top of the file limit for
// multipart boundaries/headers so the whole request is bounded but a valid
// single-file upload is never rejected for form framing.
const formOverheadBytes = 4 << 20

// imageFormat is one of the supported, decodable image formats. The key is the
// canonical format name; mime/ext are what a consistent upload must declare.
type imageFormat struct {
	mime string
	ext  string
}

var supportedImageFormats = map[string]imageFormat{
	"jpeg": {"image/jpeg", ".jpg"},
	"png":  {"image/png", ".png"},
	"webp": {"image/webp", ".webp"},
	"gif":  {"image/gif", ".gif"},
}

// enabledImageFormats returns the formats allowed for this deployment. GIF is
// disabled by default because animated GIFs are hard to validate for safe
// decoding; it is opt-in via UPLOAD_ALLOW_GIF.
func (s *server) enabledImageFormats() map[string]imageFormat {
	formats := make(map[string]imageFormat, len(supportedImageFormats))
	for name, f := range supportedImageFormats {
		if name == "gif" && !s.config.AllowGif {
			continue
		}
		formats[name] = f
	}
	return formats
}

func (s *server) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}

	maxBytes := int64(s.config.UploadMaxMB) * 1024 * 1024

	// Bound the entire multipart request BEFORE parsing, so a malicious client
	// cannot exhaust memory with form fields or a falsely sized body. The slack
	// covers multipart framing; the file itself is still capped to maxBytes.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+formOverheadBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, apiResponse[any]{
			Code:      http.StatusRequestEntityTooLarge,
			ErrorCode: "REQUEST_TOO_LARGE",
			Message:   fmt.Sprintf("Upload exceeds the %d MB limit", s.config.UploadMaxMB),
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      http.StatusBadRequest,
			ErrorCode: "IMAGE_REQUIRED",
			Message:   "Image file is required",
		})
		return
	}
	defer file.Close()

	data, err := storage.ReadAll(maxBytes, file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      http.StatusBadRequest,
			ErrorCode: "IMAGE_TOO_LARGE",
			Message:   fmt.Sprintf("Image exceeds %d MB limit", s.config.UploadMaxMB),
		})
		return
	}

	// Detect the real format from the bytes, independent of the client's claim.
	format := detectImageFormat(data)
	formats := s.enabledImageFormats()
	expected, ok := formats[format]
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      http.StatusBadRequest,
			ErrorCode: "INVALID_IMAGE_FORMAT",
			Message:   "Only jpg, png and webp images are supported",
		})
		return
	}

	// Reject spoofed MIME/extension: the declared content type and file
	// extension must agree with what the bytes actually decode to.
	if declared := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type"))); declared != "" && declared != expected.mime {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      http.StatusBadRequest,
			ErrorCode: "INVALID_IMAGE_FORMAT",
			Message:   "Image content type does not match the actual file format",
		})
		return
	}
	if ext := strings.ToLower(path.Ext(header.Filename)); ext != "" && ext != expected.ext {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{
			Code:      http.StatusBadRequest,
			ErrorCode: "INVALID_IMAGE_FORMAT",
			Message:   "Image extension does not match the actual file format",
		})
		return
	}

	// Verify the bytes are genuinely decodable and within the pixel cap. webp
	// cannot be decoded by the standard library, so we accept its RIFF/WEBP
	// magic (validated in detectImageFormat) and rely on the byte-size limit.
	if format != "webp" {
		cfg, _, decErr := image.DecodeConfig(strings.NewReader(string(data)))
		if decErr != nil {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{
				Code:      http.StatusBadRequest,
				ErrorCode: "INVALID_IMAGE_FORMAT",
				Message:   "Image file is corrupt or not decodable",
			})
			return
		}
		if cfg.Width <= 0 || cfg.Height <= 0 {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{
				Code:      http.StatusBadRequest,
				ErrorCode: "INVALID_IMAGE_FORMAT",
				Message:   "Image has invalid dimensions",
			})
			return
		}
		if int64(cfg.Width)*int64(cfg.Height) > MaxImagePixels {
			writeJSON(w, http.StatusRequestEntityTooLarge, apiResponse[any]{
				Code:      http.StatusRequestEntityTooLarge,
				ErrorCode: "IMAGE_TOO_LARGE",
				Message:   "Image dimensions exceed the maximum allowed resolution",
			})
			return
		}
	}

	objectKey := storage.BuildObjectKey(userID, string(store.UploadPurposePromptImage), header.Filename, expected.mime)
	url, err := s.imageStorage.Save(r.Context(), objectKey, expected.mime, data)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
			Code:      http.StatusInternalServerError,
			ErrorCode: "IMAGE_STORE_FAILED",
			Message:   "Failed to store image",
		})
		return
	}

	// Record the upload metadata so ownership can be proven later and the object
	// can be garbage-collected if never referenced.
	if s.uploadStore != nil {
		if _, err := s.uploadStore.RecordUpload(store.UploadRecord{
			OwnerID:     userID,
			Provider:    s.config.UploadProvider,
			Purpose:     store.UploadPurposePromptImage,
			ObjectKey:   objectKey,
			ContentType: expected.mime,
			Size:        int64(len(data)),
			Status:      store.UploadStatusPending,
		}); err != nil {
			// A metadata write failure must not leak the object without a record;
			// surface it rather than returning a dangling URL.
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{
				Code:      http.StatusInternalServerError,
				ErrorCode: "IMAGE_STORE_FAILED",
				Message:   "Failed to record upload",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, apiResponse[uploadImageResponse]{
		Code:    200,
		Message: "Success",
		Data: uploadImageResponse{
			URL:       url,
			ObjectKey: objectKey,
		},
	})
}

// detectImageFormat inspects the leading bytes to determine the real image
// format, ignoring any client-declared type. It returns "" for unsupported.
func detectImageFormat(data []byte) string {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "jpeg"
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return "png"
	}
	if len(data) >= 4 && data[0] == 'G' && data[1] == 'I' && data[2] == 'F' && data[3] == '8' {
		return "gif"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "webp"
	}
	return ""
}

// validateUploadOwnership checks that any local /uploads/ references in a
// prompt's cover/images belong to the requesting user, and marks them
// referenced so cleanup cannot remove objects that are now in use. It returns a
// stable error when a reference is missing, owned by another user, or trashed.
func (s *server) validateUploadOwnership(userID int, cover string, images []string) error {
	if s.uploadStore == nil {
		return nil
	}
	candidates := make([]string, 0, len(images)+1)
	if cover != "" {
		candidates = append(candidates, cover)
	}
	candidates = append(candidates, images...)

	var keys []string
	for _, raw := range candidates {
		url := strings.TrimSpace(raw)
		if !strings.HasPrefix(url, "/uploads/") {
			continue
		}
		objectKey := strings.TrimPrefix(url, "/uploads/")
		objectKey = strings.Trim(objectKey, "/")
		if objectKey == "" {
			continue
		}
		rec, found, err := s.uploadStore.FindUpload(objectKey)
		if err != nil {
			return err
		}
		if !found {
			return errInvalidUploadOwnership
		}
		if rec.OwnerID != userID {
			return errInvalidUploadOwnership
		}
		if rec.Status == store.UploadStatusTrashed {
			return errInvalidUploadOwnership
		}
		keys = append(keys, objectKey)
	}

	if len(keys) > 0 {
		if err := s.uploadStore.MarkUploadsReferenced(keys, userID); err != nil {
			return err
		}
	}
	return nil
}
