package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/sirupsen/logrus"
)

type NginxLogRecord struct {
	ID               int64     `json:"id"`
	IP               string    `json:"ip"`
	PageviewFlag     int       `json:"pageview_flag"`
	Timestamp        time.Time `json:"timestamp"`
	Method           string    `json:"method"`
	Url              string    `json:"url"`
	Status           int       `json:"status"`
	BytesSent        int       `json:"bytes_sent"`
	RequestLength    int       `json:"request_length"`
	RequestTimeMs    int64     `json:"request_time_ms"`
	UpstreamTimeMs   int64     `json:"upstream_response_time_ms"`
	UpstreamAddr     string    `json:"upstream_addr"`
	Host             string    `json:"host"`
	RequestID        string    `json:"request_id"`
	Fingerprint      string    `json:"fingerprint,omitempty"`
	Referer          string    `json:"referer"`
	UserBrowser      string    `json:"user_browser"`
	UserOs           string    `json:"user_os"`
	UserDevice       string    `json:"user_device"`
	DomesticLocation string    `json:"domestic_location"`
	GlobalLocation   string    `json:"global_location"`
}

type IPGeoAPIFailure struct {
	ID         int64     `json:"id"`
	IP         string    `json:"ip"`
	Source     string    `json:"source"`
	Reason     string    `json:"reason"`
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
}

type SystemNotification struct {
	ID             int64                  `json:"id"`
	Level          string                 `json:"level"`
	Category       string                 `json:"category"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Fingerprint    string                 `json:"fingerprint"`
	Occurrences    int                    `json:"occurrences"`
	CreatedAt      time.Time              `json:"created_at"`
	LastOccurredAt time.Time              `json:"last_occurred_at"`
	ReadAt         *time.Time             `json:"read_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

func sanitizeUTF8(s string) string {
	if s == "" {
		return s
	}
	if strings.IndexByte(s, 0) >= 0 {
		s = strings.ReplaceAll(s, "\x00", "")
	}
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "?")
}

const (
	maxURLBytes     = 2000
	maxRefererBytes = 2000
	maxUABytes      = 256
	maxHostBytes    = 255
	maxRequestID    = 256
	maxUpstreamAddr = 512
)

func truncateUTF8Bytes(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.ValidString(s[:cut]) {
		cut--
	}
	if cut == 0 {
		return ""
	}
	return s[:cut]
}

func sanitizeAndTruncate(s string, maxBytes int) string {
	return truncateUTF8Bytes(sanitizeUTF8(s), maxBytes)
}

func sanitizeLogRecord(log NginxLogRecord) NginxLogRecord {
	log.IP = sanitizeUTF8(log.IP)
	log.Method = sanitizeUTF8(log.Method)
	log.Url = sanitizeAndTruncate(log.Url, maxURLBytes)
	log.UpstreamAddr = sanitizeAndTruncate(log.UpstreamAddr, maxUpstreamAddr)
	log.Host = sanitizeAndTruncate(log.Host, maxHostBytes)
	log.RequestID = sanitizeAndTruncate(log.RequestID, maxRequestID)
	log.Fingerprint = sanitizeUTF8(log.Fingerprint)
	log.Referer = sanitizeAndTruncate(log.Referer, maxRefererBytes)
	log.UserBrowser = sanitizeAndTruncate(log.UserBrowser, maxUABytes)
	log.UserOs = sanitizeAndTruncate(log.UserOs, maxUABytes)
	log.UserDevice = sanitizeAndTruncate(log.UserDevice, maxUABytes)
	log.DomesticLocation = sanitizeUTF8(log.DomesticLocation)
	log.GlobalLocation = sanitizeUTF8(log.GlobalLocation)
	return log
}

type IPGeoCacheEntry struct {
	Domestic string
	Global   string
	Source   string
}

type Repository struct {
	db *sql.DB
	// 清理会重建统计表，必须与写入/重解析互斥；只在有时限的站点清理期间持有。
	maintenanceMu  sync.RWMutex
	cleanupMu      sync.Mutex
	executor       repositoryDB
	cleanupCursor  string // 有整轮预算时轮转站点，避免大站点让后面的站点长期饿死。
	cleanupPending map[string]*websiteCleanupProgress
	activeCleanup  *websiteCleanupProgress // 仅维护连接副本使用，阶段成功后才推进。
}

