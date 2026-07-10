package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var promptMu sync.RWMutex
var promptLikes = make(map[int]map[int]struct{})
var promptFavorites = make(map[int]map[int]struct{})
var promptViewHistory = make(map[int]map[int]time.Time)
var promptReports = make(map[string]Report)

type CreatePromptInput struct {
	Title        string
	Description  string
	Cover        string
	Images       []string
	Content      string
	SystemPrompt string
	Model        string
	Params       PromptParams
	CategoryID   int
	Tags         []string
	User         User
	Status       int
}

type PromptFilter struct {
	CategoryID int
	SortBy     string
	UserID     int
	Keyword    string
	Model      string
	Tag        string
}

type ReportPromptInput struct {
	PromptID int
	UserID   int
	Reason   string
	Detail   string
}

func FilterPrompts(categoryID int, sortBy string) []Prompt {
	return QueryPrompts(PromptFilter{
		CategoryID: categoryID,
		SortBy:     sortBy,
	})
}

func QueryPrompts(filter PromptFilter) []Prompt {
	promptMu.RLock()
	defer promptMu.RUnlock()

	list := make([]Prompt, 0, len(prompts))
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	model := strings.ToLower(strings.TrimSpace(filter.Model))
	tag := strings.ToLower(strings.TrimSpace(filter.Tag))

	for _, prompt := range prompts {
		if prompt.Status != 1 {
			continue
		}
		if filter.CategoryID > 0 && prompt.CategoryID != filter.CategoryID {
			continue
		}
		if filter.UserID > 0 && prompt.UserID != filter.UserID {
			continue
		}
		if model != "" && !strings.Contains(strings.ToLower(prompt.Model), model) {
			continue
		}
		if tag != "" && !hasPromptTag(prompt, tag) {
			continue
		}
		if keyword != "" && !matchesKeyword(prompt, keyword) {
			continue
		}

		list = append(list, prompt)
	}

	switch filter.SortBy {
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

func hasPromptTag(prompt Prompt, tag string) bool {
	for _, item := range prompt.Tags {
		if strings.ToLower(strings.TrimSpace(item)) == tag {
			return true
		}
	}

	return false
}

func sanitizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	cleaned := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		cleaned = append(cleaned, tag)
		if len(cleaned) >= MaxPromptTags {
			break
		}
	}

	return cleaned
}

func categoryByID(id int) (Category, bool) {
	for _, category := range categories {
		if category.ID == id {
			return category, true
		}
	}

	return Category{}, false
}

func matchesKeyword(prompt Prompt, keyword string) bool {
	if strings.Contains(strings.ToLower(prompt.Title), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(prompt.Description), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(prompt.Content), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(prompt.SystemPrompt), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(prompt.CategoryName), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(prompt.Model), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(prompt.User.Username), keyword) {
		return true
	}

	for _, tag := range prompt.Tags {
		if strings.Contains(strings.ToLower(tag), keyword) {
			return true
		}
	}

	return false
}

func FindPromptByID(id int) (Prompt, bool) {
	promptMu.RLock()
	defer promptMu.RUnlock()

	for _, prompt := range prompts {
		if prompt.ID == id && prompt.Status == 1 {
			return prompt, true
		}
	}

	return Prompt{}, false
}

func FindOwnedPromptByID(id int, userID int) (Prompt, bool) {
	promptMu.RLock()
	defer promptMu.RUnlock()

	for _, prompt := range prompts {
		if prompt.ID == id && prompt.UserID == userID && prompt.Status != -1 {
			return prompt, true
		}
	}

	return Prompt{}, false
}

func CreatePrompt(input CreatePromptInput) (Prompt, error) {
	if err := ValidatePromptModeration(input); err != nil {
		return Prompt{}, err
	}

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
		Images:       sanitizeImages(input.Images),
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
		Status:       normalizePromptStatus(input.Status),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	prompts = append([]Prompt{prompt}, prompts...)
	if prompt.Status == 1 {
		incrementCategoryCountLocked(input.CategoryID)
	}

	return prompt, nil
}

