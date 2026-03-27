// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBHost                string
	DBPort                string
	DBName                string
	DBUser                string
	DBPassword            string
	DBSSLMode             string
	APIPort               string
	JWTSecret             string
	UploadPath            string
	CORSAllowedOrigins    []string // Comma-separated list, e.g. "https://app.example.com,http://localhost:3000"
	RefreshCookieSecure   bool
	RefreshCookieSameSite string
	RefreshCookieDomain   string
}

func Load() *Config {
	return &Config{
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBName:                getEnv("DB_NAME", "poultry_farm"),
		DBUser:                getEnv("DB_USER", "postgres"),
		DBPassword:            getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:             getEnv("DB_SSLMODE", "disable"),
		APIPort:               getEnv("API_PORT", "8080"),
		JWTSecret:             getEnv("JWT_SECRET", "change-this-secret-key-in-production"),
		UploadPath:            getEnv("UPLOAD_PATH", "./uploads"),
		CORSAllowedOrigins:    parseCORSOrigins(getEnv("CORS_ALLOWED_ORIGINS", "https://d1umbk34tztlqz.cloudfront.net,https://mykolipannai.com,https://www.mykolipannai.com,https://app-dev.mykolipannai.com,http://localhost:3000,http://localhost:4300,http://localhost:5173")),
		RefreshCookieSecure:   parseBool(getEnv("REFRESH_COOKIE_SECURE", "false")),
		RefreshCookieSameSite: getEnv("REFRESH_COOKIE_SAMESITE", "Lax"),
		RefreshCookieDomain:   getEnv("REFRESH_COOKIE_DOMAIN", ""),
	}
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseCORSOrigins(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseBool(v string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false
	}
	return b
}
