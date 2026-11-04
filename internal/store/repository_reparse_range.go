package store

import (
	"fmt"
	"time"

	"github.com/likaia/nginxpulse/internal/sqlutil"
)

func (r *Repository) ClearLogsForWebsiteRange(websiteID string, start, end time.Time) error {
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()
	if websiteID == "" {
		return fmt.Errorf("websiteID 不能为空")
	}
	if !end.After(start) {
		return fmt.Errorf("时间范围无效")
	}

	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)
	_, err := r.db.Exec(
		sqlutil.ReplacePlaceholders(fmt.Sprintf(`DELETE FROM "%s" WHERE timestamp >= ? AND timestamp < ?`, tableName)),
		start.Unix(),
		end.Unix(),
	)
	if err != nil {
		return fmt.Errorf("删除网站时间段日志失败: %w", err)
	}
	return nil
}

func (r *Repository) RebuildWebsiteDerivedData(websiteID string) error {
	r.maintenanceMu.RLock()
	defer r.maintenanceMu.RUnlock()
	if websiteID == "" {
		return fmt.Errorf("websiteID 不能为空")
	}

	if err := r.backfillAggregates(websiteID); err != nil {
		return fmt.Errorf("回填聚合数据失败: %w", err)
	}
	if err := r.backfillFirstSeen(websiteID); err != nil {
		return fmt.Errorf("回填首次访问数据失败: %w", err)
	}
	if err := r.backfillSessions(websiteID); err != nil {
		return fmt.Errorf("回填会话数据失败: %w", err)
	}
	if err := r.backfillSessionAggregates(websiteID); err != nil {
		return fmt.Errorf("回填会话聚合数据失败: %w", err)
	}
	if err := r.cleanupOrphanDims(websiteID); err != nil {
		return fmt.Errorf("清理维表孤儿数据失败: %w", err)
	}
	return nil
}
