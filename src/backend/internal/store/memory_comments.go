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

func (s *MemoryCommentStore) ListByTargetPage(filter CommentFilter, page, pageSize int) ([]Comment, int, error) {
	if strings.TrimSpace(strings.ToLower(filter.TargetType)) != "prompt" {
		return []Comment{}, 0, nil
	}
	all, err := ListPromptComments(filter)
	if err != nil {
		return nil, 0, err
	}
	total := len(all)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
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
