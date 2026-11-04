package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var cleanupTestSequence atomic.Int32

// 只允许显式指定本机的专用测试库；绝不读取项目的生产 DSN。
func maintenanceTestDB(t *testing.T) (*Repository, string) {
	t.Helper()
	dsn := os.Getenv("NGINXPULSE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("set NGINXPULSE_TEST_PG_DSN to run isolated PostgreSQL maintenance tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") || u.Path != "/nginxpulse_cleanup_test" {
		t.Fatal("maintenance integration tests require localhost/nginxpulse_cleanup_test")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	r := &Repository{db: db}
	site := fmt.Sprintf("cleanup_test_%d", cleanupTestSequence.Add(1))
	t.Cleanup(func() {
		for _, suffix := range []string{"nginx_logs", "dim_ip", "dim_url", "dim_ua", "dim_referer", "dim_location", "first_seen", "sessions", "session_state", "agg_hourly", "agg_hourly_ip", "agg_daily", "agg_daily_ip", "agg_session_daily", "agg_entry_daily"} {
			if _, err := db.Exec("DROP TABLE IF EXISTS " + quoteTable(site+"_"+suffix) + " CASCADE"); err != nil {
				t.Errorf("fixture cleanup: %v", err)
			}
		}
		db.Close()
	})
	if err := createLogTable(db, site+"_nginx_logs"); err != nil {
		t.Fatal(err)
	}
	for _, create := range []func(sqlExecer, string) error{createDimTables, createLogIndexes, createAggTables, createFirstSeenTable, createSessionTables, createSessionAggTables} {
		if err := create(db, site); err != nil {
			t.Fatal(err)
		}
	}
	return r, site
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}
func scalarInt(t *testing.T, db *sql.DB, query string, args ...interface{}) int64 {
	t.Helper()
	var value int64
	if err := db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func seedMaintenanceLogs(t *testing.T, r *Repository, site string, cutoff time.Time) {
	t.Helper()
	records := []NginxLogRecord{}
	for i, stamp := range []time.Time{cutoff.Add(-time.Hour), cutoff.Add(time.Hour), time.Now()} {
		records = append(records, NginxLogRecord{IP: "192.0.2.1", Timestamp: stamp, Method: "GET", Url: fmt.Sprintf("/page-%d", i), Status: 200, BytesSent: 100, PageviewFlag: 1, Referer: "-", UserBrowser: "Chrome", UserOs: "iOS", UserDevice: "Mobile", DomesticLocation: "test", GlobalLocation: "test"})
	}
	if err := r.BatchInsertLogsForWebsite(site, records); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCleanupRetainsBoundaryStatsAndRecentData(t *testing.T) {
	r, site := maintenanceTestDB(t)
	cutoff := time.Now().AddDate(0, 0, -30).Truncate(time.Second)
	seedMaintenanceLogs(t, r, site, cutoff)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.withCleanupConnection(ctx, func(scoped *Repository) error { return scoped.cleanWebsiteLogs(site, cutoff) }); err != nil {
		t.Fatal(err)
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s_nginx_logs"`, site)); got != 2 {
		t.Fatalf("remaining logs=%d", got)
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT SUM(pv) FROM "%s_agg_daily"`, site)); got != 2 {
		t.Fatalf("remaining daily PV=%d", got)
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT SUM(pv) FROM "%s_agg_hourly"`, site)); got != 2 {
		t.Fatalf("remaining hourly PV=%d", got)
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT MIN(first_ts) FROM "%s_first_seen"`, site)); got != cutoff.Add(time.Hour).Unix() {
		t.Fatalf("first seen=%d", got)
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s_sessions" WHERE start_ts<$1`, site), cutoff.Unix()); got != 0 {
		t.Fatal("expired session retained")
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s_dim_url" WHERE url='/page-0'`, site)); got != 0 {
		t.Fatal("unreferenced expired dimension retained")
	}
	// 重跑时日志删除数为0，也应能修复上轮超时遗留的聚合数据。
	mustExec(t, r.db, fmt.Sprintf(`UPDATE "%s_agg_daily" SET pv=999`, site))
	if err := r.withCleanupConnection(ctx, func(scoped *Repository) error { return scoped.cleanWebsiteLogs(site, cutoff) }); err != nil {
		t.Fatal(err)
	}
	// 边界日必须回到实际日志计数，非边界日不会被不必要重建。
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT pv FROM "%s_agg_daily" WHERE day=$1`, site), dayBucket(cutoff)); got != 1 {
		t.Fatalf("boundary repair PV=%d", got)
	}
}

func TestPostgresOrphanCleanupPreservesSessionOnlyReferencesAndNulls(t *testing.T) {
	r, site := maintenanceTestDB(t)
	for _, dim := range []string{"ip", "url", "referer", "ua", "location"} {
		query := ""
		switch dim {
		case "ip":
			query = fmt.Sprintf(`INSERT INTO "%s_dim_ip"(id,ip) VALUES(1,'192.0.2.1'),(2,'192.0.2.2')`, site)
		case "url":
			query = fmt.Sprintf(`INSERT INTO "%s_dim_url"(id,url) VALUES(1,'/entry'),(2,'/exit'),(3,'/orphan')`, site)
		case "referer":
			query = fmt.Sprintf(`INSERT INTO "%s_dim_referer"(id,referer) VALUES(1,'-')`, site)
		case "ua":
			query = fmt.Sprintf(`INSERT INTO "%s_dim_ua"(id,browser,os,device) VALUES(1,'Chrome','iOS','Mobile')`, site)
		case "location":
			query = fmt.Sprintf(`INSERT INTO "%s_dim_location"(id,domestic,global) VALUES(1,'test','test')`, site)
		}
		mustExec(t, r.db, query)
	}
	mustExec(t, r.db, fmt.Sprintf(`INSERT INTO "%s_sessions"(ip_id,ua_id,location_id,start_ts,end_ts,entry_url_id,exit_url_id) VALUES(1,1,1,1,2,1,2)`, site))
	if err := r.cleanupOrphanDims(site); err != nil {
		t.Fatal(err)
	}
	for _, dim := range []string{"ip", "ua", "location"} {
		if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s_dim_%s" WHERE id=1`, site, dim)); got != 1 {
			t.Fatalf("session's %s was deleted", dim)
		}
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s_dim_url"`, site)); got != 2 {
		t.Fatalf("entry/exit URL refs not retained: %d", got)
	}
	// NULL 子查询值不能像 NOT IN 一样阻止孤儿删除。
	log := site + "_null_log"
	dim := site + "_null_dim"
	mustExec(t, r.db, "CREATE TABLE "+quoteTable(log)+"(ip_id BIGINT)")
	mustExec(t, r.db, "CREATE TABLE "+quoteTable(dim)+"(id BIGINT)")
	defer r.db.Exec("DROP TABLE " + quoteTable(log) + "," + quoteTable(dim))
	mustExec(t, r.db, "INSERT INTO "+quoteTable(log)+" VALUES(NULL),(1)")
	mustExec(t, r.db, "INSERT INTO "+quoteTable(dim)+" VALUES(1),(2)")
	mustExec(t, r.db, orphanDimsSQL(dim, log, "ip_id", nil))
	if got := scalarInt(t, r.db, "SELECT COUNT(*) FROM "+quoteTable(dim)); got != 1 {
		t.Fatalf("NULL anti join kept orphan: %d", got)
	}
}

