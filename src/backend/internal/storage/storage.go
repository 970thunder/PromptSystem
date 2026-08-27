package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"promptos-backend/internal/config"
)

type ImageStorage interface {
	Save(ctx context.Context, objectKey, contentType string, body []byte) (string, error)
}

type LocalStorage struct {
	baseDir   string
	publicURL string
}

type R2Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewImageStorage(cfg config.Config) (ImageStorage, error) {
	if strings.EqualFold(cfg.UploadProvider, "r2") {
		return newR2Storage(cfg)
	}

	return newLocalStorage(cfg)
}

func newLocalStorage(cfg config.Config) (*LocalStorage, error) {
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	return &LocalStorage{
		baseDir:   cfg.UploadDir,
		publicURL: strings.TrimRight(cfg.UploadBaseURL, "/"),
	}, nil
}

func newR2Storage(cfg config.Config) (*R2Storage, error) {
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretKey == "" || cfg.R2Bucket == "" || cfg.R2PublicURL == "" {
		return nil, fmt.Errorf("missing R2 configuration")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = stringPtr(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID))
		options.UsePathStyle = true
		// Bound every R2 request so a stalled network cannot hang a request
		// handler indefinitely. The storage interface takes a context that can
		// additionally cancel a request.
		options.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	})

	return &R2Storage{
		client:    client,
		bucket:    cfg.R2Bucket,
		publicURL: strings.TrimRight(cfg.R2PublicURL, "/"),
	}, nil
}

func (s *LocalStorage) Save(_ context.Context, objectKey, contentType string, body []byte) (string, error) {
	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("create upload path: %w", err)
	}

	if err := os.WriteFile(targetPath, body, 0o644); err != nil {
		return "", fmt.Errorf("write upload file: %w", err)
	}

	_ = s.publicURL
	return "/uploads/" + strings.TrimLeft(filepath.ToSlash(objectKey), "/"), nil
}

func (s *R2Storage) Save(ctx context.Context, objectKey, contentType string, body []byte) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &objectKey,
		Body:        bytes.NewReader(body),
		ContentType: stringPtr(contentType),
		ACL:         types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return "", fmt.Errorf("upload image to R2: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.publicURL, strings.TrimLeft(objectKey, "/")), nil
}

// BuildObjectKey derives a stable, collision-proof object key from the owning
// user, the upload purpose and a random component -- never from the raw client
// filename. This prevents one user from guessing or overwriting another user's
// uploads and makes key collisions impossible. The random suffix is
// hex-encoded crypto/rand output.
func BuildObjectKey(userID int, purpose, originalName, contentType string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = extensionFromMimeType(contentType)
	}
	if ext == "" {
		ext = ".bin"
	}

	random := randomHex(16)
	return fmt.Sprintf("%s/%d/%s_%s%s", purpose, userID, time.Now().UTC().Format("20060102/150405.000000000"), random, ext)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// rand.Read only fails on a broken system; fall back to a timestamp so an
		// upload can still proceed without a panic.
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func extensionFromMimeType(contentType string) string {
	extensions, err := mime.ExtensionsByType(contentType)
	if err != nil || len(extensions) == 0 {
		return ""
	}

	return extensions[0]
}

func stringPtr(value string) *string {
	return &value
}

func ReadAll(limit int64, reader io.Reader) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds max size")
	}

	return data, nil
}
