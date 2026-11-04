package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/sirupsen/logrus"
)

func isChinaGlobalLabel(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if trimmed == "中国" {
		return true
	}
	return strings.EqualFold(trimmed, "china")
}

var ipGeoISPKeywords = []string{
	"电信", "联通", "移动", "铁通", "广电", "网通", "教育网", "长城宽带", "有线", "鹏博士",
}

var ipGeoIDCKeywords = []string{
	"机房", "idc", "data center", "datacenter", "数据中心",
	"腾讯", "腾讯云",
	"阿里", "阿里云",
	"百度", "百度云",
	"华为", "华为云",
	"京东", "京东云",
	"字节", "火山", "火山引擎",
	"金山", "金山云",
	"青云", "ucloud",
	"aws", "azure", "gcp",
}

var chinaProvincePatterns = []struct {
	pattern  string
	province string
}{
	{pattern: "北京市", province: "北京"},
	{pattern: "天津市", province: "天津"},
	{pattern: "上海市", province: "上海"},
	{pattern: "重庆市", province: "重庆"},
	{pattern: "河北省", province: "河北"},
	{pattern: "山西省", province: "山西"},
	{pattern: "辽宁省", province: "辽宁"},
	{pattern: "吉林省", province: "吉林"},
	{pattern: "黑龙江省", province: "黑龙江"},
	{pattern: "江苏省", province: "江苏"},
	{pattern: "浙江省", province: "浙江"},
	{pattern: "安徽省", province: "安徽"},
	{pattern: "福建省", province: "福建"},
	{pattern: "江西省", province: "江西"},
	{pattern: "山东省", province: "山东"},
	{pattern: "河南省", province: "河南"},
	{pattern: "湖北省", province: "湖北"},
	{pattern: "湖南省", province: "湖南"},
	{pattern: "广东省", province: "广东"},
	{pattern: "海南省", province: "海南"},
	{pattern: "四川省", province: "四川"},
	{pattern: "贵州省", province: "贵州"},
	{pattern: "云南省", province: "云南"},
	{pattern: "陕西省", province: "陕西"},
	{pattern: "甘肃省", province: "甘肃"},
	{pattern: "青海省", province: "青海"},
	{pattern: "台湾省", province: "台湾"},
	{pattern: "内蒙古自治区", province: "内蒙古"},
	{pattern: "广西壮族自治区", province: "广西"},
	{pattern: "西藏自治区", province: "西藏"},
	{pattern: "宁夏回族自治区", province: "宁夏"},
	{pattern: "新疆维吾尔自治区", province: "新疆"},
	{pattern: "香港特别行政区", province: "香港"},
	{pattern: "澳门特别行政区", province: "澳门"},
	{pattern: "北京", province: "北京"},
	{pattern: "天津", province: "天津"},
	{pattern: "上海", province: "上海"},
	{pattern: "重庆", province: "重庆"},
	{pattern: "河北", province: "河北"},
	{pattern: "山西", province: "山西"},
	{pattern: "辽宁", province: "辽宁"},
	{pattern: "吉林", province: "吉林"},
	{pattern: "黑龙江", province: "黑龙江"},
	{pattern: "江苏", province: "江苏"},
	{pattern: "浙江", province: "浙江"},
	{pattern: "安徽", province: "安徽"},
	{pattern: "福建", province: "福建"},
	{pattern: "江西", province: "江西"},
	{pattern: "山东", province: "山东"},
	{pattern: "河南", province: "河南"},
	{pattern: "湖北", province: "湖北"},
	{pattern: "湖南", province: "湖南"},
	{pattern: "广东", province: "广东"},
	{pattern: "海南", province: "海南"},
	{pattern: "四川", province: "四川"},
	{pattern: "贵州", province: "贵州"},
	{pattern: "云南", province: "云南"},
	{pattern: "陕西", province: "陕西"},
	{pattern: "甘肃", province: "甘肃"},
	{pattern: "青海", province: "青海"},
	{pattern: "台湾", province: "台湾"},
	{pattern: "内蒙古", province: "内蒙古"},
	{pattern: "内蒙", province: "内蒙古"},
	{pattern: "广西", province: "广西"},
	{pattern: "西藏", province: "西藏"},
	{pattern: "宁夏", province: "宁夏"},
	{pattern: "新疆", province: "新疆"},
	{pattern: "香港", province: "香港"},
	{pattern: "澳门", province: "澳门"},
}

