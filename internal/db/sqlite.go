package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)

	"github.com/idefinity/nps-api/internal/model"
)

// Database wraps a SQLite connection pool.
type Database struct {
	db *sql.DB
}

// schema is applied on every start; CREATE TABLE IF NOT EXISTS makes it
// idempotent, so first boot creates the table and later boots are no-ops.
const schema = `
CREATE TABLE IF NOT EXISTS feedback (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    schema_version TEXT    NOT NULL,
    app            TEXT    NOT NULL,
    app_version    TEXT    NOT NULL,
    platform       TEXT    NOT NULL,
    timestamp      TEXT    NOT NULL,
    nps_rating     INTEGER NOT NULL,
    nps_category   TEXT    NOT NULL,
    timezone       TEXT,
    comment        TEXT,
    received_at    TEXT    NOT NULL
);`

// Connect opens the SQLite database at path, verifies it, and applies the
// schema. The file (and its -wal/-shm siblings) are created if absent.
func Connect(ctx context.Context, path string) (*Database, error) {
	if path == "" {
		return nil, fmt.Errorf("DB_PATH is not set")
	}

	// WAL improves concurrent read/write; busy_timeout makes brief write locks
	// wait rather than erroring; foreign_keys on for correctness.
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite allows a single writer; capping the pool at one connection keeps
	// writes serialized and avoids "database is locked" under concurrency.
	sqlDB.SetMaxOpenConns(1)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := sqlDB.ExecContext(ctx, schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	return &Database{db: sqlDB}, nil
}

// InsertFeedback stores one feedback row and sets fb.ID to the new row id.
func (d *Database) InsertFeedback(ctx context.Context, fb *model.Feedback) error {
	const q = `INSERT INTO feedback
        (schema_version, app, app_version, platform, timestamp,
         nps_rating, nps_category, timezone, comment, received_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := d.db.ExecContext(ctx, q,
		fb.SchemaVersion, fb.App, fb.AppVersion, fb.Platform, fb.Timestamp,
		fb.NPSRating, fb.NPSCategory, fb.Timezone, fb.Comment,
		fb.ReceivedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}

	if id, err := res.LastInsertId(); err == nil {
		fb.ID = id
	}
	return nil
}

// Ping verifies the database is reachable.
func (d *Database) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// DB returns the underlying *sql.DB, for health checks and tests.
func (d *Database) DB() *sql.DB {
	return d.db
}

// Close releases the connection pool.
func (d *Database) Close(ctx context.Context) error {
	return d.db.Close()
}
