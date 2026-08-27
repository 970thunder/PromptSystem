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
	keyword := strings.ToLower(capKeywordLength(strings.TrimSpace(filter.Keyword)))
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
			if list[i].Likes != list[j].Likes {
				return list[i].Likes > list[j].Likes
			}
			return list[i].ID > list[j].ID
		})
	default:
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].CreatedAt != list[j].CreatedAt {
				return list[i].CreatedAt > list[j].CreatedAt
			}
			return list[i].ID > list[j].ID
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
		return Prompt{}, ErrInvalidCategory
	}

	tags, err := NormalizePromptTags(input.Tags)
	if err != nil {
		return Prompt{}, err
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
		Tags:         tags,
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
		return Prompt{}, ErrInvalidCategory
	}

	tags, err := NormalizePromptTags(input.Tags)
	if err != nil {
		return Prompt{}, err
	}

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status == -1 {
			continue
		}
		if prompts[index].UserID != userID {
			return Prompt{}, ErrPromptForbidden
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
		prompts[index].Tags = tags
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

	return Prompt{}, ErrPromptNotFound
}

func DeletePrompt(id int, userID int) error {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status == -1 {
			continue
		}
		if prompts[index].UserID != userID {
			return ErrPromptForbidden
		}

		if prompts[index].Status == 1 {
			decrementCategoryCountLocked(prompts[index].CategoryID)
		}
		prompts[index].Status = -1
		prompts[index].UpdatedAt = time.Now().UTC().Format(time.DateOnly)
		return nil
	}

	return ErrPromptNotFound
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

	return Prompt{}, false, ErrPromptNotFound
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

	return Prompt{}, false, ErrPromptNotFound
}

// UnlikePrompt removes a like and decrements the counter in one critical
// section. Repeating the undo is a no-op (applied=false).
func UnlikePrompt(id int, userID int) (Prompt, bool, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status != 1 {
			continue
		}
		if promptLikes[id] == nil || !mapHasUser(promptLikes[id], userID) {
			return prompts[index], false, nil
		}
		delete(promptLikes[id], userID)
		if prompts[index].Likes > 0 {
			prompts[index].Likes--
		}
		return prompts[index], true, nil
	}

	return Prompt{}, false, ErrPromptNotFound
}

// UnfavoritePrompt mirrors UnlikePrompt for favorites.
func UnfavoritePrompt(id int, userID int) (Prompt, bool, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status != 1 {
			continue
		}
		if promptFavorites[id] == nil || !mapHasUser(promptFavorites[id], userID) {
			return prompts[index], false, nil
		}
		delete(promptFavorites[id], userID)
		if prompts[index].Favorites > 0 {
			prompts[index].Favorites--
		}
		return prompts[index], true, nil
	}

	return Prompt{}, false, ErrPromptNotFound
}

func mapHasUser(m map[int]struct{}, userID int) bool {
	_, ok := m[userID]
	return ok
}

// GetPromptInteractionStatus returns the per-user like/favorite state of a
// prompt. A missing or soft-deleted prompt yields ErrPromptNotFound.
func GetPromptInteractionStatus(id int, userID int) (InteractionStatus, error) {
	promptMu.RLock()
	defer promptMu.RUnlock()

	found := false
	for _, prompt := range prompts {
		if prompt.ID == id && prompt.Status == 1 {
			found = true
			break
		}
	}
	if !found {
		return InteractionStatus{}, ErrPromptNotFound
	}

	status := InteractionStatus{}
	if promptLikes[id] != nil {
		status.Liked = mapHasUser(promptLikes[id], userID)
	}
	if promptFavorites[id] != nil {
		status.Favorited = mapHasUser(promptFavorites[id], userID)
	}
	return status, nil
}

func RecordPromptView(id int, userID int) (Prompt, bool, error) {
	promptMu.Lock()
	defer promptMu.Unlock()

	for index := range prompts {
		if prompts[index].ID != id || prompts[index].Status != 1 {
			continue
		}

		// Anonymous views (userID <= 0) count toward the total views counter but
		// are deliberately NOT written to the browsing history, because they
		// cannot be attributed to a user. Each anonymous view is an independent
		// counter increment.
		if userID <= 0 {
			prompts[index].Views++
			return prompts[index], true, nil
		}

		if _, ok := promptViewHistory[userID]; !ok {
			promptViewHistory[userID] = make(map[int]time.Time)
		}
		_, existed := promptViewHistory[userID][id]
		promptViewHistory[userID][id] = time.Now().UTC()
		// One history row per (user, prompt): repeat views bump viewed_at but
		// never create a new row or a second counter increment.
		if !existed {
			prompts[index].Views++
		}

		return prompts[index], !existed, nil
	}

	return Prompt{}, false, ErrPromptNotFound
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
		return Report{}, false, ErrPromptNotFound
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

	return listUserHistoryPromptsLocked(userID)
}