func normalizeIPGeoLocation(domestic, global string) (string, string) {
	domestic = strings.TrimSpace(domestic)
	global = strings.TrimSpace(global)
	if domestic == "" || !isChinaGlobalLabel(global) {
		return domestic, global
	}

	province := extractChinaProvince(domestic)
	idc := isChinaIDCLabel(domestic)

	cleaned := stripIPGeoISPKeywords(domestic)
	if cleaned != "" && province == "" {
		province = extractChinaProvince(cleaned)
	}
	if cleaned != "" && isChinaIDCLabel(cleaned) {
		idc = true
	}
	if idc {
		if province != "" {
			return province, global
		}
		return "机房", global
	}
	if cleaned == "" {
		cleaned = "中国"
	}
	return cleaned, global
}

func stripIPGeoISPKeywords(domestic string) string {
	parts := strings.Split(domestic, "·")
	cleanedParts := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		for _, keyword := range ipGeoISPKeywords {
			part = strings.ReplaceAll(part, keyword, "")
		}
		part = strings.TrimSpace(strings.Trim(part, "-_/|,，;；()（）[]{}"))
		if part == "" || part == "0" || part == "未知" {
			continue
		}
		if isISPKeyword(part) {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		cleanedParts = append(cleanedParts, part)
	}
	return strings.Join(cleanedParts, "·")
}

func isChinaIDCLabel(value string) bool {
	clean := strings.ToLower(strings.TrimSpace(value))
	if clean == "" || clean == "0" || clean == "未知" {
		return false
	}
	for _, keyword := range ipGeoIDCKeywords {
		if strings.Contains(clean, keyword) {
			return true
		}
	}
	return false
}

func extractChinaProvince(value string) string {
	normalized := strings.NewReplacer(
		"·", " ",
		"/", " ",
		"\\", " ",
		"-", " ",
		"_", " ",
		"|", " ",
		",", " ",
		"，", " ",
		";", " ",
		"；", " ",
		"(", " ",
		")", " ",
		"（", " ",
		"）", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
	).Replace(strings.TrimSpace(value))
	for _, token := range strings.Fields(normalized) {
		clean := strings.TrimPrefix(token, "中国")
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		for _, item := range chinaProvincePatterns {
			if clean == item.pattern || strings.HasPrefix(clean, item.pattern) {
				return item.province
			}
		}
	}
	return ""
}

func isISPKeyword(value string) bool {
	clean := strings.TrimSpace(value)
	if clean == "" || clean == "0" || clean == "未知" {
		return false
	}
	regionSuffixes := []string{"省", "市", "自治区", "地区", "盟", "州", "县", "区", "特别行政区"}
	for _, suffix := range regionSuffixes {
		if strings.HasSuffix(clean, suffix) {
			return false
		}
	}
	for _, keyword := range ipGeoISPKeywords {
		if strings.Contains(clean, keyword) {
			return true
		}
	}
	return false
}

func (r *Repository) HasLogs(websiteID string) (bool, error) {
	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)
	query := fmt.Sprintf(`SELECT 1 FROM "%s" LIMIT 1`, tableName)
	var marker int
	if err := r.db.QueryRow(query).Scan(&marker); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isSQLState(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}

func sortLogsForLocking(logs []NginxLogRecord) {
	// 关键目标：让不同并发事务对相同 key 的写入顺序尽量一致，从而降低死锁概率。
	sort.SliceStable(logs, func(i, j int) bool {
		li, lj := logs[i], logs[j]
		if li.IP != lj.IP {
			return li.IP < lj.IP
		}
		if !li.Timestamp.Equal(lj.Timestamp) {
			return li.Timestamp.Before(lj.Timestamp)
		}
		if li.Url != lj.Url {
			return li.Url < lj.Url
		}
		if li.UserBrowser != lj.UserBrowser {
			return li.UserBrowser < lj.UserBrowser
		}
		if li.UserOs != lj.UserOs {
			return li.UserOs < lj.UserOs
		}
		if li.UserDevice != lj.UserDevice {
			return li.UserDevice < lj.UserDevice
		}
		if li.Referer != lj.Referer {
			return li.Referer < lj.Referer
		}
		if li.Method != lj.Method {
			return li.Method < lj.Method
		}
		if li.Status != lj.Status {
			return li.Status < lj.Status
		}
		if li.BytesSent != lj.BytesSent {
			return li.BytesSent < lj.BytesSent
		}
		return false
	})
}

