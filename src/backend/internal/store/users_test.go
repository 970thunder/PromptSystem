package store

import "testing"

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