func TestPostgresCleanupDeadlineAndConnectionSettingsDoNotLeak(t *testing.T) {
	r, _ := maintenanceTestDB(t)
	r.db.SetMaxOpenConns(1)
	r.db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := r.withCleanupConnection(ctx, func(scoped *Repository) error { _, err := scoped.database().Exec(`SELECT pg_sleep(2)`); return err })
	if err == nil || time.Since(started) > time.Second {
		t.Fatalf("deadline not respected: %v elapsed=%s", err, time.Since(started))
	}
	// 服务端主动超时也必须恢复连接设置，不污染同池后续采集。
	err = r.withCleanupConnection(context.Background(), func(scoped *Repository) error {
		if _, err := scoped.database().Exec(`SET statement_timeout='50ms'`); err != nil {
			return err
		}
		_, err := scoped.database().Exec(`SELECT pg_sleep(0.2)`)
		return err
	})
	if !isSQLState(err, "57014") {
		t.Fatalf("server timeout=%v", err)
	}
	var setting string
	if err := r.db.QueryRow(`SHOW statement_timeout`).Scan(&setting); err != nil || setting != "0" {
		t.Fatalf("leaked statement timeout %q: %v", setting, err)
	}
	mustExec(t, r.db, `SELECT pg_sleep(0.1)`)
}