// 为特定网站批量插入日志记录（带死锁重试 + 锁顺序排序）
func (r *Repository) BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error {
	if len(logs) == 0 {
		return nil
	}
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()

	// 不修改调用方的 slice，避免潜在副作用
	logsCopy := append([]NginxLogRecord(nil), logs...)
	sortLogsForLocking(logsCopy)

	const (
		maxAttempts = 5
		baseDelay   = 50 * time.Millisecond
		maxDelay    = 2 * time.Second
	)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := r.batchInsertLogsForWebsiteOnce(websiteID, logsCopy)
		if err == nil {
			return nil
		}
		lastErr = err

		// 仅对 PostgreSQL deadlock (SQLSTATE 40P01) 重试
		if !isSQLState(err, "40P01") || attempt == maxAttempts {
			return err
		}

		// 指数退避 + jitter
		delay := baseDelay * time.Duration(1<<(attempt-1))
		if delay > maxDelay {
			delay = maxDelay
		}
		jitter := time.Duration(rnd.Int63n(int64(baseDelay))) // [0, baseDelay)

		logrus.WithFields(logrus.Fields{
			"website_id": websiteID,
			"attempt":    attempt,
			"sleep":      (delay + jitter).String(),
		}).WithError(err).Warn("检测到数据库死锁(40P01)，准备重试批量写入")

		time.Sleep(delay + jitter)
	}
	return lastErr
}

