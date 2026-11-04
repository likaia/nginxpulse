package store

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/sirupsen/logrus"
)

func (r *Repository) UpdateIPGeoLocations(
	locations map[string]IPGeoCacheEntry,
	pendingLabel string,
) error {
	if len(locations) == 0 {
		return nil
	}
	for _, websiteID := range config.GetAllWebsiteIDs() {
		if err := r.updateIPGeoLocationsForWebsite(websiteID, locations, pendingLabel); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) MarkIPGeoPendingForWebsite(
	websiteID string,
	ips []string,
	pendingLabel string,
) (err error) {
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()
	if len(ips) == 0 {
		return nil
	}
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	exists, err := r.tableExists(logTable)
	if err != nil || !exists {
		return err
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
	sort.Strings(unique)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	dims, err := prepareDimStatements(tx, websiteID)
	if err != nil {
		return err
	}
	defer dims.Close()

	ipIDs, err := fetchIPIDs(tx, websiteID, unique)
	if err != nil {
		return err
	}
	if len(ipIDs) == 0 {
		return tx.Commit()
	}

	cache := newDimCaches()
	pendingKey := locationCacheKey(pendingLabel, pendingLabel)
	pendingID, err := getOrCreateDimID(
		cache.location,
		dims.insertLocation,
		dims.selectLocation,
		pendingKey,
		pendingLabel,
		pendingLabel,
	)
	if err != nil {
		return err
	}

	updateLogsStmt, err := tx.Prepare(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`UPDATE "%s" SET location_id = ? WHERE ip_id = ?`,
		logTable,
	)))
	if err != nil {
		return err
	}
	defer updateLogsStmt.Close()

	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	sessionExists, err := r.tableExists(sessionTable)
	if err != nil {
		return err
	}
	var updateSessionsStmt *sql.Stmt
	if sessionExists {
		updateSessionsStmt, err = tx.Prepare(sqlutil.ReplacePlaceholders(fmt.Sprintf(
			`UPDATE "%s" SET location_id = ? WHERE ip_id = ?`,
			sessionTable,
		)))
		if err != nil {
			return err
		}
		defer updateSessionsStmt.Close()
	}

	for _, ip := range unique {
		ipID, ok := ipIDs[ip]
		if !ok {
			continue
		}
		if _, err := updateLogsStmt.Exec(pendingID, ipID); err != nil {
			return err
		}
		if updateSessionsStmt != nil {
			if _, err := updateSessionsStmt.Exec(pendingID, ipID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) updateIPGeoLocationsForWebsite(
	websiteID string,
	locations map[string]IPGeoCacheEntry,
	pendingLabel string,
) error {
	const (
		maxAttempts = 5
		baseDelay   = 50 * time.Millisecond
		maxDelay    = 2 * time.Second
	)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := r.updateIPGeoLocationsForWebsiteOnce(websiteID, locations, pendingLabel)
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
		}).WithError(err).Warn("检测到数据库死锁(40P01)，准备重试 IP 归属地回填")

		time.Sleep(delay + jitter)
	}
	return lastErr
}

func (r *Repository) updateIPGeoLocationsForWebsiteOnce(
	websiteID string,
	locations map[string]IPGeoCacheEntry,
	pendingLabel string,
) (err error) {
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()
	logTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	exists, err := r.tableExists(logTable)
	if err != nil || !exists {
		return err
	}

	normalized := make(map[string]IPGeoCacheEntry, len(locations))
	for ip, entry := range locations {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		normalized[ip] = entry
	}
	ips := make([]string, 0, len(normalized))
	for ip := range normalized {
		ips = append(ips, ip)
	}
	if len(ips) == 0 {
		return nil
	}
	// 固定顺序，降低并发回填时的锁顺序差异
	sort.Strings(ips)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	dims, err := prepareDimStatements(tx, websiteID)
	if err != nil {
		return err
	}
	defer dims.Close()

	ipIDs, err := fetchIPIDs(tx, websiteID, ips)
	if err != nil {
		return err
	}
	if len(ipIDs) == 0 {
		return tx.Commit()
	}

	cache := newDimCaches()
	pendingKey := locationCacheKey(pendingLabel, pendingLabel)
	pendingID, err := getOrCreateDimID(
		cache.location,
		dims.insertLocation,
		dims.selectLocation,
		pendingKey,
		pendingLabel,
		pendingLabel,
	)
	if err != nil {
		return err
	}

	updateLogsStmt, err := tx.Prepare(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`UPDATE "%s" SET location_id = ? WHERE ip_id = ? AND location_id = ?`,
		logTable,
	)))
	if err != nil {
		return err
	}
	defer updateLogsStmt.Close()

	sessionTable := fmt.Sprintf("%s_sessions", websiteID)
	sessionExists, err := r.tableExists(sessionTable)
	if err != nil {
		return err
	}
	var updateSessionsStmt *sql.Stmt
	if sessionExists {
		updateSessionsStmt, err = tx.Prepare(sqlutil.ReplacePlaceholders(fmt.Sprintf(
			`UPDATE "%s" SET location_id = ? WHERE ip_id = ? AND location_id = ?`,
			sessionTable,
		)))
		if err != nil {
			return err
		}
		defer updateSessionsStmt.Close()
	}

	for _, ip := range ips {
		ipID, ok := ipIDs[ip]
		if !ok {
			continue
		}
		entry, ok := normalized[ip]
		if !ok {
			continue
		}
		domestic := strings.TrimSpace(entry.Domestic)
		global := strings.TrimSpace(entry.Global)
		domestic, global = normalizeIPGeoLocation(domestic, global)
		if domestic == "" && global == "" {
			continue
		}
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
			return err
		}
		if _, err := updateLogsStmt.Exec(locationID, ipID, pendingID); err != nil {
			return err
		}
		if updateSessionsStmt != nil {
			if _, err := updateSessionsStmt.Exec(locationID, ipID, pendingID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
