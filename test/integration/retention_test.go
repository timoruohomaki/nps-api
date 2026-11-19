package integration

import (
	"context"
	"testing"
	"time"

	"github.com/idefinity/nps-api/internal/model"
	"github.com/idefinity/nps-api/internal/retention"
)

func TestDeleteOlderThan_RemovesOnlyExpiredRows(t *testing.T) {
	ctx := context.Background()
	_, database := newTestServer(t)

	now := time.Date(2026, 11, 19, 12, 0, 0, 300_000_000, time.UTC) // .3s on purpose
	cutoff := retention.Cutoff(now, 12)                             // 2025-11-19T12:00:00.3Z

	rows := []struct {
		name       string
		receivedAt time.Time
		kept       bool
	}{
		{"two years old", cutoff.AddDate(-1, 0, 0), false},
		{"one day before cutoff", cutoff.Add(-24 * time.Hour), false},
		{"1ms before cutoff", cutoff.Add(-time.Millisecond), false},
		// Same second as the cutoff but earlier: stored as "…12:00:00Z", which
		// sorts AFTER "…12:00:00.3Z" as a string. Must still be deleted.
		{"whole second before cutoff fraction", cutoff.Truncate(time.Second), false},
		{"exactly at cutoff", cutoff, true},
		{"1ms after cutoff", cutoff.Add(time.Millisecond), true},
		{"recent", now.Add(-time.Hour), true},
	}
	for _, r := range rows {
		fb := &model.Feedback{SchemaVersion: "1.1", App: "rokdsk", Timestamp: "2026-01-01T00:00:00Z",
			NPSRating: 8, NPSCategory: "passive", Comment: r.name, ReceivedAt: r.receivedAt}
		if err := database.InsertFeedback(ctx, fb); err != nil {
			t.Fatalf("insert %q: %v", r.name, err)
		}
	}

	n, err := database.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if n != 4 {
		t.Errorf("deleted %d rows, want 4", n)
	}

	items, err := database.ListFeedback(ctx, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	left := map[string]bool{}
	for _, it := range items {
		left[it.Comment] = true
	}
	for _, r := range rows {
		if left[r.name] != r.kept {
			t.Errorf("%s: kept=%v, want %v", r.name, left[r.name], r.kept)
		}
	}

	// A second run deletes nothing more.
	if n, _ := database.DeleteOlderThan(ctx, cutoff); n != 0 {
		t.Errorf("second run deleted %d rows, want 0", n)
	}
}