func (r *Repository) batchInsertLogsForWebsiteOnce(websiteID string, logs []NginxLogRecord) (err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 准备批量插入语句
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	dims, err := prepareDimStatements(tx, websiteID)
	if err != nil {
		return err
	}
	defer dims.Close()
	aggs, err := prepareAggStatements(tx, websiteID)
	if err != nil {
		return err
	}
	defer aggs.Close()
	firstSeenStmt, err := prepareFirstSeenStatement(tx, websiteID)
	if err != nil {
		return err
	}
	defer firstSeenStmt.Close()
	lockFirstSeenStmt, err := tx.Prepare(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`SELECT pg_advisory_xact_lock(hashtext('%s:first_seen'), hashint8(?))`,
		websiteID,
	)))
	if err != nil {
		return err
	}
	defer lockFirstSeenStmt.Close()
	sessions, err := prepareSessionStatements(tx, websiteID)
	if err != nil {
		return err
	}
	defer sessions.Close()

	cache := newDimCaches()
	aggBatch := newAggBatch()
	sessionCache := make(map[string]sessionState)
	// 会话聚合：在事务内先累加，提交前收敛落库，避免每条新会话都去争抢同一天聚合行。
	sessionAggDaily := make(map[string]int64)
	sessionAggEntry := make(map[string]map[int64]int64)
	// 会话更新/状态：在事务内先累加，提交前按稳定顺序落库，避免在维表写入/锁等待期间提前持有 sessions 行锁。
	sessionUpdates := make(map[int64]*pendingSessionUpdate)
	sessionStateUpserts := make(map[string]pendingSessionStateUpsert)
	lockedSessionKeys := make(map[string]struct{})
	// 将 first_seen 的写入从“每条日志一次 upsert”改为“本批次去重后按 ip_id 顺序写入”，降低死锁概率与锁竞争。
	firstSeenMinTs := make(map[int64]int64)
	logRows := make([]logInsertRow, 0, len(logs))
	sanitizedLogs := make([]NginxLogRecord, 0, len(logs))

	// 执行批量插入
	for _, log := range logs {
		log = sanitizeLogRecord(log)

		ipID, err := getOrCreateDimID(
			cache.ip, dims.insertIP, dims.selectIP, log.IP, log.IP,
		)
		if err != nil {
			return err
		}

		urlID, err := getOrCreateDimID(
			cache.url, dims.insertURL, dims.selectURL, log.Url, log.Url,
		)
		if err != nil {
			return err
		}

		refererID, err := getOrCreateDimID(
			cache.referer, dims.insertReferer, dims.selectReferer, log.Referer, log.Referer,
		)
		if err != nil {
			return err
		}

		uaKey := uaCacheKey(log.UserBrowser, log.UserOs, log.UserDevice)
		uaID, err := getOrCreateDimID(
			cache.ua, dims.insertUA, dims.selectUA, uaKey,
			log.UserBrowser, log.UserOs, log.UserDevice,
		)
		if err != nil {
			return err
		}

		locationKey := locationCacheKey(log.DomesticLocation, log.GlobalLocation)
		locationID, err := getOrCreateDimID(
			cache.location, dims.insertLocation, dims.selectLocation, locationKey,
			log.DomesticLocation, log.GlobalLocation,
		)
		if err != nil {
			return err
		}

		ts := log.Timestamp.Unix()
		logRows = append(logRows, logInsertRow{
			ipID:           ipID,
			pageviewFlag:   log.PageviewFlag,
			timestamp:      ts,
			method:         log.Method,
			urlID:          urlID,
			statusCode:     log.Status,
			bytesSent:      log.BytesSent,
			requestLength:  log.RequestLength,
			requestTimeMs:  log.RequestTimeMs,
			upstreamTimeMs: log.UpstreamTimeMs,
			upstreamAddr:   log.UpstreamAddr,
			host:           log.Host,
			requestID:      log.RequestID,
			fingerprint:    log.Fingerprint,
			refererID:      refererID,
			uaID:           uaID,
			locationID:     locationID,
		})
		sanitizedLogs = append(sanitizedLogs, log)
	}

	insertedCounts, err := bulkInsertLogRows(tx, logTable, logRows)
	if err != nil {
		return err
	}

	for i, row := range logRows {
		log := sanitizedLogs[i]
		if row.fingerprint != "" {
			key := logInsertConflictKey(row.timestamp, row.fingerprint)
			if insertedCounts[key] <= 0 {
				continue
			}
			insertedCounts[key]--
		}

		if log.PageviewFlag == 1 {
			if prev, ok := firstSeenMinTs[row.ipID]; !ok || row.timestamp < prev {
				firstSeenMinTs[row.ipID] = row.timestamp
			}
			if err := updateSessionFromLog(
				sessions,
				sessionCache,
				sessionAggDaily,
				sessionAggEntry,
				sessionUpdates,
				sessionStateUpserts,
				lockedSessionKeys,
				row.ipID,
				row.uaID,
				row.locationID,
				row.urlID,
				row.timestamp,
			); err != nil {
				return err
			}
		}

		aggBatch.add(log, row.ipID)
	}

	// 统一顺序写入 first_seen：按 ip_id 升序，避免不同事务对同一批 key 的锁顺序不一致。
	if len(firstSeenMinTs) > 0 {
		ipIDs := make([]int64, 0, len(firstSeenMinTs))
		for ipID := range firstSeenMinTs {
			ipIDs = append(ipIDs, ipID)
		}
		sort.Slice(ipIDs, func(i, j int) bool { return ipIDs[i] < ipIDs[j] })
		for _, ipID := range ipIDs {
			if _, err := lockFirstSeenStmt.Exec(ipID); err != nil {
				return err
			}
			if _, err := firstSeenStmt.Exec(ipID, firstSeenMinTs[ipID]); err != nil {
				return err
			}
		}
	}

	if err := applyAggUpdates(aggs, aggBatch); err != nil {
		return err
	}

	// 在提交前的收敛阶段一次性写入会话聚合，并在每个 day 上使用 advisory lock 将并发写串行化（避免死锁）。
	if err := applySessionAggUpdatesWithLocks(sessions, sessionAggDaily, sessionAggEntry); err != nil {
		return err
	}

	// 会话 UPDATE / session_state upsert：收敛到事务末尾一次性落库，尽量缩短 sessions 行锁持有时间。
	if err := applySessionUpdates(sessions, sessionUpdates); err != nil {
		return err
	}
	if err := applySessionStateUpserts(sessions, sessionStateUpserts); err != nil {
		return err
	}

	return tx.Commit()
}

