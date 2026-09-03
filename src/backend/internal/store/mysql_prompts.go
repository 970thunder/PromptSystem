package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type MySQLPromptStore struct {
	db *sql.DB

	// allowedImageDomains holds the allowlist of HTTPS hostnames permitted for
	// remote cover/image references. When empty, only paths under /uploads/ are
	// accepted. Populate it via SetAllowedImageDomains with the deployment's R2
	// or image CDN hostnames.
	allowedImageDomains []string
}

func NewMySQLPromptStore(db *sql.DB) *MySQLPromptStore {
	return &MySQLPromptStore{db: db}
}

// SetAllowedImageDomains configures the HTTPS hostnames that may be referenced by
// prompt cover/image fields. Call it at startup from the config layer. Empty or
// nil is safe: only local /uploads/ paths are then persisted.
func (s *MySQLPromptStore) SetAllowedImageDomains(domains []string) {
	s.allowedImageDomains = append([]string(nil), domains...)
}

func (s *MySQLPromptStore) Query(filter PromptFilter) ([]Prompt, error) {
	prompts, _, err := s.queryPage(filter, 1, 100)
	return prompts, err
}

func (s *MySQLPromptStore) QueryPage(filter PromptFilter, page, pageSize int) ([]Prompt, int, error) {
	return s.queryPage(filter, page, pageSize)
}

func (s *MySQLPromptStore) HomeSummary() (HomeSummary, error) {
	var summary HomeSummary
	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM prompts p
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 1 AND u.status = 1
	`).Scan(&summary.PromptCount); err != nil {
		return summary, err
	}
	if err := s.db.QueryRow(`
		SELECT COUNT(DISTINCT p.user_id) FROM prompts p
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 1 AND u.status = 1
	`).Scan(&summary.CreatorCount); err != nil {
		return summary, err
	}
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(p.views), 0) FROM prompts p
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 1 AND u.status = 1
	`).Scan(&summary.TotalViews); err != nil {
		return summary, err
	}

	tagRows, err := s.db.Query(`
	SELECT pt.tag FROM prompt_tags pt
		JOIN prompts p ON p.id = pt.prompt_id
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 1 AND u.status = 1
		GROUP BY pt.tag ORDER BY COUNT(*) DESC LIMIT 8
	`)
	if err != nil {
		return summary, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var tag string
		if err := tagRows.Scan(&tag); err != nil {
			return summary, err
		}
		summary.HotTags = append(summary.HotTags, tag)
	}

	catRows, err := s.db.Query(`
	SELECT c.name FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 1 AND u.status = 1
		GROUP BY c.id, c.name ORDER BY COUNT(*) DESC LIMIT 8
	`)
	if err != nil {
		return summary, err
	}
	defer catRows.Close()
	for catRows.Next() {
		var name string
		if err := catRows.Scan(&name); err != nil {
			return summary, err
		}
		summary.HotCategories = append(summary.HotCategories, name)
	}
	return summary, nil
}

// ListCategories returns categories from the database (prompt type first).
func (s *MySQLPromptStore) ListCategories() ([]Category, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.icon,
		(SELECT COUNT(*) FROM prompts p JOIN users u ON u.id = p.user_id
		 WHERE p.category_id = c.id AND p.status = 1 AND u.status = 1) AS cnt
		FROM categories c
		WHERE c.type = 1
		ORDER BY c.sort ASC, c.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Icon, &cat.Count); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