// listUserHistoryPromptsLocked returns the requesting user's browsing history,
// newest view first, restricted to published (status == 1) prompts so soft
// deleted prompts never leak into a history response. Caller must hold
// promptMu read lock.
func listUserHistoryPromptsLocked(userID int) []Prompt {
	visitedAt := promptViewHistory[userID]
	list := make([]Prompt, 0, len(visitedAt))
	for _, prompt := range prompts {
		if prompt.Status != 1 {
			continue
		}
		if _, ok := visitedAt[prompt.ID]; ok {
			list = append(list, prompt)
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return visitedAt[list[i].ID].After(visitedAt[list[j].ID])
	})

	return list
}

// ListUserHistoryPagePrompts returns a page of the user's browsing history plus
// the total, filtering out soft-deleted prompts, mirroring the MySQL store's
// pagination semantics.
func ListUserHistoryPagePrompts(userID, page, pageSize int) ([]Prompt, int) {
	promptMu.RLock()
	defer promptMu.RUnlock()

	all := listUserHistoryPromptsLocked(userID)
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
	return all[start:end], total
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
		return ErrPromptNotFound
	}
	if input.UserID <= 0 {
		return fmt.Errorf("invalid user")
	}

	reason := strings.TrimSpace(input.Reason)
	if !ValidReportReason(reason) {
		return ErrInvalidReportReason
	}
	if len([]rune(strings.TrimSpace(input.Detail))) > MaxReportDetailRunes {
		return fmt.Errorf("report detail must be %d characters or fewer", MaxReportDetailRunes)
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

// normalizeTagValue trims surrounding whitespace and collapses any run of
// internal whitespace to a single space, so tags like "  摄影  大师  " become the
// canonical "摄影 大师". This keeps persisted tags consistent and prevents
// duplicates that differ only in spacing.
// MaxSearchKeywordLen bounds the length of a LIKE keyword. Boundless LIKE terms
// degrade index usage and can be abused for expensive scans; MySQL ingestion of
// overly long patterns is also wasted work. Cap it (in runes) since CJK keywords
// are common and rune-based is exact.
const MaxSearchKeywordLen = 64

// capKeywordLength truncates a search keyword to MaxSearchKeywordLen runes. It
// preserves multi-byte characters (e.g. Chinese) by slicing on rune boundaries.
func capKeywordLength(keyword string) string {
	runes := []rune(keyword)
	if len(runes) <= MaxSearchKeywordLen {
		return keyword
	}
	return string(runes[:MaxSearchKeywordLen])
}

func normalizeTagValue(tag string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(tag)), " ")
}

// NormalizePromptTags normalizes a prompt's tag list for persistence: each tag
// is trimmed and whitespace-collapsed, tags are deduplicated (case-preserving),
// the count is capped at MaxPromptTags, and every individual tag is capped at
// MaxPromptTagLength. It returns ErrInvalidTag if a tag is entirely whitespace
// or longer than MaxPromptTagLength after normalization.
func NormalizePromptTags(tags []string) ([]string, error) {
	if len(tags) > MaxPromptTags {
		tags = tags[:MaxPromptTags]
	}
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tagValue := normalizeTagValue(tag)
		if tagValue == "" {
			return nil, ErrInvalidTag
		}
		if len([]rune(tagValue)) > MaxPromptTagLength {
			return nil, ErrInvalidTag
		}
		if _, exists := seen[tagValue]; exists {
			continue
		}
		seen[tagValue] = struct{}{}
		result = append(result, tagValue)
		if len(result) >= MaxPromptTags {
			break
		}
	}

	return result, nil
}

// QueryPagePrompts applies the same filtering as QueryPrompts but returns one
// page plus the total, mirroring the MySQL store semantics.
func QueryPagePrompts(filter PromptFilter, page, pageSize int) ([]Prompt, int) {
	all := QueryPrompts(filter)
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
	return all[start:end], total
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
