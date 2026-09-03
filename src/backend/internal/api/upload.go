package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path"
	"strings"
	"time"

	"promptos-backend/internal/storage"
	"promptos-backend/internal/store"
)

var errInvalidUploadOwnership = errors.New("invalid upload ownership")
var errUploadConcurrency = errors.New("upload concurrency limit reached")
var errUploadDailyQuota = errors.New("upload daily quota exceeded")
var errUploadTotalQuota = errors.New("upload total quota exceeded")
var errUploadQuotaUnavailable = errors.New("upload quota unavailable")

const (
	defaultUploadMaxConcurrent = 4
	defaultUploadDailyQuotaMB  = 100
	defaultUploadTotalQuotaMB  = 2048
)

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
	if !s.enforceRateLimits(r.Context(), w, "upload", rateLimitRule{bucket: rateLimitIP(r), limit: 30, window: time.Hour}) {
		return
	}
	uploadSucceeded := false
	defer func() {
		if s.metrics != nil {
			s.metrics.observeUpload(uploadSucceeded)
		}
	}()

	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse[any]{Code: 401, Message: "Unauthorized"})
		return
	}
	if !s.acquireUploadSlot(w) {
		return
	}
	defer s.releaseUploadSlot()

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
		cfg, _, decErr := image.DecodeConfig(bytes.NewReader(data))
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

	releaseQuota, err := s.reserveUploadQuota(r.Context(), userID, int64(len(data)))
	if err != nil {
		status := http.StatusServiceUnavailable
		errorCode := "UPLOAD_QUOTA_UNAVAILABLE"
		message := "Upload quota is temporarily unavailable"
		switch {
		case errors.Is(err, errUploadDailyQuota):
			status = http.StatusTooManyRequests
			errorCode = "UPLOAD_DAILY_QUOTA_EXCEEDED"
			message = "Daily upload quota exceeded"
		case errors.Is(err, errUploadTotalQuota):
			status = http.StatusInsufficientStorage
			errorCode = "UPLOAD_CAPACITY_EXCEEDED"
			message = "Upload storage capacity reached"
		}
		writeJSON(w, status, apiResponse[any]{Code: status, ErrorCode: errorCode, Message: message})
		return
	}
	defer releaseQuota()

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
	uploadSucceeded = true
}

func (s *server) acquireUploadSlot(w http.ResponseWriter) bool {
	s.uploadOnce.Do(func() {
		limit := s.config.UploadMaxConcurrent
		if limit <= 0 {
			limit = defaultUploadMaxConcurrent
		}
		s.uploadSlots = make(chan struct{}, limit)
	})
	select {
	case s.uploadSlots <- struct{}{}:
		return true
	default:
		w.Header().Set("Retry-After", "1")
		writeJSON(w, http.StatusTooManyRequests, apiResponse[any]{
			Code: http.StatusTooManyRequests, ErrorCode: "UPLOAD_CONCURRENCY_LIMITED", Message: "Too many uploads in progress",
		})
		return false
	}
}

func (s *server) releaseUploadSlot() {
	select {
	case <-s.uploadSlots:
	default:
	}
}

type uploadQuotaCounter interface {
	IncrementBy(context.Context, string, int64, time.Duration) (int64, time.Duration, error)
}

func (s *server) reserveUploadQuota(ctx context.Context, userID int, size int64) (func(), error) {
	if size <= 0 {
		return nil, errUploadTotalQuota
	}
	totalLimit := s.config.UploadTotalQuotaMB
	if totalLimit <= 0 {
		totalLimit = defaultUploadTotalQuotaMB
	}
	totalBytes := int64(totalLimit) * 1024 * 1024

	// Keep a process-local reservation across the storage write and metadata
	// insert. This closes the race where concurrent requests all pass the
	// persisted SUM check before either one has recorded its upload.
	s.uploadQuotaMu.Lock()
	persisted, err := s.activeUploadBytes()
	if err != nil {
		s.uploadQuotaMu.Unlock()
		return nil, errUploadQuotaUnavailable
	}
	if persisted+s.uploadReservedBytes+size > totalBytes {
		s.uploadQuotaMu.Unlock()
		return nil, errUploadTotalQuota
	}
	s.uploadReservedBytes += size
	s.uploadQuotaMu.Unlock()

	release := func() {
		s.uploadQuotaMu.Lock()
		s.uploadReservedBytes -= size
		if s.uploadReservedBytes < 0 {
			s.uploadReservedBytes = 0
		}
		s.uploadQuotaMu.Unlock()
	}

	dailyLimit := s.config.UploadDailyQuotaMB
	if dailyLimit <= 0 {
		dailyLimit = defaultUploadDailyQuotaMB
	}
	dailyBytes := int64(dailyLimit) * 1024 * 1024
	dayKey := fmt.Sprintf("%d:%s", userID, time.Now().UTC().Format("2006-01-02"))
	if counter, ok := s.cache.(uploadQuotaCounter); ok {
		used, _, err := counter.IncrementBy(ctx, "promptos:quota:upload:daily:"+dayKey, size, 25*time.Hour)
		if err != nil {
			release()
			return nil, errUploadQuotaUnavailable
		}
		if used > dailyBytes {
			release()
			return nil, errUploadDailyQuota
		}
	} else {
		s.uploadQuotaMu.Lock()
		if s.uploadDailyUsage == nil {
			s.uploadDailyUsage = make(map[string]int64)
		}
		used := s.uploadDailyUsage[dayKey] + size
		if used > dailyBytes {
			s.uploadQuotaMu.Unlock()
			release()
			return nil, errUploadDailyQuota
		}
		s.uploadDailyUsage[dayKey] = used
		s.uploadQuotaMu.Unlock()
	}
	return release, nil
}

func (s *server) activeUploadBytes() (int64, error) {
	if s.uploadStore == nil {
		return 0, nil
	}
	return s.uploadStore.ActiveUploadBytes()
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
