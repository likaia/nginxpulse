package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/sirupsen/logrus"
)

func (r *Repository) cleanupOrphanDims(websiteID string) error {
	if r.executor == nil {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupWebsiteTimeout)
		defer cancel()
		return r.withCleanupConnection(ctx, func(scoped *Repository) error { return scoped.cleanupOrphanDims(websiteID) })
	}
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	hasIPID, err := r.tableHasColumn(logTable, "ip_id")
	if err != nil || !hasIPID {
		return err
	}

	type dimSpec struct {
		table  string
		column string
		refs   map[string][]string
	}
	dims := []dimSpec{
		{table: fmt.Sprintf("%s_dim_ip", websiteID), column: "ip_id", refs: map[string][]string{
			"first_seen": {"ip_id"}, "agg_hourly_ip": {"ip_id"}, "agg_daily_ip": {"ip_id"}, "sessions": {"ip_id"}, "session_state": {"ip_id"},
		}},
		{table: fmt.Sprintf("%s_dim_url", websiteID), column: "url_id", refs: map[string][]string{"sessions": {"entry_url_id", "exit_url_id"}, "agg_entry_daily": {"entry_url_id"}}},
		{table: fmt.Sprintf("%s_dim_referer", websiteID), column: "referer_id"},
		{table: fmt.Sprintf("%s_dim_ua", websiteID), column: "ua_id", refs: map[string][]string{"sessions": {"ua_id"}, "session_state": {"ua_id"}}},
		{table: fmt.Sprintf("%s_dim_location", websiteID), column: "location_id", refs: map[string][]string{"sessions": {"location_id"}}},
	}

	for i, dim := range dims {
		if r.activeCleanup != nil && i < r.activeCleanup.orphanDim {
			continue
		}
		exists, err := r.tableExists(dim.table)
		if err != nil {
			return err
		}
		if !exists {
			if r.activeCleanup != nil {
				r.activeCleanup.orphanDim = i + 1
			}
			continue
		}
		var refs []string
		for suffix, columns := range dim.refs {
			table := fmt.Sprintf("%s_%s", websiteID, suffix)
			ok, err := r.tableExists(table)
			if err != nil {
				return err
			}
			if ok {
				for _, column := range columns {
					refs = append(refs, fmt.Sprintf(`NOT EXISTS (SELECT 1 FROM %s AS ref WHERE ref.%s = d.id)`, quoteTable(table), column))
				}
			}
		}
		sort.Strings(refs)
		started := time.Now()
		result, err := r.database().Exec(orphanDimsSQL(dim.table, logTable, dim.column, refs))
		if err != nil {
			return fmt.Errorf("table=%s: %w", dim.table, err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		logrus.WithFields(logrus.Fields{"website_id": websiteID, "table": dim.table, "rows": count, "duration_ms": time.Since(started).Milliseconds()}).Info("清理维表孤儿数据完成")
		if r.activeCleanup != nil {
			r.activeCleanup.orphanDim = i + 1
		}
	}
	return nil
}

func orphanDimsSQL(dimTable, logTable, column string, refs []string) string {
	predicates := []string{fmt.Sprintf(`NOT EXISTS (SELECT 1 FROM %s AS l WHERE l.%s = d.id)`, quoteTable(logTable), column)}
	predicates = append(predicates, refs...)
	return fmt.Sprintf(`DELETE FROM %s AS d WHERE %s`, quoteTable(dimTable), strings.Join(predicates, " AND "))
}

func (r *Repository) clearDimTablesForWebsite(websiteID string) error {
	dimTables := []string{
		fmt.Sprintf("%s_dim_ip", websiteID),
		fmt.Sprintf("%s_dim_url", websiteID),
		fmt.Sprintf("%s_dim_referer", websiteID),
		fmt.Sprintf("%s_dim_ua", websiteID),
		fmt.Sprintf("%s_dim_location", websiteID),
	}
	for _, table := range dimTables {
		exists, err := r.tableExists(table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := r.db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) clearFirstSeenForWebsite(websiteID string) error {
	table := fmt.Sprintf("%s_first_seen", websiteID)
	exists, err := r.tableExists(table)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := r.db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err != nil {
		return err
	}
	return nil
}

func (r *Repository) clearSessionTablesForWebsite(websiteID string) error {
	tables := []string{
		fmt.Sprintf("%s_sessions", websiteID),
		fmt.Sprintf("%s_session_state", websiteID),
	}
	for _, table := range tables {
		exists, err := r.tableExists(table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := r.db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) clearSessionAggTablesForWebsite(websiteID string) error {
	tables := []string{
		fmt.Sprintf("%s_agg_session_daily", websiteID),
		fmt.Sprintf("%s_agg_entry_daily", websiteID),
	}
	for _, table := range tables {
		exists, err := r.tableExists(table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := r.db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ensureWebsiteSchema(websiteID string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	exists, err := r.tableExists(logTable)
	if err != nil {
		return err
	}

	if !exists {
		if err := createDimTables(r.db, websiteID); err != nil {
			return err
		}
		if err := createLogTable(r.db, logTable); err != nil {
			return err
		}
		if err := createLogIndexes(r.db, websiteID); err != nil {
			return err
		}
		if err := createAggTables(r.db, websiteID); err != nil {
			return err
		}
		if err := createFirstSeenTable(r.db, websiteID); err != nil {
			return err
		}
		if err := createSessionTables(r.db, websiteID); err != nil {
			return err
		}
		if err := createSessionAggTables(r.db, websiteID); err != nil {
			return err
		}
		if err := r.backfillAggregatesIfEmpty(websiteID); err != nil {
			return err
		}
		if err := r.backfillFirstSeenIfEmpty(websiteID); err != nil {
			return err
		}
		if err := r.backfillSessionsIfEmpty(websiteID); err != nil {
			return err
		}
		return r.backfillSessionAggregatesIfEmpty(websiteID)
	}

	hasIPID, err := r.tableHasColumn(logTable, "ip_id")
	if err != nil {
		return err
	}

	if !hasIPID {
		return r.migrateLegacyLogs(websiteID)
	}

	if err := createDimTables(r.db, websiteID); err != nil {
		return err
	}
	if err := r.ensureLogTraceColumns(logTable); err != nil {
		return err
	}
	if err := createLogIndexes(r.db, websiteID); err != nil {
		return err
	}
	if err := createAggTables(r.db, websiteID); err != nil {
		return err
	}
	if err := createFirstSeenTable(r.db, websiteID); err != nil {
		return err
	}
	if err := createSessionTables(r.db, websiteID); err != nil {
		return err
	}
	if err := createSessionAggTables(r.db, websiteID); err != nil {
		return err
	}
	if err := r.backfillAggregatesIfEmpty(websiteID); err != nil {
		return err
	}
	if err := r.backfillFirstSeenIfEmpty(websiteID); err != nil {
		return err
	}
	if err := r.backfillSessionsIfEmpty(websiteID); err != nil {
		return err
	}
	return r.backfillSessionAggregatesIfEmpty(websiteID)
}

func (r *Repository) EnsureWebsiteSchema(websiteID string) error {
	return r.ensureWebsiteSchema(websiteID)
}

func (r *Repository) migrateLegacyLogs(websiteID string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	newLogTable := fmt.Sprintf("%s_nginx_logs_new", websiteID)

	logrus.WithField("website", websiteID).Info("检测到旧日志表结构，开始迁移")

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, newLogTable)); err != nil {
		return err
	}
	if err := createDimTables(tx, websiteID); err != nil {
		return err
	}
	if err := createLogTable(tx, newLogTable); err != nil {
		return err
	}
	if err := createAggTables(tx, websiteID); err != nil {
		return err
	}
	if err := createFirstSeenTable(tx, websiteID); err != nil {
		return err
	}
	if err := createSessionTables(tx, websiteID); err != nil {
		return err
	}
	if err := createSessionAggTables(tx, websiteID); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s_dim_ip"(ip) SELECT DISTINCT ip FROM "%s" ON CONFLICT DO NOTHING`,
		websiteID, logTable,
	)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s_dim_url"(url) SELECT DISTINCT url FROM "%s" ON CONFLICT DO NOTHING`,
		websiteID, logTable,
	)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s_dim_referer"(referer) SELECT DISTINCT referer FROM "%s" ON CONFLICT DO NOTHING`,
		websiteID, logTable,
	)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s_dim_ua"(browser, os, device)
         SELECT DISTINCT user_browser, user_os, user_device FROM "%s"
         ON CONFLICT DO NOTHING`,
		websiteID, logTable,
	)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s_dim_location"(domestic, global)
         SELECT DISTINCT domestic_location, global_location FROM "%s"
         ON CONFLICT DO NOTHING`,
		websiteID, logTable,
	)); err != nil {
		return err
	}

	_, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s"(
            ip_id, pageview_flag, timestamp, method, url_id,
            status_code, bytes_sent, referer_id, ua_id, location_id
        )
        SELECT
            ip.id, l.pageview_flag, l.timestamp, l.method, url.id,
            l.status_code, l.bytes_sent, ref.id, ua.id, loc.id
        FROM "%s" l
        JOIN "%s_dim_ip" ip ON ip.ip = l.ip
        JOIN "%s_dim_url" url ON url.url = l.url
        JOIN "%s_dim_referer" ref ON ref.referer = l.referer
        JOIN "%s_dim_ua" ua
            ON ua.browser = l.user_browser AND ua.os = l.user_os AND ua.device = l.user_device
        JOIN "%s_dim_location" loc
            ON loc.domestic = l.domestic_location AND loc.global = l.global_location`,
		newLogTable, logTable,
		websiteID, websiteID, websiteID, websiteID, websiteID,
	))
	if err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(`DROP TABLE "%s"`, logTable)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s"`, newLogTable, logTable)); err != nil {
		return err
	}
	if err := createLogIndexes(tx, websiteID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := r.backfillAggregates(websiteID); err != nil {
		return err
	}
	if err := r.backfillFirstSeen(websiteID); err != nil {
		return err
	}
	if err := r.backfillSessions(websiteID); err != nil {
		return err
	}
	if err := r.backfillSessionAggregates(websiteID); err != nil {
		return err
	}

	logrus.WithField("website", websiteID).Info("旧日志表迁移完成")
	return nil
}

