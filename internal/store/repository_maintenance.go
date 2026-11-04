package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/sirupsen/logrus"
)

const (
	cleanupWebsiteTimeout   = 45 * time.Second
	cleanupStatementTimeout = 30 * time.Second
	cleanupBatchSize        = 20000
)

var ErrMaintenanceBusy = errors.New("日志写入或重解析进行中，本轮让出清理任务")

type websiteCleanupProgress struct {
	cutoff        time.Time
	retentionDays int
	step          int
	orphanDim     int
	deleted       int64
}

// repositoryDB 让维护专用连接的 Context/会话超时覆盖所有嵌套清理和重建步骤。
// 正常业务仍使用原连接池，不会继承维护任务的 statement_timeout。
type repositoryDB interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
}

func (r *Repository) database() repositoryDB {
	if r.executor != nil {
		return r.executor
	}
	return r.db
}

type cleanupConnection struct {
	ctx  context.Context
	conn *sql.Conn
}

func (c *cleanupConnection) Exec(q string, args ...interface{}) (sql.Result, error) {
	return c.conn.ExecContext(c.ctx, q, args...)
}
func (c *cleanupConnection) Query(q string, args ...interface{}) (*sql.Rows, error) {
	return c.conn.QueryContext(c.ctx, q, args...)
}
func (c *cleanupConnection) QueryRow(q string, args ...interface{}) *sql.Row {
	return c.conn.QueryRowContext(c.ctx, q, args...)
}
func (c *cleanupConnection) Begin() (*sql.Tx, error) { return c.conn.BeginTx(c.ctx, nil) }

// BeginTx 的 Context 本身不能中断后续无 Context 的 tx.Exec；显式传递到每条重建语句。
type cleanupTx struct {
	*sql.Tx
	ctx context.Context
}

func (tx *cleanupTx) Exec(q string, args ...interface{}) (sql.Result, error) {
	return tx.Tx.ExecContext(tx.ctx, q, args...)
}

func (r *Repository) beginCleanupTx() (*cleanupTx, error) {
	tx, err := r.database().Begin()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	if connection, ok := r.executor.(*cleanupConnection); ok {
		ctx = connection.ctx
	}
	return &cleanupTx{Tx: tx, ctx: ctx}, nil
}

func (r *Repository) withCleanupConnection(ctx context.Context, run func(*Repository) error) error {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// 始终重置后再归还连接池；取消导致连接不可用时丢弃，不能污染正常采集连接。
	defer func() {
		resetCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, resetErr := conn.ExecContext(resetCtx, `RESET statement_timeout; RESET lock_timeout; RESET idle_in_transaction_session_timeout`); resetErr != nil {
			_ = conn.Raw(func(interface{}) error { return driver.ErrBadConn })
		}
	}()
	if _, err := conn.ExecContext(ctx, `SELECT set_config('statement_timeout', $1, false),
		set_config('lock_timeout', '1s', false), set_config('idle_in_transaction_session_timeout', '30s', false)`,
		fmt.Sprintf("%dms", cleanupStatementTimeout.Milliseconds())); err != nil {
		return err
	}
	scoped := &Repository{db: r.db, executor: &cleanupConnection{ctx: ctx, conn: conn}}
	return run(scoped)
}

