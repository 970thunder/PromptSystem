package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var commentMu sync.RWMutex
var commentLikes = make(map[int]map[int]struct{})

type CreateCommentInput struct {
	TargetType string
	TargetID   int
	User       User
	Content    string
	ParentID   *int
}

func validateCommentInput(input CreateCommentInput) error {
	targetType := strings.TrimSpace(strings.ToLower(input.TargetType))
	if targetType != "prompt" {
		return errors.New("invalid comment target")
	}
	if input.TargetID <= 0 {
		return errors.New("invalid comment target")
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return errors.New("comment content is required")
	}
	if len([]rune(content)) > 1000 {
		return errors.New("comment content must be 1000 characters or fewer")
	}
	if input.User.ID <= 0 {
		return errors.New("invalid user")
	}

	return nil
}

func buildCommentTree(source []Comment) []Comment {
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

	sort.SliceStable(roots, func(i, j int) bool {
		return roots[i].CreatedAt > roots[j].CreatedAt
	})

	for i := range roots {
		sortRepliesByTime(&roots[i])
	}

	return roots
}

func sortRepliesByTime(comment *Comment) {
	sort.SliceStable(comment.Replies, func(i, j int) bool {
		return comment.Replies[i].CreatedAt < comment.Replies[j].CreatedAt
	})

	for i := range comment.Replies {
		sortRepliesByTime(&comment.Replies[i])
	}
}

func ListPromptComments(targetID int) ([]Comment, error) {
	commentMu.RLock()
	defer commentMu.RUnlock()

	filtered := make([]Comment, 0)
	for _, comment := range comments {
		if comment.TargetType == "prompt" && comment.TargetID == targetID {
			filtered = append(filtered, comment)
		}
	}

	return buildCommentTree(filtered), nil
}

func CreateComment(input CreateCommentInput) (Comment, error) {
	if err := validateCommentInput(input); err != nil {
		return Comment{}, err
	}

	if _, ok := FindPromptByID(input.TargetID); !ok {
		return Comment{}, errors.New("prompt not found")
	}

	commentMu.Lock()
	defer commentMu.Unlock()

	var parentID *int
	if input.ParentID != nil {
		parent, found := findCommentByIDLocked(*input.ParentID)
		if !found {
			return Comment{}, errors.New("parent comment not found")
		}
		if parent.TargetType != "prompt" || parent.TargetID != input.TargetID {
			return Comment{}, errors.New("parent comment does not match prompt")
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

	return Comment{}, false, fmt.Errorf("comment not found")
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
