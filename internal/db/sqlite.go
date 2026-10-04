package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)

	"github.com/idefinity/nps-api/internal/crypto"
	"github.com/idefinity/nps-api/internal/model"
)

// Database wraps a SQLite connection pool and the field cipher used to encrypt
// PII columns (comment, timezone) at rest.
type Database struct {
	db  *sql.DB
	enc *crypto.Cipher
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
// schema. The file (and its -wal/-shm siblings) are created if absent. enc is the
// field cipher for PII columns; pass a passthrough cipher to disable encryption.
func Connect(ctx context.Context, path string, enc *crypto.Cipher) (*Database, error) {
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

	return &Database{db: sqlDB, enc: enc}, nil
}

// InsertFeedback stores one feedback row and sets fb.ID to the new row id. The
// comment and timezone fields are encrypted at rest; the caller's struct is not
// mutated (fb.Comment/fb.Timezone stay plaintext).
func (d *Database) InsertFeedback(ctx context.Context, fb *model.Feedback) error {
	encComment, err := d.enc.Encrypt(fb.Comment)
	if err != nil {
		return fmt.Errorf("encrypt comment: %w", err)
	}
	encTimezone, err := d.enc.Encrypt(fb.Timezone)
	if err != nil {
		return fmt.Errorf("encrypt timezone: %w", err)
	}

	const q = `INSERT INTO feedback
        (schema_version, app, app_version, platform, timestamp,
         nps_rating, nps_category, timezone, comment, received_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := d.db.ExecContext(ctx, q,
		fb.SchemaVersion, fb.App, fb.AppVersion, fb.Platform, fb.Timestamp,
		fb.NPSRating, fb.NPSCategory, encTimezone, encComment,
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

// ListFeedback returns rows oldest-first, decrypting the comment and timezone
// fields. A year > 0 restricts results to that calendar year by received_at (the
// server-authoritative UTC receipt time); year <= 0 returns all rows.
func (d *Database) ListFeedback(ctx context.Context, year int) ([]model.Feedback, error) {
	q := `SELECT id, schema_version, app, app_version, platform, timestamp,
        nps_rating, nps_category, timezone, comment, received_at
        FROM feedback`
	var args []any

	if year > 0 {
		// received_at is stored as RFC3339Nano UTC (…Z), so a lexicographic range
		// over the ISO-8601 string selects the calendar year correctly.
		lo := fmt.Sprintf("%04d-01-01T00:00:00Z", year)
		hi := fmt.Sprintf("%04d-01-01T00:00:00Z", year+1)
		q += ` WHERE received_at >= ? AND received_at < ?`
		args = append(args, lo, hi)
	}
	q += ` ORDER BY id`

	rows, err := d.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Feedback
	for rows.Next() {
		var (
			fb         model.Feedback
			tz         string
			comment    string
			receivedAt string
		)
		if err := rows.Scan(
			&fb.ID, &fb.SchemaVersion, &fb.App, &fb.AppVersion, &fb.Platform,
			&fb.Timestamp, &fb.NPSRating, &fb.NPSCategory, &tz, &comment, &receivedAt,
		); err != nil {
			return nil, err
		}

		if fb.Timezone, err = d.enc.Decrypt(tz); err != nil {
			return nil, fmt.Errorf("decrypt timezone (id %d): %w", fb.ID, err)
		}
		if fb.Comment, err = d.enc.Decrypt(comment); err != nil {
			return nil, fmt.Errorf("decrypt comment (id %d): %w", fb.ID, err)
		}
		fb.ReceivedAt, _ = time.Parse(time.RFC3339Nano, receivedAt)

		out = append(out, fb)
	}
	return out, rows.Err()
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