// CategoryExists reports whether a prompt-type category exists.
func (s *MySQLPromptStore) CategoryExists(id int) (bool, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM categories WHERE id = ? AND type = 1`, id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *MySQLPromptStore) queryPage(filter PromptFilter, page, pageSize int) ([]Prompt, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}

	baseQuery := `
		SELECT
			p.id, p.title, p.description, p.cover, p.images, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE p.status = 1 AND u.status = 1
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
	if tag := strings.TrimSpace(filter.Tag); tag != "" {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM prompt_tags filter_tags WHERE filter_tags.prompt_id = p.id AND LOWER(filter_tags.tag) = ?
		)`)
		args = append(args, strings.ToLower(tag))
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		keyword = capKeywordLength(keyword)
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
		baseQuery += " ORDER BY p.likes DESC, p.id DESC"
	} else {
		baseQuery += " ORDER BY p.created_at DESC, p.id DESC"
	}

	countQuery := `
		SELECT COUNT(DISTINCT p.id)
		FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 1 AND u.status = 1
	`
	var countArgs []any
	if len(conditions) > 0 {
		countQuery += " AND " + strings.Join(conditions, " AND ")
		countArgs = args
	}

	var total int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		prompt, err := scanPrompt(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		prompts = append(prompts, prompt)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return prompts, total, nil
}

func (s *MySQLPromptStore) FindByID(id int) (Prompt, bool, error) {
	return s.findOne(`
		WHERE p.id = ? AND p.status = 1 AND u.status = 1
		GROUP BY p.id
	`, id)
}

func (s *MySQLPromptStore) FindOwnedByID(id int, userID int) (Prompt, bool, error) {
	return s.findOne(`
		WHERE p.id = ? AND p.user_id = ? AND p.status <> -1
		GROUP BY p.id
	`, id, userID)
}

func (s *MySQLPromptStore) findOne(whereClause string, args ...any) (Prompt, bool, error) {
	row := s.db.QueryRow(`
		SELECT
			p.id, p.title, p.description, p.cover, p.images, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		`+whereClause, args...)

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
	if err := ValidatePromptModeration(input); err != nil {
		return Prompt{}, err
	}

	category, ok := categoryByID(input.CategoryID)
	if !ok {
		return Prompt{}, ErrInvalidCategory
	}

	tags, err := NormalizePromptTags(input.Tags)
	if err != nil {
		return Prompt{}, err
	}

	if err := ValidateImageURLs(input.Images, s.allowedImageDomains); err != nil {
		return Prompt{}, err
	}
	if cover := strings.TrimSpace(input.Cover); cover != "" {
		if err := ValidateImageURL(cover, s.allowedImageDomains); err != nil {
			return Prompt{}, err
		}
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Prompt{}, err
	}
	defer tx.Rollback()

	paramsJSON, err := json.Marshal(input.Params)
	if err != nil {
		return Prompt{}, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO prompts (
			title, description, cover, images, content, system_prompt, model, params,
			category_id, user_id, views, likes, favorites, status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 0, ?)
	`,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		strings.TrimSpace(input.Cover),
		mustJSON(sanitizeImages(input.Images)),
		strings.TrimSpace(input.Content),
		strings.TrimSpace(input.SystemPrompt),
		strings.TrimSpace(input.Model),
		string(paramsJSON),
		input.CategoryID,
		input.User.ID,
		normalizePromptStatus(input.Status),
	)
	if err != nil {
		return Prompt{}, err
	}

	promptID, err := result.LastInsertId()
	if err != nil {
		return Prompt{}, err
	}

	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO prompt_tags (prompt_id, tag) VALUES (?, ?)`, promptID, tag); err != nil {
			return Prompt{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Prompt{}, err
	}

	prompt, found, err := s.FindOwnedByID(int(promptID), input.User.ID)
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
			Status:       normalizePromptStatus(input.Status),
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}

	return prompt, nil
}

func (s *MySQLPromptStore) Update(id int, userID int, input CreatePromptInput) (Prompt, error) {
	if err := ValidatePromptModeration(input); err != nil {
		return Prompt{}, err
	}

	category, ok := categoryByID(input.CategoryID)
	if !ok {
		return Prompt{}, ErrInvalidCategory
	}

	tags, err := NormalizePromptTags(input.Tags)
	if err != nil {
		return Prompt{}, err
	}

	if err := ValidateImageURLs(input.Images, s.allowedImageDomains); err != nil {
		return Prompt{}, err
	}
	if cover := strings.TrimSpace(input.Cover); cover != "" {
		if err := ValidateImageURL(cover, s.allowedImageDomains); err != nil {
			return Prompt{}, err
		}
	}

	current, found, err := s.FindOwnedByID(id, userID)
	if err != nil {
		return Prompt{}, err
	}
	if !found {
		var ownerID int
		if err := s.db.QueryRow(`SELECT user_id FROM prompts WHERE id = ? AND status <> -1`, id).Scan(&ownerID); errors.Is(err, sql.ErrNoRows) {
			return Prompt{}, ErrPromptNotFound
		} else if err != nil {
			return Prompt{}, err
		}
		if ownerID != userID {
			return Prompt{}, ErrPromptForbidden
		}
		return Prompt{}, ErrPromptNotFound
	}
	if current.UserID != userID {
		return Prompt{}, ErrPromptForbidden
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Prompt{}, err
	}
	defer tx.Rollback()

	paramsJSON, err := json.Marshal(input.Params)
	if err != nil {
		return Prompt{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE prompts
		SET title = ?, description = ?, cover = ?, images = ?, content = ?, system_prompt = ?, model = ?, params = ?, category_id = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND status <> -1
	`,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		strings.TrimSpace(input.Cover),
		mustJSON(sanitizeImages(input.Images)),
		strings.TrimSpace(input.Content),
		strings.TrimSpace(input.SystemPrompt),
		strings.TrimSpace(input.Model),
		string(paramsJSON),
		input.CategoryID,
		normalizePromptStatus(input.Status),
		id,
		userID,
	); err != nil {
		return Prompt{}, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM prompt_tags WHERE prompt_id = ?`, id); err != nil {
		return Prompt{}, err
	}

	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO prompt_tags (prompt_id, tag) VALUES (?, ?)`, id, tag); err != nil {
			return Prompt{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Prompt{}, err
	}

	prompt, found, err := s.FindOwnedByID(id, userID)
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
			Images:       sanitizeImages(input.Images),
			Content:      strings.TrimSpace(input.Content),
			SystemPrompt: strings.TrimSpace(input.SystemPrompt),
			Model:        strings.TrimSpace(input.Model),
			Params:       input.Params,
			CategoryID:   input.CategoryID,
			CategoryName: category.Name,
			Tags:         tags,
			UserID:       userID,
			User:         current.User,
			Views:        current.Views,
			Likes:        current.Likes,
			Favorites:    current.Favorites,
			Status:       normalizePromptStatus(input.Status),
			CreatedAt:    current.CreatedAt,
			UpdatedAt:    now,
		}, nil
	}

	return prompt, nil
}

