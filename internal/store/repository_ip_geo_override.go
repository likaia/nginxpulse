package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/sqlutil"
)

type IPGeoManualOverride struct {
	IP        string    `json:"ip"`
	Domestic  string    `json:"domestic"`
	Global    string    `json:"global"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *Repository) ensureIPGeoManualOverrideTable() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "ip_geo_manual_overrides" (
            ip TEXT PRIMARY KEY,
            domestic TEXT NOT NULL,
            global TEXT NOT NULL,
            note TEXT NOT NULL DEFAULT '',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE INDEX IF NOT EXISTS idx_ip_geo_manual_overrides_updated_at ON "ip_geo_manual_overrides"(updated_at)`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetIPGeoManualOverride(ip string) (*IPGeoManualOverride, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil, nil
	}

	row := r.db.QueryRow(
		`SELECT ip, domestic, global, note, created_at, updated_at
         FROM "ip_geo_manual_overrides"
         WHERE ip = $1`,
		ip,
	)

	var entry IPGeoManualOverride
	if err := row.Scan(
		&entry.IP,
		&entry.Domestic,
		&entry.Global,
		&entry.Note,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (r *Repository) UpsertIPGeoManualOverride(entry IPGeoManualOverride) error {
	ip := strings.TrimSpace(entry.IP)
	domestic := strings.TrimSpace(entry.Domestic)
	global := strings.TrimSpace(entry.Global)
	note := strings.TrimSpace(entry.Note)
	if ip == "" || domestic == "" || global == "" {
		return fmt.Errorf("IP 和归属地不能为空")
	}

	_, err := r.db.Exec(
		`INSERT INTO "ip_geo_manual_overrides" (ip, domestic, global, note)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (ip) DO UPDATE SET
             domestic = EXCLUDED.domestic,
             global = EXCLUDED.global,
             note = EXCLUDED.note,
             updated_at = NOW()`,
		ip,
		domestic,
		global,
		note,
	)
	return err
}

func (r *Repository) DeleteIPGeoManualOverride(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	_, err := r.db.Exec(`DELETE FROM "ip_geo_manual_overrides" WHERE ip = $1`, ip)
	return err
}

func (r *Repository) GetEffectiveIPGeo(ip, pendingLabel string) (IPGeoCacheEntry, bool, string, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return IPGeoCacheEntry{}, false, "", nil
	}

	override, err := r.GetIPGeoManualOverride(ip)
	if err != nil {
		return IPGeoCacheEntry{}, false, "", err
	}
	if override != nil {
		return IPGeoCacheEntry{
			Domestic: override.Domestic,
			Global:   override.Global,
			Source:   "manual",
		}, true, override.Note, nil
	}

	cached, err := r.GetIPGeoCache([]string{ip})
	if err != nil {
		return IPGeoCacheEntry{}, false, "", err
	}
	if entry, ok := cached[ip]; ok {
		return entry, false, "", nil
	}

	pending := strings.TrimSpace(pendingLabel)
	if pending == "" {
		pending = "待解析"
	}
	return IPGeoCacheEntry{
		Domestic: pending,
		Global:   pending,
		Source:   "pending",
	}, false, "", nil
}

func (r *Repository) ApplyIPGeoLocationForIP(ip string, entry IPGeoCacheEntry) (int64, int64, int, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return 0, 0, 0, nil
	}

	totalLogs := int64(0)
	totalSessions := int64(0)
	updatedWebsites := 0

	for _, websiteID := range config.GetAllWebsiteIDs() {
		logsAffected, sessionsAffected, err := r.applyIPGeoLocationForWebsite(websiteID, ip, entry)
		if err != nil {
			return totalLogs, totalSessions, updatedWebsites, err
		}
		if logsAffected > 0 || sessionsAffected > 0 {
			updatedWebsites++
		}
		totalLogs += logsAffected
		totalSessions += sessionsAffected
	}

	return totalLogs, totalSessions, updatedWebsites, nil
}

func (r *Repository) MarkIPGeoPendingForIP(ip, pendingLabel string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	for _, websiteID := range config.GetAllWebsiteIDs() {
		if err := r.MarkIPGeoPendingForWebsite(websiteID, []string{ip}, pendingLabel); err != nil {
			return err
		}
	}
	return r.UpsertIPGeoPending([]string{ip})
}

func (r *Repository) applyIPGeoLocationForWebsite(
	websiteID string,
	ip string,
	entry IPGeoCacheEntry,
) (logsAffected int64, sessionsAffected int64, err error) {
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	exists, err := r.tableExists(logTable)
	if err != nil || !exists {
		return 0, 0, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	dims, err := prepareDimStatements(tx, websiteID)
	if err != nil {
		return 0, 0, err
	}
	defer dims.Close()

	ipIDs, err := fetchIPIDs(tx, websiteID, []string{ip})
	if err != nil {
		return 0, 0, err
	}
	ipID, ok := ipIDs[ip]
	if !ok {
		return 0, 0, tx.Commit()
	}

	cache := newDimCaches()
	domestic, global := normalizeIPGeoLocation(strings.TrimSpace(entry.Domestic), strings.TrimSpace(entry.Global))
	locationKey := locationCacheKey(domestic, global)
	locationID, err := getOrCreateDimID(
		cache.location,
		dims.insertLocation,
		dims.selectLocation,
		locationKey,
		domestic,
		global,
	)
	if err != nil {
		return 0, 0, err
	}

	logResult, err := tx.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(
			`UPDATE "%s" SET location_id = ? WHERE ip_id = ?`,
			logTable,
		)),
		locationID,
		ipID,
	)
	if err != nil {
		return 0, 0, err
	}
	logsAffected, _ = logResult.RowsAffected()

	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	sessionExists, err := r.tableExists(sessionTable)
	if err != nil {
		return 0, 0, err
	}
	if sessionExists {
		sessionResult, execErr := tx.Exec(
			sqlutil.ReplacePlaceholders(fmt.Sprintf(
				`UPDATE "%s" SET location_id = ? WHERE ip_id = ?`,
				sessionTable,
			)),
			locationID,
			ipID,
		)
		if execErr != nil {
			return 0, 0, execErr
		}
		sessionsAffected, _ = sessionResult.RowsAffected()
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return logsAffected, sessionsAffected, nil
}
