package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("DB_PATH")

	cfg := Load()

	if cfg.Port != "8081" {
		t.Errorf("expected default port 8081, got %s", cfg.Port)
	}
	if cfg.DBPath != "nps.db" {
		t.Errorf("expected default DBPath nps.db, got %s", cfg.DBPath)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("DB_PATH", "/data/nps.db")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_PATH")
	}()

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if cfg.DBPath != "/data/nps.db" {
		t.Errorf("expected DBPath /data/nps.db, got %s", cfg.DBPath)
	}
}
