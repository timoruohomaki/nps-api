package config

import "os"

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port            string
	MongoURI        string
	MongoDatabase   string
	SentryDSN       string
	SentryEnv       string
	SentryTraceRate float64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8081"),
		MongoURI:        getEnv("MONGODB_URI", ""),
		MongoDatabase:   getEnv("MONGODB_DATABASE", "nps"),
		SentryDSN:       getEnv("SENTRY_DSN", ""),
		SentryEnv:       getEnv("SENTRY_ENVIRONMENT", "development"),
		SentryTraceRate: 1.0,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
