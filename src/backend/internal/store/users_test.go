package store

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestPublicUserJSONDoesNotExposePrivateFields(t *testing.T) {
	user := AuthUser{ID: 1, Username: "Public", Email: "private@example.com", Status: 1, GitHubID: 42}
	encoded, err := json.Marshal(ToPublicUser(user))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"email", "status", "hasGitHubBound", "private@example.com"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("public user leaked %q: %s", forbidden, text)
		}
	}
}

func TestPromptAuthorJSONDoesNotExposeEmailOrAccountStatus(t *testing.T) {
	prompt := Prompt{ID: 1, User: User{ID: 2, Email: "private@example.com", Status: 1}}
	encoded, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(encoded)
	if strings.Contains(text, "private@example.com") || strings.Contains(text, `"user":{"id":2,"username":"","avatar":"","email"`) {
		t.Fatalf("prompt author leaked private fields: %s", text)
	}
}

func TestUserStorePasswordTooLong(t *testing.T) {
	userStore := NewUserStore()
	longPassword := strings.Repeat("a", maxPasswordBytes+1)

	if _, err := userStore.Register("TooLong", "long@example.com", longPassword); err != ErrPasswordTooLong {
		t.Fatalf("Register() error = %v, want ErrPasswordTooLong", err)
	}
}

func TestUserStoreBumpSessionVersionInvalidatesTokens(t *testing.T) {
	userStore := NewUserStore()

	user, err := userStore.Register("SessionUser", "session@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	before := user.SessionVer

	if err := userStore.BumpSessionVersion("session@example.com"); err != nil {
		t.Fatalf("BumpSessionVersion() error = %v", err)
	}

	after, _ := userStore.FindByID(user.ID)
	if after.SessionVer <= before {
		t.Fatalf("expected session version to increase, got before=%d after=%d", before, after.SessionVer)
	}
}

func TestUserStoreRegisterAndAuthenticate(t *testing.T) {
	userStore := NewUserStore()

	user, err := userStore.Register("Tester", "tester@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.PasswordHash == "StrongPass123!" {
		t.Fatalf("password should be hashed")
	}

	authUser, err := userStore.Authenticate("tester@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if authUser.Email != "tester@example.com" {
		t.Fatalf("expected authenticated email tester@example.com, got %s", authUser.Email)
	}
}

func TestUserStoreResetPassword(t *testing.T) {
	userStore := NewUserStore()

	if _, err := userStore.Register("Reset User", "reset@example.com", "OldPass123!"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := userStore.ResetPassword("reset@example.com", "NewPass123!"); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	if _, err := userStore.Authenticate("reset@example.com", "OldPass123!"); err == nil {
		t.Fatal("expected old password to fail after reset")
	}

	if _, err := userStore.Authenticate("reset@example.com", "NewPass123!"); err != nil {
		t.Fatalf("Authenticate() with new password error = %v", err)
	}
}

func TestUserStoreUpsertGitHubUserResolvesUsernameCollision(t *testing.T) {
	userStore := NewUserStore()

	user, err := userStore.UpsertGitHubUser(4242, "Astra Lab", "github-user@users.noreply.github.com", "")
	if err != nil {
		t.Fatalf("UpsertGitHubUser() error = %v", err)
	}

	if user.Username == "Astra Lab" {
		t.Fatalf("expected username collision to be resolved, got %q", user.Username)
	}
	if user.GitHubID != 4242 {
		t.Fatalf("expected github id 4242, got %d", user.GitHubID)
	}
}

func TestUserStoreUpsertGitHubUserCreatesNewAccount(t *testing.T) {
	userStore := NewUserStore()

	user, err := userStore.UpsertGitHubUser(5252, "octocat", "octocat@users.noreply.github.com", "https://example.com/avatar.png")
	if err != nil {
		t.Fatalf("UpsertGitHubUser() error = %v", err)
	}

	if user.ID == 0 {
		t.Fatal("expected created user id")
	}
	if user.GitHubID != 5252 {
		t.Fatalf("expected github id 5252, got %d", user.GitHubID)
	}
	if user.Email != "octocat@users.noreply.github.com" {
		t.Fatalf("expected github email persisted, got %s", user.Email)
	}
	if user.PasswordHash != "" {
		t.Fatal("github-only account should not have a password hash")
	}
}

func TestUserStoreUpsertGitHubUserBindsExistingEmailAccount(t *testing.T) {
	userStore := NewUserStore()

	registered, err := userStore.Register("Email User", "bind-me@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	bound, err := userStore.UpsertGitHubUser(6262, "email-user", "bind-me@example.com", "https://example.com/avatar.png")
	if err != nil {
		t.Fatalf("UpsertGitHubUser() error = %v", err)
	}

	if bound.ID != registered.ID {
		t.Fatalf("expected github login to bind existing user %d, got %d", registered.ID, bound.ID)
	}
	if bound.GitHubID != 6262 {
		t.Fatalf("expected github id 6262, got %d", bound.GitHubID)
	}

	authUser, err := userStore.Authenticate("bind-me@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("Authenticate() after github bind error = %v", err)
	}
	if authUser.ID != registered.ID {
		t.Fatalf("expected email login to keep same user %d, got %d", registered.ID, authUser.ID)
	}
}

func TestUserStoreDeleteAccountAnonymizesAndRevokes(t *testing.T) {
	userStore := NewUserStore()
	user, err := userStore.Register("Delete Me", "delete-me@example.com", "StrongPass123!")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	prompt, err := CreatePrompt(CreatePromptInput{
		Title:      "retained prompt",
		Content:    "content",
		Model:      "gpt-4o",
		CategoryID: 1,
		Tags:       []string{"privacy"},
		User:       User{ID: user.ID, Username: user.Username, Email: user.Email, Status: 1},
		Status:     1,
	})
	if err != nil {
		t.Fatalf("CreatePrompt() error = %v", err)
	}
	if _, _, err := LikePrompt(prompt.ID, user.ID); err != nil {
		t.Fatalf("LikePrompt() error = %v", err)
	}
	if _, _, err := FavoritePrompt(prompt.ID, user.ID); err != nil {
		t.Fatalf("FavoritePrompt() error = %v", err)
	}
	if _, _, err := RecordPromptView(prompt.ID, user.ID); err != nil {
		t.Fatalf("RecordPromptView() error = %v", err)
	}

	if err := userStore.DeleteAccount(user.ID); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}
	deleted, found := userStore.FindByID(user.ID)
	if !found || deleted.Status != 0 || deleted.PasswordHash != "" || deleted.GitHubID != 0 {
		t.Fatalf("account was not disabled and scrubbed: found=%v user=%+v", found, deleted)
	}
	if deleted.Email != "deleted+"+strconv.Itoa(user.ID)+"@invalid.promptos.local" {
		t.Fatalf("unexpected anonymized email: %q", deleted.Email)
	}
	if _, err := userStore.Authenticate("delete-me@example.com", "StrongPass123!"); err == nil {
		t.Fatal("disabled account must not authenticate")
	}
	if _, found := FindPromptByID(prompt.ID); found {
		t.Fatal("prompt owned by disabled account must not remain public")
	}
	if got := ListUserHistoryPrompts(user.ID); len(got) != 0 {
		t.Fatalf("expected history to be cleared, got %d rows", len(got))
	}
	if err := userStore.DeleteAccount(user.ID); err != nil {
		t.Fatalf("repeated DeleteAccount() should be idempotent: %v", err)
	}
}
