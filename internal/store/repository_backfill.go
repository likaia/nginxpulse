package store

import (
	"fmt"
	"time"

	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/sirupsen/logrus"
)

func (r *Repository) backfillAggregatesIfEmpty(websiteID string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	aggHourly := fmt.Sprintf("%s_agg_hourly", websiteID)

	hasAgg, err := r.tableHasRows(aggHourly)
	if err != nil {
		return err
	}
	if hasAgg {
		return nil
	}

	hasLogs, err := r.tableHasRows(logTable)
	if err != nil || !hasLogs {
		return err
	}

	return r.backfillAggregates(websiteID)
}

func (r *Repository) backfillFirstSeenIfEmpty(websiteID string) error {
	table := fmt.Sprintf("%s_first_seen", websiteID)
	hasFirstSeen, err := r.tableHasRows(table)
	if err != nil {
		return err
	}
	if hasFirstSeen {
		return nil
	}

	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	hasLogs, err := r.tableHasRows(logTable)
	if err != nil || !hasLogs {
		return err
	}

	return r.backfillFirstSeen(websiteID)
}

func (r *Repository) backfillSessionsIfEmpty(websiteID string) error {
	table := fmt.Sprintf("%s_sessions", websiteID)
	hasSessions, err := r.tableHasRows(table)
	if err != nil {
		return err
	}
	if hasSessions {
		return nil
	}

	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	hasLogs, err := r.tableHasRows(logTable)
	if err != nil || !hasLogs {
		return err
	}

	return r.backfillSessions(websiteID)
}

func (r *Repository) backfillSessionAggregatesIfEmpty(websiteID string) error {
	dailyTable := fmt.Sprintf("%s_agg_session_daily", websiteID)
	entryTable := fmt.Sprintf("%s_agg_entry_daily", websiteID)

	hasDaily, err := r.tableHasRows(dailyTable)
	if err != nil {
		return err
	}
	hasEntry, err := r.tableHasRows(entryTable)
	if err != nil {
		return err
	}
	if hasDaily && hasEntry {
		return nil
	}

	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	hasSessions, err := r.tableHasRows(sessionTable)
	if err != nil || !hasSessions {
		return err
	}

	return r.backfillSessionAggregates(websiteID)
}

func (r *Repository) backfillAggregates(websiteID string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	aggHourly := fmt.Sprintf("%s_agg_hourly", websiteID)
	aggHourlyIP := fmt.Sprintf("%s_agg_hourly_ip", websiteID)
	aggDaily := fmt.Sprintf("%s_agg_daily", websiteID)
	aggDailyIP := fmt.Sprintf("%s_agg_daily_ip", websiteID)

	logrus.WithField("website", websiteID).Info("开始回填聚合数据")

	tx, err := r.database().Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, aggHourly)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, aggHourlyIP)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, aggDaily)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, aggDailyIP)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (bucket, pv, traffic, s2xx, s3xx, s4xx, s5xx, other)
         SELECT
             (timestamp / 3600) * 3600 AS bucket,
             SUM(CASE WHEN pageview_flag = 1 THEN 1 ELSE 0 END) AS pv,
             SUM(CASE WHEN pageview_flag = 1 THEN bytes_sent ELSE 0 END) AS traffic,
             SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) AS s2xx,
             SUM(CASE WHEN status_code >= 300 AND status_code < 400 THEN 1 ELSE 0 END) AS s3xx,
             SUM(CASE WHEN status_code >= 400 AND status_code < 500 THEN 1 ELSE 0 END) AS s4xx,
             SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS s5xx,
             SUM(CASE WHEN status_code < 200 OR status_code >= 600 THEN 1 ELSE 0 END) AS other
         FROM "%s"
         GROUP BY bucket`, aggHourly, logTable,
	)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (bucket, ip_id)
         SELECT
             (timestamp / 3600) * 3600 AS bucket,
             ip_id
         FROM "%s"
         WHERE pageview_flag = 1
         GROUP BY bucket, ip_id
         ON CONFLICT DO NOTHING`, aggHourlyIP, logTable,
	)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (day, pv, traffic, s2xx, s3xx, s4xx, s5xx, other)
         SELECT
             date(to_timestamp(timestamp)) AS day,
             SUM(CASE WHEN pageview_flag = 1 THEN 1 ELSE 0 END) AS pv,
             SUM(CASE WHEN pageview_flag = 1 THEN bytes_sent ELSE 0 END) AS traffic,
             SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) AS s2xx,
             SUM(CASE WHEN status_code >= 300 AND status_code < 400 THEN 1 ELSE 0 END) AS s3xx,
             SUM(CASE WHEN status_code >= 400 AND status_code < 500 THEN 1 ELSE 0 END) AS s4xx,
             SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS s5xx,
             SUM(CASE WHEN status_code < 200 OR status_code >= 600 THEN 1 ELSE 0 END) AS other
         FROM "%s"
         GROUP BY day`, aggDaily, logTable,
	)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (day, ip_id)
         SELECT
             date(to_timestamp(timestamp)) AS day,
             ip_id
         FROM "%s"
         WHERE pageview_flag = 1
         GROUP BY day, ip_id
         ON CONFLICT DO NOTHING`, aggDailyIP, logTable,
	)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logrus.WithField("website", websiteID).Info("聚合数据回填完成")
	return nil
}