func TestPostgresCleanupRollbackKeepsSessionStateOnTimeout(t *testing.T) {
	r, site := maintenanceTestDB(t)
	cutoff := time.Now().AddDate(0, 0, -30)
	seedMaintenanceLogs(t, r, site, cutoff)
	before := scalarInt(t, r.db, fmt.Sprintf(`SELECT SUM(session_id) FROM "%s_session_state"`, site))
	fn := site + "_slow_state"
	mustExec(t, r.db, fmt.Sprintf(`CREATE FUNCTION "%s"() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(2); RETURN NEW; END $$`, fn))
	defer r.db.Exec("DROP FUNCTION " + quoteTable(fn) + "() CASCADE")
	mustExec(t, r.db, fmt.Sprintf(`CREATE TRIGGER slow_state BEFORE INSERT ON "%s_session_state" FOR EACH ROW EXECUTE FUNCTION "%s"()`, site, fn))
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := r.withCleanupConnection(ctx, func(scoped *Repository) error { return scoped.cleanupSessions(site, cutoff) })
	if err == nil {
		t.Fatal("slow rebuild unexpectedly completed")
	}
	if time.Since(started) > time.Second {
		t.Fatal("transaction statement ignored cleanup Context")
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT SUM(session_id) FROM "%s_session_state"`, site)); got != before {
		t.Fatalf("state was cleared by failed rebuild: %d != %d", got, before)
	}
}

func TestPostgresCleanupYieldsToIngestionAndReleasesGateAfterFailure(t *testing.T) {
	r, site := maintenanceTestDB(t)
	seedMaintenanceLogs(t, r, site, time.Now().AddDate(0, 0, -400))
	r.maintenanceMu.RLock()
	err := r.CleanOldLogsContext(context.Background())
	r.maintenanceMu.RUnlock()
	if !errors.Is(err, ErrMaintenanceBusy) {
		t.Fatalf("did not yield to write: %v", err)
	}
	fn := site + "_slow_delete"
	mustExec(t, r.db, fmt.Sprintf(`CREATE FUNCTION "%s"() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(2); RETURN OLD; END $$`, fn))
	defer r.db.Exec("DROP FUNCTION " + quoteTable(fn) + "() CASCADE")
	mustExec(t, r.db, fmt.Sprintf(`CREATE TRIGGER slow_delete BEFORE DELETE ON "%s_nginx_logs_default" FOR EACH ROW EXECUTE FUNCTION "%s"()`, site, fn))
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err = r.CleanOldLogsContext(ctx)
	if err == nil || !strings.Contains(err.Error(), "logs") {
		t.Fatalf("failure hidden: %v", err)
	}
	if !r.maintenanceMu.TryLock() {
		t.Fatal("cleanup retained write gate after timeout")
	}
	r.maintenanceMu.Unlock()
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s_nginx_logs"`, site)); got != 3 {
		t.Fatalf("canceled delete partially committed: %d", got)
	}
	if err := r.BatchInsertLogsForWebsite(site, []NginxLogRecord{{IP: "192.0.2.5", Timestamp: time.Now(), Method: "GET", Url: "/after-timeout", Status: 200, PageviewFlag: 1}}); err != nil {
		t.Fatalf("ingestion failed after canceled cleanup: %v", err)
	}
}