// CleanOldLogs 清理保留天数之前的日志数据
func (r *Repository) CleanOldLogs() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return r.CleanOldLogsContext(ctx)
}

// ClearLogsForWebsite 清空指定网站的日志数据
func (r *Repository) ClearLogsForWebsite(websiteID string) error {
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()
	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)
	if _, err := r.db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, tableName)); err != nil {
		return fmt.Errorf("清空网站日志失败: %w", err)
	}
	if err := r.clearDimTablesForWebsite(websiteID); err != nil {
		return fmt.Errorf("清空网站维表失败: %w", err)
	}
	if err := r.clearFirstSeenForWebsite(websiteID); err != nil {
		return fmt.Errorf("清空网站首次访问数据失败: %w", err)
	}
	if err := r.clearAggregateTablesForWebsite(websiteID); err != nil {
		return fmt.Errorf("清空网站聚合表失败: %w", err)
	}
	if err := r.clearSessionTablesForWebsite(websiteID); err != nil {
		return fmt.Errorf("清空网站会话表失败: %w", err)
	}
	if err := r.clearSessionAggTablesForWebsite(websiteID); err != nil {
		return fmt.Errorf("清空网站会话聚合表失败: %w", err)
	}
	return nil
}

// ClearAllLogs 清空所有网站的日志数据
func (r *Repository) ClearAllLogs() error {
	for _, id := range config.GetAllWebsiteIDs() {
		if err := r.ClearLogsForWebsite(id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) createTables() error {
	if err := r.ensureIPGeoCacheTable(); err != nil {
		return err
	}
	if err := r.ensureIPGeoManualOverrideTable(); err != nil {
		return err
	}
	if err := r.ensureIPGeoPendingTable(); err != nil {
		return err
	}
	if err := r.ensureIPGeoAPIFailureTable(); err != nil {
		return err
	}
	if err := r.ensureSystemNotificationTable(); err != nil {
		return err
	}
	for _, id := range config.GetAllWebsiteIDs() {
		if err := r.ensureWebsiteSchema(id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ensureIPGeoCacheTable() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "ip_geo_cache" (
            ip TEXT PRIMARY KEY,
            domestic TEXT NOT NULL,
            global TEXT NOT NULL,
            source TEXT NOT NULL DEFAULT 'unknown',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE INDEX IF NOT EXISTS idx_ip_geo_cache_created_at ON "ip_geo_cache"(created_at)`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ensureIPGeoPendingTable() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "ip_geo_pending" (
            ip TEXT PRIMARY KEY,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE INDEX IF NOT EXISTS idx_ip_geo_pending_updated_at ON "ip_geo_pending"(updated_at)`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ensureIPGeoAPIFailureTable() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "ip_geo_api_failures" (
            id BIGSERIAL PRIMARY KEY,
            ip TEXT NOT NULL,
            source TEXT NOT NULL DEFAULT 'ip-api',
            reason TEXT NOT NULL DEFAULT 'unknown',
            error TEXT NOT NULL DEFAULT '',
            status_code INT NOT NULL DEFAULT 0,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ip_geo_api_failures_created_at ON "ip_geo_api_failures"(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_ip_geo_api_failures_ip ON "ip_geo_api_failures"(ip)`,
		`CREATE INDEX IF NOT EXISTS idx_ip_geo_api_failures_ip_created_at ON "ip_geo_api_failures"(ip, created_at)`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ensureSystemNotificationTable() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "system_notifications" (
            id BIGSERIAL PRIMARY KEY,
            level TEXT NOT NULL,
            category TEXT NOT NULL,
            title TEXT NOT NULL,
            message TEXT NOT NULL,
            fingerprint TEXT UNIQUE,
            occurrences INT NOT NULL DEFAULT 1,
            metadata JSONB,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            last_occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            read_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_system_notifications_created_at ON "system_notifications"(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_system_notifications_last_occurred ON "system_notifications"(last_occurred_at)`,
		`CREATE INDEX IF NOT EXISTS idx_system_notifications_read_at ON "system_notifications"(read_at)`,
		`CREATE INDEX IF NOT EXISTS idx_system_notifications_unread_last_occurred ON "system_notifications"(last_occurred_at DESC) WHERE read_at IS NULL`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
