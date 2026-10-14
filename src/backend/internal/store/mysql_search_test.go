package store

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"promptos-backend/internal/database"

	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLKeywordSearchFulltextParity verifies on a real MySQL that the
// FULLTEXT-backed keyword search keeps the legacy OR-LIKE semantics: big-text
// matches (ngram phrase), small-column matches (category/model/username/tag),
// single-rune fallback, and soft-deleted exclusion. Runs only with
// PROMPTOS_TEST_MYSQL_DSN (CI provides it).
func TestMySQLKeywordSearchFulltextParity(t *testing.T) {
	dsn := testMySQLDSN(t)
	base, databaseName := seedTestDSNParts(t, dsn)
	testName := databaseName + "_search_" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")

	admin, err := sql.Open("mysql", base+"mysql")
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + testName); err != nil {
		t.Fatalf("drop stale test database: %v", err)
	}
	if _, err := admin.Exec("CREATE DATABASE " + testName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS " + testName) }()

	db, err := sql.Open("mysql", base+testName+"?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	if err := database.RunMigrations(db, "../../sql/migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	users := NewMySQLUserStore(db)
	prompts := NewMySQLPromptStore(db)

	owner, err := users.Register("city_photographer", "search-owner@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("register owner: %v", err)
	}
	other, err := users.Register("painter", "search-other@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("register other: %v", err)
	}
	toUser := func(u AuthUser) User {
		return User{ID: u.ID, Username: u.Username, Email: u.Email, Status: u.Status}
	}

	newInput := func(user User, title, description, content string) CreatePromptInput {
		return CreatePromptInput{
			Title:       title,
			Description: description,
			Content:     content,
			Model:       "gpt-4o",
			CategoryID:  1,
			Tags:        []string{"search-it"},
			User:        user,
			Status:      1,
		}
	}

	titleHit, err := prompts.Create(newInput(toUser(owner), "夜间城市摄影指南", "urban night", "content"))
	if err != nil {
		t.Fatalf("create title hit: %v", err)
	}
	descHit, err := prompts.Create(newInput(toUser(other), "constant title", "一篇关于摄影后期的入门", "content"))
	if err != nil {
		t.Fatalf("create description hit: %v", err)
	}
	usernameHit, err := prompts.Create(newInput(toUser(owner), "constant title two", "unrelated", "content"))
	if err != nil {
		t.Fatalf("create username hit: %v", err)
	}
	draftHit, err := prompts.Create(newInput(toUser(other), "草稿摄影秘籍", "draft", "content"))
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := prompts.Delete(draftHit.ID, other.ID); err != nil {
		t.Fatalf("soft-delete draft: %v", err)
	}

	waitForFulltextIndex(t, db, "摄影", 2)

	// Big-text keyword: ngram phrase path must find title/description hits,
	// never the soft-deleted draft.
	ids := searchIDs(t, prompts, "摄影")
	if !ids[titleHit.ID] || !ids[descHit.ID] {
		t.Fatalf("big-text keyword misses: ids=%v want %d and %d", ids, titleHit.ID, descHit.ID)
	}
	if ids[draftHit.ID] {
		t.Fatalf("soft-deleted draft surfaced: %v", ids)
	}

	// Small-column keyword: username substring still matches.
	ids = searchIDs(t, prompts, "photographer")
	if !ids[usernameHit.ID] {
		t.Fatalf("username keyword misses: ids=%v want %d", ids, usernameHit.ID)
	}

	// Single-rune keyword: below ngram_token_size, legacy LIKE fallback.
	ids = searchIDs(t, prompts, "夜")
	if !ids[titleHit.ID] {
		t.Fatalf("single-rune fallback misses title: ids=%v want %d", ids, titleHit.ID)
	}

	// ASCII multi-word keyword with exact substring semantics.
	ascii, err := prompts.Create(newInput(toUser(other), "Code Review Helper", "ascii path", "content"))
	if err != nil {
		t.Fatalf("create ascii prompt: %v", err)
	}
	waitForFulltextIndex(t, db, "Code Review", 1)
	ids = searchIDs(t, prompts, "Code Review")
	if !ids[ascii.ID] {
		t.Fatalf("ascii phrase keyword misses: ids=%v want %d", ids, ascii.ID)
	}
	if ids[titleHit.ID] || ids[descHit.ID] || ids[usernameHit.ID] {
		t.Fatalf("ascii keyword over-matched: %v", ids)
	}
}

// searchIDs runs a keyword page query and returns the matched prompt ids.
func searchIDs(t *testing.T, prompts *MySQLPromptStore, keyword string) map[int]bool {
	t.Helper()
	page, _, err := prompts.QueryPage(PromptFilter{Keyword: keyword}, 1, 100)
	if err != nil {
		t.Fatalf("QueryPage(keyword=%q) error = %v", keyword, err)
	}
	ids := make(map[int]bool, len(page))
	for _, prompt := range page {
		ids[prompt.ID] = true
	}
	return ids
}

// waitForFulltextIndex polls until the InnoDB FULLTEXT cache exposes the
// seeded rows for the phrase, keeping the test deterministic.
func waitForFulltextIndex(t *testing.T, db *sql.DB, phrase string, wantAtLeast int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM prompts WHERE MATCH(title, description, content, system_prompt) AGAINST(? IN BOOLEAN MODE)`,
			`"`+phrase+`"`,
		).Scan(&n); err == nil && n >= wantAtLeast {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("fulltext index did not expose %q within timeout", phrase)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
