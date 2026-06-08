package store

import "strings"

type MemoryCommentStore struct{}

func NewMemoryCommentStore() *MemoryCommentStore {
	return &MemoryCommentStore{}
}

func (s *MemoryCommentStore) ListByTarget(targetType string, targetID int) ([]Comment, error) {
	if strings.TrimSpace(strings.ToLower(targetType)) != "prompt" {
		return []Comment{}, nil
	}

	return ListPromptComments(targetID)
}

func (s *MemoryCommentStore) Create(input CreateCommentInput) (Comment, error) {
	return CreateComment(input)
}

func (s *MemoryCommentStore) Like(id int, userID int) (Comment, bool, error) {
	return LikeComment(id, userID)
}
