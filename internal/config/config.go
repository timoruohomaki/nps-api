package config

import "os"

type Config struct {
	Port            string
	MongoURI        string
	MongoDatabase   string
	SentryDSN       string
	SentryEnv       string
	SentryTraceRate float64
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
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
