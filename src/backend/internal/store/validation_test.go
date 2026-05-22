package store

import (
	"strings"
	"testing"
)

func TestTruncateUsername(t *testing.T) {
	long := strings.Repeat("a", 50)
	got := truncateUsername(long)
	if len(got) != maxUsernameLen {
		t.Fatalf("expected length %d, got %d", maxUsernameLen, len(got))
	}
}

func TestGitHubUsernameCandidates(t *testing.T) {
	candidates := githubUsernameCandidates("Astra Lab", 583231)
	if len(candidates) < 2 {
		t.Fatalf("expected multiple candidates, got %#v", candidates)
	}
	if candidates[0] != "Astra Lab" {
		t.Fatalf("expected primary login preserved, got %q", candidates[0])
	}

	longLogin := strings.Repeat("x", 50)
	for _, candidate := range githubUsernameCandidates(longLogin, 42) {
		if len(candidate) > maxUsernameLen {
			t.Fatalf("candidate longer than %d: %q", maxUsernameLen, candidate)
		}
	}
}

func TestUsernameWithGitHubSuffixFitsLimit(t *testing.T) {
	got := usernameWithGitHubSuffix(strings.Repeat("y", 50), 999)
	if len(got) > maxUsernameLen {
		t.Fatalf("expected suffix username within %d chars, got len %d: %q", maxUsernameLen, len(got), got)
	}
	if !strings.HasSuffix(got, "-999") {
		t.Fatalf("expected github id suffix, got %q", got)
	}
}
