package config

import (
	"os"
	"strings"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port             string
	DBPath           string
	EncKey           string
	APIKeys          []string
	ReadAPIKeys      []string
	AllowedPlatforms []string
	SentryDSN        string
	SentryEnv        string
	SentryTraceRate  float64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8081"),
		// Default is relative (cwd) so `go run ./cmd/server` works out of the
		// box; the container sets DB_PATH=/data/nps.db onto a persistent volume.
		DBPath: getEnv("DB_PATH", "nps.db"),
		// base64-encoded 32-byte AES-256 key for comment/timezone encryption.
		// Empty disables encryption (dev); production must set it.
		EncKey: getEnv("FEEDBACK_ENC_KEY", ""),
		// Accepted X-API-Key values for POSTing feedback. Empty = check disabled.
		APIKeys: getEnvCSV("API_KEYS", nil),
		// Consumer keys for the GET analytics query (returns decrypted PII).
		// Empty = read endpoint disabled (fail closed).
		ReadAPIKeys: getEnvCSV("READ_API_KEYS", nil),
		// Platform allowlist for feedback validation; default keeps back-compat.
		AllowedPlatforms: getEnvCSV("ALLOWED_PLATFORMS", []string{"macOS", "Windows"}),
		SentryDSN:        getEnv("SENTRY_DSN", ""),
		SentryEnv:        getEnv("SENTRY_ENVIRONMENT", "development"),
		SentryTraceRate:  1.0,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvCSV parses a comma-separated env value into a slice, trimming whitespace
// and dropping empty entries; returns fallback when unset or all-blank.
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