func (r *Repository) tableExists(tableName string) (bool, error) {
	row := r.database().QueryRow(sqlutil.ReplacePlaceholders(
		`SELECT 1
         FROM pg_class c
         JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'public'
           AND c.relkind IN ('r', 'p')
           AND c.relname = ?`,
	), tableName)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) tableHasRows(tableName string) (bool, error) {
	exists, err := r.tableExists(tableName)
	if err != nil || !exists {
		return false, err
	}
	row := r.db.QueryRow(fmt.Sprintf(`SELECT 1 FROM "%s" LIMIT 1`, tableName))
	var value int
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) tableHasColumn(tableName, columnName string) (bool, error) {
	rows, err := r.database().Query(sqlutil.ReplacePlaceholders(
		`SELECT 1
         FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = ? AND column_name = ?
         LIMIT 1`,
	), tableName, columnName)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		return true, nil
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func createDimTables(execer sqlExecer, websiteID string) error {
	stmts := []string{
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_dim_ip" (
                id BIGSERIAL PRIMARY KEY,
                ip TEXT NOT NULL UNIQUE
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_dim_url" (
                id BIGSERIAL PRIMARY KEY,
                url TEXT NOT NULL UNIQUE
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_dim_referer" (
                id BIGSERIAL PRIMARY KEY,
                referer TEXT NOT NULL UNIQUE
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_dim_ua" (
                id BIGSERIAL PRIMARY KEY,
                browser TEXT NOT NULL,
                os TEXT NOT NULL,
                device TEXT NOT NULL,
                UNIQUE(browser, os, device)
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_dim_location" (
                id BIGSERIAL PRIMARY KEY,
                domestic TEXT NOT NULL,
                global TEXT NOT NULL,
                UNIQUE(domestic, global)
            )`, websiteID,
		),
	}

	for _, stmt := range stmts {
		if _, err := execer.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func createLogTable(execer sqlExecer, tableName string) error {
	stmt := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS "%s" (
            id BIGSERIAL NOT NULL,
            ip_id BIGINT NOT NULL,
            pageview_flag SMALLINT NOT NULL DEFAULT 0,
            timestamp BIGINT NOT NULL,
            method TEXT NOT NULL,
            url_id BIGINT NOT NULL,
            status_code INT NOT NULL,
            bytes_sent BIGINT NOT NULL,
            request_length BIGINT NOT NULL DEFAULT 0,
            request_time_ms BIGINT NOT NULL DEFAULT 0,
            upstream_response_time_ms BIGINT NOT NULL DEFAULT 0,
            upstream_addr TEXT NOT NULL DEFAULT '',
            host TEXT NOT NULL DEFAULT '',
            request_id TEXT NOT NULL DEFAULT '',
            fingerprint TEXT,
            referer_id BIGINT NOT NULL,
            ua_id BIGINT NOT NULL,
            location_id BIGINT NOT NULL,
            PRIMARY KEY (id, timestamp)
        ) PARTITION BY RANGE (timestamp)`, tableName,
	)
	_, err := execer.Exec(stmt)
	if err != nil {
		return err
	}
	partition := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS "%s_default" PARTITION OF "%s" DEFAULT`,
		tableName, tableName,
	)
	_, err = execer.Exec(partition)
	return err
}

func (r *Repository) ensureLogTraceColumns(tableName string) error {
	type columnSpec struct {
		name       string
		definition string
	}
	columns := []columnSpec{
		{name: "request_length", definition: `BIGINT NOT NULL DEFAULT 0`},
		{name: "request_time_ms", definition: `BIGINT NOT NULL DEFAULT 0`},
		{name: "upstream_response_time_ms", definition: `BIGINT NOT NULL DEFAULT 0`},
		{name: "upstream_addr", definition: `TEXT NOT NULL DEFAULT ''`},
		{name: "host", definition: `TEXT NOT NULL DEFAULT ''`},
		{name: "request_id", definition: `TEXT NOT NULL DEFAULT ''`},
		{name: "fingerprint", definition: `TEXT`},
	}

	for _, column := range columns {
		hasColumn, err := r.tableHasColumn(tableName, column.name)
		if err != nil {
			return err
		}
		if hasColumn {
			continue
		}
		if _, err := r.db.Exec(fmt.Sprintf(
			`ALTER TABLE "%s" ADD COLUMN %s %s`,
			tableName, column.name, column.definition,
		)); err != nil {
			return err
		}
	}
	return nil
}

func createLogIndexes(execer sqlExecer, websiteID string) error {
	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)
	stmts := []string{
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON "%s"(timestamp)`,
			websiteID, tableName,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_pv_ts_ip ON "%s"(timestamp, ip_id) WHERE pageview_flag = 1`,
			websiteID, tableName,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_session_key ON "%s"(ip_id, ua_id, timestamp) WHERE pageview_flag = 1`,
			websiteID, tableName,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_ip_location ON "%s"(ip_id, location_id)`,
			websiteID, tableName,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_location_ip ON "%s"(location_id, ip_id)`,
			websiteID, tableName,
		),
		fmt.Sprintf(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_log_fingerprint ON "%s"(timestamp, fingerprint)`,
			websiteID, tableName,
		),
	}
	for _, stmt := range stmts {
		if _, err := execer.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func createAggTables(execer sqlExecer, websiteID string) error {
	stmts := []string{
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_agg_hourly" (
                bucket BIGINT PRIMARY KEY,
                pv BIGINT NOT NULL DEFAULT 0,
                traffic BIGINT NOT NULL DEFAULT 0,
                s2xx BIGINT NOT NULL DEFAULT 0,
                s3xx BIGINT NOT NULL DEFAULT 0,
                s4xx BIGINT NOT NULL DEFAULT 0,
                s5xx BIGINT NOT NULL DEFAULT 0,
                other BIGINT NOT NULL DEFAULT 0
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_agg_hourly_ip" (
                bucket BIGINT NOT NULL,
                ip_id BIGINT NOT NULL,
                PRIMARY KEY(bucket, ip_id)
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_agg_daily" (
                day DATE PRIMARY KEY,
                pv BIGINT NOT NULL DEFAULT 0,
                traffic BIGINT NOT NULL DEFAULT 0,
                s2xx BIGINT NOT NULL DEFAULT 0,
                s3xx BIGINT NOT NULL DEFAULT 0,
                s4xx BIGINT NOT NULL DEFAULT 0,
                s5xx BIGINT NOT NULL DEFAULT 0,
                other BIGINT NOT NULL DEFAULT 0
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_agg_daily_ip" (
                day DATE NOT NULL,
                ip_id BIGINT NOT NULL,
                PRIMARY KEY(day, ip_id)
            )`, websiteID,
		),
	}

	for _, stmt := range stmts {
		if _, err := execer.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func createFirstSeenTable(execer sqlExecer, websiteID string) error {
	stmt := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS "%s_first_seen" (
            ip_id BIGINT PRIMARY KEY,
            first_ts BIGINT NOT NULL
        )`, websiteID,
	)
	_, err := execer.Exec(stmt)
	return err
}

func createSessionTables(execer sqlExecer, websiteID string) error {
	stmts := []string{
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_sessions" (
                id BIGSERIAL PRIMARY KEY,
                ip_id BIGINT NOT NULL,
                ua_id BIGINT NOT NULL,
                location_id BIGINT NOT NULL,
                start_ts BIGINT NOT NULL,
                end_ts BIGINT NOT NULL,
                entry_url_id BIGINT NOT NULL,
                exit_url_id BIGINT NOT NULL,
                page_count INT NOT NULL DEFAULT 1
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_sessions_start ON "%s_sessions"(start_ts)`,
			websiteID, websiteID,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_sessions_start_entry ON "%s_sessions"(start_ts, entry_url_id)`,
			websiteID, websiteID,
		),
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_sessions_key ON "%s_sessions"(ip_id, ua_id, end_ts)`,
			websiteID, websiteID,
		),
		// 支持 IP 归属地回填：UPDATE ... WHERE ip_id = ? AND location_id = ?
		fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS idx_%s_sessions_ip_loc ON "%s_sessions"(ip_id, location_id)`,
			websiteID, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_session_state" (
                ip_id BIGINT NOT NULL,
                ua_id BIGINT NOT NULL,
                session_id BIGINT NOT NULL,
                last_ts BIGINT NOT NULL,
                PRIMARY KEY(ip_id, ua_id)
            )`, websiteID,
		),
	}
	for _, stmt := range stmts {
		if _, err := execer.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func createSessionAggTables(execer sqlExecer, websiteID string) error {
	stmts := []string{
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_agg_session_daily" (
                day DATE PRIMARY KEY,
                sessions BIGINT NOT NULL DEFAULT 0
            )`, websiteID,
		),
		fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_agg_entry_daily" (
                day DATE NOT NULL,
                entry_url_id BIGINT NOT NULL,
                count BIGINT NOT NULL DEFAULT 0,
                PRIMARY KEY(day, entry_url_id)
            )`, websiteID,
		),
	}
	for _, stmt := range stmts {
		if _, err := execer.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