func (s *MySQLPromptStore) Delete(id int, userID int) error {
	if _, found, err := s.FindOwnedByID(id, userID); err != nil {
		return err
	} else if !found {
		if prompt, publicFound, lookupErr := s.FindByID(id); lookupErr != nil {
			return lookupErr
		} else if publicFound && prompt.UserID != userID {
			return ErrPromptForbidden
		}
		return ErrPromptNotFound
	}

	result, err := s.db.ExecContext(context.Background(), `
		UPDATE prompts
		SET status = -1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND status <> -1
	`, id, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrPromptNotFound
	}

	return nil
}

func (s *MySQLPromptStore) Like(id int, userID int) (Prompt, bool, error) {
	return s.applyEngagement("likes", "likes", id, userID)
}

func (s *MySQLPromptStore) Favorite(id int, userID int) (Prompt, bool, error) {
	return s.applyEngagement("favorites", "favorites", id, userID)
}

// Unlike removes a like in the same transaction that decrements the counter, so
// a crash can never leave a like row without its counter (or vice versa). It is
// idempotent: undoing a like that was never set is a no-op and reports
// applied=false.
func (s *MySQLPromptStore) Unlike(id int, userID int) (Prompt, bool, error) {
	return s.removeEngagement("likes", "likes", id, userID)
}

// Unfavorite mirrors Unlike for the favorites table.
func (s *MySQLPromptStore) Unfavorite(id int, userID int) (Prompt, bool, error) {
	return s.removeEngagement("favorites", "favorites", id, userID)
}