func NewRepository() (*Repository, error) {
	cfg := config.ReadConfig()
	db, err := openPostgres(cfg.Database)
	if err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

func openPostgres(cfg config.DatabaseConfig) (*sql.DB, error) {
	if cfg.Driver == "" {
		cfg.Driver = "postgres"
	}
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("仅支持 postgres 驱动，当前为: %s", cfg.Driver)
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("数据库 DSN 不能为空")
	}

	pgConfig, err := pgx.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("解析数据库 DSN 失败: %w", err)
	}

	db := stdlib.OpenDB(*pgConfig)
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != "" {
		if parsed, err := time.ParseDuration(cfg.ConnMaxLifetime); err == nil {
			db.SetConnMaxLifetime(parsed)
		} else {
			logrus.WithFields(logrus.Fields{
				"conn_max_lifetime": cfg.ConnMaxLifetime,
			}).WithError(err).Warn("无效的数据库连接最大生命周期配置，已忽略")
		}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 PostgreSQL 失败(host=%s db=%s user=%s): %w",
			pgConfig.Host, pgConfig.Database, pgConfig.User, err)
	}
	warnPostgresRole(db)

	return db, nil
}

// Existing external database accounts must keep working across upgrades. Report
// excessive privileges without changing a shared role or blocking startup.
func warnPostgresRole(db *sql.DB) {
	privileged, err := postgresRoleHasElevatedPrivileges(db)
	if err != nil {
		logrus.WithError(err).WithField("security_event", "postgres_role_check_failed").Warn("检查 PostgreSQL 账号权限失败，将继续使用现有数据库连接")
		return
	}
	if privileged {
		logrus.WithField("security_event", "postgres_role_privileges").Warn("数据库账号权限过高，将继续使用现有连接以兼容升级；建议使用仅拥有应用数据库及其表的普通账号，移除集群管理及服务器文件/程序权限（包括角色成员权限）")
	}
}

// Include both the login and effective roles, plus inherited/SET ROLE privileges.
func postgresRoleHasElevatedPrivileges(db *sql.DB) (bool, error) {
	var privileged bool
	err := db.QueryRow(`WITH RECURSIVE reachable_roles AS (
        SELECT oid, rolname, rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
        FROM pg_roles WHERE rolname IN (current_user, session_user)
        UNION
        SELECT p.oid, p.rolname, p.rolsuper, p.rolcreatedb, p.rolcreaterole, p.rolreplication, p.rolbypassrls
        FROM reachable_roles r
        JOIN pg_auth_members m ON m.member = r.oid
        JOIN pg_roles p ON p.oid = m.roleid
    ) SELECT EXISTS (
        SELECT 1 FROM reachable_roles
        WHERE rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls
           OR rolname IN ('pg_read_server_files', 'pg_write_server_files', 'pg_execute_server_program')
    )`).Scan(&privileged)
	if err != nil {
		return false, err
	}
	return privileged, nil
}

// 初始化数据库
func (r *Repository) Init() error {
	return r.createTables()
}

