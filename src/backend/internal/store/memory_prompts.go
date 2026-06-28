package store

type MemoryPromptStore struct{}

func NewMemoryPromptStore() *MemoryPromptStore {
	return &MemoryPromptStore{}
}

func (s *MemoryPromptStore) Query(filter PromptFilter) ([]Prompt, error) {
	return QueryPrompts(filter), nil
}

func (s *MemoryPromptStore) FindByID(id int) (Prompt, bool, error) {
	prompt, ok := FindPromptByID(id)
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

func (s *MemoryPromptStore) ListUserFavorites(userID int) ([]Prompt, error) {
	return ListUserFavoritePrompts(userID), nil
}

func (s *MemoryPromptStore) ListUserLikes(userID int) ([]Prompt, error) {
	return ListUserLikedPrompts(userID), nil
}

func (s *MemoryPromptStore) ListUserHistory(userID int) ([]Prompt, error) {
	return ListUserHistoryPrompts(userID), nil
}