func (s *MySQLPromptStore) removeEngagement(table, counterColumn string, id, userID int) (Prompt, bool, error) {
	// The status=1 guard, the detail-row DELETE and the counter decrement share
	// one transaction so the removal is atomic and a soft-deleted prompt can
	// never be unliked/unfavorited.
	tx, err := s.db.Begin()
	if err != nil {
		return Prompt{}, false, err
	}
	defer tx.Rollback()

	var published int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM prompts p JOIN users u ON u.id = p.user_id WHERE p.id = ? AND p.status = 1 AND u.status = 1`, id).Scan(&published); err != nil {
		return Prompt{}, false, err
	}
	if published == 0 {
		return Prompt{}, false, ErrPromptNotFound
	}

	result, err := tx.Exec(
		"DELETE FROM "+table+" WHERE user_id = ? AND target_type = 'prompt' AND target_id = ?",
		userID,
		id,
	)
	if err != nil {
		return Prompt{}, false, err
	}

	// RowsAffected>0 means a row was actually deleted; only then do we decrement
	// the counter, keeping the idempotency contract (repeat undo = no-op).
	affected, err := result.RowsAffected()
	if err != nil {
		return Prompt{}, false, err
	}
	removed := affected > 0

	if removed {
		if _, err := tx.Exec(
			"UPDATE prompts SET "+counterColumn+" = "+counterColumn+" - 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 1 AND "+counterColumn+" > 0",
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
		return Prompt{}, false, ErrPromptNotFound
	}

	return prompt, removed, nil
}

// GetInteractionStatus returns the per-user like/favorite state of a prompt. A
// missing or soft-deleted prompt yields ErrPromptNotFound so callers cannot
// infer existence from the response.
func (s *MySQLPromptStore) GetInteractionStatus(id int, userID int) (InteractionStatus, error) {
	var published int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM prompts p JOIN users u ON u.id = p.user_id WHERE p.id = ? AND p.status = 1 AND u.status = 1`, id).Scan(&published); err != nil {
		return InteractionStatus{}, err
	}
	if published == 0 {
		return InteractionStatus{}, ErrPromptNotFound
	}

	status := InteractionStatus{}
	var liked int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM likes WHERE user_id = ? AND target_type = 'prompt' AND target_id = ?`, userID, id).Scan(&liked); err != nil {
		return InteractionStatus{}, err
	}
	status.Liked = liked > 0

	var favorited int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM favorites WHERE user_id = ? AND target_type = 'prompt' AND target_id = ?`, userID, id).Scan(&favorited); err != nil {
		return InteractionStatus{}, err
	}
	status.Favorited = favorited > 0

	return status, nil
}

