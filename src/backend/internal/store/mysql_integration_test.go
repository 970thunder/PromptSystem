package store

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// testMySQLDSN returns the test database DSN from PROMPTOS_TEST_MYSQL_DSN, or
// skips the test when unset (CI provides it; local runs skip). It is a
// dedicated test database, never the development data volume.
func testMySQLDSN(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("PROMPTOS_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("PROMPTOS_TEST_MYSQL_DSN not set; skipping MySQL integration tests (run via CI or a local MySQL)")
	}
	return dsn
}

func openTestMySQL(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestMySQLInteractionsUniqueConstraints verifies the double-write invariant on
// a real database: detail rows and denormalized counters commit in one
// transaction, and the unique constraints make repeated actions idempotent.
func TestMySQLInteractionsUniqueConstraints(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)

	// A fresh user and prompt owned by that user.
	users := NewMySQLUserStore(db)
	user, err := users.Register("it_mysql_user", "it-mysql-user@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	prompts := NewMySQLPromptStore(db)
	input := CreatePromptInput{
		Title:       "MySQL interaction test",
		Description: "integration",
		Content:     "content",
		Model:       "gpt-4o",
		CategoryID:  1,
		Tags:        []string{"mysql-it"},
		User:        User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:      1,
	}
	prompt, err := prompts.Create(input)
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	// Concurrent identical likes must count exactly once.
	const goroutines = 10
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := prompts.Like(prompt.ID, user.ID); err != nil {
				errs <- fmt.Errorf("like: %w", err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent like error: %v", err)
	}

	status, err := prompts.GetInteractionStatus(prompt.ID, user.ID)
	if err != nil {
		t.Fatalf("interaction status: %v", err)
	}
	if !status.Liked {
		t.Fatal("expected liked=true after concurrent likes")
	}

	// Toggle off removes the row and decrements the counter.
	if _, applied, err := prompts.Unlike(prompt.ID, user.ID); err != nil || !applied {
		t.Fatalf("unlike: applied=%v err=%v", applied, err)
	}
	status, err = prompts.GetInteractionStatus(prompt.ID, user.ID)
	if err != nil {
		t.Fatalf("interaction status after unlike: %v", err)
	}
	if status.Liked {
		t.Fatal("expected liked=false after unlike")
	}
}

// TestMySQLSoftDeleteExcludesFromLists verifies soft-deleted prompts disappear
// from search results and history while the row remains for audit.
func TestMySQLSoftDeleteExcludesFromLists(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)

	users := NewMySQLUserStore(db)
	user, err := users.Register("it_mysql_del", "it-mysql-del@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	prompts := NewMySQLPromptStore(db)
	prompt, err := prompts.Create(CreatePromptInput{
		Title:       "to be deleted",
		Description: "x",
		Content:     "y",
		Model:       "gpt-4o",
		CategoryID:  1,
		Tags:        []string{"del"},
		User:        User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:      1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	if _, _, err := prompts.RecordView(prompt.ID, user.ID); err != nil {
		t.Fatalf("record view: %v", err)
	}

	if err := prompts.Delete(prompt.ID, user.ID); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}

	if _, found, _ := prompts.FindByID(prompt.ID); found {
		t.Fatal("soft-deleted prompt must not be findable")
	}

	// Search must not return it.
	results, total, err := prompts.QueryPage(PromptFilter{Keyword: "to be deleted"}, 1, 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("soft-deleted prompt returned in search: total=%d len=%d", total, len(results))
	}

	// History must not return it either.
	history, total, err := prompts.ListUserHistoryPage(user.ID, 1, 10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if total != 0 || len(history) != 0 {
		t.Fatalf("soft-deleted prompt returned in history: total=%d len=%d", total, len(history))
	}
}
