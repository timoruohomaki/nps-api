package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")
	os.Unsetenv("ALLOWED_PLATFORMS")
	os.Unsetenv("API_KEYS")

	cfg := Load()

	if cfg.Port != "8081" {
		t.Errorf("expected default port 8081, got %s", cfg.Port)
	}
	if cfg.MongoDatabase != "nps" {
		t.Errorf("expected default database nps, got %s", cfg.MongoDatabase)
	}
	if cfg.MongoURI != "" {
		t.Errorf("expected empty MongoURI, got %s", cfg.MongoURI)
	}
	if len(cfg.AllowedPlatforms) != 2 || cfg.AllowedPlatforms[0] != "macOS" || cfg.AllowedPlatforms[1] != "Windows" {
		t.Errorf("expected default platforms [macOS Windows], got %v", cfg.AllowedPlatforms)
	}
	if len(cfg.APIKeys) != 0 {
		t.Errorf("expected no API keys by default, got %v", cfg.APIKeys)
	}
}

func TestLoad_CSVEnvVars(t *testing.T) {
	os.Setenv("ALLOWED_PLATFORMS", "macOS, Windows ,iOS,Android")
	os.Setenv("API_KEYS", "key-one, key-two")
	defer func() {
		os.Unsetenv("ALLOWED_PLATFORMS")
		os.Unsetenv("API_KEYS")
	}()

	cfg := Load()

	want := []string{"macOS", "Windows", "iOS", "Android"}
	if len(cfg.AllowedPlatforms) != len(want) {
		t.Fatalf("expected %v, got %v", want, cfg.AllowedPlatforms)
	}
	for i, w := range want {
		if cfg.AllowedPlatforms[i] != w {
			t.Errorf("AllowedPlatforms[%d]: expected %q, got %q", i, w, cfg.AllowedPlatforms[i])
		}
	}

	if len(cfg.APIKeys) != 2 || cfg.APIKeys[0] != "key-one" || cfg.APIKeys[1] != "key-two" {
		t.Errorf("expected API keys [key-one key-two], got %v", cfg.APIKeys)
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
