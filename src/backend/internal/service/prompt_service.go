package service

import (
	"context"
	"errors"
	"strings"

	"promptos-backend/internal/store"
)

// ErrInvalidUploadOwnership is returned when a prompt tries to reference a
// local upload that is missing, trashed, or owned by another account.
var ErrInvalidUploadOwnership = errors.New("invalid upload ownership")

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
	keys, err := s.prompts.ListReferencedUploadKeys(userID)
	if err != nil {
		return err
	}
	_, err = s.uploads.UnreferenceUploadsByOwner(userID, keys)
	return err
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
