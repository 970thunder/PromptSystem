package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"promptos-backend/internal/store"
)

// ErrInvalidUploadOwnership is returned when a prompt tries to reference a
// local upload that is missing, trashed, or owned by another account.
var ErrInvalidUploadOwnership = errors.New("invalid upload ownership")

// ErrUploadReferenceFinalize identifies a prompt write whose upload metadata
// could not be finalized even after the compensating reconciliation attempt.
// The prompt remains durable and a later retry can repair the references.
var ErrUploadReferenceFinalize = errors.New("upload reference finalization failed")

// ErrUploadLifecycle identifies a failure while reconciling all of the owner's
// retained prompt references. It is retry-safe and never marks stale objects as
// referenced after the prompt set has changed.
var ErrUploadLifecycle = errors.New("upload lifecycle reconciliation failed")

// PromptService owns prompt business operations that span a store and public
// cache invalidation. HTTP handlers only translate the result into an API
// response; they do not decide when an interaction changes cached aggregates.
type PromptService struct {
	prompts    store.PromptManager
	uploads    store.UploadManager
	invalidate func(context.Context)
}

func NewPromptService(prompts store.PromptManager, uploads store.UploadManager, invalidate func(context.Context)) *PromptService {
	return &PromptService{prompts: prompts, uploads: uploads, invalidate: invalidate}
}

func (s *PromptService) Like(ctx context.Context, id, userID int) (store.Prompt, bool, error) {
	return s.interact(ctx, func() (store.Prompt, bool, error) { return s.prompts.Like(id, userID) })
}

func (s *PromptService) Unlike(ctx context.Context, id, userID int) (store.Prompt, bool, error) {
	return s.interact(ctx, func() (store.Prompt, bool, error) { return s.prompts.Unlike(id, userID) })
}

func (s *PromptService) Favorite(ctx context.Context, id, userID int) (store.Prompt, bool, error) {
	return s.interact(ctx, func() (store.Prompt, bool, error) { return s.prompts.Favorite(id, userID) })
}

func (s *PromptService) Unfavorite(ctx context.Context, id, userID int) (store.Prompt, bool, error) {
	return s.interact(ctx, func() (store.Prompt, bool, error) { return s.prompts.Unfavorite(id, userID) })
}

func (s *PromptService) Report(ctx context.Context, id, userID int, reason, detail string) (store.Report, bool, error) {
	return s.prompts.Report(id, userID, reason, detail)
}

func (s *PromptService) GetInteractionStatus(id, userID int) (store.InteractionStatus, error) {
	return s.prompts.GetInteractionStatus(id, userID)
}

func (s *PromptService) RecordView(id, userID int) (store.Prompt, bool, error) {
	return s.prompts.RecordView(id, userID)
}

func (s *PromptService) Create(ctx context.Context, input store.CreatePromptInput) (store.Prompt, error) {
	prompt, err := s.prompts.Create(input)
	if err == nil && s.invalidate != nil {
		s.invalidate(ctx)
	}
	return prompt, err
}

func (s *PromptService) Update(ctx context.Context, id, userID int, input store.CreatePromptInput) (store.Prompt, error) {
	prompt, err := s.prompts.Update(id, userID, input)
	if err == nil && s.invalidate != nil {
		s.invalidate(ctx)
	}
	return prompt, err
}

func (s *PromptService) Delete(ctx context.Context, id, userID int) error {
	if err := s.prompts.Delete(id, userID); err != nil {
		return err
	}
	if s.invalidate != nil {
		s.invalidate(ctx)
	}
	return nil
}

func (s *PromptService) interact(ctx context.Context, operation func() (store.Prompt, bool, error)) (store.Prompt, bool, error) {
	prompt, applied, err := operation()
	if err == nil && applied && s.invalidate != nil {
		s.invalidate(ctx)
	}
	return prompt, applied, err
}