func UpdatePrompt(id int, userID int, input CreatePromptInput) (Prompt, error) {
	if err := ValidatePromptModeration(input); err != nil {
		return Prompt{}, err
	}

	promptMu.Lock()
	defer promptMu.Unlock()

	category, ok := findCategoryByID(input.CategoryID)
	if !ok {
		return Prompt{}, fmt.Errorf("invalid category")
	}

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status == -1 {
			continue
		}
		if prompts[index].UserID != userID {
			return Prompt{}, fmt.Errorf("forbidden")
		}

		previousCategoryID := prompts[index].CategoryID
		previousStatus := prompts[index].Status
		prompts[index].Title = strings.TrimSpace(input.Title)
		prompts[index].Description = strings.TrimSpace(input.Description)
		prompts[index].Cover = strings.TrimSpace(input.Cover)
		prompts[index].Images = sanitizeImages(input.Images)
		prompts[index].Content = strings.TrimSpace(input.Content)
		prompts[index].SystemPrompt = strings.TrimSpace(input.SystemPrompt)
		prompts[index].Model = strings.TrimSpace(input.Model)
		prompts[index].Params = input.Params
		prompts[index].CategoryID = input.CategoryID
		prompts[index].CategoryName = category.Name
		prompts[index].Tags = normalizeTags(input.Tags)
		prompts[index].Status = normalizePromptStatus(input.Status)
		prompts[index].UpdatedAt = time.Now().UTC().Format(time.DateOnly)

		if previousStatus == 1 && prompts[index].Status != 1 {
			decrementCategoryCountLocked(previousCategoryID)
		} else if previousStatus != 1 && prompts[index].Status == 1 {
			incrementCategoryCountLocked(input.CategoryID)
		} else if prompts[index].Status == 1 && previousCategoryID != input.CategoryID {
			decrementCategoryCountLocked(previousCategoryID)
			incrementCategoryCountLocked(input.CategoryID)
		}

		return prompts[index], nil
	}

	return Prompt{}, fmt.Errorf("prompt not found")
}

func DeletePrompt(id int, userID int) error {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status == -1 {
			continue
		}
		if prompts[index].UserID != userID {
			return fmt.Errorf("forbidden")
		}

		if prompts[index].Status == 1 {
			decrementCategoryCountLocked(prompts[index].CategoryID)
		}
		prompts[index].Status = -1
		prompts[index].UpdatedAt = time.Now().UTC().Format(time.DateOnly)
		return nil
	}

	return fmt.Errorf("prompt not found")
}

func LikePrompt(id int, userID int) (Prompt, bool, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status != 1 {
			continue
		}

		if _, ok := promptLikes[id]; !ok {
			promptLikes[id] = make(map[int]struct{})
		}
		if _, exists := promptLikes[id][userID]; exists {
			return prompts[index], false, nil
		}

		promptLikes[id][userID] = struct{}{}
		prompts[index].Likes++
		return prompts[index], true, nil
	}

	return Prompt{}, false, fmt.Errorf("prompt not found")
}

func FavoritePrompt(id int, userID int) (Prompt, bool, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status != 1 {
			continue
		}

		if _, ok := promptFavorites[id]; !ok {
			promptFavorites[id] = make(map[int]struct{})
		}
		if _, exists := promptFavorites[id][userID]; exists {
			return prompts[index], false, nil
		}

		promptFavorites[id][userID] = struct{}{}
		prompts[index].Favorites++
		return prompts[index], true, nil
	}

	return Prompt{}, false, fmt.Errorf("prompt not found")
}

func RecordPromptView(id int, userID int) (Prompt, bool, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status != 1 {
			continue
		}

		if _, ok := promptViewHistory[userID]; !ok {
			promptViewHistory[userID] = make(map[int]time.Time)
		}
		_, existed := promptViewHistory[userID][id]
		promptViewHistory[userID][id] = time.Now().UTC()
		if !existed {
			prompts[index].Views++
		}

		return prompts[index], !existed, nil
	}

	return Prompt{}, false, fmt.Errorf("prompt not found")
}

