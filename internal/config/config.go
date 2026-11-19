package config

import (
	"fmt"
	"os"
	"strconv"
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
	RetentionMonths  int // 0 = cleanup disabled
	SentryDSN        string
	SentryEnv        string
	SentryTraceRate  float64
}

// Load reads configuration from environment variables with sensible defaults.
// It returns an error for a value that is set but malformed, so the service
// fails to start instead of silently ignoring it.
func Load() (*Config, error) {
	// Feedback older than this many months is deleted (privacy policy: 12).
	retention, err := getEnvNonNegInt("RETENTION_MONTHS", 12)
	if err != nil {
		return nil, err
	}
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
		RetentionMonths:  retention,
		SentryTraceRate:  1.0,
	}, nil
}

// getEnvNonNegInt parses a non-negative integer env value; unset = fallback.
func getEnvNonNegInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer, got %q", key, raw)
	}
	return n, nil
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