// CleanOldLogsContext 保留原有保留天数与统计重建口径。清理只使用一个专用连接，
// 每站点/每条语句都有时限，遇到写入则让出；失败返回调度器退避，不再静默成功。
func (r *Repository) CleanOldLogsContext(ctx context.Context) error {
	if !r.cleanupMu.TryLock() {
		return ErrMaintenanceBusy
	}
	defer r.cleanupMu.Unlock()
	retentionDays := config.ReadConfig().System.LogRetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	if r.cleanupPending == nil {
		r.cleanupPending = make(map[string]*websiteCleanupProgress)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT c.relname FROM pg_class c
		JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE n.nspname='public' AND c.relkind IN ('r','p') AND c.relispartition=false
		AND c.relname LIKE '%\_nginx_logs' ESCAPE '\' ORDER BY c.relname`)
	if err != nil {
		return fmt.Errorf("查询清理站点失败: %w", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, name)
	}
	err = rows.Err()
	rows.Close() // 在申请维护连接前释放游标，兼容单连接池。
	if err != nil {
		return err
	}
	for i, table := range tables {
		if table == r.cleanupCursor {
			tables = append(append([]string{}, tables[i+1:]...), tables[:i+1]...)
			break
		}
	}
	var failures []error
	for _, table := range tables {
		if ctx.Err() != nil {
			return errors.Join(append(failures, ctx.Err())...)
		}
		if !r.maintenanceMu.TryLock() {
			if len(failures) > 0 {
				return errors.Join(failures...)
			}
			return ErrMaintenanceBusy
		}
		siteCtx, cancel := context.WithTimeout(ctx, cleanupWebsiteTimeout)
		started := time.Now()
		websiteID := strings.TrimSuffix(table, "_nginx_logs")
		progress := r.cleanupPending[websiteID]
		if progress == nil || progress.retentionDays != retentionDays {
			progress = &websiteCleanupProgress{cutoff: cutoff, retentionDays: retentionDays}
			r.cleanupPending[websiteID] = progress
		}
		logrus.WithFields(logrus.Fields{"website_id": websiteID, "step": progress.step, "orphan_dim": progress.orphanDim, "cutoff": progress.cutoff.Format(time.RFC3339), "timeout": cleanupWebsiteTimeout.String()}).Info("开始清理站点数据")
		err := r.withCleanupConnection(siteCtx, func(scoped *Repository) error {
			return scoped.resumeWebsiteCleanup(websiteID, progress)
		})
		cancel()
		r.maintenanceMu.Unlock()
		r.cleanupCursor = table
		fields := logrus.Fields{"website_id": websiteID, "duration_ms": time.Since(started).Milliseconds(), "timeout": cleanupWebsiteTimeout.String(), "step": progress.step, "orphan_dim": progress.orphanDim, "deleted": progress.deleted}
		if err != nil {
			logrus.WithFields(fields).WithError(err).Warn("站点数据清理未完成，将退避重试")
			failures = append(failures, fmt.Errorf("网站 %s 清理失败: %w", websiteID, err))
		} else {
			delete(r.cleanupPending, websiteID)
			logrus.WithFields(fields).Info("站点数据清理完成")
		}
	}
	return errors.Join(failures...)
}

func (r *Repository) cleanWebsiteLogs(websiteID string, cutoff time.Time) error {
	return r.resumeWebsiteCleanup(websiteID, &websiteCleanupProgress{cutoff: cutoff})
}

func (r *Repository) resumeWebsiteCleanup(websiteID string, progress *websiteCleanupProgress) error {
	r.activeCleanup = progress
	if progress.step == 0 {
		if err := r.cleanExpiredLogBatches(websiteID, progress); err != nil {
			return err
		}
		progress.step = 1
	}
	// 重试继续未完成阶段，避免反复全量回填 first_seen，使后续维表永远得不到时间预算。
	for i, step := range []struct {
		name string
		run  func() error
	}{
		{"aggregates", func() error { return r.cleanupAggregates(websiteID, progress.cutoff) }},
		{"sessions", func() error { return r.cleanupSessions(websiteID, progress.cutoff) }},
		{"orphan_dims", func() error { return r.cleanupOrphanDims(websiteID) }},
	} {
		if progress.step > i+1 {
			continue
		}
		started := time.Now()
		if err := step.run(); err != nil {
			return fmt.Errorf("stage=%s: %w", step.name, err)
		}
		progress.step = i + 2
		logrus.WithFields(logrus.Fields{"website_id": websiteID, "stage": step.name, "duration_ms": time.Since(started).Milliseconds()}).Info("数据清理阶段完成")
	}
	logrus.WithFields(logrus.Fields{"website_id": websiteID, "deleted": progress.deleted, "cutoff": progress.cutoff.Format(time.RFC3339)}).Info("过期数据清理完成")
	return nil
}

func (r *Repository) cleanExpiredLogBatches(websiteID string, progress *websiteCleanupProgress) error {
	table := fmt.Sprintf("%s_nginx_logs", websiteID)
	// 在物理分区内按 ctid 小批删除。直接对分区父表做 tableoid/ctid 联表删除
	// 可能让每个批次都全扫所有分区；这里显式列出叶子表，ctid 不会跨分区误匹配。
	rows, err := r.database().Query(`SELECT c.relname FROM pg_partition_tree($1::regclass) p
		JOIN pg_class c ON c.oid=p.relid JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE p.isleaf AND n.nspname='public'
		UNION SELECT relname FROM pg_class WHERE oid=$1::regclass AND relkind='r'
		ORDER BY relname`, quoteTable(table))
	if err != nil {
		return fmt.Errorf("stage=log_partitions: %w", err)
	}
	var partitions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		partitions = append(partitions, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, partition := range partitions {
		for {
			started := time.Now()
			result, err := r.database().Exec(expiredLogBatchSQL(partition), progress.cutoff.Unix(), cleanupBatchSize)
			if err != nil {
				return fmt.Errorf("stage=logs table=%s: %w", partition, err)
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			progress.deleted += count
			logrus.WithFields(logrus.Fields{"website_id": websiteID, "stage": "logs", "rows": count, "duration_ms": time.Since(started).Milliseconds()}).Debug("清理过期日志批次")
			if count < cleanupBatchSize {
				break
			}
		}
	}
	return nil
}

func expiredLogBatchSQL(table string) string {
	return fmt.Sprintf(`WITH expired AS (
		SELECT ctid FROM ONLY %s WHERE timestamp < $1 ORDER BY timestamp LIMIT $2
	) DELETE FROM ONLY %s AS l USING expired e WHERE l.ctid=e.ctid`, quoteTable(table), quoteTable(table))
}

func quoteTable(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }
