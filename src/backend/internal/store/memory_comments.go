package store

import "strings"

type MemoryCommentStore struct{}

func NewMemoryCommentStore() *MemoryCommentStore {
	return &MemoryCommentStore{}
}

func (s *MemoryCommentStore) ListByTarget(filter CommentFilter) ([]Comment, error) {
	if strings.TrimSpace(strings.ToLower(filter.TargetType)) != "prompt" {
		return []Comment{}, nil
	}

	return ListPromptComments(filter)
}

func (s *MemoryCommentStore) Create(input CreateCommentInput) (Comment, error) {
	return CreateComment(input)
}

func (s *MemoryCommentStore) Like(id int, userID int) (Comment, bool, error) {
	return LikeComment(id, userID)
}

func (s *MemoryCommentStore) Report(input ReportCommentInput) (Report, bool, error) {
	return ReportComment(input)
}
