package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv               string
	Port                 string
	JWTSecret            string
	JWTExpireHours       int
	AuthCookieEnabled    bool
	UploadProvider       string
	UploadDir            string
	UploadBaseURL        string
	UploadMaxMB          int
	UploadMaxConcurrent  int
	UploadDailyQuotaMB   int
	UploadTotalQuotaMB   int
	AllowGif             bool
	R2AccountID          string
	R2Endpoint           string
	R2AccessKeyID        string
	R2SecretKey          string
	R2Bucket             string
	R2PublicURL          string
	AllowedImageDomains  []string
	MySQLHost            string
	MySQLPort            string
	MySQLUser            string
	MySQLPass            string
	MySQLDB              string
	MySQLMigrationUser   string
	MySQLMigrationPass   string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetimeMin int
	RedisHost            string
	RedisPort            string
	RedisPass            string
	AllowedOrigin        string
	GitHubClientID       string
	GitHubClientSecret   string
	GitHubOAuthEnabled   bool
	GitHubRedirectURI    string
	FrontendURL          string
	SMTPHost             string
	SMTPPort             string
	SMTPUser             string
	SMTPPassword         string
	SMTPFrom             string
}

// DefaultAllowedOrigins is the comma-separated origin list used in development.
const DefaultAllowedOrigins = "*"

// Load reads configuration from the environment. It never fails; use Validate
// to check that the resulting config is safe to run.
func Load() Config {
	return Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		Port:                 getEnv("PORT", "8080"),
		JWTSecret:            getEnv("JWT_SECRET", "promptos-dev-secret-change-me"),
		JWTExpireHours:       getEnvAsInt("JWT_EXPIRE_HOURS", 72),
		AuthCookieEnabled:    getEnvAsBool("AUTH_COOKIE_ENABLED", true),
		UploadProvider:       getEnv("UPLOAD_PROVIDER", "local"),
		UploadDir:            getEnv("UPLOAD_DIR", "./uploads"),
		UploadBaseURL:        getEnv("UPLOAD_BASE_URL", "http://localhost:8080"),
		UploadMaxMB:          getEnvAsInt("UPLOAD_MAX_MB", 10),
		UploadMaxConcurrent:  getEnvAsInt("UPLOAD_MAX_CONCURRENT", 4),
		UploadDailyQuotaMB:   getEnvAsInt("UPLOAD_DAILY_QUOTA_MB", 100),
		UploadTotalQuotaMB:   getEnvAsInt("UPLOAD_TOTAL_QUOTA_MB", 2048),
		AllowGif:             getEnvAsBool("UPLOAD_ALLOW_GIF", false),
		R2AccountID:          getEnv("R2_ACCOUNT_ID", ""),
		R2Endpoint:           getEnv("R2_ENDPOINT", getEnv("S3_ENDPOINT", "")),
		R2AccessKeyID:        getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:          getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:             getEnv("R2_BUCKET", ""),
		R2PublicURL:          getEnv("R2_PUBLIC_URL", ""),
		AllowedImageDomains:  splitDomains(getEnv("ALLOWED_IMAGE_DOMAINS", ""), getEnv("R2_PUBLIC_URL", "")),
		MySQLHost:            getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:            getEnv("MYSQL_PORT", "3306"),
		MySQLUser:            getEnv("MYSQL_USER", "root"),
		MySQLPass:            getEnv("MYSQL_PASSWORD", "root"),
		MySQLDB:              getEnv("MYSQL_DATABASE", "promptos"),
		MySQLMigrationUser:   getEnv("MYSQL_MIGRATION_USER", ""),
		MySQLMigrationPass:   getEnv("MYSQL_MIGRATION_PASSWORD", ""),
		DBMaxOpenConns:       getEnvAsInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:       getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetimeMin: getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		RedisHost:            getEnv("REDIS_HOST", "localhost"),
		RedisPort:            getEnv("REDIS_PORT", "6379"),
		RedisPass:            getEnv("REDIS_PASSWORD", ""),
		AllowedOrigin:        getEnv("ALLOWED_ORIGIN", "*"),
		GitHubClientID:       getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:   getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubOAuthEnabled:   getEnvAsBool("GITHUB_OAUTH_ENABLED", false),
		GitHubRedirectURI:    getEnv("GITHUB_REDIRECT_URI", ""),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
		SMTPHost:             getEnv("SMTP_HOST", ""),
		SMTPPort:             getEnv("SMTP_PORT", "587"),
		SMTPUser:             getEnv("SMTP_USER", ""),
		SMTPPassword:         getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:             getEnv("SMTP_FROM", ""),
	}
}

