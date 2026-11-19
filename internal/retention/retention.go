// Package retention deletes feedback older than the configured retention
// period (RETENTION_MONTHS). The privacy policy promises feedback is deleted
// automatically 12 months after it is received.
package retention

import (
	"context"
	"log/slog"
	"time"
)

// Interval is how often the cleanup runs after the initial run at startup.
const Interval = 24 * time.Hour

// Deleter is the storage operation the cleanup needs (implemented by *db.Database).
type Deleter interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

// Cutoff returns the oldest received_at that is still retained: now minus
// months calendar months, in UTC.
func Cutoff(now time.Time, months int) time.Time {
	return now.UTC().AddDate(0, -months, 0)
}

// Run deletes expired feedback once immediately and then every interval until
// ctx is cancelled (graceful shutdown). months <= 0 disables the cleanup. now
// supplies the current time (time.Now in production; fixed in tests).
func Run(ctx context.Context, d Deleter, months int, interval time.Duration, now func() time.Time) {
	if months <= 0 {
		slog.Warn("retention cleanup disabled (RETENTION_MONTHS=0) — feedback is kept indefinitely")
		return
	}
	slog.Info("retention cleanup enabled", "months", months, "interval", interval.String())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		runOnce(ctx, d, months, now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func runOnce(ctx context.Context, d Deleter, months int, now time.Time) {
	cutoff := Cutoff(now, months)
	n, err := d.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		if ctx.Err() != nil {
			return // shutting down; the next start will retry
		}
		slog.Error("retention cleanup failed", "cutoff", cutoff.Format(time.RFC3339), "error", err)
		return
	}
	slog.Info("retention cleanup", "deleted", n, "cutoff", cutoff.Format(time.RFC3339))
}
