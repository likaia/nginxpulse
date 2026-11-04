package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/likaia/nginxpulse/internal/ingest"
	"github.com/likaia/nginxpulse/internal/store"
)

func TestCleanupBackoffAndSuccessSchedule(t *testing.T) {
	now := time.Date(2026, 11, 4, 15, 0, 0, 0, time.FixedZone("CST", 8*3600))
	calls := 0
	failure := errors.New("statement timeout")
	w := newCleanupWorker(func(ctx context.Context) error {
		calls++
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > cleanupRunTimeout {
			t.Fatal("missing cleanup budget")
		}
		return failure
	})
	w.now = func() time.Time { return now }
	for i, want := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 16 * time.Minute, 32 * time.Minute, time.Hour, time.Hour} {
		w.tryRun(context.Background())
		if w.nextRun.Sub(now) != want {
			t.Fatalf("failure %d retry=%s, want %s", i+1, w.nextRun.Sub(now), want)
		}
		previous := calls
		w.tryRun(context.Background())
		if calls != previous {
			t.Fatal("retried before backoff elapsed")
		}
		now = w.nextRun
	}
	failure = nil
	w.tryRun(context.Background())
	if w.failures != 0 || !w.nextRun.Equal(nextCleanupTime(now)) {
		t.Fatal("success did not reset backoff/schedule")
	}
	w.tryRun(context.Background())
	if calls != 9 {
		t.Fatal("successful cleanup ran again before next schedule")
	}
}

func TestCleanupBusyYieldsWithoutIncreasingFailureBackoff(t *testing.T) {
	now := time.Now()
	w := newCleanupWorker(func(context.Context) error { return store.ErrMaintenanceBusy })
	w.now = func() time.Time { return now }
	w.tryRun(context.Background())
	if w.failures != 0 || w.nextRun.Sub(now) != time.Minute {
		t.Fatal("ingestion contention should yield, not count as SQL failure")
	}
}

func TestNextCleanupTimeAcrossMidnightAndTwoAM(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	for _, tc := range []struct{ hour, minute, day int }{{0, 0, 4}, {1, 59, 4}, {2, 0, 5}, {23, 59, 5}} {
		now := time.Date(2026, 11, 4, tc.hour, tc.minute, 0, 0, zone)
		want := time.Date(2026, 11, tc.day, 2, 0, 0, 0, zone)
		if got := nextCleanupTime(now); !got.Equal(want) {
			t.Fatalf("at %s next=%s want=%s", now, got, want)
		}
	}
}

type schedulerTestParser struct {
	scans      atomic.Int32
	cleanCalls atomic.Int32
	clean      func(context.Context) error
}

func (p *schedulerTestParser) ScanNginxLogs() []ingest.ParserResult { p.scans.Add(1); return nil }
func (*schedulerTestParser) BackfillHistory(time.Duration, int64) ingest.BackfillResult {
	return ingest.BackfillResult{}
}
func (*schedulerTestParser) ProcessPendingIPGeo(int) int { return 0 }
func (p *schedulerTestParser) CleanOldLogs(ctx context.Context) error {
	p.cleanCalls.Add(1)
	return p.clean(ctx)
}

func TestBlockedCleanupDoesNotStopSchedulerOrOverlapAndCancelsOnShutdown(t *testing.T) {
	started := make(chan struct{})
	ended := make(chan struct{})
	p := &schedulerTestParser{}
	p.clean = func(ctx context.Context) error {
		if p.scans.Load() == 0 {
			t.Error("cleanup ran before the initial scan")
		}
		close(started)
		<-ctx.Done()
		close(ended)
		return ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); runScheduler(ctx, p, 5*time.Millisecond) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup not started")
	}
	deadline := time.Now().Add(2 * time.Second)
	for p.scans.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if p.scans.Load() < 5 {
		t.Fatal("blocked cleanup stopped subsequent scans")
	}
	if p.cleanCalls.Load() != 1 {
		t.Fatal("cleanup jobs overlapped")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup prevented shutdown")
	}
	select {
	case <-ended:
	default:
		t.Fatal("cleanup context not canceled")
	}
}

func TestSchedulerDoesNotRunAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &schedulerTestParser{}
	runScheduler(ctx, p, time.Millisecond)
	if p.scans.Load() != 0 || p.cleanCalls.Load() != 0 {
		t.Fatal("canceled scheduler still started work")
	}
}

func TestCleanupTriggerCoalescesAndCanceledRunDoesNotMarkSuccess(t *testing.T) {
	w := newCleanupWorker(func(ctx context.Context) error { return nil })
	for i := 0; i < 100; i++ {
		w.trigger()
	}
	if len(w.wake) != 1 {
		t.Fatal("cleanup trigger queue grew")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.tryRun(ctx)
	if !w.nextRun.IsZero() {
		t.Fatal("canceled run marked complete")
	}
}
