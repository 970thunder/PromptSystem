package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv         string
	Port           string
	JWTSecret      string
	JWTExpireHours int
	UploadProvider string
	UploadDir      string
	UploadBaseURL  string
	UploadMaxMB    int
	R2AccountID    string
	R2AccessKeyID  string
	R2SecretKey    string
	R2Bucket       string
	R2PublicURL    string
	MySQLHost      string
	MySQLPort      string
	MySQLUser      string
	MySQLPass      string
	MySQLDB        string
	RedisHost      string
	RedisPort      string
	RedisPass      string
	AllowedOrigin  string
}

func Load() Config {
	return Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "promptos-dev-secret-change-me"),
		JWTExpireHours: getEnvAsInt("JWT_EXPIRE_HOURS", 72),
		UploadProvider: getEnv("UPLOAD_PROVIDER", "local"),
		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
		UploadBaseURL:  getEnv("UPLOAD_BASE_URL", "http://localhost:8080"),
		UploadMaxMB:    getEnvAsInt("UPLOAD_MAX_MB", 10),
		R2AccountID:    getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:    getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:       getEnv("R2_BUCKET", ""),
		R2PublicURL:    getEnv("R2_PUBLIC_URL", ""),
		MySQLHost:      getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:      getEnv("MYSQL_PORT", "3306"),
		MySQLUser:      getEnv("MYSQL_USER", "root"),
		MySQLPass:      getEnv("MYSQL_PASSWORD", "root"),
		MySQLDB:        getEnv("MYSQL_DATABASE", "promptos"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPass:      getEnv("REDIS_PASSWORD", ""),
		AllowedOrigin:  getEnv("ALLOWED_ORIGIN", "*"),
	}
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
