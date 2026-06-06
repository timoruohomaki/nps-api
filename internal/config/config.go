package config

import (
	"os"
	"strings"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port             string
	MongoURI         string
	MongoDatabase    string
	SentryDSN        string
	SentryEnv        string
	SentryTraceRate  float64
	AllowedPlatforms []string
	APIKeys          []string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8081"),
		MongoURI:         getEnv("MONGODB_URI", ""),
		MongoDatabase:    getEnv("MONGODB_DATABASE", "nps"),
		SentryDSN:        getEnv("SENTRY_DSN", ""),
		SentryEnv:        getEnv("SENTRY_ENVIRONMENT", "development"),
		SentryTraceRate:  1.0,
		AllowedPlatforms: getEnvCSV("ALLOWED_PLATFORMS", []string{"macOS", "Windows"}),
		APIKeys:          getEnvCSV("API_KEYS", nil),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvCSV(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