func (r *Repository) backfillFirstSeen(websiteID string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	firstSeenTable := fmt.Sprintf("%s_first_seen", websiteID)

	logrus.WithField("website", websiteID).Info("开始回填首次访问数据")

	tx, err := r.beginCleanupTx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, firstSeenTable)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (ip_id, first_ts)
         SELECT ip_id, MIN(timestamp)
         FROM "%s"
         WHERE pageview_flag = 1
         GROUP BY ip_id`, firstSeenTable, logTable,
	)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logrus.WithField("website", websiteID).Info("首次访问数据回填完成")
	return nil
}

func (r *Repository) backfillSessions(websiteID string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	stateTable := fmt.Sprintf("%s_session_state", websiteID)

	logrus.WithField("website", websiteID).Info("开始回填会话数据")

	tx, err := r.database().Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, sessionTable)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, stateTable)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`WITH ordered AS (
            SELECT id, ip_id, ua_id, location_id, url_id, timestamp,
                   CASE
                       WHEN LAG(timestamp) OVER (
                           PARTITION BY ip_id, ua_id ORDER BY timestamp, id
                       ) IS NULL
                       OR timestamp - LAG(timestamp) OVER (
                           PARTITION BY ip_id, ua_id ORDER BY timestamp, id
                       ) > %d THEN 1
                       ELSE 0
                   END AS new_session
            FROM "%s"
            WHERE pageview_flag = 1
        ),
        sessions AS (
            SELECT *,
                   SUM(new_session) OVER (
                       PARTITION BY ip_id, ua_id ORDER BY timestamp, id
                       ROWS UNBOUNDED PRECEDING
                   ) AS session_no
            FROM ordered
        ),
        ranked AS (
            SELECT *,
                   ROW_NUMBER() OVER (
                       PARTITION BY ip_id, ua_id, session_no ORDER BY timestamp, id
                   ) AS rn_asc,
                   ROW_NUMBER() OVER (
                       PARTITION BY ip_id, ua_id, session_no ORDER BY timestamp DESC, id DESC
                   ) AS rn_desc
            FROM sessions
        )
        INSERT INTO "%s" (ip_id, ua_id, location_id, start_ts, end_ts, entry_url_id, exit_url_id, page_count)
        SELECT
            ip_id,
            ua_id,
            MAX(CASE WHEN rn_asc = 1 THEN location_id END) AS location_id,
            MIN(timestamp) AS start_ts,
            MAX(timestamp) AS end_ts,
            MAX(CASE WHEN rn_asc = 1 THEN url_id END) AS entry_url_id,
            MAX(CASE WHEN rn_desc = 1 THEN url_id END) AS exit_url_id,
            COUNT(*) AS page_count
        FROM ranked
        GROUP BY ip_id, ua_id, session_no`,
		sessionGapSeconds, logTable, sessionTable,
	)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (ip_id, ua_id, session_id, last_ts)
         SELECT DISTINCT ON (ip_id, ua_id) ip_id, ua_id, id, end_ts
         FROM "%s"
         ORDER BY ip_id, ua_id, end_ts DESC, id DESC
         ON CONFLICT(ip_id, ua_id) DO UPDATE SET
             session_id = excluded.session_id,
             last_ts = excluded.last_ts`,
		stateTable, sessionTable,
	)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logrus.WithField("website", websiteID).Info("会话数据回填完成")
	return nil
}

