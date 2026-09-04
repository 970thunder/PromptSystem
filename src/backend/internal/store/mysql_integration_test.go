package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

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

// TestMySQLReportAndFollowTransactions verifies that target validation,
// idempotent detail writes, and derived follow counts observe one committed
// transaction even when identical requests race.
func TestMySQLReportAndFollowTransactions(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	suffix := time.Now().UnixNano()
	users := NewMySQLUserStore(db)
	follower, err := users.Register(
		fmt.Sprintf("tx_follower_%d", suffix),
		fmt.Sprintf("tx-follower-%d@example.com", suffix),
		"StrongPass123!",
	)
	if err != nil {
		t.Fatalf("register follower: %v", err)
	}
	target, err := users.Register(
		fmt.Sprintf("tx_target_%d", suffix),
		fmt.Sprintf("tx-target-%d@example.com", suffix),
		"StrongPass123!",
	)
	if err != nil {
		t.Fatalf("register target: %v", err)
	}

	prompts := NewMySQLPromptStore(db)
	prompt, err := prompts.Create(CreatePromptInput{
		Title:      fmt.Sprintf("transaction prompt %d", suffix),
		Content:    "transaction test",
		Model:      "gpt-4o",
		CategoryID: 1,
		Tags:       []string{"transaction"},
		User:       User{ID: target.ID, Username: target.Username, Status: 1},
		Status:     1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	const followWorkers = 8
	followErrs := make(chan error, followWorkers)
	var followWG sync.WaitGroup
	for i := 0; i < followWorkers; i++ {
		followWG.Add(1)
		go func() {
			defer followWG.Done()
			if _, _, err := users.Follow(follower.ID, target.ID); err != nil {
				followErrs <- err
			}
		}()
	}
	followWG.Wait()
	close(followErrs)
	for err := range followErrs {
		t.Fatalf("concurrent follow: %v", err)
	}
	status, err := users.FollowStatus(target.ID, follower.ID)
	if err != nil {
		t.Fatalf("follow status: %v", err)
	}
	if !status.Following || status.FollowerCount != 1 {
		t.Fatalf("follow status = %+v, want following=true and one follower", status)
	}

	if _, applied, err := users.Unfollow(follower.ID, target.ID); err != nil || !applied {
		t.Fatalf("unfollow: applied=%v err=%v, want applied", applied, err)
	}
	if _, applied, err := users.Unfollow(follower.ID, target.ID); err != nil || applied {
		t.Fatalf("duplicate unfollow: applied=%v err=%v, want no-op", applied, err)
	}

	const reportWorkers = 8
	reportErrs := make(chan error, reportWorkers)
	var reportWG sync.WaitGroup
	for i := 0; i < reportWorkers; i++ {
		reportWG.Add(1)
		go func() {
			defer reportWG.Done()
			if _, _, err := prompts.Report(prompt.ID, follower.ID, ReportReasonSpam, "transaction report"); err != nil {
				reportErrs <- err
			}
		}()
	}
	reportWG.Wait()
	close(reportErrs)
	for err := range reportErrs {
		t.Fatalf("concurrent prompt report: %v", err)
	}
	var reportCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reports WHERE user_id = ? AND target_type = 'prompt' AND target_id = ?`, follower.ID, prompt.ID).Scan(&reportCount); err != nil {
		t.Fatalf("count prompt reports: %v", err)
	}
	if reportCount != 1 {
		t.Fatalf("prompt report rows = %d, want 1", reportCount)
	}

	comments := NewMySQLCommentStore(db)
	comment, err := comments.Create(CreateCommentInput{
		TargetType: "prompt",
		TargetID:   prompt.ID,
		User:       User{ID: follower.ID, Username: follower.Username, Status: 1},
		Content:    "transaction comment",
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if _, _, err := comments.Report(ReportCommentInput{
		CommentID: comment.ID,
		UserID:    target.ID,
		Reason:    ReportReasonAbuse,
		Detail:    "comment report",
	}); err != nil {
		t.Fatalf("report comment: %v", err)
	}
	if _, applied, err := comments.Report(ReportCommentInput{
		CommentID: comment.ID,
		UserID:    target.ID,
		Reason:    ReportReasonNsfw,
		Detail:    "duplicate should reuse row",
	}); err != nil || applied {
		t.Fatalf("duplicate comment report: applied=%v err=%v, want no-op", applied, err)
	}

	deletedPrompt, err := prompts.Create(CreatePromptInput{
		Title:      fmt.Sprintf("deleted transaction prompt %d", suffix),
		Content:    "deleted",
		Model:      "gpt-4o",
		CategoryID: 1,
		Tags:       []string{"transaction"},
		User:       User{ID: target.ID, Username: target.Username, Status: 1},
		Status:     1,
	})
	if err != nil {
		t.Fatalf("create deleted prompt: %v", err)
	}
	if err := prompts.Delete(deletedPrompt.ID, target.ID); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}
	if _, _, err := prompts.Report(deletedPrompt.ID, follower.ID, ReportReasonOther, "must be rejected"); !errors.Is(err, ErrPromptNotFound) {
		t.Fatalf("report deleted prompt error = %v, want ErrPromptNotFound", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM reports WHERE user_id = ? AND target_type = 'prompt' AND target_id = ?`, follower.ID, deletedPrompt.ID).Scan(&reportCount); err != nil {
		t.Fatalf("count deleted prompt reports: %v", err)
	}
	if reportCount != 0 {
		t.Fatalf("deleted prompt report rows = %d, want 0", reportCount)
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

func TestMySQLPublicQueriesExcludeDisabledAuthors(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	users := NewMySQLUserStore(db)
	suffix := time.Now().UnixNano()
	prompts := NewMySQLPromptStore(db)
	before, err := prompts.HomeSummary()
	if err != nil {
		t.Fatalf("summary before create: %v", err)
	}
	user, err := users.Register(fmt.Sprintf("disabled_%d", suffix), fmt.Sprintf("disabled-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	prompt, err := prompts.Create(CreatePromptInput{
		Title:       fmt.Sprintf("disabled author %d", suffix),
		Description: "must stay private",
		Content:     "content",
		Model:       "gpt-4o",
		CategoryID:  1,
		Tags:        []string{"visibility"},
		User:        User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:      1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	if _, err := db.Exec("UPDATE users SET status = 0 WHERE id = ?", user.ID); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	results, total, err := prompts.QueryPage(PromptFilter{Keyword: prompt.Title}, 1, 10)
	if err != nil {
		t.Fatalf("query disabled author: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("disabled author's prompt returned in search: total=%d len=%d", total, len(results))
	}
	if _, found, err := prompts.FindByID(prompt.ID); err != nil {
		t.Fatalf("find disabled author prompt: %v", err)
	} else if found {
		t.Fatal("disabled author's prompt returned from public detail lookup")
	}
	after, err := prompts.HomeSummary()
	if err != nil {
		t.Fatalf("summary after disable: %v", err)
	}
	if after.PromptCount != before.PromptCount || after.CreatorCount != before.CreatorCount || after.TotalViews != before.TotalViews {
		t.Fatalf("disabled author affected public summary: before=%+v after=%+v", before, after)
	}
}

func TestMySQLDeleteAccountAnonymizesAndClearsPersonalRows(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	users := NewMySQLUserStore(db)
	suffix := time.Now().UnixNano()
	user, err := users.Register(fmt.Sprintf("delete_%d", suffix), fmt.Sprintf("delete-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	other, err := users.Register(fmt.Sprintf("delete_other_%d", suffix), fmt.Sprintf("delete-other-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register other: %v", err)
	}
	prompts := NewMySQLPromptStore(db)
	prompt, err := prompts.Create(CreatePromptInput{
		Title:      fmt.Sprintf("delete account prompt %d", suffix),
		Content:    "retained content",
		Model:      "gpt-4o",
		CategoryID: 1,
		Tags:       []string{"privacy"},
		User:       User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:     1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM likes WHERE user_id = ? OR target_id = ?", user.ID, prompt.ID)
		_, _ = db.Exec("DELETE FROM favorites WHERE user_id = ? OR target_id = ?", user.ID, prompt.ID)
		_, _ = db.Exec("DELETE FROM view_histories WHERE user_id = ? OR prompt_id = ?", user.ID, prompt.ID)
		_, _ = db.Exec("DELETE FROM follows WHERE follower_id IN (?, ?) OR following_id IN (?, ?)", user.ID, other.ID, user.ID, other.ID)
		_, _ = db.Exec("DELETE FROM prompt_tags WHERE prompt_id = ?", prompt.ID)
		_, _ = db.Exec("DELETE FROM prompts WHERE id = ?", prompt.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id IN (?, ?)", user.ID, other.ID)
	}()

	if _, _, err := prompts.Like(prompt.ID, user.ID); err != nil {
		t.Fatalf("like: %v", err)
	}
	if _, _, err := prompts.Favorite(prompt.ID, user.ID); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if _, _, err := prompts.RecordView(prompt.ID, user.ID); err != nil {
		t.Fatalf("view: %v", err)
	}
	if _, _, err := users.Follow(other.ID, user.ID); err != nil {
		t.Fatalf("follow: %v", err)
	}
	beforeVersion := user.SessionVer
	if err := users.DeleteAccount(user.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	deleted, found := users.FindByID(user.ID)
	if !found || deleted.Status != 0 || deleted.PasswordHash != "" || deleted.GitHubID != 0 {
		t.Fatalf("account was not scrubbed: found=%v user=%+v", found, deleted)
	}
	if deleted.Username != fmt.Sprintf("deleted-user-%d", user.ID) || deleted.Email != fmt.Sprintf("deleted+%d@invalid.promptos.local", user.ID) {
		t.Fatalf("unexpected anonymized identity: %+v", deleted)
	}
	if deleted.SessionVer != beforeVersion+1 {
		t.Fatalf("session version = %d, want %d", deleted.SessionVer, beforeVersion+1)
	}
	if _, err := users.Authenticate(user.Email, "StrongPass123!"); err == nil {
		t.Fatal("disabled account must not authenticate")
	}
	var likes, favorites, history, follows int
	if err := db.QueryRow("SELECT likes FROM prompts WHERE id = ?", prompt.ID).Scan(&likes); err != nil {
		t.Fatalf("prompt likes: %v", err)
	}
	if err := db.QueryRow("SELECT favorites FROM prompts WHERE id = ?", prompt.ID).Scan(&favorites); err != nil {
		t.Fatalf("prompt favorites: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM view_histories WHERE user_id = ?", user.ID).Scan(&history); err != nil {
		t.Fatalf("history rows: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM follows WHERE follower_id = ? OR following_id = ?", user.ID, user.ID).Scan(&follows); err != nil {
		t.Fatalf("follow rows: %v", err)
	}
	if likes != 0 || favorites != 0 || history != 0 || follows != 0 {
		t.Fatalf("personal rows remain: likes=%d favorites=%d history=%d follows=%d", likes, favorites, history, follows)
	}
	if _, found, err := prompts.FindByID(prompt.ID); err != nil {
		t.Fatalf("find retained prompt: %v", err)
	} else if found {
		t.Fatal("disabled author's prompt must not be publicly visible")
	}
	if err := users.DeleteAccount(user.ID); err != nil {
		t.Fatalf("repeated DeleteAccount should be idempotent: %v", err)
	}
}

func TestMySQLCommentPopularSortsBeforePagination(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	users := NewMySQLUserStore(db)
	suffix := time.Now().UnixNano()
	user, err := users.Register(fmt.Sprintf("comment_sort_%d", suffix), fmt.Sprintf("comment-sort-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	prompts := NewMySQLPromptStore(db)
	prompt, err := prompts.Create(CreatePromptInput{
		Title:      fmt.Sprintf("comment sort prompt %d", suffix),
		Content:    "content",
		Model:      "gpt-4o",
		CategoryID: 1,
		Tags:       []string{"comments"},
		User:       User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:     1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	comments := NewMySQLCommentStore(db)
	created := make([]Comment, 0, 3)
	for _, content := range []string{"low", "high", "middle"} {
		comment, err := comments.Create(CreateCommentInput{
			TargetType: "prompt",
			TargetID:   prompt.ID,
			User:       User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
			Content:    content,
		})
		if err != nil {
			t.Fatalf("create comment %q: %v", content, err)
		}
		created = append(created, comment)
	}
	if _, err := db.Exec(`UPDATE comments SET likes = CASE id WHEN ? THEN 1 WHEN ? THEN 7 WHEN ? THEN 3 END WHERE id IN (?, ?, ?)`,
		created[0].ID, created[1].ID, created[2].ID, created[0].ID, created[1].ID, created[2].ID); err != nil {
		t.Fatalf("set comment likes: %v", err)
	}

	page, total, err := comments.ListByTargetPage(CommentFilter{TargetType: "prompt", TargetID: prompt.ID, SortBy: "popular"}, 1, 1)
	if err != nil {
		t.Fatalf("list popular comments: %v", err)
	}
	if total != 3 || len(page) != 1 {
		t.Fatalf("popular page = len %d total %d, want len 1 total 3", len(page), total)
	}
	if page[0].ID != created[1].ID || page[0].Likes != 7 {
		t.Fatalf("popular page returned %+v, want highest-liked comment %d", page[0], created[1].ID)
	}

	var plan string
	if err := db.QueryRow(`EXPLAIN FORMAT=JSON
		SELECT c.id
		FROM comments c
		WHERE c.target_type = 'prompt' AND c.target_id = ? AND c.parent_id IS NULL
		ORDER BY c.likes DESC, c.created_at DESC, c.id DESC
		LIMIT 1`, prompt.ID).Scan(&plan); err != nil {
		t.Fatalf("explain popular comment query: %v", err)
	}
	if !strings.Contains(plan, "idx_target_parent_likes") {
		t.Fatalf("popular comment plan does not expose idx_target_parent_likes: %s", plan)
	}
}

func TestMySQLPolymorphicIntegrityAuditDetectsUnsupportedAndOrphanRows(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	users := NewMySQLUserStore(db)
	suffix := time.Now().UnixNano()
	user, err := users.Register(fmt.Sprintf("integrity_%d", suffix), fmt.Sprintf("integrity-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	missingID := int(suffix % 2000000000)
	if _, err := db.Exec(`INSERT INTO comments (target_type, target_id, user_id, content) VALUES ('prompt', ?, ?, 'orphan')`, missingID, user.ID); err != nil {
		t.Fatalf("insert orphan comment: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO likes (user_id, target_type, target_id) VALUES (?, 'unknown', ?)`, user.ID, missingID); err != nil {
		t.Fatalf("insert unsupported like: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO favorites (user_id, target_type, target_id) VALUES (?, 'prompt', ?)`, user.ID, missingID); err != nil {
		t.Fatalf("insert orphan favorite: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO reports (user_id, target_type, target_id, reason) VALUES (?, 'unknown', ?, 'other')`, user.ID, missingID); err != nil {
		t.Fatalf("insert unsupported report: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM comments WHERE user_id = ? AND target_id = ?", user.ID, missingID)
		_, _ = db.Exec("DELETE FROM likes WHERE user_id = ? AND target_id = ?", user.ID, missingID)
		_, _ = db.Exec("DELETE FROM favorites WHERE user_id = ? AND target_id = ?", user.ID, missingID)
		_, _ = db.Exec("DELETE FROM reports WHERE user_id = ? AND target_id = ?", user.ID, missingID)
	})

	report, err := AuditMySQLPolymorphicIntegrity(db)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if report.OrphanComments < 1 || report.OrphanFavorites < 1 || report.UnsupportedLikes < 1 || report.UnsupportedReports < 1 {
		t.Fatalf("audit report = %+v, want each injected violation", report)
	}
	if report.Total() < 4 {
		t.Fatalf("audit total = %d, want at least 4", report.Total())
	}
}

func TestMySQLPromptCounterAuditDetectsDrift(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	users := NewMySQLUserStore(db)
	suffix := time.Now().UnixNano()
	user, err := users.Register(fmt.Sprintf("counter_%d", suffix), fmt.Sprintf("counter-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	prompts := NewMySQLPromptStore(db)
	prompt, err := prompts.Create(CreatePromptInput{
		Title:      fmt.Sprintf("counter audit prompt %d", suffix),
		Content:    "content",
		Model:      "gpt-4o",
		CategoryID: 1,
		Tags:       []string{"counter"},
		User:       User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:     1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	if _, _, err := prompts.Like(prompt.ID, user.ID); err != nil {
		t.Fatalf("like: %v", err)
	}
	if _, _, err := prompts.RecordView(prompt.ID, 0); err != nil {
		t.Fatalf("anonymous view: %v", err)
	}
	if _, _, err := prompts.RecordView(prompt.ID, user.ID); err != nil {
		t.Fatalf("logged-in view: %v", err)
	}
	cleanReport, err := AuditMySQLPromptCounters(db)
	if err != nil {
		t.Fatalf("clean counter audit: %v", err)
	}
	if cleanReport.Total() != 0 {
		t.Fatalf("clean counter audit report = %+v, want no drift", cleanReport)
	}
	if _, err := db.Exec("UPDATE prompts SET likes = 0 WHERE id = ?", prompt.ID); err != nil {
		t.Fatalf("inject counter drift: %v", err)
	}
	report, err := AuditMySQLPromptCounters(db)
	if err != nil {
		t.Fatalf("counter audit: %v", err)
	}
	if report.LikeDrift < 1 || report.Total() < 1 {
		t.Fatalf("counter audit report = %+v, want like drift", report)
	}
}

// TestMySQLModerationSetPromptStatusWritesAudit is a regression test for the
// S-14 production drill: SetPromptStatus used to UPDATE a non-existent
// `prompt` table, so admin takedowns failed with a 500 and no audit event was
// appended. It also verifies the hash chain links consecutive events.
func TestMySQLModerationSetPromptStatusWritesAudit(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)

	suffix := time.Now().UnixNano()
	users := NewMySQLUserStore(db)
	author, err := users.Register(fmt.Sprintf("it_mod_author_%d", suffix), fmt.Sprintf("it-mod-author-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register author: %v", err)
	}
	admin, err := users.Register(fmt.Sprintf("it_mod_admin_%d", suffix), fmt.Sprintf("it-mod-admin-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}

	prompts := NewMySQLPromptStore(db)
	prompt, err := prompts.Create(CreatePromptInput{
		Title:       "MySQL moderation regression",
		Description: "integration",
		Content:     "content",
		Model:       "gpt-4o",
		CategoryID:  1,
		Tags:        []string{"mysql-it"},
		User:        User{ID: author.ID, Username: author.Username, Email: author.Email, Status: 1},
		Status:      1,
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	moderation := NewMySQLModerationStore(db)
	if _, err := db.Exec(`INSERT INTO user_roles (user_id, role) VALUES (?, 'admin')`, admin.ID); err != nil {
		t.Fatalf("grant admin role: %v", err)
	}

	if err := moderation.SetPromptStatus(prompt.ID, admin.ID, -1, "integration takedown"); err != nil {
		t.Fatalf("SetPromptStatus takedown: %v", err)
	}

	var status int
	if err := db.QueryRow(`SELECT status FROM prompts WHERE id = ?`, prompt.ID).Scan(&status); err != nil {
		t.Fatalf("read prompt status: %v", err)
	}
	if status != -1 {
		t.Fatalf("expected prompt status -1 after takedown, got %d", status)
	}

	events, total, err := moderation.ListAuditEvents(1, 10)
	if err != nil {
		t.Fatalf("ListAuditEvents: %v", err)
	}
	if total < 1 || len(events) == 0 {
		t.Fatal("expected at least one audit event after takedown")
	}
	latest := events[0]
	if latest.Action != "content.status" {
		t.Fatalf("expected action content.status, got %q", latest.Action)
	}

	if err := moderation.SetPromptStatus(prompt.ID, admin.ID, 1, "integration restore"); err != nil {
		t.Fatalf("SetPromptStatus restore: %v", err)
	}
	events, _, err = moderation.ListAuditEvents(1, 10)
	if err != nil {
		t.Fatalf("ListAuditEvents after restore: %v", err)
	}
	if len(events) < 2 {
		t.Fatal("expected two audit events after takedown and restore")
	}
	if events[0].PrevHash != events[1].EventHash {
		t.Fatal("expected audit hash chain to link restore event to takedown event")
	}
}