// Validate enforces production-safe defaults and rejects configurations that
// would leak secrets or misbehave in a non-development environment. It returns
// the offending variable name so callers can log it without printing values.
func (c Config) Validate() error {
	if !c.IsDevelopment() && !c.IsTest() && !c.IsProduction() {
		return fmt.Errorf("APP_ENV must be one of development, test, or production")
	}
	prod := c.IsProduction()

	if err := validatePort(c.Port); err != nil {
		return err
	}
	if err := validateIntEnv("JWT_EXPIRE_HOURS", c.JWTExpireHours); err != nil {
		return err
	}
	if err := validateIntEnv("UPLOAD_MAX_MB", c.UploadMaxMB); err != nil {
		return err
	}
	if c.UploadMaxConcurrent < 0 {
		return errors.New("UPLOAD_MAX_CONCURRENT must not be negative")
	}
	if c.UploadDailyQuotaMB < 0 {
		return errors.New("UPLOAD_DAILY_QUOTA_MB must not be negative")
	}
	if c.UploadTotalQuotaMB < 0 {
		return errors.New("UPLOAD_TOTAL_QUOTA_MB must not be negative")
	}
	// Zero means "use the documented code default" for programmatic test
	// configs; values loaded from the environment are always positive.
	if c.DBMaxOpenConns < 0 || c.DBMaxIdleConns < 0 || c.DBConnMaxLifetimeMin < 0 {
		return errors.New("database pool settings must not be negative")
	}
	if c.DBMaxOpenConns > 0 && c.DBMaxIdleConns > c.DBMaxOpenConns {
		return errors.New("DB_MAX_IDLE_CONNS cannot exceed DB_MAX_OPEN_CONNS")
	}

	if prod && (strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == "promptos-dev-secret-change-me") {
		return errors.New("JWT_SECRET must be set to a strong secret in non-development environments")
	}
	if prod && strings.EqualFold(c.MySQLPass, "root") {
		return errors.New("MYSQL_PASSWORD must not be the default root password in non-development environments")
	}
	if prod && strings.EqualFold(strings.TrimSpace(c.MySQLUser), "root") {
		return errors.New("MYSQL_USER must be a dedicated application user in production")
	}
	if prod && (strings.TrimSpace(c.MySQLMigrationUser) == "" || strings.TrimSpace(c.MySQLMigrationPass) == "") {
		return errors.New("MYSQL_MIGRATION_USER and MYSQL_MIGRATION_PASSWORD must be configured in production")
	}
	if prod && strings.EqualFold(strings.TrimSpace(c.MySQLMigrationUser), strings.TrimSpace(c.MySQLUser)) {
		return errors.New("MYSQL_MIGRATION_USER must differ from MYSQL_USER in production")
	}
	if prod && strings.TrimSpace(c.RedisPass) == "" {
		return errors.New("REDIS_PASSWORD must be configured in production")
	}
	if prod && c.AllowedOrigin == "*" {
		return errors.New("ALLOWED_ORIGIN must be an explicit origin list (not *) in non-development environments")
	}
	if prod {
		if strings.TrimSpace(c.SMTPHost) == "" || strings.TrimSpace(c.SMTPFrom) == "" {
			return errors.New("SMTP_HOST and SMTP_FROM must be configured in production")
		}
		if err := validatePort(c.SMTPPort); err != nil {
			return fmt.Errorf("SMTP_PORT: %w", err)
		}
		if (strings.TrimSpace(c.SMTPUser) == "") != (strings.TrimSpace(c.SMTPPassword) == "") {
			return errors.New("SMTP_USER and SMTP_PASSWORD must be configured together")
		}
	}
	if c.GitHubOAuthEnabled && (strings.TrimSpace(c.GitHubClientID) == "" || strings.TrimSpace(c.GitHubClientSecret) == "") {
		return errors.New("GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be configured when GitHub OAuth is enabled")
	}

	if strings.Contains(c.AllowedOrigin, "*") && strings.Contains(c.AllowedOrigin, ",") {
		return errors.New("ALLOWED_ORIGIN cannot mix wildcard * with an explicit origin list")
	}

	for _, origin := range splitOrigins(c.AllowedOrigin) {
		if !validOrigin(origin) {
			return fmt.Errorf("ALLOWED_ORIGIN contains invalid origin %q", origin)
		}
	}

	return nil
}

func (c Config) IsDevelopment() bool { return c.AppEnv == "development" }
func (c Config) IsTest() bool        { return c.AppEnv == "test" }
func (c Config) IsProduction() bool  { return c.AppEnv == "production" }

// AllowedOrigins returns the parsed comma-separated allowlist. An empty or
// wildcard list yields an empty slice so callers can treat it as "any origin".
func (c Config) AllowedOrigins() []string {
	return splitOrigins(c.AllowedOrigin)
}

func validatePort(port string) error {
	if strings.TrimSpace(port) == "" {
		return errors.New("PORT must not be empty")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("PORT %q is not a valid TCP port", port)
	}
	return nil
}

func validateIntEnv(name string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be a positive integer", name)
	}
	return nil
}

func splitOrigins(list string) []string {
	var origins []string
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	return origins
}

func validOrigin(origin string) bool {
	if origin == "*" {
		return true
	}
	return strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getEnvAsBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

// splitDomains merges an explicit comma-separated domain allowlist with an
// optional fallback (the R2 public host), normalizing each to a bare hostname.
func splitDomains(domainList, fallback string) []string {
	raw := []string{}
	for _, part := range strings.Split(domainList, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			raw = append(raw, part)
		}
	}
	if len(raw) == 0 && fallback != "" {
		raw = append(raw, fallback)
	}

	domains := make([]string, 0, len(raw))
	for _, d := range raw {
		d = strings.TrimSpace(d)
		d = strings.TrimPrefix(d, "http://")
		d = strings.TrimPrefix(d, "https://")
		d = strings.Trim(d, "/")
		if d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}
