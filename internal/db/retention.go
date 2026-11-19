package db

import (
	"context"
	"fmt"
	"time"
)

// DeleteOlderThan permanently deletes feedback received strictly before cutoff
// and returns the number of rows removed. A row received exactly at cutoff is
// kept.
//
// received_at is stored as RFC3339Nano UTC, which drops trailing fractional
// zeros ("…:00Z" vs "…:00.3Z"), so a plain string comparison can misorder two
// times within the same second. julianday() parses both sides into real
// timestamps and compares them correctly; a value it cannot parse yields NULL,
// so such a row is never deleted by mistake.
func (d *Database) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := d.db.ExecContext(ctx,
		`DELETE FROM feedback WHERE julianday(received_at) < julianday(?)`,
		cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("delete expired feedback: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count deleted feedback: %w", err)
	}
	return n, nil
}
