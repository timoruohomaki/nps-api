package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might be set
	os.Unsetenv("PORT")
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.MongoDatabase != "nps" {
		t.Errorf("expected default database nps, got %s", cfg.MongoDatabase)
	}
	if cfg.MongoURI != "" {
		t.Errorf("expected empty MongoURI, got %s", cfg.MongoURI)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	os.Setenv("MONGODB_DATABASE", "testdb")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("MONGODB_URI")
		os.Unsetenv("MONGODB_DATABASE")
	}()

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("expected MongoURI mongodb://localhost:27017, got %s", cfg.MongoURI)
	}
	if cfg.MongoDatabase != "testdb" {
		t.Errorf("expected database testdb, got %s", cfg.MongoDatabase)
	}
}
