package store

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// newTestPrompt creates a fresh published prompt through the in-memory store and
// returns it so each test gets an isolated subject instead of mutating the seed
// prompts (IDs 101..106). The returned Prompt carries the newly allocated ID.
func newTestPrompt(t *testing.T, owner int) Prompt {
	t.Helper()
	p, err := CreatePrompt(CreatePromptInput{
		Title:       "Test Prompt",
		Description: "Test description",
		Cover:       "https://example.com/cover.png",
		Content:     "Test content",
		Model:       "gpt-4",
		CategoryID:  1,
		Tags:        []string{"test"},
		User:        User{ID: owner, Username: "u", Email: "u@example.com", Status: 1},
		Status:      1,
	})
	if err != nil {
		t.Fatalf("CreatePrompt() error = %v", err)
	}
	return p
}

// runConcurrent runs the given action concurrently n times and returns the
// number of times it reported applied==true.
func runConcurrent(n int, action func(i int, applied *bool)) int {
	var wg sync.WaitGroup
	var mu sync.Mutex
	appliedCount := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var applied bool
			action(i, &applied)
			if applied {
				mu.Lock()
				appliedCount++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return appliedCount
}

func TestConcurrentIdenticalLikesCountOnce(t *testing.T) {
	p := newTestPrompt(t, 9001)
	const user = 8001
	store := NewMemoryPromptStore()

	applied := runConcurrent(20, func(_ int, out *bool) {
		_, a, err := store.Like(p.ID, user)
		if err != nil {
			t.Errorf("Like() error = %v", err)
		}
		*out = a
	})

	if applied != 1 {
		t.Fatalf("expected exactly 1 applied like out of 20, got %d", applied)
	}
	after, _, err := store.FindByID(p.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if after.Likes != p.Likes+1 {
		t.Fatalf("expected Likes to increase by exactly 1 (base %d), got %d", p.Likes, after.Likes)
	}
	if len(promptLikes[p.ID]) != 1 {
		t.Fatalf("expected exactly 1 unique like row for prompt %d, got %d", p.ID, len(promptLikes[p.ID]))
	}
}

func TestConcurrentIdenticalFavoritesCountOnce(t *testing.T) {
	p := newTestPrompt(t, 9002)
	const user = 8002
	store := NewMemoryPromptStore()

	applied := runConcurrent(20, func(_ int, out *bool) {
		_, a, err := store.Favorite(p.ID, user)
		if err != nil {
			t.Errorf("Favorite() error = %v", err)
		}
		*out = a
	})

	if applied != 1 {
		t.Fatalf("expected exactly 1 applied favorite out of 20, got %d", applied)
	}
	after, _, err := store.FindByID(p.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if after.Favorites != p.Favorites+1 {
		t.Fatalf("expected Favorites to increase by exactly 1 (base %d), got %d", p.Favorites, after.Favorites)
	}
	if len(promptFavorites[p.ID]) != 1 {
		t.Fatalf("expected exactly 1 unique favorite row for prompt %d, got %d", p.ID, len(promptFavorites[p.ID]))
	}
}

func TestConcurrentIdenticalLikesOnCommentCountOnce(t *testing.T) {
	// Reuse the seed comment (ID 1001) as the target.
	const commentID = 1001
	const user = 8003
	store := NewMemoryCommentStore()

	applied := runConcurrent(20, func(_ int, out *bool) {
		_, a, err := store.Like(commentID, user)
		if err != nil {
			t.Errorf("Comment Like() error = %v", err)
		}
		*out = a
	})

	if applied != 1 {
		t.Fatalf("expected exactly 1 applied comment like out of 20, got %d", applied)
	}
	if len(commentLikes[commentID]) != 1 {
		t.Fatalf("expected exactly 1 unique comment like row for comment %d, got %d", commentID, len(commentLikes[commentID]))
	}
}

func TestConcurrentIdenticalLoggedInViewsCountOnce(t *testing.T) {
	p := newTestPrompt(t, 9003)
	const user = 8004
	store := NewMemoryPromptStore()

	applied := runConcurrent(20, func(_ int, out *bool) {
		_, a, err := store.RecordView(p.ID, user)
		if err != nil {
			t.Errorf("RecordView() error = %v", err)
		}
		*out = a
	})

	if applied != 1 {
		t.Fatalf("expected exactly 1 applied view out of 20, got %d", applied)
	}
	after, _, err := store.FindByID(p.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if after.Views != p.Views+1 {
		t.Fatalf("expected Views to increase by exactly 1 (base %d), got %d", p.Views, after.Views)
	}
	if len(promptViewHistory[user]) != 1 {
		t.Fatalf("expected exactly 1 history row for user %d, got %d", user, len(promptViewHistory[user]))
	}
}

func TestConcurrentIdenticalReportsCountOnce(t *testing.T) {
	p := newTestPrompt(t, 9004)
	const user = 8005
	store := NewMemoryPromptStore()
	before := len(promptReports)

	applied := runConcurrent(20, func(_ int, out *bool) {
		_, a, err := store.Report(p.ID, user, ReportReasonSpam, "test")
		if err != nil {
			t.Errorf("Report() error = %v", err)
		}
		*out = a
	})

	if applied != 1 {
		t.Fatalf("expected exactly 1 applied report out of 20, got %d", applied)
	}
	if len(promptReports) != before+1 {
		t.Fatalf("expected exactly 1 new report row, got %d new", len(promptReports)-before)
	}
}

func TestAnonymousViewIncrementsCounterButWritesNoHistory(t *testing.T) {
	p := newTestPrompt(t, 9005)
	store := NewMemoryPromptStore()
	if _, ok := promptViewHistory[0]; ok {
		t.Fatal("test precondition: anonymous user 0 should have no history initially")
	}

	applied := runConcurrent(5, func(_ int, out *bool) {
		_, a, err := store.RecordView(p.ID, 0)
		if err != nil {
			t.Errorf("RecordView(anonymous) error = %v", err)
		}
		*out = a
	})

	if applied != 5 {
		t.Fatalf("expected every anonymous view to apply (count toward total), got %d/5", applied)
	}
	after, _, err := store.FindByID(p.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if after.Views != p.Views+5 {
		t.Fatalf("expected Views to increase by 5 anonymous views, got %d", after.Views)
	}
	if h, ok := promptViewHistory[0]; ok && len(h) != 0 {
		t.Fatalf("anonymous views must not write history, got %d rows for user 0", len(h))
	}
	history := ListUserHistoryPrompts(0)
	if len(history) != 0 {
		t.Fatalf("anonymous user must have empty history, got %d entries", len(history))
	}
}

func TestLoggedInRepeatViewUpdatesHistoryNotCount(t *testing.T) {
	p := newTestPrompt(t, 9006)
	const user = 8006
	store := NewMemoryPromptStore()

	first, applied, err := store.RecordView(p.ID, user)
	if err != nil {
		t.Fatalf("RecordView() first error = %v", err)
	}
	if !applied {
		t.Fatal("expected first view to apply")
	}

	second, applied, err := store.RecordView(p.ID, user)
	if err != nil {
		t.Fatalf("RecordView() second error = %v", err)
	}
	if applied {
		t.Fatal("expected repeat view not to apply (no double count)")
	}
	if second.Views != first.Views {
		t.Fatalf("expected repeat view not to change counter, got %d -> %d", first.Views, second.Views)
	}
	if len(promptViewHistory[user]) != 1 {
		t.Fatalf("expected exactly 1 history row for repeat views, got %d", len(promptViewHistory[user]))
	}
}

func TestListUserHistoryPageFiltersDeletedAndPaginates(t *testing.T) {
	const user = 8007
	const owner = 9007
	store := NewMemoryPromptStore()

	p1 := newTestPrompt(t, owner)
	p2 := newTestPrompt(t, owner)
	p3 := newTestPrompt(t, owner)

	if _, _, err := store.RecordView(p1.ID, user); err != nil {
		t.Fatalf("RecordView p1 error = %v", err)
	}
	if _, _, err := store.RecordView(p2.ID, user); err != nil {
		t.Fatalf("RecordView p2 error = %v", err)
	}
	if _, _, err := store.RecordView(p3.ID, user); err != nil {
		t.Fatalf("RecordView p3 error = %v", err)
	}

	if err := store.Delete(p3.ID, owner); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	page, total, err := store.ListUserHistoryPage(user, 1, 10)
	if err != nil {
		t.Fatalf("ListUserHistoryPage() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 after deleting one prompt, got %d", total)
	}
	for _, p := range page {
		if p.ID == p3.ID {
			t.Fatalf("soft-deleted prompt %d leaked into history", p3.ID)
		}
	}

	// Pagination: 2 visible items, page 1 size 1 -> 1 item, page 2 -> 1 item.
	page1, total1, err := store.ListUserHistoryPage(user, 1, 1)
	if err != nil {
		t.Fatalf("ListUserHistoryPage(page=1) error = %v", err)
	}
	if total1 != 2 || len(page1) != 1 {
		t.Fatalf("expected total 2 and page 1 len 1, got total %d len %d", total1, len(page1))
	}
	page2, _, err := store.ListUserHistoryPage(user, 2, 1)
	if err != nil {
		t.Fatalf("ListUserHistoryPage(page=2) error = %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected page 2 len 1, got %d", len(page2))
	}
}

func TestInvalidReportReasonRejected(t *testing.T) {
	p := newTestPrompt(t, 9008)
	const user = 8008
	store := NewMemoryPromptStore()
	before := len(promptReports)

	if _, _, err := store.Report(p.ID, user, "not-a-valid-reason", "detail"); !errors.Is(err, ErrInvalidReportReason) {
		t.Fatalf("expected ErrInvalidReportReason, got %v", err)
	}
	if len(promptReports) != before {
		t.Fatalf("failed report must not create a row, got %d new", len(promptReports)-before)
	}

	if _, _, err := store.Report(p.ID, user, ReportReasonSpam, string(make([]rune, MaxReportDetailRunes+1))); err == nil {
		t.Fatal("expected detail longer than MaxReportDetailRunes to be rejected")
	}
}

func TestDuplicateReportDoesNotDoubleCount(t *testing.T) {
	p := newTestPrompt(t, 9009)
	const user = 8009
	store := NewMemoryPromptStore()

	first, applied, err := store.Report(p.ID, user, ReportReasonAbuse, "first")
	if err != nil {
		t.Fatalf("Report() first error = %v", err)
	}
	if !applied {
		t.Fatal("expected first report to apply")
	}

	second, applied, err := store.Report(p.ID, user, ReportReasonNsfw, "second")
	if err != nil {
		t.Fatalf("Report() duplicate error = %v", err)
	}
	if applied {
		t.Fatal("expected duplicate report not to apply")
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate report to reuse id %d, got %d", first.ID, second.ID)
	}
}

func TestReportOnDeletedPromptReturnsNotFound(t *testing.T) {
	const owner = 9010
	const user = 8010
	store := NewMemoryPromptStore()
	p := newTestPrompt(t, owner)

	if err := store.Delete(p.ID, owner); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, _, err := store.Report(p.ID, user, ReportReasonOther, "x"); !errors.Is(err, ErrPromptNotFound) {
		t.Fatalf("expected ErrPromptNotFound for report on deleted prompt, got %v", err)
	}
	if _, _, err := store.Like(p.ID, user); !errors.Is(err, ErrPromptNotFound) {
		t.Fatalf("expected ErrPromptNotFound for like on deleted prompt, got %v", err)
	}
	if _, _, err := store.RecordView(p.ID, user); !errors.Is(err, ErrPromptNotFound) {
		t.Fatalf("expected ErrPromptNotFound for view on deleted prompt, got %v", err)
	}

	// A failed interaction must leave no half-write: no like row, no view row,
	// no report row for the deleted target.
	if len(promptLikes[p.ID]) != 0 {
		t.Fatalf("failed like must not leave a like row on deleted prompt, got %d", len(promptLikes[p.ID]))
	}
	if len(promptViewHistory[user]) != 0 {
		t.Fatalf("failed view must not leave a history row for deleted prompt, got %d", len(promptViewHistory[user]))
	}
	key := fmt.Sprintf("prompt:%d:%d", user, p.ID)
	if r, exists := promptReports[key]; exists && r.ID != 0 {
		t.Fatalf("failed report must not leave a report row for deleted prompt: %+v", r)
	}
}

func TestCommentReportInvalidReasonRejected(t *testing.T) {
	const commentID = 1001
	const user = 8011
	store := NewMemoryCommentStore()
	before := len(commentReports)

	if _, _, err := store.Report(ReportCommentInput{CommentID: commentID, UserID: user, Reason: "bogus", Detail: "x"}); !errors.Is(err, ErrInvalidReportReason) {
		t.Fatalf("expected ErrInvalidReportReason, got %v", err)
	}
	if len(commentReports) != before {
		t.Fatalf("failed comment report must not create a row, got %d new", len(commentReports)-before)
	}
}

func TestCommentReportOnMissingCommentReturnsNotFound(t *testing.T) {
	const user = 8012
	store := NewMemoryCommentStore()
	if _, _, err := store.Report(ReportCommentInput{CommentID: 999999, UserID: user, Reason: ReportReasonSpam, Detail: "x"}); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}
