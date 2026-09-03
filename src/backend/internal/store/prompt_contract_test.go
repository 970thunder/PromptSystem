package store

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// runPromptManagerContract exercises the behavior that every PromptManager
// implementation must share. The same assertions run against memory and real
// MySQL stores so development fallback cannot silently drift from production.
func runPromptManagerContract(t *testing.T, manager PromptManager, owner User, viewerID int) {
	t.Helper()

	input := CreatePromptInput{
		Title:       fmt.Sprintf("contract prompt %d", time.Now().UnixNano()),
		Description: "contract description",
		Content:     "contract content",
		Model:       "gpt-4o",
		CategoryID:  1,
		Tags:        []string{" Contract ", "contract"},
		User:        owner,
		Status:      1,
	}
	prompt, err := manager.Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(prompt.Tags) != 1 || prompt.Tags[0] != "contract" {
		t.Fatalf("Create() tags = %#v, want [contract]", prompt.Tags)
	}

	page, total, err := manager.QueryPage(PromptFilter{Keyword: prompt.Title}, 1, 1)
	if err != nil {
		t.Fatalf("QueryPage() error = %v", err)
	}
	if total != 1 || len(page) != 1 || page[0].ID != prompt.ID {
		t.Fatalf("QueryPage() = len %d total %d page=%+v, want one created prompt", len(page), total, page)
	}
	if found, ok, err := manager.FindByID(prompt.ID); err != nil || !ok || found.ID != prompt.ID {
		t.Fatalf("FindByID() = %+v found=%v err=%v", found, ok, err)
	}

	liked, applied, err := manager.Like(prompt.ID, viewerID)
	if err != nil || !applied || liked.Likes != prompt.Likes+1 {
		t.Fatalf("Like() = likes=%d applied=%v err=%v", liked.Likes, applied, err)
	}
	if _, applied, err := manager.Like(prompt.ID, viewerID); err != nil || applied {
		t.Fatalf("duplicate Like() = applied=%v err=%v, want no-op", applied, err)
	}
	status, err := manager.GetInteractionStatus(prompt.ID, viewerID)
	if err != nil || !status.Liked {
		t.Fatalf("GetInteractionStatus() = %+v err=%v, want liked", status, err)
	}
	if _, applied, err := manager.Unlike(prompt.ID, viewerID); err != nil || !applied {
		t.Fatalf("Unlike() = applied=%v err=%v, want applied", applied, err)
	}

	if _, applied, err := manager.Favorite(prompt.ID, viewerID); err != nil || !applied {
		t.Fatalf("Favorite() = applied=%v err=%v, want applied", applied, err)
	}
	status, err = manager.GetInteractionStatus(prompt.ID, viewerID)
	if err != nil || !status.Favorited {
		t.Fatalf("GetInteractionStatus() = %+v err=%v, want favorited", status, err)
	}
	if _, applied, err := manager.Unfavorite(prompt.ID, viewerID); err != nil || !applied {
		t.Fatalf("Unfavorite() = applied=%v err=%v, want applied", applied, err)
	}

	if _, applied, err := manager.RecordView(prompt.ID, viewerID); err != nil || !applied {
		t.Fatalf("RecordView() = applied=%v err=%v, want first view applied", applied, err)
	}
	if _, applied, err := manager.RecordView(prompt.ID, viewerID); err != nil || applied {
		t.Fatalf("repeat RecordView() = applied=%v err=%v, want no-op", applied, err)
	}
	history, total, err := manager.ListUserHistoryPage(viewerID, 1, 10)
	if err != nil || total < 1 || len(history) < 1 {
		t.Fatalf("ListUserHistoryPage() = len %d total %d err=%v, want viewed prompt", len(history), total, err)
	}

	if _, err := manager.Update(prompt.ID, viewerID, input); !errors.Is(err, ErrPromptForbidden) {
		t.Fatalf("Update() as non-owner error = %v, want ErrPromptForbidden", err)
	}
	if err := manager.Delete(prompt.ID, owner.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, found, err := manager.FindByID(prompt.ID); err != nil || found {
		t.Fatalf("FindByID() after delete = found=%v err=%v, want hidden", found, err)
	}
}

func TestPromptManagerContractMemory(t *testing.T) {
	runPromptManagerContract(t, NewMemoryPromptStore(), User{
		ID:       98001,
		Username: "contract-memory-owner",
		Status:   1,
	}, 98002)
}

func TestPromptManagerContractMySQL(t *testing.T) {
	dsn := testMySQLDSN(t)
	db := openTestMySQL(t, dsn)
	users := NewMySQLUserStore(db)
	suffix := time.Now().UnixNano()
	owner, err := users.Register(fmt.Sprintf("contract_owner_%d", suffix), fmt.Sprintf("contract-owner-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register owner: %v", err)
	}
	viewer, err := users.Register(fmt.Sprintf("contract_viewer_%d", suffix), fmt.Sprintf("contract-viewer-%d@example.com", suffix), "StrongPass123!")
	if err != nil {
		t.Fatalf("register viewer: %v", err)
	}
	runPromptManagerContract(t, NewMySQLPromptStore(db), User{
		ID:       owner.ID,
		Username: owner.Username,
		Email:    owner.Email,
		Status:   owner.Status,
	}, viewer.ID)
}
