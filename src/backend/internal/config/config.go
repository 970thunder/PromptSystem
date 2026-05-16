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
