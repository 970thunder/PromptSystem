package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type MySQLPromptStore struct {
	db *sql.DB
}

func NewMySQLPromptStore(db *sql.DB) *MySQLPromptStore {
	return &MySQLPromptStore{db: db}
}

func (s *MySQLPromptStore) Query(filter PromptFilter) ([]Prompt, error) {
	baseQuery := `
		SELECT
			p.id, p.title, p.description, p.cover, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE p.status = 1
	`

	var (
		conditions []string
		args       []any
	)

	if filter.CategoryID > 0 {
		conditions = append(conditions, "p.category_id = ?")
		args = append(args, filter.CategoryID)
	}
	if filter.UserID > 0 {
		conditions = append(conditions, "p.user_id = ?")
		args = append(args, filter.UserID)
	}
	if model := strings.TrimSpace(filter.Model); model != "" {
		conditions = append(conditions, "LOWER(p.model) LIKE ?")
		args = append(args, "%"+strings.ToLower(model)+"%")
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		conditions = append(conditions, `(LOWER(p.title) LIKE ? OR LOWER(p.description) LIKE ? OR LOWER(p.content) LIKE ? OR LOWER(p.system_prompt) LIKE ? OR LOWER(c.name) LIKE ? OR LOWER(p.model) LIKE ? OR LOWER(u.username) LIKE ? OR EXISTS (
			SELECT 1 FROM prompt_tags search_tags WHERE search_tags.prompt_id = p.id AND LOWER(search_tags.tag) LIKE ?
		))`)
		args = append(args, like, like, like, like, like, like, like, like)
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " GROUP BY p.id"
	if filter.SortBy == "popular" {
		baseQuery += " ORDER BY p.likes DESC, p.created_at DESC"
	} else {
		baseQuery += " ORDER BY p.created_at DESC, p.id DESC"
	}

	rows, err := s.db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		prompt, err := scanPrompt(rows.Scan)
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, prompt)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prompts, nil
}

func (s *MySQLPromptStore) FindByID(id int) (Prompt, bool, error) {
	row := s.db.QueryRow(`
		SELECT
			p.id, p.title, p.description, p.cover, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE p.id = ? AND p.status = 1
		GROUP BY p.id
	`, id)

	prompt, err := scanPrompt(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Prompt{}, false, nil
	}
	if err != nil {
		return Prompt{}, false, err
	}

	return prompt, true, nil
}

