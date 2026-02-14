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
	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/handler"
	"github.com/idefinity/nps-api/internal/middleware"
)

func main() {
	cfg := config.Load()

	initSentry(cfg)

	database, cleanup := connectMongo(cfg)
	defer cleanup()

	mux := handler.RegisterRoutes(database)

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

func connectMongo(cfg *config.Config) (*db.Database, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		slog.Error("MongoDB connection failed", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to MongoDB")

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