// 关闭数据库连接
func (r *Repository) Close() error {
	logrus.Info("关闭数据库")
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// 获取数据库连接
func (r *Repository) GetDB() *sql.DB {
	return r.db
}

func (r *Repository) GetIPGeoCache(ips []string) (map[string]IPGeoCacheEntry, error) {
	results := make(map[string]IPGeoCacheEntry)
	if len(ips) == 0 {
		return results, nil
	}

	unique := make([]string, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		unique = append(unique, ip)
	}
	if len(unique) == 0 {
		return results, nil
	}

	placeholders := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, ip := range unique {
		placeholders[i] = "?"
		args[i] = ip
	}

	query := fmt.Sprintf(`SELECT ip, domestic, global, source FROM "ip_geo_cache" WHERE ip IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.Query(sqlutil.ReplacePlaceholders(query), args...)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip, domestic, global, source string
		if err := rows.Scan(&ip, &domestic, &global, &source); err != nil {
			return results, err
		}
		results[ip] = IPGeoCacheEntry{
			Domestic: domestic,
			Global:   global,
			Source:   source,
		}
	}
	if err := rows.Err(); err != nil {
		return results, err
	}

	overrideQuery := fmt.Sprintf(
		`SELECT ip, domestic, global FROM "ip_geo_manual_overrides" WHERE ip IN (%s)`,
		strings.Join(placeholders, ","),
	)
	overrideRows, err := r.db.Query(sqlutil.ReplacePlaceholders(overrideQuery), args...)
	if err != nil {
		return results, err
	}
	defer overrideRows.Close()

	for overrideRows.Next() {
		var ip, domestic, global string
		if err := overrideRows.Scan(&ip, &domestic, &global); err != nil {
			return results, err
		}
		results[ip] = IPGeoCacheEntry{
			Domestic: domestic,
			Global:   global,
			Source:   "manual",
		}
	}
	if err := overrideRows.Err(); err != nil {
		return results, err
	}

	return results, nil
}

func (r *Repository) ClearIPGeoCache() error {
	_, err := r.db.Exec(`DELETE FROM "ip_geo_cache"`)
	return err
}

func (r *Repository) DeleteIPGeoCache(ips []string) error {
	if len(ips) == 0 {
		return nil
	}

	unique := make([]string, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		unique = append(unique, ip)
	}
	if len(unique) == 0 {
		return nil
	}

	placeholders := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, ip := range unique {
		placeholders[i] = "?"
		args[i] = ip
	}
	query := fmt.Sprintf(`DELETE FROM "ip_geo_cache" WHERE ip IN (%s)`, strings.Join(placeholders, ","))
	_, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	return err
}

func (r *Repository) ClearIPGeoPending() error {
	_, err := r.db.Exec(`DELETE FROM "ip_geo_pending"`)
	return err
}

func (r *Repository) UpsertIPGeoPending(ips []string) error {
	if len(ips) == 0 {
		return nil
	}

	values := make([]string, 0, len(ips))
	args := make([]interface{}, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		values = append(values, "(?)")
		args = append(args, ip)
	}
	if len(values) == 0 {
		return nil
	}

	query := fmt.Sprintf(`INSERT INTO "ip_geo_pending" (ip)
        VALUES %s
        ON CONFLICT (ip) DO UPDATE SET
            updated_at = NOW()`, strings.Join(values, ","))

	_, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	return err
}

func (r *Repository) FetchIPGeoPending(limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.Query(
		sqlutil.ReplacePlaceholders(`SELECT ip FROM "ip_geo_pending" ORDER BY updated_at ASC LIMIT ?`),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ips := make([]string, 0, limit)
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ips, nil
}

func (r *Repository) FetchIPGeoPendingWithCooldown(limit int, cutoff time.Time) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	if cutoff.IsZero() {
		return r.FetchIPGeoPending(limit)
	}
	rows, err := r.db.Query(
		sqlutil.ReplacePlaceholders(`SELECT p.ip
         FROM "ip_geo_pending" AS p
         WHERE NOT EXISTS (
             SELECT 1 FROM "ip_geo_api_failures" AS f
             WHERE f.ip = p.ip AND f.created_at >= ?
         )
         ORDER BY p.updated_at ASC
         LIMIT ?`),
		cutoff, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ips := make([]string, 0, limit)
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ips, nil
}

func (r *Repository) DeleteIPGeoPending(ips []string) error {
	if len(ips) == 0 {
		return nil
	}

	unique := make([]string, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		unique = append(unique, ip)
	}
	if len(unique) == 0 {
		return nil
	}

	placeholders := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, ip := range unique {
		placeholders[i] = "?"
		args[i] = ip
	}
	query := fmt.Sprintf(`DELETE FROM "ip_geo_pending" WHERE ip IN (%s)`, strings.Join(placeholders, ","))
	_, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	return err
}

func (r *Repository) HasIPGeoPending() (bool, error) {
	row := r.db.QueryRow(`SELECT 1 FROM "ip_geo_pending" LIMIT 1`)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) CountIPGeoPending() (int64, error) {
	row := r.db.QueryRow(`SELECT COUNT(*) FROM "ip_geo_pending"`)
	var total int64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repository) FetchPendingIPGeoFromLogs(websiteID, pendingLabel string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	exists, err := r.tableExists(logTable)
	if err != nil || !exists {
		return nil, err
	}
	ipTable := fmt.Sprintf("%s_dim_ip", websiteID)
	locationTable := fmt.Sprintf("%s_dim_location", websiteID)

	query := fmt.Sprintf(
		`SELECT DISTINCT ip.ip
         FROM "%s" AS l
         JOIN "%s" AS ip ON l.ip_id = ip.id
         JOIN "%s" AS loc ON l.location_id = loc.id
         WHERE loc.domestic = ? AND loc.global = ?
         LIMIT ?`,
		logTable, ipTable, locationTable,
	)
	rows, err := r.db.Query(sqlutil.ReplacePlaceholders(query), pendingLabel, pendingLabel, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ips := make([]string, 0, limit)
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ips, nil
}

func (r *Repository) UpsertIPGeoCache(entries map[string]IPGeoCacheEntry) error {
	if len(entries) == 0 {
		return nil
	}

	values := make([]string, 0, len(entries))
	args := make([]interface{}, 0, len(entries)*4)
	for ip, entry := range entries {
		if ip == "" {
			continue
		}
		domestic, global := normalizeIPGeoLocation(strings.TrimSpace(entry.Domestic), strings.TrimSpace(entry.Global))
		source := strings.TrimSpace(entry.Source)
		if source == "" {
			source = "unknown"
		}
		values = append(values, "(?, ?, ?, ?)")
		args = append(args, ip, domestic, global, source)
	}
	if len(values) == 0 {
		return nil
	}

	query := fmt.Sprintf(`INSERT INTO "ip_geo_cache" (ip, domestic, global, source)
        VALUES %s
        ON CONFLICT (ip) DO UPDATE SET
            domestic = excluded.domestic,
            global = excluded.global,
            source = excluded.source,
            updated_at = NOW()`, strings.Join(values, ","))

	_, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	return err
}

func (r *Repository) TrimIPGeoCache(limit int) error {
	if limit <= 0 {
		return nil
	}

	var total int64
	row := r.db.QueryRow(`SELECT COUNT(*) FROM "ip_geo_cache"`)
	if err := row.Scan(&total); err != nil {
		return err
	}
	if total <= int64(limit) {
		return nil
	}
	excess := total - int64(limit)

	_, err := r.db.Exec(
		`DELETE FROM "ip_geo_cache"
         WHERE ip IN (
             SELECT ip FROM "ip_geo_cache"
             ORDER BY created_at ASC
             LIMIT $1
         )`, excess,
	)
	return err
}

func (r *Repository) InsertIPGeoAPIFailures(
	failures map[string]string,
	source string,
	detail string,
	statusCode int,
) error {
	if len(failures) == 0 {
		return nil
	}
	if strings.TrimSpace(source) == "" {
		source = "ip-api"
	}
	values := make([]string, 0, len(failures))
	args := make([]interface{}, 0, len(failures)*5)
	seen := make(map[string]struct{}, len(failures))
	for ip, reason := range failures {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		if strings.TrimSpace(reason) == "" {
			reason = "unknown"
		}
		values = append(values, "(?, ?, ?, ?, ?)")
		args = append(args, ip, source, reason, detail, statusCode)
	}
	if len(values) == 0 {
		return nil
	}
	query := fmt.Sprintf(
		`INSERT INTO "ip_geo_api_failures" (ip, source, reason, error, status_code)
         VALUES %s`,
		strings.Join(values, ","),
	)
	_, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	return err
}

func (r *Repository) ListIPGeoAPIFailures(page, pageSize int) ([]IPGeoAPIFailure, bool, error) {
	return r.ListIPGeoAPIFailuresFiltered("", "", "", page, pageSize)
}

func (r *Repository) ListIPGeoAPIFailuresFiltered(
	websiteID string,
	reason string,
	keyword string,
	page int,
	pageSize int,
) ([]IPGeoAPIFailure, bool, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	joinClause := ""
	whereParts := make([]string, 0, 3)
	args := make([]interface{}, 0, 5)

	websiteID = strings.TrimSpace(websiteID)
	if websiteID != "" {
		ipTable := fmt.Sprintf("%s_dim_ip", websiteID)
		exists, err := r.tableExists(ipTable)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return []IPGeoAPIFailure{}, false, nil
		}
		joinClause = fmt.Sprintf(`JOIN "%s" AS ip ON f.ip = ip.ip`, ipTable)
	}

	reason = strings.TrimSpace(reason)
	if reason != "" {
		whereParts = append(whereParts, "f.reason = ?")
		args = append(args, reason)
	}

	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		whereParts = append(whereParts, "f.ip ILIKE ?")
		args = append(args, "%"+keyword+"%")
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	query := fmt.Sprintf(
		`SELECT f.id, f.ip, f.source, f.reason, f.error, f.status_code, f.created_at
         FROM "ip_geo_api_failures" AS f
         %s
         %s
         ORDER BY f.created_at DESC
         LIMIT ? OFFSET ?`,
		joinClause, whereClause,
	)
	args = append(args, pageSize+1, offset)

	rows, err := r.db.Query(sqlutil.ReplacePlaceholders(query), args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	failures := make([]IPGeoAPIFailure, 0, pageSize)
	hasMore := false
	for rows.Next() {
		var entry IPGeoAPIFailure
		if err := rows.Scan(
			&entry.ID,
			&entry.IP,
			&entry.Source,
			&entry.Reason,
			&entry.Error,
			&entry.StatusCode,
			&entry.CreatedAt,
		); err != nil {
			return nil, false, err
		}
		if len(failures) < pageSize {
			failures = append(failures, entry)
		} else {
			hasMore = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return failures, hasMore, nil
}

func (r *Repository) ClearIPGeoAPIFailuresFiltered(websiteID, reason, keyword string) (int64, error) {
	whereParts := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)

	websiteID = strings.TrimSpace(websiteID)
	if websiteID != "" {
		ipTable := fmt.Sprintf("%s_dim_ip", websiteID)
		exists, err := r.tableExists(ipTable)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, nil
		}
		whereParts = append(
			whereParts,
			fmt.Sprintf(`EXISTS (SELECT 1 FROM "%s" AS ip WHERE ip.ip = f.ip)`, ipTable),
		)
	}

	reason = strings.TrimSpace(reason)
	if reason != "" {
		whereParts = append(whereParts, "f.reason = ?")
		args = append(args, reason)
	}

	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		whereParts = append(whereParts, "f.ip ILIKE ?")
		args = append(args, "%"+keyword+"%")
	}

	query := `DELETE FROM "ip_geo_api_failures" AS f`
	if len(whereParts) > 0 {
		query += " WHERE " + strings.Join(whereParts, " AND ")
	}

	result, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *Repository) DeleteIPGeoAPIFailuresByIPs(ips []string) (int64, error) {
	if len(ips) == 0 {
		return 0, nil
	}
	values := make([]string, 0, len(ips))
	args := make([]interface{}, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		values = append(values, "?")
		args = append(args, ip)
	}
	if len(values) == 0 {
		return 0, nil
	}
	query := fmt.Sprintf(`DELETE FROM "ip_geo_api_failures" WHERE ip IN (%s)`, strings.Join(values, ","))
	result, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *Repository) CountIPGeoAPIFailures() (int64, error) {
	var count int64
	row := r.db.QueryRow(`SELECT COUNT(*) FROM "ip_geo_api_failures"`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) CreateSystemNotification(entry SystemNotification) (int64, error) {
	level := strings.TrimSpace(entry.Level)
	category := strings.TrimSpace(entry.Category)
	title := strings.TrimSpace(entry.Title)
	message := strings.TrimSpace(entry.Message)
	fingerprint := strings.TrimSpace(entry.Fingerprint)
	if level == "" {
		level = "info"
	}
	if category == "" {
		category = "system"
	}
	if title == "" {
		title = "系统通知"
	}
	if message == "" {
		message = "-"
	}

	var metadataJSON []byte
	if entry.Metadata != nil {
		if encoded, err := json.Marshal(entry.Metadata); err == nil {
			metadataJSON = encoded
		}
	}

	if fingerprint == "" {
		row := r.db.QueryRow(
			`INSERT INTO "system_notifications" (level, category, title, message, metadata)
             VALUES ($1, $2, $3, $4, $5)
             RETURNING id`,
			level, category, title, message, metadataJSON,
		)
		var id int64
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}

	row := r.db.QueryRow(
		`INSERT INTO "system_notifications"
            (level, category, title, message, fingerprint, occurrences, metadata, last_occurred_at)
         VALUES ($1, $2, $3, $4, $5, 1, $6, NOW())
         ON CONFLICT (fingerprint) DO UPDATE SET
            level = EXCLUDED.level,
            category = EXCLUDED.category,
            title = EXCLUDED.title,
            message = EXCLUDED.message,
            metadata = COALESCE(EXCLUDED.metadata, "system_notifications".metadata),
            occurrences = "system_notifications".occurrences + 1,
            last_occurred_at = NOW(),
            read_at = NULL
         RETURNING id`,
		level, category, title, message, fingerprint, metadataJSON,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateSystemNotificationWithCount(entry SystemNotification, count int) (int64, error) {
	if count <= 0 {
		count = 1
	}
	level := strings.TrimSpace(entry.Level)
	category := strings.TrimSpace(entry.Category)
	title := strings.TrimSpace(entry.Title)
	message := strings.TrimSpace(entry.Message)
	fingerprint := strings.TrimSpace(entry.Fingerprint)
	if level == "" {
		level = "info"
	}
	if category == "" {
		category = "system"
	}
	if title == "" {
		title = "系统通知"
	}
	if message == "" {
		message = "-"
	}

	var metadataJSON []byte
	if entry.Metadata != nil {
		if encoded, err := json.Marshal(entry.Metadata); err == nil {
			metadataJSON = encoded
		}
	}

	if fingerprint == "" {
		return r.CreateSystemNotification(entry)
	}

	row := r.db.QueryRow(
		`INSERT INTO "system_notifications"
            (level, category, title, message, fingerprint, occurrences, metadata, last_occurred_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
         ON CONFLICT (fingerprint) DO UPDATE SET
            level = EXCLUDED.level,
            category = EXCLUDED.category,
            title = EXCLUDED.title,
            message = EXCLUDED.message,
            metadata = COALESCE(EXCLUDED.metadata, "system_notifications".metadata),
            occurrences = "system_notifications".occurrences + EXCLUDED.occurrences,
            last_occurred_at = NOW(),
            read_at = NULL
         RETURNING id`,
		level, category, title, message, fingerprint, count, metadataJSON,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) ListSystemNotifications(page, pageSize int, unreadOnly bool) ([]SystemNotification, bool, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	where := ""
	args := make([]interface{}, 0, 3)
	if unreadOnly {
		where = "WHERE read_at IS NULL"
	}
	query := fmt.Sprintf(
		`SELECT id, level, category, title, message, fingerprint, occurrences, metadata, created_at, last_occurred_at, read_at
         FROM "system_notifications"
         %s
         ORDER BY last_occurred_at DESC
         LIMIT ? OFFSET ?`,
		where,
	)
	args = append(args, pageSize+1, offset)

	rows, err := r.db.Query(sqlutil.ReplacePlaceholders(query), args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	notifications := make([]SystemNotification, 0, pageSize)
	hasMore := false
	for rows.Next() {
		var entry SystemNotification
		var metadataBytes []byte
		var readAt sql.NullTime
		if err := rows.Scan(
			&entry.ID,
			&entry.Level,
			&entry.Category,
			&entry.Title,
			&entry.Message,
			&entry.Fingerprint,
			&entry.Occurrences,
			&metadataBytes,
			&entry.CreatedAt,
			&entry.LastOccurredAt,
			&readAt,
		); err != nil {
			return nil, false, err
		}
		if readAt.Valid {
			entry.ReadAt = &readAt.Time
		}
		if len(metadataBytes) > 0 {
			_ = json.Unmarshal(metadataBytes, &entry.Metadata)
		}
		if len(notifications) < pageSize {
			notifications = append(notifications, entry)
		} else {
			hasMore = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return notifications, hasMore, nil
}

func (r *Repository) MarkSystemNotificationsRead(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	values := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		values = append(values, "?")
		args = append(args, id)
	}
	if len(values) == 0 {
		return nil
	}
	query := fmt.Sprintf(
		`UPDATE "system_notifications"
         SET read_at = NOW()
         WHERE read_at IS NULL AND id IN (%s)`,
		strings.Join(values, ","),
	)
	_, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	return err
}

func (r *Repository) MarkAllSystemNotificationsRead() error {
	_, err := r.db.Exec(`UPDATE "system_notifications" SET read_at = NOW() WHERE read_at IS NULL`)
	return err
}

func (r *Repository) DeleteSystemNotifications(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	values := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		values = append(values, "?")
		args = append(args, id)
	}
	if len(values) == 0 {
		return 0, nil
	}
	query := fmt.Sprintf(
		`DELETE FROM "system_notifications"
         WHERE id IN (%s)`,
		strings.Join(values, ","),
	)
	result, err := r.db.Exec(sqlutil.ReplacePlaceholders(query), args...)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *Repository) DeleteAllSystemNotifications() (int64, error) {
	result, err := r.db.Exec(`DELETE FROM "system_notifications"`)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *Repository) DeleteSystemNotificationByFingerprint(fingerprint string) (int64, error) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return 0, nil
	}
	result, err := r.db.Exec(`DELETE FROM "system_notifications" WHERE fingerprint = $1`, fingerprint)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *Repository) GetSystemNotificationUnreadCount() (int64, error) {
	var count int64
	row := r.db.QueryRow(`SELECT COUNT(*) FROM "system_notifications" WHERE read_at IS NULL`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
