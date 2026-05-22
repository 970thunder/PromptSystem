package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestScanAuthUserHandlesNullAvatarAndBio(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)

	user, found, err := scanAuthUser(func(dest ...any) error {
		*(dest[0].(*int)) = 42
		*(dest[1].(*string)) = "octocat"
		*(dest[2].(*sql.NullString)) = sql.NullString{}
		*(dest[3].(*string)) = "octocat@users.noreply.github.com"
		*(dest[4].(*sql.NullInt64)) = sql.NullInt64{Int64: 99, Valid: true}
		*(dest[5].(*sql.NullString)) = sql.NullString{}
		*(dest[6].(*sql.NullString)) = sql.NullString{}
		*(dest[7].(*int)) = 1
		*(dest[8].(*int)) = 0
		*(dest[9].(*int)) = 1
		*(dest[10].(*time.Time)) = createdAt
		return nil
	})
	if err != nil {
		t.Fatalf("scanAuthUser() error = %v", err)
	}
	if !found {
		t.Fatal("scanAuthUser() found = false, want true")
	}
	if user.ID != 42 || user.Username != "octocat" {
		t.Fatalf("unexpected user identity: %+v", user)
	}
	if user.Avatar != "" || user.Bio != "" {
		t.Fatalf("expected empty avatar/bio for NULL columns, got avatar=%q bio=%q", user.Avatar, user.Bio)
	}
	if user.GitHubID != 99 {
		t.Fatalf("expected github id 99, got %d", user.GitHubID)
	}
}