// ValidateUploadOwnership verifies all local upload references before a
// prompt write. It does not mutate lifecycle state, so a failed prompt write
// cannot accidentally protect a temporary object from garbage collection.
func (s *PromptService) ValidateUploadOwnership(userID int, cover string, images []string) error {
	if s.uploads == nil {
		return nil
	}
	candidates := make([]string, 0, len(images)+1)
	if cover != "" {
		candidates = append(candidates, cover)
	}
	candidates = append(candidates, images...)

	for _, raw := range candidates {
		url := strings.TrimSpace(raw)
		if !strings.HasPrefix(url, "/uploads/") {
			continue
		}
		objectKey := strings.Trim(strings.TrimPrefix(url, "/uploads/"), "/")
		if objectKey == "" {
			continue
		}
		record, found, err := s.uploads.FindUpload(objectKey)
		if err != nil {
			return err
		}
		if !found || record.OwnerID != userID || record.Status == store.UploadStatusTrashed {
			return ErrInvalidUploadOwnership
		}
	}
	return nil
}

func (s *PromptService) MarkUploadsReferenced(userID int, cover string, images []string) error {
	if s.uploads == nil {
		return nil
	}
	keys := uploadKeys(cover, images)
	return s.uploads.MarkUploadsReferenced(keys, userID)
}

func (s *PromptService) ReconcileUploads(userID int) error {
	if s.uploads == nil {
		return nil
	}
	if s.prompts == nil {
		return fmt.Errorf("%w: prompt store unavailable", ErrUploadLifecycle)
	}
	keys, err := s.prompts.ListReferencedUploadKeys(userID)
	if err != nil {
		return err
	}
	// Mark current references first. This is the compensating step for a prompt
	// transaction that committed just before upload metadata became unavailable;
	// a later retry can never garbage-collect an object still used by a prompt.
	if err := s.uploads.MarkUploadsReferenced(keys, userID); err != nil {
		return err
	}
	_, err = s.uploads.UnreferenceUploadsByOwner(userID, keys)
	return err
}

// FinalizeUploadReferences closes the cross-store prompt/upload boundary. The
// prompt write is already atomic in its own store; metadata finalization is
// retried by reconciliation, which marks all current references before moving
// stale records back to pending. A remaining error is explicit and retry-safe.
func (s *PromptService) FinalizeUploadReferences(userID int, cover string, images []string) error {
	markErr := s.MarkUploadsReferenced(userID, cover, images)
	reconcileErr := s.ReconcileUploads(userID)
	if markErr != nil {
		if reconcileErr != nil {
			return fmt.Errorf("%w: mark=%v; reconcile=%v", ErrUploadReferenceFinalize, markErr, reconcileErr)
		}
		// The compensating pass repaired every current reference. The caller can
		// safely observe success even though the first metadata write transiently
		// failed.
		return nil
	}
	if reconcileErr != nil {
		return fmt.Errorf("%w: %v", ErrUploadLifecycle, reconcileErr)
	}
	return nil
}

func uploadKeys(cover string, images []string) []string {
	candidates := make([]string, 0, len(images)+1)
	if cover != "" {
		candidates = append(candidates, cover)
	}
	candidates = append(candidates, images...)
	keys := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, raw := range candidates {
		trimmed := strings.TrimSpace(raw)
		if !strings.HasPrefix(trimmed, "/uploads/") {
			continue
		}
		key := strings.Trim(strings.TrimPrefix(trimmed, "/uploads/"), "/")
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

// CommentService keeps comment interactions and reports out of HTTP handlers.
type CommentService struct {
	comments store.CommentManager
}

func NewCommentService(comments store.CommentManager) *CommentService {
	return &CommentService{comments: comments}
}

func (s *CommentService) Like(id, userID int) (store.Comment, bool, error) {
	return s.comments.Like(id, userID)
}

func (s *CommentService) Create(input store.CreateCommentInput) (store.Comment, error) {
	return s.comments.Create(input)
}

func (s *CommentService) Report(input store.ReportCommentInput) (store.Report, bool, error) {
	return s.comments.Report(input)
}
