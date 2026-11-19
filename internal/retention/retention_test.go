package retention

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeDeleter struct {
	mu      sync.Mutex
	cutoffs []time.Time
	calls   chan struct{}
}

func newFake() *fakeDeleter { return &fakeDeleter{calls: make(chan struct{}, 100)} }

func (f *fakeDeleter) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	f.mu.Lock()
	f.cutoffs = append(f.cutoffs, cutoff)
	f.mu.Unlock()
	f.calls <- struct{}{}
	return 0, nil
}

var fixedNow = time.Date(2026, 11, 19, 12, 30, 0, 0, time.UTC)

func TestCutoff_IsCalendarMonthsBackInUTC(t *testing.T) {
	helsinki := time.FixedZone("EET", 2*3600)
	got := Cutoff(fixedNow.In(helsinki), 12)
	want := time.Date(2025, 11, 19, 12, 30, 0, 0, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("Cutoff = %v, want %v (UTC)", got, want)
	}
}

func TestRun_DisabledNeverDeletes(t *testing.T) {
	f := newFake()
	done := make(chan struct{})
	go func() {
		Run(context.Background(), f, 0, time.Millisecond, func() time.Time { return fixedNow })
		close(done)
	}()
	select {
	case <-done: // returns immediately when disabled
	case <-time.After(time.Second):
		t.Fatal("Run with months=0 did not return")
	}
	if len(f.cutoffs) != 0 {
		t.Errorf("disabled cleanup deleted %d times", len(f.cutoffs))
	}
}

func TestRun_RunsAtStartupThenPeriodicallyUntilCancelled(t *testing.T) {
	f := newFake()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Run(ctx, f, 12, 5*time.Millisecond, func() time.Time { return fixedNow })
		close(done)
	}()

	for i := 0; i < 3; i++ { // startup run + at least two ticks
		select {
		case <-f.calls:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected run %d", i+1)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	want := time.Date(2025, 11, 19, 12, 30, 0, 0, time.UTC)
	if !f.cutoffs[0].Equal(want) {
		t.Errorf("cutoff = %v, want %v", f.cutoffs[0], want)
	}
}
