package config

import (
	"os"
	"strings"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port            string
	DBPath          string
	EncKey          string
	APIKeys         []string
	SentryDSN       string
	SentryEnv       string
	SentryTraceRate float64
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
		// Comma-separated accepted X-API-Key values for the feedback endpoint.
		// Empty disables the check (all requests pass).
		APIKeys:         splitList(getEnv("API_KEYS", "")),
		SentryDSN:       getEnv("SENTRY_DSN", ""),
		SentryEnv:       getEnv("SENTRY_ENVIRONMENT", "development"),
		SentryTraceRate: 1.0,
	}
}

// splitList parses a comma-separated env value into a slice, trimming whitespace
// and dropping empty entries.
func splitList(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
