package config

import "os"

type Config struct {
	AppEnv        string
	Port          string
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPass     string
	MySQLDB       string
	RedisHost     string
	RedisPort     string
	RedisPass     string
	AllowedOrigin string
}

func Load() Config {
	return Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		Port:          getEnv("PORT", "8080"),
		MySQLHost:     getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLUser:     getEnv("MYSQL_USER", "root"),
		MySQLPass:     getEnv("MYSQL_PASSWORD", "root"),
		MySQLDB:       getEnv("MYSQL_DATABASE", "promptos"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPass:     getEnv("REDIS_PASSWORD", ""),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
