package worker

import (
	"context"
	"errors"
	"time"

	"github.com/likaia/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

const cleanupRunTimeout = 2 * time.Minute

// 只有这个工作器拥有清理调度状态；触发信号合并，绝不叠加并发任务。
type cleanupWorker struct {
	wake     chan struct{}
	clean    func(context.Context) error
	nextRun  time.Time
	failures int
	now      func() time.Time
}

func newCleanupWorker(clean func(context.Context) error) *cleanupWorker {
	return &cleanupWorker{wake: make(chan struct{}, 1), clean: clean, now: time.Now}
}

func (w *cleanupWorker) trigger() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *cleanupWorker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
			if ctx.Err() != nil {
				return
			}
			w.tryRun(ctx)
		}
	}
}

func (w *cleanupWorker) tryRun(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	now := w.now()
	if now.Before(w.nextRun) {
		return
	}
	started := time.Now()
	logrus.WithFields(logrus.Fields{"timeout": cleanupRunTimeout.String(), "failures": w.failures}).Info("开始后台清理过期数据")
	runCtx, cancel := context.WithTimeout(ctx, cleanupRunTimeout)
	err := w.clean(runCtx)
	// 即使底层误将取消当成成功，也不能将本轮标记为完成。
	if err == nil {
		err = runCtx.Err()
	}
	cancel()
	if ctx.Err() != nil {
		return
	}
	fields := logrus.Fields{"duration_ms": time.Since(started).Milliseconds(), "timeout": cleanupRunTimeout.String()}
	if err != nil {
		delay := time.Minute
		if !errors.Is(err, store.ErrMaintenanceBusy) {
			w.failures++
			delay = cleanupRetryDelay(w.failures)
		}
		w.nextRun = w.now().Add(delay)
		fields["retry_at"] = w.nextRun.Format(time.RFC3339)
		fields["failures"] = w.failures
		logrus.WithFields(fields).WithError(err).Warn("过期数据清理未完成，采集调度继续运行")
		return
	}
	w.failures = 0
	w.nextRun = nextCleanupTime(w.now())
	fields["next_run"] = w.nextRun.Format(time.RFC3339)
	logrus.WithFields(fields).Info("本轮过期数据清理完成")
}

func cleanupRetryDelay(failures int) time.Duration {
	delay := time.Minute
	for i := 1; i < failures && delay < time.Hour; i++ {
		delay *= 2
	}
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func nextCleanupTime(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
