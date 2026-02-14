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

	// Initialize Sentry
	if cfg.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.SentryEnv,
			TracesSampleRate: cfg.SentryTraceRate,
		})
		if err != nil {
			slog.Error("failed to initialize Sentry", "error", err)
		} else {
			slog.Info("Sentry initialized", "environment", cfg.SentryEnv)
			defer sentry.Flush(2 * time.Second)
		}
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := db.Connect(ctx, cfg.MongoURI)
	if err != nil {
		slog.Error("MongoDB connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Disconnect(context.Background(), client)
	slog.Info("connected to MongoDB")

	database := client.Database(cfg.MongoDatabase)

	// Setup routes
	mux := http.NewServeMux()
	feedbackHandler := handler.NewFeedbackHandler(database)

	mux.HandleFunc("GET /health", handler.HealthCheck)
	mux.HandleFunc("POST /api/v1/feedback", feedbackHandler.Submit)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      middleware.Logging(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		slog.Info("shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("server starting", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