func (s *MySQLPromptStore) Create(input CreatePromptInput) (Prompt, error) {
	category, ok := categoryByID(input.CategoryID)
	if !ok {
		return Prompt{}, errors.New("invalid category")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Prompt{}, err
	}
	defer tx.Rollback()

	paramsJSON, err := json.Marshal(input.Params)
	if err != nil {
		return Prompt{}, err
	}

	result, err := tx.Exec(`
		INSERT INTO prompts (
			title, description, cover, content, system_prompt, model, params,
			category_id, user_id, views, likes, favorites, status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 1)
	`,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		strings.TrimSpace(input.Cover),
		strings.TrimSpace(input.Content),
		strings.TrimSpace(input.SystemPrompt),
		strings.TrimSpace(input.Model),
		string(paramsJSON),
		input.CategoryID,
		input.User.ID,
	)
	if err != nil {
		return Prompt{}, err
	}

	promptID, err := result.LastInsertId()
	if err != nil {
		return Prompt{}, err
	}

	for _, tag := range sanitizeTags(input.Tags) {
		if _, err := tx.Exec(`INSERT INTO prompt_tags (prompt_id, tag) VALUES (?, ?)`, promptID, tag); err != nil {
			return Prompt{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Prompt{}, err
	}

	prompt, found, err := s.FindByID(int(promptID))
	if err != nil {
		return Prompt{}, err
	}
	if !found {
		now := time.Now().UTC().Format("2006-01-02")
		return Prompt{
			ID:           int(promptID),
			Title:        strings.TrimSpace(input.Title),
			Description:  strings.TrimSpace(input.Description),
			Cover:        strings.TrimSpace(input.Cover),
			Content:      strings.TrimSpace(input.Content),
			SystemPrompt: strings.TrimSpace(input.SystemPrompt),
			Model:        strings.TrimSpace(input.Model),
			Params:       input.Params,
			CategoryID:   input.CategoryID,
			CategoryName: category.Name,
			Tags:         sanitizeTags(input.Tags),
			UserID:       input.User.ID,
			User:         input.User,
			Status:       1,
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}

	return prompt, nil
}

func (s *MySQLPromptStore) Update(id int, userID int, input CreatePromptInput) (Prompt, error) {
	category, ok := categoryByID(input.CategoryID)
	if !ok {
		return Prompt{}, errors.New("invalid category")
	}

	current, found, err := s.FindByID(id)
	if err != nil {
		return Prompt{}, err
	}
	if !found {
		return Prompt{}, errors.New("prompt not found")
	}
	if current.UserID != userID {
		return Prompt{}, errors.New("forbidden")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Prompt{}, err
	}
	defer tx.Rollback()

	paramsJSON, err := json.Marshal(input.Params)
	if err != nil {
		return Prompt{}, err
	}

	if _, err := tx.Exec(`
		UPDATE prompts
		SET title = ?, description = ?, cover = ?, content = ?, system_prompt = ?, model = ?, params = ?, category_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND status = 1
	`,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		strings.TrimSpace(input.Cover),
		strings.TrimSpace(input.Content),
		strings.TrimSpace(input.SystemPrompt),
		strings.TrimSpace(input.Model),
		string(paramsJSON),
		input.CategoryID,
		id,
		userID,
	); err != nil {
		return Prompt{}, err
	}

	if _, err := tx.Exec(`DELETE FROM prompt_tags WHERE prompt_id = ?`, id); err != nil {
		return Prompt{}, err
	}

	for _, tag := range sanitizeTags(input.Tags) {
		if _, err := tx.Exec(`INSERT INTO prompt_tags (prompt_id, tag) VALUES (?, ?)`, id, tag); err != nil {
			return Prompt{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Prompt{}, err
	}

	prompt, found, err := s.FindByID(id)
	if err != nil {
		return Prompt{}, err
	}
	if !found {
		now := time.Now().UTC().Format("2006-01-02")
		return Prompt{
			ID:           id,
			Title:        strings.TrimSpace(input.Title),
			Description:  strings.TrimSpace(input.Description),
			Cover:        strings.TrimSpace(input.Cover),
			Content:      strings.TrimSpace(input.Content),
			SystemPrompt: strings.TrimSpace(input.SystemPrompt),
			Model:        strings.TrimSpace(input.Model),
			Params:       input.Params,
			CategoryID:   input.CategoryID,
			CategoryName: category.Name,
			Tags:         sanitizeTags(input.Tags),
			UserID:       userID,
			User:         current.User,
			Views:        current.Views,
			Likes:        current.Likes,
			Favorites:    current.Favorites,
			Status:       current.Status,
			CreatedAt:    current.CreatedAt,
			UpdatedAt:    now,
		}, nil
	}

	return prompt, nil
}

func (s *MySQLPromptStore) Delete(id int, userID int) error {
	result, err := s.db.Exec(`
		UPDATE prompts
		SET status = -1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND status = 1
	`, id, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		prompt, found, lookupErr := s.FindByID(id)
		if lookupErr != nil {
			return lookupErr
		}
		if !found {
			return errors.New("prompt not found")
		}
		if prompt.UserID != userID {
			return errors.New("forbidden")
		}
	}

	return nil
}

func (s *MySQLPromptStore) Like(id int, userID int) (Prompt, bool, error) {
	return s.applyEngagement("likes", "likes", id, userID)
}

func (s *MySQLPromptStore) Favorite(id int, userID int) (Prompt, bool, error) {
	return s.applyEngagement("favorites", "favorites", id, userID)
}

func (s *MySQLPromptStore) applyEngagement(table string, counterColumn string, id int, userID int) (Prompt, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Prompt{}, false, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"INSERT IGNORE INTO "+table+" (user_id, target_type, target_id) VALUES (?, 'prompt', ?)",
		userID,
		id,
	)
	if err != nil {
		return Prompt{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Prompt{}, false, err
	}
	applied := affected > 0

	if applied {
		if _, err := tx.Exec(
			"UPDATE prompts SET "+counterColumn+" = "+counterColumn+" + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 1",
			id,
		); err != nil {
			return Prompt{}, false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Prompt{}, false, err
	}

	prompt, found, err := s.FindByID(id)
	if err != nil {
		return Prompt{}, false, err
	}
	if !found {
		return Prompt{}, false, errors.New("prompt not found")
	}

	return prompt, applied, nil
}

func scanPrompt(scan func(dest ...any) error) (Prompt, error) {
	var (
		prompt          Prompt
		paramsRaw       string
		tagList         string
		userCreatedAt   time.Time
		promptCreatedAt time.Time
		promptUpdatedAt time.Time
	)

	err := scan(
		&prompt.ID,
		&prompt.Title,
		&prompt.Description,
		&prompt.Cover,
		&prompt.Content,
		&prompt.SystemPrompt,
		&prompt.Model,
		&paramsRaw,
		&prompt.CategoryID,
		&prompt.CategoryName,
		&prompt.UserID,
		&prompt.User.Username,
		&prompt.User.Avatar,
		&prompt.User.Email,
		&prompt.User.Bio,
		&prompt.User.Level,
		&prompt.User.Experience,
		&prompt.User.Status,
		&userCreatedAt,
		&prompt.Views,
		&prompt.Likes,
		&prompt.Favorites,
		&prompt.Status,
		&promptCreatedAt,
		&promptUpdatedAt,
		&tagList,
	)
	if err != nil {
		return Prompt{}, err
	}

	if strings.TrimSpace(paramsRaw) != "" {
		if err := json.Unmarshal([]byte(paramsRaw), &prompt.Params); err != nil {
			return Prompt{}, err
		}
	}

	prompt.User.ID = prompt.UserID
	prompt.User.CreatedAt = userCreatedAt.UTC().Format("2006-01-02")
	prompt.CreatedAt = promptCreatedAt.UTC().Format("2006-01-02")
	prompt.UpdatedAt = promptUpdatedAt.UTC().Format("2006-01-02")
	prompt.Tags = splitTags(tagList)

	return prompt, nil
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	parts := strings.Split(raw, "||")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}
