package store

import (
	"database/sql"
	"encoding/json"
	"sort"
	"time"
)

func SeedMySQLData(db *sql.DB, includeDemo bool) error {
	if err := seedCategories(db); err != nil {
		return err
	}
	if !includeDemo {
		return nil
	}

	if err := seedUsers(db); err != nil {
		return err
	}

	if err := seedPrompts(db); err != nil {
		return err
	}

	return nil
}

func seedCategories(db *sql.DB) error {
	for index, category := range categories {
		if _, err := db.Exec(`
			INSERT INTO categories (id, name, icon, sort, type)
			VALUES (?, ?, ?, ?, 1)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				icon = VALUES(icon),
				sort = VALUES(sort),
				type = VALUES(type)
		`,
			category.ID,
			category.Name,
			category.Icon,
			index+1,
		); err != nil {
			return err
		}
	}

	return nil
}

func seedUsers(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	for _, user := range seedAuthUsers() {
		createdAt, err := time.Parse("2006-01-02", user.CreatedAt)
		if err != nil {
			return err
		}

		if _, err := db.Exec(`
			INSERT INTO users (id, username, avatar, email, password, bio, level, experience, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			user.ID,
			user.Username,
			user.Avatar,
			user.Email,
			user.PasswordHash,
			user.Bio,
			user.Level,
			user.Experience,
			user.Status,
			createdAt,
			createdAt,
		); err != nil {
			return err
		}
	}

	return nil
}

func seedAuthUsers() []AuthUser {
	store := NewUserStore()
	userMap := make(map[int]AuthUser, len(store.users)+len(prompts))

	for _, user := range store.users {
		userMap[user.ID] = user
	}

	defaultHash := mustHashPassword("PromptOS123!")
	for _, prompt := range prompts {
		if _, exists := userMap[prompt.User.ID]; exists {
			continue
		}

		userMap[prompt.User.ID] = AuthUser{
			ID:           prompt.User.ID,
			Username:     prompt.User.Username,
			Avatar:       prompt.User.Avatar,
			Email:        prompt.User.Email,
			PasswordHash: defaultHash,
			Bio:          prompt.User.Bio,
			Level:        prompt.User.Level,
			Experience:   prompt.User.Experience,
			Status:       prompt.User.Status,
			CreatedAt:    prompt.User.CreatedAt,
		}
	}

	ids := make([]int, 0, len(userMap))
	for id := range userMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	result := make([]AuthUser, 0, len(ids))
	for _, id := range ids {
		result = append(result, userMap[id])
	}

	return result
}

func seedPrompts(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM prompts`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, prompt := range prompts {
		paramsJSON, err := json.Marshal(prompt.Params)
		if err != nil {
			return err
		}

		createdAt, err := time.Parse("2006-01-02", prompt.CreatedAt)
		if err != nil {
			return err
		}
		updatedAt, err := time.Parse("2006-01-02", prompt.UpdatedAt)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(`
			INSERT INTO prompts (
				id, title, description, cover, content, system_prompt, model, params,
				category_id, user_id, views, likes, favorites, status, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			prompt.ID,
			prompt.Title,
			prompt.Description,
			prompt.Cover,
			prompt.Content,
			prompt.SystemPrompt,
			prompt.Model,
			string(paramsJSON),
			prompt.CategoryID,
			prompt.UserID,
			prompt.Views,
			prompt.Likes,
			prompt.Favorites,
			prompt.Status,
			createdAt,
			updatedAt,
		); err != nil {
			return err
		}

		tags, err := NormalizePromptTags(prompt.Tags)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			if _, err := tx.Exec(`INSERT INTO prompt_tags (prompt_id, tag) VALUES (?, ?)`, prompt.ID, tag); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
