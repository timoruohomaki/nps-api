package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("ALLOWED_PLATFORMS")
	os.Unsetenv("API_KEYS")
	os.Unsetenv("READ_API_KEYS")

	cfg := mustLoad(t)

	if cfg.Port != "8081" {
		t.Errorf("expected default port 8081, got %s", cfg.Port)
	}
	if cfg.DBPath != "nps.db" {
		t.Errorf("expected default DBPath nps.db, got %s", cfg.DBPath)
	}
	if len(cfg.AllowedPlatforms) != 2 || cfg.AllowedPlatforms[0] != "macOS" || cfg.AllowedPlatforms[1] != "Windows" {
		t.Errorf("expected default platforms [macOS Windows], got %v", cfg.AllowedPlatforms)
	}
	if len(cfg.APIKeys) != 0 {
		t.Errorf("expected no API keys by default, got %v", cfg.APIKeys)
	}
	if len(cfg.ReadAPIKeys) != 0 {
		t.Errorf("expected no read API keys by default, got %v", cfg.ReadAPIKeys)
	}
}

func TestLoad_CSVEnvVars(t *testing.T) {
	os.Setenv("ALLOWED_PLATFORMS", "macOS, Windows ,iOS,Android")
	os.Setenv("API_KEYS", "key-one, key-two")
	os.Setenv("READ_API_KEYS", "consumer-one")
	defer func() {
		os.Unsetenv("ALLOWED_PLATFORMS")
		os.Unsetenv("API_KEYS")
		os.Unsetenv("READ_API_KEYS")
	}()

	cfg := mustLoad(t)

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
	if len(cfg.ReadAPIKeys) != 1 || cfg.ReadAPIKeys[0] != "consumer-one" {
		t.Errorf("expected read API keys [consumer-one], got %v", cfg.ReadAPIKeys)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("DB_PATH", "/data/nps.db")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_PATH")
	}()

	cfg := mustLoad(t)

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if cfg.DBPath != "/data/nps.db" {
		t.Errorf("expected DBPath /data/nps.db, got %s", cfg.DBPath)
	}
}

func mustLoad(t *testing.T) *Config {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestLoad_RetentionMonths(t *testing.T) {
	t.Setenv("RETENTION_MONTHS", "")
	if got := mustLoad(t).RetentionMonths; got != 12 {
		t.Errorf("default RetentionMonths = %d, want 12", got)
	}

	t.Setenv("RETENTION_MONTHS", "0")
	if got := mustLoad(t).RetentionMonths; got != 0 {
		t.Errorf("RETENTION_MONTHS=0 gave %d, want 0 (disabled)", got)
	}

	t.Setenv("RETENTION_MONTHS", "24")
	if got := mustLoad(t).RetentionMonths; got != 24 {
		t.Errorf("RETENTION_MONTHS=24 gave %d", got)
	}

	for _, bad := range []string{"-1", "twelve", "1.5"} {
		t.Setenv("RETENTION_MONTHS", bad)
		if _, err := Load(); err == nil {
			t.Errorf("RETENTION_MONTHS=%q should be rejected", bad)
		}
	}
}