func ReportPrompt(id int, userID int, reason string, detail string) (Report, bool, error) {
	input := ReportPromptInput{
		PromptID: id,
		UserID:   userID,
		Reason:   reason,
		Detail:   detail,
	}
	if err := validateReportPromptInput(input); err != nil {
		return Report{}, false, err
	}

	promptMu.Lock()
	defer promptMu.Unlock()

	found := false
	for _, prompt := range prompts {
		if prompt.ID == id && prompt.Status == 1 {
			found = true
			break
		}
	}
	if !found {
		return Report{}, false, fmt.Errorf("prompt not found")
	}

	key := fmt.Sprintf("prompt:%d:%d", userID, id)
	if report, exists := promptReports[key]; exists {
		return report, false, nil
	}

	report := Report{
		ID:         nextPromptReportIDLocked(),
		UserID:     userID,
		TargetType: "prompt",
		TargetID:   id,
		Reason:     strings.TrimSpace(reason),
		Detail:     strings.TrimSpace(detail),
		Status:     "pending",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	promptReports[key] = report

	return report, true, nil
}

func ListUserFavoritePrompts(userID int) []Prompt {
	promptMu.RLock()
	defer promptMu.RUnlock()

	ids := promptFavoritesByUserLocked(userID)
	return promptsByIDLocked(ids)
}

func ListUserLikedPrompts(userID int) []Prompt {
	promptMu.RLock()
	defer promptMu.RUnlock()

	ids := promptLikesByUserLocked(userID)
	return promptsByIDLocked(ids)
}

func ListUserHistoryPrompts(userID int) []Prompt {
	promptMu.RLock()
	defer promptMu.RUnlock()

	visitedAt := promptViewHistory[userID]
	ids := make([]int, 0, len(visitedAt))
	for promptID := range visitedAt {
		ids = append(ids, promptID)
	}

	list := promptsByIDLocked(ids)
	sort.SliceStable(list, func(i, j int) bool {
		return visitedAt[list[i].ID].After(visitedAt[list[j].ID])
	})

	return list
}

func ListUserDraftPrompts(userID int) []Prompt {
	promptMu.RLock()
	defer promptMu.RUnlock()

	list := make([]Prompt, 0)
	for _, prompt := range prompts {
		if prompt.UserID == userID && prompt.Status == 0 {
			list = append(list, prompt)
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})

	return list
}

func validateReportPromptInput(input ReportPromptInput) error {
	if input.PromptID <= 0 {
		return fmt.Errorf("invalid prompt id")
	}
	if input.UserID <= 0 {
		return fmt.Errorf("invalid user")
	}

	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return fmt.Errorf("report reason is required")
	}
	if len([]rune(reason)) > 80 {
		return fmt.Errorf("report reason must be 80 characters or fewer")
	}
	if len([]rune(strings.TrimSpace(input.Detail))) > 500 {
		return fmt.Errorf("report detail must be 500 characters or fewer")
	}

	return nil
}

func nextPromptReportIDLocked() int {
	maxID := 0
	for _, report := range promptReports {
		if report.ID > maxID {
			maxID = report.ID
		}
	}

	return maxID + 1
}

func promptFavoritesByUserLocked(userID int) []int {
	ids := make([]int, 0)
	for promptID, users := range promptFavorites {
		if _, ok := users[userID]; ok {
			ids = append(ids, promptID)
		}
	}

	return ids
}

func promptLikesByUserLocked(userID int) []int {
	ids := make([]int, 0)
	for promptID, users := range promptLikes {
		if _, ok := users[userID]; ok {
			ids = append(ids, promptID)
		}
	}

	return ids
}

func promptsByIDLocked(ids []int) []Prompt {
	allowed := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}

	list := make([]Prompt, 0, len(ids))
	for _, prompt := range prompts {
		if _, ok := allowed[prompt.ID]; ok {
			list = append(list, prompt)
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})

	return list
}

func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
		if len(result) >= MaxPromptTags {
			break
		}
	}

	return result
}

func sanitizeImages(images []string) []string {
	seen := make(map[string]struct{}, len(images))
	cleaned := make([]string, 0, len(images))
	for _, raw := range images {
		image := strings.TrimSpace(raw)
		if image == "" {
			continue
		}
		if _, exists := seen[image]; exists {
			continue
		}
		seen[image] = struct{}{}
		cleaned = append(cleaned, image)
		if len(cleaned) >= 12 {
			break
		}
	}

	return cleaned
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

func decrementCategoryCountLocked(categoryID int) {
	for index := range categories {
		if categories[index].ID == categoryID && categories[index].Count > 0 {
			categories[index].Count--
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

func normalizePromptStatus(status int) int {
	if status == 0 {
		return 0
	}

	return 1
}
