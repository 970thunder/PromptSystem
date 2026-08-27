package store

import (
	"strings"
	"testing"
)

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