func TestPostgresExpiredBatchesStayInsidePhysicalPartitions(t *testing.T) {
	r, site := maintenanceTestDB(t)
	cutoff := time.Now().AddDate(0, 0, -30).Truncate(time.Second)
	oldPartition := site + "_nginx_logs_old"
	mustExec(t, r.db, fmt.Sprintf(`CREATE TABLE %s PARTITION OF %s FOR VALUES FROM (MINVALUE) TO (%d)`, quoteTable(oldPartition), quoteTable(site+"_nginx_logs"), cutoff.Unix()))
	seedMaintenanceLogs(t, r, site, cutoff)
	// 两个物理分区会重复使用相同 ctid；不能因为删除旧分区行而误删新分区行。
	oldCTID, newCTID := "", ""
	if err := r.db.QueryRow("SELECT ctid::text FROM " + quoteTable(oldPartition) + " LIMIT 1").Scan(&oldCTID); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow("SELECT ctid::text FROM " + quoteTable(site+"_nginx_logs_default") + " LIMIT 1").Scan(&newCTID); err != nil {
		t.Fatal(err)
	}
	if oldCTID != newCTID {
		t.Fatal("fixture must exercise matching ctids across partitions")
	}
	if err := r.withCleanupConnection(context.Background(), func(scoped *Repository) error { return scoped.cleanWebsiteLogs(site, cutoff) }); err != nil {
		t.Fatal(err)
	}
	if got := scalarInt(t, r.db, "SELECT COUNT(*) FROM "+quoteTable(oldPartition)); got != 0 {
		t.Fatal("old partition not cleaned")
	}
	if got := scalarInt(t, r.db, "SELECT COUNT(*) FROM "+quoteTable(site+"_nginx_logs_default")); got != 2 {
		t.Fatal("recent partition rows deleted")
	}
}

