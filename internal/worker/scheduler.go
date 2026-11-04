package worker

import (
	"context"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/ingest"
	"github.com/likaia/nginxpulse/internal/logging"
	"github.com/sirupsen/logrus"
)

// InitialScan performs an initial log scan after startup.
func InitialScan(parser *ingest.LogParser, interval time.Duration) {
	logrus.Info("****** 2 初始扫描 ******")
	ExecutePeriodicTasks(parser, interval)
}

// RunScheduler executes periodic tasks on a ticker until ctx is canceled.
func RunScheduler(ctx context.Context, parser *ingest.LogParser, interval time.Duration) {
	runScheduler(ctx, parser, interval)
}

type periodicParser interface {
	ScanNginxLogs() []ingest.ParserResult
	BackfillHistory(time.Duration, int64) ingest.BackfillResult
	ProcessPendingIPGeo(int) int
	CleanOldLogs(context.Context) error
}

func runScheduler(ctx context.Context, parser periodicParser, interval time.Duration) {
	if ctx.Err() != nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	cleanup := newCleanupWorker(parser.CleanOldLogs)
	cleanupCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); cleanup.run(cleanupCtx) }()
	defer func() { cancel(); <-done }()
	// 初次扫描和后续扫描由同一调度器串行发起，避免启动扫描尚未完成又启动第二轮。
	logrus.Info("****** 2 初始扫描 ******")
	executePeriodicTasks(parser, interval)
	cleanup.trigger()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	iteration := 0

	for {
		select {
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			iteration++
			logrus.WithFields(logrus.Fields{"iteration": iteration}).Info("定期任务开始")
			executePeriodicTasks(parser, interval)
			cleanup.trigger()
		case <-ctx.Done():
			return
		}
	}
}

// ExecutePeriodicTasks 优先采集；数据清理由独立的、可取消且有退避的工作器执行。
func ExecutePeriodicTasks(parser *ingest.LogParser, interval time.Duration) {
	executePeriodicTasks(parser, interval)
}

func executePeriodicTasks(parser periodicParser, interval time.Duration) {
	{ // 1 日志轮转
		if err := logging.RotateLogFile(); err != nil {
			logrus.WithError(err).Warn("日志轮转失败")
		}
	}

	{ // 3 Nginx日志扫描
		startTime := time.Now()
		results := parser.ScanNginxLogs()
		totalDuration := time.Since(startTime)

		totalEntries := 0
		successCount := 0

		for _, result := range results {
			if result.WebName == "" {
				continue
			}

			totalEntries += result.TotalEntries

			if result.Success {
				successCount++
				if result.TotalEntries > 0 {
					logrus.Infof("网站 %s (%s) 扫描完成: %d 条记录, 耗时 %.2fs",
						result.WebName, result.WebID, result.TotalEntries, result.Duration.Seconds())
				}
			} else {
				logrus.Warnf("网站 %s (%s) 扫描失败: %s",
					result.WebName, result.WebID, result.Error)
			}
		}

		if totalEntries > 0 {
			logrus.Infof("Nginx日志扫描完成: %d/%d 个站点成功, 共 %d 条记录, 总耗时 %.2fs",
				successCount, len(results), totalEntries, totalDuration.Seconds())
		}
	}

	{ // 4 历史日志回填
		backfillDuration, backfillBytes := backfillBudget(interval)
		backfillResult := parser.BackfillHistory(backfillDuration, backfillBytes)
		if backfillResult.ProcessedBytes > 0 {
			logrus.Infof("历史日志回填完成: %d 条记录, %.2f MB",
				backfillResult.ProcessedEntries,
				float64(backfillResult.ProcessedBytes)/(1024*1024))
		}
	}

	{ // 5 IP 归属地回填
		processed := parser.ProcessPendingIPGeo(0)
		if processed > 0 {
			logrus.Infof("IP 归属地回填完成: %d 个 IP", processed)
		}
	}
}

func backfillBudget(interval time.Duration) (time.Duration, int64) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	duration := interval / 3
	if duration < 500*time.Millisecond {
		duration = 500 * time.Millisecond
	}
	maxDuration := config.GetBackfillMaxDurationPerRun()
	if duration > maxDuration {
		duration = maxDuration
	}
	return duration, config.GetBackfillMaxBytesPerRun()
}