func (r *Repository) backfillSessionAggregates(websiteID string) error {
	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	dailyTable := fmt.Sprintf("%s_agg_session_daily", websiteID)
	entryTable := fmt.Sprintf("%s_agg_entry_daily", websiteID)

	logrus.WithField("website", websiteID).Info("开始回填会话聚合数据")

	tx, err := r.database().Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, dailyTable)); err != nil {
		return err
	}
	if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, entryTable)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (day, sessions)
         SELECT
             date(to_timestamp(start_ts)) AS day,
             COUNT(*)
         FROM "%s"
         GROUP BY day`, dailyTable, sessionTable,
	)); err != nil {
		return err
	}

	if _, err = tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (day, entry_url_id, count)
         SELECT
             date(to_timestamp(start_ts)) AS day,
             entry_url_id,
             COUNT(*)
        FROM "%s"
        GROUP BY day, entry_url_id`, entryTable, sessionTable,
	)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logrus.WithField("website", websiteID).Info("会话聚合数据回填完成")
	return nil
}

func (r *Repository) cleanupAggregates(websiteID string, cutoff time.Time) error {
	aggHourly := fmt.Sprintf("%s_agg_hourly", websiteID)
	aggHourlyIP := fmt.Sprintf("%s_agg_hourly_ip", websiteID)
	aggDaily := fmt.Sprintf("%s_agg_daily", websiteID)
	aggDailyIP := fmt.Sprintf("%s_agg_daily_ip", websiteID)

	hasAgg, err := r.tableExists(aggHourly)
	if err != nil || !hasAgg {
		return err
	}

	cutoffHour := hourBucket(cutoff)
	cutoffDay := dayBucket(cutoff)

	if _, err := r.database().Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE bucket < ?`, aggHourly)),
		cutoffHour,
	); err != nil {
		return err
	}
	if _, err := r.database().Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE bucket < ?`, aggHourlyIP)),
		cutoffHour,
	); err != nil {
		return err
	}
	if _, err := r.database().Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day < ?`, aggDaily)),
		cutoffDay,
	); err != nil {
		return err
	}
	if _, err := r.database().Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day < ?`, aggDailyIP)),
		cutoffDay,
	); err != nil {
		return err
	}

	if err := r.rebuildHourlyAggregate(websiteID, cutoffHour); err != nil {
		return err
	}
	if err := r.rebuildDailyAggregate(websiteID, cutoffDay); err != nil {
		return err
	}
	return r.rebuildFirstSeen(websiteID)
}

