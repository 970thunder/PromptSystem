package auth

import (
	"testing"
	"time"
)

func TestTokenManagerGenerateAndVerify(t *testing.T) {
	manager := NewTokenManager("test-secret", time.Hour)

	token, err := manager.Generate(9, "user@example.com")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.Subject != "9" {
		t.Fatalf("expected subject 9, got %s", claims.Subject)
	}

	if claims.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", claims.Email)
	}
}
