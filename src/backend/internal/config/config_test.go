package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "8080")
	t.Setenv("JWT_SECRET", "promptos-dev-secret-change-me")
	t.Setenv("MYSQL_PASSWORD", "root")
	t.Setenv("ALLOWED_ORIGIN", "*")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")

	cfg := Load()
	if cfg.AppEnv != "development" {
		t.Fatalf("expected development env, got %q", cfg.AppEnv)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development defaults should validate: %v", err)
	}
}

func TestValidateRejectsWeakProductionSecret(t *testing.T) {
	cfg := Config{
		AppEnv:             "production",
		Port:               "8080",
		JWTSecret:          "promptos-dev-secret-change-me",
		MySQLPass:          "strong-password",
		AllowedOrigin:      "https://example.com",
		GitHubClientID:     "id",
		GitHubClientSecret: "secret",
		JWTExpireHours:     72,
		UploadMaxMB:        10,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for weak production JWT secret")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET in error, got %v", err)
	}
}

func TestValidateRejectsDefaultRootPasswordInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:             "production",
		Port:               "8080",
		JWTSecret:          "super-strong-secret",
		MySQLPass:          "root",
		AllowedOrigin:      "https://example.com",
		GitHubClientID:     "id",
		GitHubClientSecret: "secret",
		JWTExpireHours:     72,
		UploadMaxMB:        10,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for default root password")
	}
}

func TestValidateRejectsWildcardOriginInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:             "production",
		Port:               "8080",
		JWTSecret:          "super-strong-secret",
		MySQLPass:          "strong-password",
		AllowedOrigin:      "*",
		GitHubClientID:     "id",
		GitHubClientSecret: "secret",
		JWTExpireHours:     72,
		UploadMaxMB:        10,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for wildcard origin in production")
	}
}

func TestValidateRejectsMissingOAuthInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:             "production",
		Port:               "8080",
		JWTSecret:          "super-strong-secret",
		MySQLPass:          "strong-password",
		AllowedOrigin:      "https://example.com",
		GitHubClientID:     "",
		GitHubClientSecret: "",
		GitHubOAuthEnabled: true,
		JWTExpireHours:     72,
		UploadMaxMB:        10,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing OAuth config in production")
	}
}

func TestValidateAllowsDisabledOAuthInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:             "production",
		Port:               "8080",
		JWTSecret:          "super-strong-secret",
		MySQLPass:          "strong-password",
		MySQLUser:          "promptos_app",
		MySQLMigrationUser: "promptos_migrator",
		MySQLMigrationPass: "migration-password",
		AllowedOrigin:      "https://example.com",
		JWTExpireHours:     72,
		UploadMaxMB:        10,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled OAuth should validate: %v", err)
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := Config{
		AppEnv:         "development",
		Port:           "not-a-port",
		JWTSecret:      "promptos-dev-secret-change-me",
		MySQLPass:      "root",
		AllowedOrigin:  "*",
		JWTExpireHours: 72,
		UploadMaxMB:    10,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid port")
	}
}

func TestAllowedOriginsParsing(t *testing.T) {
	cfg := Config{AllowedOrigin: "https://a.com, http://b.com"}
	origins := cfg.AllowedOrigins()
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d: %#v", len(origins), origins)
	}
	if origins[0] != "https://a.com" || origins[1] != "http://b.com" {
		t.Fatalf("unexpected origins: %#v", origins)
	}
}

func TestValidateRejectsMixedWildcardAndList(t *testing.T) {
	cfg := Config{
		AppEnv:         "development",
		Port:           "8080",
		JWTSecret:      "promptos-dev-secret-change-me",
		MySQLPass:      "root",
		AllowedOrigin:  "*, https://example.com",
		JWTExpireHours: 72,
		UploadMaxMB:    10,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for mixing wildcard with explicit origin list")
	}
}