func (r *Repository) cleanupSessions(websiteID string, cutoff time.Time) error {
	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	stateTable := fmt.Sprintf("%s_session_state", websiteID)

	exists, err := r.tableExists(sessionTable)
	if err != nil || !exists {
		return err
	}

	if _, err := r.database().Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE start_ts < ?`, sessionTable)),
		cutoff.Unix(),
	); err != nil {
		return err
	}

	stateExists, err := r.tableExists(stateTable)
	if err != nil || !stateExists {
		return err
	}

	// 不能把清空和回填拆成两个自动提交语句：超时必须保留原 session_state。
	tx, err := r.beginCleanupTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM "%s"`, stateTable)); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf(
		`INSERT INTO "%s" (ip_id, ua_id, session_id, last_ts)
         SELECT DISTINCT ON (ip_id, ua_id) ip_id, ua_id, id, end_ts
         FROM "%s"
         ORDER BY ip_id, ua_id, end_ts DESC, id DESC
         ON CONFLICT(ip_id, ua_id) DO UPDATE SET
             session_id = excluded.session_id,
             last_ts = excluded.last_ts`,
		stateTable, sessionTable,
	)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return r.cleanupSessionAggregates(websiteID, cutoff)
}

func (r *Repository) cleanupSessionAggregates(websiteID string, cutoff time.Time) error {
	dailyTable := fmt.Sprintf("%s_agg_session_daily", websiteID)
	entryTable := fmt.Sprintf("%s_agg_entry_daily", websiteID)

	hasDaily, err := r.tableExists(dailyTable)
	if err != nil || !hasDaily {
		return err
	}

	cutoffDay := dayBucket(cutoff)

	if _, err := r.database().Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day < ?`, dailyTable)),
		cutoffDay,
	); err != nil {
		return err
	}

	hasEntry, err := r.tableExists(entryTable)
	if err != nil {
		return err
	}
	if hasEntry {
		if _, err := r.database().Exec(
			sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day < ?`, entryTable)),
			cutoffDay,
		); err != nil {
			return err
		}
	}

	return r.rebuildSessionAggregatesForDay(websiteID, cutoffDay)
}

