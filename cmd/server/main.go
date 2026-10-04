package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/idefinity/nps-api/internal/config"
	"github.com/idefinity/nps-api/internal/crypto"
	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/handler"
	"github.com/idefinity/nps-api/internal/middleware"
)

func main() {
	cfg := config.Load()

	initSentry(cfg)

	enc, err := crypto.New(cfg.EncKey)
	if err != nil {
		// A configured-but-malformed key must not silently fall back to
		// plaintext — fail closed.
		slog.Error("invalid FEEDBACK_ENC_KEY", "error", err)
		os.Exit(1)
	}
	if !enc.Enabled() {
		slog.Warn("FEEDBACK_ENC_KEY not set — feedback comment/timezone stored UNENCRYPTED")
	} else {
		slog.Info("field encryption enabled", "fields", "comment,timezone")
	}

	database, cleanup := connectDB(cfg, enc)
	defer cleanup()

	if len(cfg.APIKeys) == 0 {
		slog.Warn("API_KEYS not set — feedback POST accepts requests without an API key")
	} else {
		slog.Info("API key auth enabled for POST", "keys", len(cfg.APIKeys))
	}
	if len(cfg.ReadAPIKeys) == 0 {
		slog.Warn("READ_API_KEYS not set — GET feedback (analytics) endpoint is disabled")
	} else {
		slog.Info("analytics read endpoint enabled", "keys", len(cfg.ReadAPIKeys))
	}

	mux := handler.RegisterRoutes(database, cfg.APIKeys, cfg.ReadAPIKeys)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      middleware.Logging(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go awaitShutdown(srv)

	slog.Info("server starting", "port", cfg.Port, "prefix", "/nps")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func initSentry(cfg *config.Config) {
	if cfg.SentryDSN == "" {
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.SentryEnv,
		TracesSampleRate: cfg.SentryTraceRate,
	})
	if err != nil {
		slog.Error("failed to initialize Sentry", "error", err)
		return
	}
	slog.Info("Sentry initialized", "environment", cfg.SentryEnv)
}

func connectDB(cfg *config.Config, enc *crypto.Cipher) (*db.Database, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.DBPath, enc)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database ready", "path", cfg.DBPath)

	cleanup := func() {
		sentry.Flush(2 * time.Second)
		database.Close(context.Background())
	}
	return database, cleanup
}

func awaitShutdown(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
}
