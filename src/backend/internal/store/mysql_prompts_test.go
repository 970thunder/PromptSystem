package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestScanPromptHandlesNullUserBioAndAvatar(t *testing.T) {
	userCreatedAt := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	promptCreatedAt := time.Date(2026, 5, 23, 8, 0, 0, 0, time.UTC)
	promptUpdatedAt := time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC)

	prompt, err := scanPrompt(func(dest ...any) error {
		*(dest[0].(*int)) = 7
		*(dest[1].(*string)) = "Test Prompt"
		*(dest[2].(*sql.NullString)) = sql.NullString{String: "desc", Valid: true}
		*(dest[3].(*sql.NullString)) = sql.NullString{}
		*(dest[4].(*sql.NullString)) = sql.NullString{String: `["https://example.com/a.png"]`, Valid: true}
		*(dest[5].(*string)) = "content body"
		*(dest[6].(*sql.NullString)) = sql.NullString{}
		*(dest[7].(*string)) = "gpt-4"
		*(dest[8].(*sql.NullString)) = sql.NullString{}
		*(dest[9].(*int)) = 1
		*(dest[10].(*string)) = "摄影"
		*(dest[11].(*int)) = 42
		*(dest[12].(*string)) = "octocat"
		*(dest[13].(*sql.NullString)) = sql.NullString{}
		*(dest[14].(*string)) = "octocat@users.noreply.github.com"
		*(dest[15].(*sql.NullString)) = sql.NullString{}
		*(dest[16].(*int)) = 1
		*(dest[17].(*int)) = 0
		*(dest[18].(*int)) = 1
		*(dest[19].(*time.Time)) = userCreatedAt
		*(dest[20].(*int)) = 10
		*(dest[21].(*int)) = 2
		*(dest[22].(*int)) = 1
		*(dest[23].(*int)) = 1
		*(dest[24].(*time.Time)) = promptCreatedAt
		*(dest[25].(*time.Time)) = promptUpdatedAt
		*(dest[26].(*string)) = "tag1||tag2"
		return nil
	})
	if err != nil {
		t.Fatalf("scanPrompt() error = %v", err)
	}

	if prompt.ID != 7 || prompt.Title != "Test Prompt" {
		t.Fatalf("unexpected prompt identity: %+v", prompt)
	}
	if prompt.User.Avatar != "" || prompt.User.Bio != "" {
		t.Fatalf("expected empty avatar/bio for NULL user columns, got avatar=%q bio=%q", prompt.User.Avatar, prompt.User.Bio)
	}
	if prompt.Cover != "" || prompt.SystemPrompt != "" {
		t.Fatalf("expected empty cover/system_prompt for NULL columns, got cover=%q system_prompt=%q", prompt.Cover, prompt.SystemPrompt)
	}
	if len(prompt.Images) != 1 || prompt.Images[0] != "https://example.com/a.png" {
		t.Fatalf("unexpected images: %+v", prompt.Images)
	}
	if len(prompt.Tags) != 2 || prompt.Tags[0] != "tag1" || prompt.Tags[1] != "tag2" {
		t.Fatalf("unexpected tags: %+v", prompt.Tags)
	}
}

func TestScanPromptHandlesNullParams(t *testing.T) {
	now := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)

	prompt, err := scanPrompt(func(dest ...any) error {
		*(dest[0].(*int)) = 1
		*(dest[1].(*string)) = "Title"
		*(dest[2].(*sql.NullString)) = sql.NullString{}
		*(dest[3].(*sql.NullString)) = sql.NullString{}
		*(dest[4].(*sql.NullString)) = sql.NullString{}
		*(dest[5].(*string)) = "content"
		*(dest[6].(*sql.NullString)) = sql.NullString{}
		*(dest[7].(*string)) = ""
		*(dest[8].(*sql.NullString)) = sql.NullString{}
		*(dest[9].(*int)) = 1
		*(dest[10].(*string)) = "摄影"
		*(dest[11].(*int)) = 1
		*(dest[12].(*string)) = "user"
		*(dest[13].(*sql.NullString)) = sql.NullString{}
		*(dest[14].(*string)) = "user@example.com"
		*(dest[15].(*sql.NullString)) = sql.NullString{}
		*(dest[16].(*int)) = 1
		*(dest[17].(*int)) = 0
		*(dest[18].(*int)) = 1
		*(dest[19].(*time.Time)) = now
		*(dest[20].(*int)) = 0
		*(dest[21].(*int)) = 0
		*(dest[22].(*int)) = 0
		*(dest[23].(*int)) = 1
		*(dest[24].(*time.Time)) = now
		*(dest[25].(*time.Time)) = now
		*(dest[26].(*string)) = ""
		return nil
	})
	if err != nil {
		t.Fatalf("scanPrompt() error = %v", err)
	}

	if prompt.Params != (PromptParams{}) {
		t.Fatalf("expected zero params for NULL column, got %+v", prompt.Params)
	}
}
