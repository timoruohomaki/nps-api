package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// migrations upgrade the schema step by step. migrations[i] moves the database
// from PRAGMA user_version i to i+1; each runs in its own transaction together
// with the version bump, so a failed step leaves the database untouched. Append
// new steps; never edit or reorder applied ones.
var migrations = []func(ctx context.Context, tx *sql.Tx) error{
	migrateNullableConsentFields, // 0 → 1
}

// migrate applies every pending migration. It is called from Connect after the
// base schema (version 0) is ensured, so fresh and existing databases converge.
func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	if version > len(migrations) {
		return fmt.Errorf("database schema version %d is newer than this binary supports (%d)",
			version, len(migrations))
	}

	for v := version; v < len(migrations); v++ {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("migration %d: begin: %w", v+1, err)
		}
		if err := migrations[v](ctx, tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", v+1, err)
		}
		// PRAGMA does not accept bound parameters; v+1 is an int we control.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: set user_version: %w", v+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d: commit: %w", v+1, err)
		}
		slog.Info("database migrated", "schema_version", v+1)
	}
	return nil
}

// migrateNullableConsentFields drops NOT NULL from app_version and platform
// (payload schema 1.1 sends them only with user consent). SQLite cannot alter a
// column constraint, so the table is rebuilt and all rows copied, keeping ids
// and the AUTOINCREMENT counter. Empty strings become NULL.
func migrateNullableConsentFields(ctx context.Context, tx *sql.Tx) error {
	var seq sql.NullInt64
	err := tx.QueryRowContext(ctx,
		`SELECT seq FROM sqlite_sequence WHERE name = 'feedback'`).Scan(&seq)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read sequence: %w", err)
	}

	stmts := []string{
		`CREATE TABLE feedback_new (
            id             INTEGER PRIMARY KEY AUTOINCREMENT,
            schema_version TEXT    NOT NULL,
            app            TEXT    NOT NULL,
            app_version    TEXT,
            platform       TEXT,
            timestamp      TEXT    NOT NULL,
            nps_rating     INTEGER NOT NULL,
            nps_category   TEXT    NOT NULL,
            timezone       TEXT,
            comment        TEXT,
            received_at    TEXT    NOT NULL
        )`,
		`INSERT INTO feedback_new
            (id, schema_version, app, app_version, platform, timestamp,
             nps_rating, nps_category, timezone, comment, received_at)
         SELECT id, schema_version, app, NULLIF(app_version, ''), NULLIF(platform, ''),
             timestamp, nps_rating, nps_category, timezone, comment, received_at
         FROM feedback`,
		`DROP TABLE feedback`,
		`ALTER TABLE feedback_new RENAME TO feedback`,
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}

	if !seq.Valid {
		return nil
	}
	// Copying explicit ids leaves seq at MAX(id); restore the old counter so ids
	// of deleted trailing rows are never reused.
	res, err := tx.ExecContext(ctx,
		`UPDATE sqlite_sequence SET seq = MAX(seq, ?) WHERE name = 'feedback'`, seq.Int64)
	if err == nil {
		if n, _ := res.RowsAffected(); n == 0 {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO sqlite_sequence (name, seq) VALUES ('feedback', ?)`, seq.Int64)
		}
	}
	if err != nil {
		return fmt.Errorf("restore sequence: %w", err)
	}
	return nil
}