func (r *Repository) rebuildSessionAggregatesForDay(websiteID, day string) error {
	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	dailyTable := fmt.Sprintf("%s_agg_session_daily", websiteID)
	entryTable := fmt.Sprintf("%s_agg_entry_daily", websiteID)

	start, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return err
	}
	end := start.Add(24 * time.Hour)

	tx, err := r.beginCleanupTx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day = ?`, dailyTable)),
		day,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`INSERT INTO "%s" (day, sessions)
         SELECT ?, COUNT(*)
         FROM "%s"
         WHERE start_ts >= ? AND start_ts < ?
         HAVING COUNT(*) > 0`, dailyTable, sessionTable,
	)), day, start.Unix(), end.Unix()); err != nil {
		return err
	}

	hasEntry, err := r.tableExists(entryTable)
	if err != nil {
		return err
	}
	if hasEntry {
		if _, err = tx.Exec(
			sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day = ?`, entryTable)),
			day,
		); err != nil {
			return err
		}
		if _, err = tx.Exec(sqlutil.ReplacePlaceholders(fmt.Sprintf(
			`INSERT INTO "%s" (day, entry_url_id, count)
             SELECT ?, entry_url_id, COUNT(*)
             FROM "%s"
             WHERE start_ts >= ? AND start_ts < ?
             GROUP BY entry_url_id`, entryTable, sessionTable,
		)), day, start.Unix(), end.Unix()); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) rebuildHourlyAggregate(websiteID string, bucket int64) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	aggHourly := fmt.Sprintf("%s_agg_hourly", websiteID)
	aggHourlyIP := fmt.Sprintf("%s_agg_hourly_ip", websiteID)

	start := bucket
	end := bucket + 3600

	tx, err := r.beginCleanupTx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE bucket = ?`, aggHourly)),
		bucket,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE bucket = ?`, aggHourlyIP)),
		bucket,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`INSERT INTO "%s" (bucket, pv, traffic, s2xx, s3xx, s4xx, s5xx, other)
         SELECT
             (timestamp / 3600) * 3600 AS bucket,
             SUM(CASE WHEN pageview_flag = 1 THEN 1 ELSE 0 END) AS pv,
             SUM(CASE WHEN pageview_flag = 1 THEN bytes_sent ELSE 0 END) AS traffic,
             SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) AS s2xx,
             SUM(CASE WHEN status_code >= 300 AND status_code < 400 THEN 1 ELSE 0 END) AS s3xx,
             SUM(CASE WHEN status_code >= 400 AND status_code < 500 THEN 1 ELSE 0 END) AS s4xx,
             SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS s5xx,
             SUM(CASE WHEN status_code < 200 OR status_code >= 600 THEN 1 ELSE 0 END) AS other
         FROM "%s"
         WHERE timestamp >= ? AND timestamp < ?
         GROUP BY bucket`, aggHourly, logTable,
	)), start, end); err != nil {
		return err
	}

	if _, err = tx.Exec(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`INSERT INTO "%s" (bucket, ip_id)
         SELECT
             (timestamp / 3600) * 3600 AS bucket,
             ip_id
         FROM "%s"
         WHERE pageview_flag = 1 AND timestamp >= ? AND timestamp < ?
         GROUP BY bucket, ip_id
         ON CONFLICT DO NOTHING`, aggHourlyIP, logTable,
	)), start, end); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) rebuildDailyAggregate(websiteID string, day string) error {
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	aggDaily := fmt.Sprintf("%s_agg_daily", websiteID)
	aggDailyIP := fmt.Sprintf("%s_agg_daily_ip", websiteID)

	start, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return err
	}
	end := start.Add(24 * time.Hour)

	tx, err := r.beginCleanupTx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day = ?`, aggDaily)),
		day,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE day = ?`, aggDailyIP)),
		day,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`INSERT INTO "%s" (day, pv, traffic, s2xx, s3xx, s4xx, s5xx, other)
         SELECT
             date(to_timestamp(timestamp)) AS day,
             SUM(CASE WHEN pageview_flag = 1 THEN 1 ELSE 0 END) AS pv,
             SUM(CASE WHEN pageview_flag = 1 THEN bytes_sent ELSE 0 END) AS traffic,
             SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) AS s2xx,
             SUM(CASE WHEN status_code >= 300 AND status_code < 400 THEN 1 ELSE 0 END) AS s3xx,
             SUM(CASE WHEN status_code >= 400 AND status_code < 500 THEN 1 ELSE 0 END) AS s4xx,
             SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS s5xx,
             SUM(CASE WHEN status_code < 200 OR status_code >= 600 THEN 1 ELSE 0 END) AS other
         FROM "%s"
         WHERE timestamp >= ? AND timestamp < ?
         GROUP BY day`, aggDaily, logTable,
	)), start.Unix(), end.Unix()); err != nil {
		return err
	}

	if _, err = tx.Exec(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`INSERT INTO "%s" (day, ip_id)
         SELECT
             date(to_timestamp(timestamp)) AS day,
             ip_id
         FROM "%s"
         WHERE pageview_flag = 1 AND timestamp >= ? AND timestamp < ?
         GROUP BY day, ip_id
         ON CONFLICT DO NOTHING`, aggDailyIP, logTable,
	)), start.Unix(), end.Unix()); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) clearAggregateTablesForWebsite(websiteID string) error {
	aggTables := []string{
		fmt.Sprintf("%s_agg_hourly", websiteID),
		fmt.Sprintf("%s_agg_hourly_ip", websiteID),
		fmt.Sprintf("%s_agg_daily", websiteID),
		fmt.Sprintf("%s_agg_daily_ip", websiteID),
	}
	for _, table := range aggTables {
		exists, err := r.tableExists(table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := r.database().Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) rebuildFirstSeen(websiteID string) error {
	table := fmt.Sprintf("%s_first_seen", websiteID)
	exists, err := r.tableExists(table)
	if err != nil || !exists {
		return err
	}
	return r.backfillFirstSeen(websiteID)
}
