package store

import (
	"fmt"
	"regexp"
	"strings"
)

const maxUsernameLen = 39

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func truncateUsername(username string) string {
	username = strings.TrimSpace(username)
	if len(username) <= maxUsernameLen {
		return username
	}

	return username[:maxUsernameLen]
}

func formatGitHubUsername(login string, githubID int64) string {
	login = truncateUsername(strings.TrimSpace(login))
	if login != "" {
		return login
	}
	if githubID > 0 {
		return truncateUsername(fmt.Sprintf("user-%d", githubID))
	}

	return "user"
}

func usernameWithGitHubSuffix(base string, githubID int64) string {
	suffix := fmt.Sprintf("-%d", githubID)
	base = truncateUsername(strings.TrimSpace(base))
	if base == "" {
		base = "user"
	}

	maxBase := maxUsernameLen - len(suffix)
	if maxBase < 1 {
		maxBase = 1
	}
	if len(base) > maxBase {
		base = base[:maxBase]
	}

	return base + suffix
}

func githubUsernameCandidates(login string, githubID int64) []string {
	primary := formatGitHubUsername(login, githubID)
	candidates := []string{primary}
	if githubID > 0 {
		candidates = append(candidates, usernameWithGitHubSuffix(primary, githubID))
		fallback := truncateUsername(fmt.Sprintf("user-%d", githubID))
		candidates = append(candidates, fallback)
	}

	seen := make(map[string]struct{}, len(candidates))
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}

	return unique
}

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(email) > 100 {
		return false
	}

	return emailPattern.MatchString(email)
}