func TestPostgresOrphanQueryAvoidsMaterializedNotInAtLowWorkMem(t *testing.T) {
	r, site := maintenanceTestDB(t)
	dim, logs := site+"_plan_dim", site+"_plan_logs"
	mustExec(t, r.db, "CREATE TABLE "+quoteTable(dim)+" (id BIGINT PRIMARY KEY)")
	mustExec(t, r.db, "CREATE TABLE "+quoteTable(logs)+" (ip_id BIGINT)")
	defer r.db.Exec("DROP TABLE " + quoteTable(dim) + "," + quoteTable(logs))
	mustExec(t, r.db, "INSERT INTO "+quoteTable(dim)+" SELECT generate_series(1,10001)")
	mustExec(t, r.db, "INSERT INTO "+quoteTable(logs)+" SELECT 1+(g%10000) FROM generate_series(1,350000) g")
	mustExec(t, r.db, "ANALYZE "+quoteTable(dim))
	mustExec(t, r.db, "ANALYZE "+quoteTable(logs))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := r.withCleanupConnection(ctx, func(scoped *Repository) error {
		if _, err := scoped.database().Exec(`SET work_mem='64kB'`); err != nil {
			return err
		}
		defer scoped.database().Exec(`RESET work_mem`)
		rows, err := scoped.database().Query("EXPLAIN " + orphanDimsSQL(dim, logs, "ip_id", nil))
		if err != nil {
			return err
		}
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				return err
			}
			plan.WriteString(line)
			plan.WriteByte('\n')
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if !strings.Contains(plan.String(), "Anti Join") || strings.Contains(plan.String(), "SubPlan") {
			return fmt.Errorf("not an anti join: %s", plan.String())
		}
		started := time.Now()
		_, err = scoped.database().Exec(orphanDimsSQL(dim, logs, "ip_id", nil))
		t.Logf("350,000 log rows, 10,001 dimensions, 64kB work_mem: orphan cleanup=%s", time.Since(started))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := scalarInt(t, r.db, "SELECT COUNT(*) FROM "+quoteTable(dim)); got != 10000 {
		t.Fatalf("incorrect orphan count: %d", got)
	}
}

func TestPostgresCleanupResumesFailedStageWithoutRepeatingCompletedRebuilds(t *testing.T) {
	r, site := maintenanceTestDB(t)
	cutoff := time.Now().AddDate(0, 0, -30).Truncate(time.Second)
	seedMaintenanceLogs(t, r, site, cutoff)
	fn := site + "_fail_stage"
	mustExec(t, r.db, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected cleanup failure'; END $$`, quoteTable(fn)))
	defer r.db.Exec("DROP FUNCTION " + quoteTable(fn) + "() CASCADE")
	mustExec(t, r.db, fmt.Sprintf(`CREATE TRIGGER fail_orphan BEFORE DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s()`, quoteTable(site+"_dim_url"), quoteTable(fn)))
	progress := &websiteCleanupProgress{cutoff: cutoff}
	run := func() error {
		return r.withCleanupConnection(context.Background(), func(scoped *Repository) error { return scoped.resumeWebsiteCleanup(site, progress) })
	}
	if err := run(); err == nil || !strings.Contains(err.Error(), "orphan_dims") {
		t.Fatalf("expected last stage failure, got %v", err)
	}
	if progress.step != 3 || progress.orphanDim != 1 {
		t.Fatalf("checkpoint lost: %+v", progress)
	}
	if got := scalarInt(t, r.db, "SELECT COUNT(*) FROM "+quoteTable(site+"_nginx_logs")); got != 2 {
		t.Fatal("expired log batch not committed")
	}
	mustExec(t, r.db, "DROP TRIGGER fail_orphan ON "+quoteTable(site+"_dim_url"))
	// 如果重试又去全量回填 first_seen，这个触发器会使测试失败。
	mustExec(t, r.db, fmt.Sprintf(`CREATE TRIGGER fail_rebuild BEFORE DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s()`, quoteTable(site+"_first_seen"), quoteTable(fn)))
	if err := run(); err != nil {
		t.Fatalf("retry repeated completed stage: %v", err)
	}
	if progress.step != 4 || progress.deleted != 1 || progress.orphanDim != 5 {
		t.Fatalf("incorrect completed progress: %+v", progress)
	}
}

func TestPostgresCleanupRepairsDerivedDataAfterCommittedDeleteAndFailedRebuild(t *testing.T) {
	r, site := maintenanceTestDB(t)
	cutoff := time.Now().AddDate(0, 0, -30).Truncate(time.Second)
	seedMaintenanceLogs(t, r, site, cutoff)
	before := scalarInt(t, r.db, fmt.Sprintf(`SELECT MIN(first_ts) FROM "%s_first_seen"`, site))
	fn := site + "_fail_firstseen"
	mustExec(t, r.db, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected first_seen failure'; END $$`, quoteTable(fn)))
	defer r.db.Exec("DROP FUNCTION " + quoteTable(fn) + "() CASCADE")
	mustExec(t, r.db, fmt.Sprintf(`CREATE TRIGGER fail_firstseen BEFORE INSERT ON %s FOR EACH ROW EXECUTE FUNCTION %s()`, quoteTable(site+"_first_seen"), quoteTable(fn)))
	progress := &websiteCleanupProgress{cutoff: cutoff}
	run := func() error {
		return r.withCleanupConnection(context.Background(), func(scoped *Repository) error { return scoped.resumeWebsiteCleanup(site, progress) })
	}
	if err := run(); err == nil || !strings.Contains(err.Error(), "aggregates") {
		t.Fatalf("rebuild failure hidden: %v", err)
	}
	if progress.step != 1 {
		t.Fatalf("failed rebuild incorrectly marked complete: %+v", progress)
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT MIN(first_ts) FROM "%s_first_seen"`, site)); got != before {
		t.Fatal("failed first_seen rebuild was not rolled back")
	}
	if got := scalarInt(t, r.db, "SELECT COUNT(*) FROM "+quoteTable(site+"_nginx_logs")); got != 2 {
		t.Fatal("expected already committed log deletion")
	}
	mustExec(t, r.db, "DROP TRIGGER fail_firstseen ON "+quoteTable(site+"_first_seen"))
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if progress.deleted != 1 {
		t.Fatal("retry double-counted deletion")
	}
	if got := scalarInt(t, r.db, fmt.Sprintf(`SELECT MIN(first_ts) FROM "%s_first_seen"`, site)); got != cutoff.Add(time.Hour).Unix() {
		t.Fatalf("retry did not repair first_seen: %d", got)
	}
}