func (s *MySQLPromptStore) RecordView(id int, userID int) (Prompt, bool, error) {
	// The status=1 guard, the detail-row write, and the counter increment all
	// run in one transaction so a crash can never leave a view counted without
	// its history row (or vice versa), and a soft-deleted prompt is never
	// counted.
	tx, err := s.db.Begin()
	if err != nil {
		return Prompt{}, false, err
	}
	defer tx.Rollback()

	var published int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM prompts p JOIN users u ON u.id = p.user_id WHERE p.id = ? AND p.status = 1 AND u.status = 1`, id).Scan(&published); err != nil {
		return Prompt{}, false, err
	}
	if published == 0 {
		return Prompt{}, false, ErrPromptNotFound
	}

	// Anonymous views (userID <= 0) increment the total views counter only and
	// never touch view_histories, because they cannot be attributed to a user.
	// Each anonymous view is an independent counter increment.
	if userID <= 0 {
		if _, err := tx.Exec(`UPDATE prompts SET views = views + 1 WHERE id = ? AND status = 1`, id); err != nil {
			return Prompt{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return Prompt{}, false, err
		}
		return s.reloadPrompt(id)
	}

	result, err := tx.Exec(`
		INSERT INTO view_histories (user_id, prompt_id, viewed_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE viewed_at = CURRENT_TIMESTAMP
	`, userID, id)
	if err != nil {
		return Prompt{}, false, err
	}

	// affected == 1 only for an inserted row (a new unique (user, prompt)).
	// A duplicate-key update affects 0 rows here because the column value is
	// unchanged, so a repeat view never double-counts. This is the idempotency
	// contract for logged-in views.
	affected, err := result.RowsAffected()
	if err != nil {
		return Prompt{}, false, err
	}
	applied := affected == 1
	if applied {
		if _, err := tx.Exec(`UPDATE prompts SET views = views + 1 WHERE id = ? AND status = 1`, id); err != nil {
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
		return Prompt{}, false, ErrPromptNotFound
	}
	return prompt, applied, nil
}

// reloadPrompt loads a published prompt for a view response.
func (s *MySQLPromptStore) reloadPrompt(id int) (Prompt, bool, error) {
	prompt, found, err := s.FindByID(id)
	if err != nil {
		return Prompt{}, false, err
	}
	if !found {
		return Prompt{}, false, ErrPromptNotFound
	}
	return prompt, true, nil
}

func (s *MySQLPromptStore) Report(id int, userID int, reason string, detail string) (Report, bool, error) {
	input := ReportPromptInput{
		PromptID: id,
		UserID:   userID,
		Reason:   reason,
		Detail:   detail,
	}
	if err := validateReportPromptInput(input); err != nil {
		return Report{}, false, err
	}

	if _, found, err := s.FindByID(id); err != nil {
		return Report{}, false, err
	} else if !found {
		return Report{}, false, ErrPromptNotFound
	}

	result, err := s.db.Exec(`
		INSERT IGNORE INTO reports (user_id, target_type, target_id, reason, detail, status)
		VALUES (?, 'prompt', ?, ?, ?, 'pending')
	`, userID, id, strings.TrimSpace(reason), strings.TrimSpace(detail))
	if err != nil {
		return Report{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Report{}, false, err
	}

	report, found, err := s.findPromptReport(userID, "prompt", id)
	if err != nil {
		return Report{}, false, err
	}
	if !found {
		return Report{}, false, ErrReportNotFound
	}

	return report, affected > 0, nil
}

func (s *MySQLPromptStore) ListUserFavorites(userID int) ([]Prompt, error) {
	return s.listUserEngagements("favorites", userID)
}

func (s *MySQLPromptStore) ListUserLikes(userID int) ([]Prompt, error) {
	return s.listUserEngagements("likes", userID)
}

func (s *MySQLPromptStore) ListUserHistory(userID int) ([]Prompt, error) {
	rows, err := s.db.Query(`
		SELECT
			p.id, p.title, p.description, p.cover, p.images, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM view_histories vh
		JOIN prompts p ON p.id = vh.prompt_id
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE vh.user_id = ? AND p.status = 1 AND u.status = 1
		GROUP BY p.id, vh.viewed_at
		ORDER BY vh.viewed_at DESC, p.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPromptRows(rows)
}

// ListUserHistoryPage returns one page of the requesting user's browsing
// history (newest view first) plus the total, excluding soft-deleted prompts
// (status = 1 join) and pushing pagination down to the database.
func (s *MySQLPromptStore) ListUserHistoryPage(userID, page, pageSize int) ([]Prompt, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}

	var total int
	if err := s.db.QueryRow(`
		SELECT COUNT(DISTINCT p.id)
		FROM view_histories vh
		JOIN prompts p ON p.id = vh.prompt_id
		JOIN users u ON u.id = p.user_id
		WHERE vh.user_id = ? AND p.status = 1 AND u.status = 1
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
		SELECT
			p.id, p.title, p.description, p.cover, p.images, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM view_histories vh
		JOIN prompts p ON p.id = vh.prompt_id
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE vh.user_id = ? AND p.status = 1 AND u.status = 1
		GROUP BY p.id
		ORDER BY MAX(vh.viewed_at) DESC, p.id DESC
		LIMIT ? OFFSET ?
	`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list, err := scanPromptRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *MySQLPromptStore) ListUserDrafts(userID int) ([]Prompt, error) {
	rows, err := s.db.Query(`
		SELECT
			p.id, p.title, p.description, p.cover, p.images, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM prompts p
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE p.user_id = ? AND p.status = 0
		GROUP BY p.id
		ORDER BY p.updated_at DESC, p.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPromptRows(rows)
}

func (s *MySQLPromptStore) listUserEngagements(table string, userID int) ([]Prompt, error) {
	if table != "favorites" && table != "likes" {
		return nil, errors.New("invalid engagement table")
	}

	rows, err := s.db.Query(`
		SELECT
			p.id, p.title, p.description, p.cover, p.images, p.content, p.system_prompt, p.model, p.params,
			p.category_id, c.name, p.user_id,
			u.username, u.avatar, u.email, u.bio, u.level, u.experience, u.status, u.created_at,
			p.views, p.likes, p.favorites, p.status, p.created_at, p.updated_at,
			COALESCE(GROUP_CONCAT(pt.tag ORDER BY pt.id SEPARATOR '||'), '')
		FROM `+table+` e
		JOIN prompts p ON p.id = e.target_id AND e.target_type = 'prompt'
		JOIN categories c ON c.id = p.category_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN prompt_tags pt ON pt.prompt_id = p.id
		WHERE e.user_id = ? AND p.status = 1 AND u.status = 1
		GROUP BY p.id, e.created_at
		ORDER BY e.created_at DESC, p.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPromptRows(rows)
}

func scanPromptRows(rows *sql.Rows) ([]Prompt, error) {
	list := make([]Prompt, 0)
	for rows.Next() {
		prompt, err := scanPrompt(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, prompt)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *MySQLPromptStore) applyEngagement(table string, counterColumn string, id int, userID int) (Prompt, bool, error) {
	// The status=1 guard, the detail-table insert, and the counter increment
	// share one transaction so a crash can never write a like/favorite row
	// without bumping the counter (or bump the counter without the row).
	tx, err := s.db.Begin()
	if err != nil {
		return Prompt{}, false, err
	}
	defer tx.Rollback()

	var published int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM prompts p JOIN users u ON u.id = p.user_id WHERE p.id = ? AND p.status = 1 AND u.status = 1`, id).Scan(&published); err != nil {
		return Prompt{}, false, err
	}
	if published == 0 {
		return Prompt{}, false, ErrPromptNotFound
	}

	result, err := tx.Exec(
		"INSERT IGNORE INTO "+table+" (user_id, target_type, target_id) VALUES (?, 'prompt', ?)",
		userID,
		id,
	)
	if err != nil {
		return Prompt{}, false, err
	}

	// The (user_id, target_type, target_id) unique constraint makes the insert
	// idempotent: a repeat action inserts nothing, so the counter increments
	// exactly once across any number of concurrent identical calls.
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
		return Prompt{}, false, ErrPromptNotFound
	}

	return prompt, applied, nil
}

func (s *MySQLPromptStore) findPromptReport(userID int, targetType string, targetID int) (Report, bool, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, target_type, target_id, reason, detail, status, created_at
		FROM reports
		WHERE user_id = ? AND target_type = ? AND target_id = ?
	`, userID, targetType, targetID)

	var (
		report    Report
		createdAt time.Time
	)
	if err := row.Scan(
		&report.ID,
		&report.UserID,
		&report.TargetType,
		&report.TargetID,
		&report.Reason,
		&report.Detail,
		&report.Status,
		&createdAt,
	); errors.Is(err, sql.ErrNoRows) {
		return Report{}, false, nil
	} else if err != nil {
		return Report{}, false, err
	}

	report.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return report, true, nil
}

func scanPrompt(scan func(dest ...any) error) (Prompt, error) {
	var (
		prompt          Prompt
		description     sql.NullString
		cover           sql.NullString
		imagesRaw       sql.NullString
		systemPrompt    sql.NullString
		paramsRaw       sql.NullString
		userAvatar      sql.NullString
		userBio         sql.NullString
		tagList         string
		userCreatedAt   time.Time
		promptCreatedAt time.Time
		promptUpdatedAt time.Time
	)

	err := scan(
		&prompt.ID,
		&prompt.Title,
		&description,
		&cover,
		&imagesRaw,
		&prompt.Content,
		&systemPrompt,
		&prompt.Model,
		&paramsRaw,
		&prompt.CategoryID,
		&prompt.CategoryName,
		&prompt.UserID,
		&prompt.User.Username,
		&userAvatar,
		&prompt.User.Email,
		&userBio,
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

	if description.Valid {
		prompt.Description = description.String
	}
	if cover.Valid {
		prompt.Cover = cover.String
	}
	if imagesRaw.Valid && strings.TrimSpace(imagesRaw.String) != "" {
		if err := json.Unmarshal([]byte(imagesRaw.String), &prompt.Images); err != nil {
			return Prompt{}, err
		}
	}
	if systemPrompt.Valid {
		prompt.SystemPrompt = systemPrompt.String
	}
	if userAvatar.Valid {
		prompt.User.Avatar = userAvatar.String
	}
	if userBio.Valid {
		prompt.User.Bio = userBio.String
	}

	if paramsRaw.Valid && strings.TrimSpace(paramsRaw.String) != "" {
		if err := json.Unmarshal([]byte(paramsRaw.String), &prompt.Params); err != nil {
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

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}
