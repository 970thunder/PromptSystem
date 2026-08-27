package store

type MemoryPromptStore struct{}

func NewMemoryPromptStore() *MemoryPromptStore {
	return &MemoryPromptStore{}
}

func (s *MemoryPromptStore) Query(filter PromptFilter) ([]Prompt, error) {
	return QueryPrompts(filter), nil
}

func (s *MemoryPromptStore) QueryPage(filter PromptFilter, page, pageSize int) ([]Prompt, int, error) {
	prompts, total := QueryPagePrompts(filter, page, pageSize)
	return prompts, total, nil
}

func (s *MemoryPromptStore) HomeSummary() (HomeSummary, error) {
	prompts := QueryPrompts(PromptFilter{})
	summary := HomeSummary{PromptCount: len(prompts)}
	creators := map[int]struct{}{}
	tagCount := map[string]int{}
	catCount := map[string]int{}
	for _, p := range prompts {
		creators[p.UserID] = struct{}{}
		summary.TotalViews += int64(p.Views)
		for _, tag := range p.Tags {
			tagCount[tag]++
		}
		catCount[p.CategoryName]++
	}
	summary.CreatorCount = len(creators)
	summary.HotTags = topKeys(tagCount, 8)
	summary.HotCategories = topKeys(catCount, 8)
	return summary, nil
}

func topKeys(m map[string]int, limit int) []string {
	type kv struct {
		k string
		v int
	}
	list := make([]kv, 0, len(m))
	for k, v := range m {
		list = append(list, kv{k, v})
	}
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].v > list[j-1].v; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
	out := make([]string, 0, limit)
	for i := 0; i < len(list) && i < limit; i++ {
		out = append(out, list[i].k)
	}
	return out
}

func (s *MemoryPromptStore) FindByID(id int) (Prompt, bool, error) {
	prompt, ok := FindPromptByID(id)
	return prompt, ok, nil
}

func (s *MemoryPromptStore) FindOwnedByID(id int, userID int) (Prompt, bool, error) {
	prompt, ok := FindOwnedPromptByID(id, userID)
	return prompt, ok, nil
}

func (s *MemoryPromptStore) Create(input CreatePromptInput) (Prompt, error) {
	return CreatePrompt(input)
}

func (s *MemoryPromptStore) Update(id int, userID int, input CreatePromptInput) (Prompt, error) {
	return UpdatePrompt(id, userID, input)
}

func (s *MemoryPromptStore) Delete(id int, userID int) error {
	return DeletePrompt(id, userID)
}

func (s *MemoryPromptStore) Like(id int, userID int) (Prompt, bool, error) {
	return LikePrompt(id, userID)
}

func (s *MemoryPromptStore) Favorite(id int, userID int) (Prompt, bool, error) {
	return FavoritePrompt(id, userID)
}

func (s *MemoryPromptStore) RecordView(id int, userID int) (Prompt, bool, error) {
	return RecordPromptView(id, userID)
}

func (s *MemoryPromptStore) Report(id int, userID int, reason string, detail string) (Report, bool, error) {
	return ReportPrompt(id, userID, reason, detail)
}

func (s *MemoryPromptStore) ListUserFavorites(userID int) ([]Prompt, error) {
	return ListUserFavoritePrompts(userID), nil
}

func (s *MemoryPromptStore) ListUserLikes(userID int) ([]Prompt, error) {
	return ListUserLikedPrompts(userID), nil
}

func (s *MemoryPromptStore) ListUserHistory(userID int) ([]Prompt, error) {
	return ListUserHistoryPrompts(userID), nil
}

func (s *MemoryPromptStore) ListUserDrafts(userID int) ([]Prompt, error) {
	return ListUserDraftPrompts(userID), nil
}
