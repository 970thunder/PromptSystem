package store

import "testing"

func TestIsValidEmail(t *testing.T) {
	cases := map[string]bool{
		"user@example.com":  true,
		"bad-email":         false,
		"@missing.com":      false,
		"spaces @x.com":     false,
	}

	for email, expected := range cases {
		if IsValidEmail(email) != expected {
			t.Fatalf("IsValidEmail(%q) = %v, want %v", email, !expected, expected)
		}
	}
}
