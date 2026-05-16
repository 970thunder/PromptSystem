package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var promptMu sync.RWMutex

type CreatePromptInput struct {
	Title        string
	Description  string
	Cover        string
	Content      string
	SystemPrompt string
	Model        string
	Params       PromptParams
	CategoryID   int
	Tags         []string
	User         User
}

func FilterPrompts(categoryID int, sortBy string) []Prompt {
	promptMu.RLock()
	defer promptMu.RUnlock()

	list := make([]Prompt, 0, len(prompts))
	for _, prompt := range prompts {
		if categoryID == 0 || prompt.CategoryID == categoryID {
			list = append(list, prompt)
		}
	}

	switch sortBy {
	case "popular":
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Likes > list[j].Likes
		})
	default:
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].CreatedAt > list[j].CreatedAt
		})
	}

	return list
}

func FindPromptByID(id int) (Prompt, bool) {
	promptMu.RLock()
	defer promptMu.RUnlock()

	for _, prompt := range prompts {
		if prompt.ID == id {
			return prompt, true
		}
	}

	return Prompt{}, false
}

func CreatePrompt(input CreatePromptInput) (Prompt, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	category, ok := findCategoryByID(input.CategoryID)
	if !ok {
		return Prompt{}, fmt.Errorf("invalid category")
	}

	now := time.Now().UTC().Format(time.DateOnly)
	prompt := Prompt{
		ID:           nextPromptIDLocked(),
		Title:        strings.TrimSpace(input.Title),
		Description:  strings.TrimSpace(input.Description),
		Cover:        strings.TrimSpace(input.Cover),
		Content:      strings.TrimSpace(input.Content),
		SystemPrompt: strings.TrimSpace(input.SystemPrompt),
		Model:        strings.TrimSpace(input.Model),
		Params:       input.Params,
		CategoryID:   input.CategoryID,
		CategoryName: category.Name,
		Tags:         normalizeTags(input.Tags),
		UserID:       input.User.ID,
		User:         input.User,
		Views:        0,
		Likes:        0,
		Favorites:    0,
		Status:       1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	prompts = append([]Prompt{prompt}, prompts...)
	incrementCategoryCountLocked(input.CategoryID)

	return prompt, nil
}

func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}

func findCategoryByID(categoryID int) (Category, bool) {
	for _, category := range categories {
		if category.ID == categoryID {
			return category, true
		}
	}

	return Category{}, false
}

func incrementCategoryCountLocked(categoryID int) {
	for index := range categories {
		if categories[index].ID == categoryID {
			categories[index].Count++
			return
		}
	}
}

func nextPromptIDLocked() int {
	maxID := 100
	for _, prompt := range prompts {
		if prompt.ID > maxID {
			maxID = prompt.ID
		}
	}

	return maxID + 1
}
