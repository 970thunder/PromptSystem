package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var commentMu sync.RWMutex
var commentLikes = make(map[int]map[int]struct{})
var commentReports = make(map[string]Report)

type CommentFilter struct {
	TargetType string
	TargetID   int
	SortBy     string
}

type CreateCommentInput struct {
	TargetType string
	TargetID   int
	User       User
	Content    string
	ParentID   *int
}

type ReportCommentInput struct {
	CommentID int
	UserID    int
	Reason    string
	Detail    string
}

func normalizeCommentSort(sortBy string) string {
	switch strings.TrimSpace(strings.ToLower(sortBy)) {
	case "popular", "oldest":
		return strings.TrimSpace(strings.ToLower(sortBy))
	default:
		return "latest"
	}
}

func validateCommentInput(input CreateCommentInput) error {
	targetType := strings.TrimSpace(strings.ToLower(input.TargetType))
	if targetType != "prompt" {
		return ErrInvalidCommentTarget
	}
	if input.TargetID <= 0 {
		return ErrInvalidCommentTarget
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return ErrInvalidCommentContent
	}
	if len([]rune(content)) > 1000 {
		return ErrInvalidCommentContent
	}
	if err := ValidateCommentModeration(content); err != nil {
		return err
	}
	if input.User.ID <= 0 {
		return ErrInvalidCommentUser
	}

	return nil
}

func validateReportCommentInput(input ReportCommentInput) error {
	if input.CommentID <= 0 {
		return ErrInvalidCommentID
	}
	if input.UserID <= 0 {
		return ErrInvalidUser
	}

	reason := strings.TrimSpace(input.Reason)
	if !ValidReportReason(reason) {
		return ErrInvalidReportReason
	}
	if len([]rune(strings.TrimSpace(input.Detail))) > MaxReportDetailRunes {
		return ErrReportDetailTooLong
	}

	return nil
}

func buildCommentTree(source []Comment, sortBy string) []Comment {
	if len(source) == 0 {
		return []Comment{}
	}

	cloned := make([]Comment, len(source))
	index := make(map[int]int, len(source))
	for i, item := range source {
		cloned[i] = item
		cloned[i].Replies = []Comment{}
		index[item.ID] = i
	}

	roots := make([]Comment, 0, len(cloned))
	for _, item := range cloned {
		if item.ParentID != nil {
			if parentIndex, ok := index[*item.ParentID]; ok {
				parent := &cloned[parentIndex]
				parent.Replies = append(parent.Replies, item)
				continue
			}
		}
		roots = append(roots, item)
	}

	sortComments(roots, sortBy)

	for i := range roots {
		sortRepliesByTime(&roots[i])
	}

	return roots
}

func sortComments(comments []Comment, sortBy string) {
	switch normalizeCommentSort(sortBy) {
	case "popular":
		sort.SliceStable(comments, func(i, j int) bool {
			if comments[i].Likes == comments[j].Likes {
				return comments[i].CreatedAt > comments[j].CreatedAt
			}
			return comments[i].Likes > comments[j].Likes
		})
	case "oldest":
		sort.SliceStable(comments, func(i, j int) bool {
			return comments[i].CreatedAt < comments[j].CreatedAt
		})
	default:
		sort.SliceStable(comments, func(i, j int) bool {
			return comments[i].CreatedAt > comments[j].CreatedAt
		})
	}
}

func sortRepliesByTime(comment *Comment) {
	sort.SliceStable(comment.Replies, func(i, j int) bool {
		return comment.Replies[i].CreatedAt < comment.Replies[j].CreatedAt
	})

	for i := range comment.Replies {
		sortRepliesByTime(&comment.Replies[i])
	}
}

func ListPromptComments(filter CommentFilter) ([]Comment, error) {
	commentMu.RLock()
	defer commentMu.RUnlock()

	targetType := strings.TrimSpace(strings.ToLower(filter.TargetType))
	if targetType != "prompt" {
		return []Comment{}, nil
	}

	filtered := make([]Comment, 0)
	for _, comment := range comments {
		if comment.TargetType == targetType && comment.TargetID == filter.TargetID {
			filtered = append(filtered, comment)
		}
	}

	return buildCommentTree(filtered, filter.SortBy), nil
}

func CreateComment(input CreateCommentInput) (Comment, error) {
	if err := validateCommentInput(input); err != nil {
		return Comment{}, err
	}

	if _, ok := FindPromptByID(input.TargetID); !ok {
		return Comment{}, ErrPromptNotFound
	}

	commentMu.Lock()
	defer commentMu.Unlock()

	var parentID *int
	if input.ParentID != nil {
		parent, found := findCommentByIDLocked(*input.ParentID)
		if !found {
			return Comment{}, ErrCommentParentNotFound
		}
		if parent.TargetType != "prompt" || parent.TargetID != input.TargetID {
			return Comment{}, ErrCommentParentMismatch
		}
		value := parent.ID
		parentID = &value
	}

	comment := Comment{
		ID:         nextCommentIDLocked(),
		TargetType: "prompt",
		TargetID:   input.TargetID,
		UserID:     input.User.ID,
		User:       input.User,
		Content:    strings.TrimSpace(input.Content),
		Likes:      0,
		ParentID:   parentID,
		Replies:    []Comment{},
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	comments = append(comments, comment)
	return comment, nil
}

func LikeComment(id int, userID int) (Comment, bool, error) {
	commentMu.Lock()
	defer commentMu.Unlock()

	for index := range comments {
		if comments[index].ID != id {
			continue
		}

		if _, ok := commentLikes[id]; !ok {
			commentLikes[id] = make(map[int]struct{})
		}
		if _, exists := commentLikes[id][userID]; exists {
			return comments[index], false, nil
		}

		commentLikes[id][userID] = struct{}{}
		comments[index].Likes++
		return comments[index], true, nil
	}

	return Comment{}, false, ErrCommentNotFound
}

func ReportComment(input ReportCommentInput) (Report, bool, error) {
	if err := validateReportCommentInput(input); err != nil {
		return Report{}, false, err
	}

	commentMu.Lock()
	defer commentMu.Unlock()

	if _, found := findCommentByIDLocked(input.CommentID); !found {
		return Report{}, false, ErrCommentNotFound
	}

	key := fmt.Sprintf("%d:%d", input.UserID, input.CommentID)
	if report, exists := commentReports[key]; exists {
		return report, false, nil
	}

	report := Report{
		ID:         nextReportIDLocked(),
		UserID:     input.UserID,
		TargetType: "comment",
		TargetID:   input.CommentID,
		Reason:     strings.TrimSpace(input.Reason),
		Detail:     strings.TrimSpace(input.Detail),
		Status:     "pending",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	commentReports[key] = report

	return report, true, nil
}

func findCommentByIDLocked(id int) (Comment, bool) {
	for _, comment := range comments {
		if comment.ID == id {
			return comment, true
		}
	}

	return Comment{}, false
}

func nextCommentIDLocked() int {
	maxID := 0
	for _, comment := range comments {
		if comment.ID > maxID {
			maxID = comment.ID
		}
	}

	return maxID + 1
}

func nextReportIDLocked() int {
	maxID := 0
	for _, report := range commentReports {
		if report.ID > maxID {
			maxID = report.ID
		}
	}

	return maxID + 1
}
