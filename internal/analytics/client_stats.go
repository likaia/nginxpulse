package analytics

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/likaia/nginxpulse/internal/timeutil"
)

type ClientStats struct {
	Key       []string `json:"key"`        // 统计项的键
	PV        []int    `json:"pv"`         // 页面浏览量
	UV        []int    `json:"uv"`         // 独立访客数
	PVPercent []int    `json:"pv_percent"` // PV 百分比
	UVPercent []int    `json:"uv_percent"` // UV 百分比
}

func (s ClientStats) GetType() string {
	return "client"
}

type ClientStatsManager struct {
	repo      *store.Repository
	statsType string
}

func clientStatsOrderByExpr(statsType string) string {
	if statsType == "url" {
		return "pv DESC, uv DESC"
	}
	return "uv DESC, pv DESC"
}

func NewURLStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "url",
	}
}

func NewrefererStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "referer",
	}
}

func NewRefererIPStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "referer_ip",
	}
}

func NewBrowserStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_browser",
	}
}

func NewOsStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_os",
	}
}

func NewDeviceStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_device",
	}
}

func NewLocationStatsManager(userRepoPtr *store.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "location",
	}
}

// 实现 StatsManager 接口
func (s *ClientStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := ClientStats{
		Key:       make([]string, 0),
		PV:        make([]int, 0),
		UV:        make([]int, 0),
		PVPercent: make([]int, 0),
		UVPercent: make([]int, 0),
	}

	statsType := s.statsType
	locationType := ""
	joinClause := ""
	if s.statsType == "location" {
		locationType, _ = query.ExtraParam["locationType"].(string)
		switch locationType {
		case "domestic", "city":
			statsType = "domestic"
		case "global":
			statsType = "global"
		default:
			return result, fmt.Errorf("locationType 参数无效，必须为 domestic、city 或 global")
		}
	}
	selectExpr := statsType
	groupExpr := statsType
	if s.statsType == "location" && locationType == "domestic" {
		selectExpr = fmt.Sprintf(
			"CASE WHEN position('·' in loc.%[1]s) > 0 THEN substring(loc.%[1]s from 1 for position('·' in loc.%[1]s) - 1) ELSE loc.%[1]s END",
			statsType,
		)
		groupExpr = selectExpr
	}
	if s.statsType == "location" && locationType == "city" {
		selectExpr = fmt.Sprintf(
			"CASE WHEN position('·' in loc.%[1]s) > 0 THEN substring(loc.%[1]s from position('·' in loc.%[1]s) + 1) ELSE loc.%[1]s END",
			statsType,
		)
		groupExpr = selectExpr
	}
	if s.statsType == "referer" {
		internalCond := ""
		if website, ok := config.GetWebsiteByID(query.WebsiteID); ok {
			internalCond = buildInternalRefererCondition(website.Domains, "r.referer")
		}
		if internalCond != "" {
			selectExpr = fmt.Sprintf(
				"CASE WHEN r.referer = '-' OR r.referer = '' THEN '直接输入网址访问' WHEN %s THEN '站内访问' ELSE r.referer END",
				internalCond,
			)
		} else {
			selectExpr = "CASE WHEN r.referer = '-' OR r.referer = '' THEN '直接输入网址访问' ELSE r.referer END"
		}
		groupExpr = selectExpr
	}
	limit, _ := query.ExtraParam["limit"].(int)
	timeRange := query.ExtraParam["timeRange"].(string)
	startTime, endTime, err := timeutil.TimePeriod(timeRange)
	if err != nil {
		return result, err
	}
	urlFilter := ""
	if value, ok := query.ExtraParam["urlFilter"].(string); ok {
		urlFilter = strings.TrimSpace(value)
	}

	extraCondition := ""
	switch s.statsType {
	case "url":
		joinClause = fmt.Sprintf(`JOIN "%s_dim_url" u ON u.id = l.url_id`, query.WebsiteID)
		selectExpr = "u.url"
		groupExpr = "u.url"
	case "referer":
		joinClause = fmt.Sprintf(`JOIN "%s_dim_referer" r ON r.id = l.referer_id`, query.WebsiteID)
	case "referer_ip":
		joinClause = fmt.Sprintf(
			`JOIN "%s_dim_referer" r ON r.id = l.referer_id JOIN "%s_dim_ip" ip ON ip.id = l.ip_id`,
			query.WebsiteID,
			query.WebsiteID,
		)
		selectExpr = "ip.ip"
		groupExpr = "ip.ip"
		sourceKind, _ := query.ExtraParam["sourceKind"].(string)
		if sourceCondition := buildRefererSourceCondition(sourceKind, "r.referer"); sourceCondition != "" {
			extraCondition += " AND " + sourceCondition
		}
	case "user_browser":
		joinClause = fmt.Sprintf(`JOIN "%s_dim_ua" ua ON ua.id = l.ua_id`, query.WebsiteID)
		selectExpr = "ua.browser"
		groupExpr = "ua.browser"
	case "user_os":
		joinClause = fmt.Sprintf(`JOIN "%s_dim_ua" ua ON ua.id = l.ua_id`, query.WebsiteID)
		selectExpr = "ua.os"
		groupExpr = "ua.os"
	case "user_device":
		joinClause = fmt.Sprintf(`JOIN "%s_dim_ua" ua ON ua.id = l.ua_id`, query.WebsiteID)
		selectExpr = "ua.device"
		groupExpr = "ua.device"
	case "location":
		joinClause = fmt.Sprintf(`JOIN "%s_dim_location" loc ON loc.id = l.location_id`, query.WebsiteID)
		if locationType == "global" {
			selectExpr = "loc.global"
			groupExpr = "loc.global"
		}
	}
	if s.statsType == "location" && (locationType == "domestic" || locationType == "city") {
		extraCondition += " AND loc.global = '中国'"
	}
	if urlFilter != "" {
		if s.statsType == "url" {
			extraCondition += " AND u.url LIKE ?"
		} else {
			joinClause += fmt.Sprintf(` JOIN "%s_dim_url" filter_url ON filter_url.id = l.url_id`, query.WebsiteID)
			extraCondition += " AND filter_url.url LIKE ?"
		}
	}

	// 构建、执行查询
	orderByExpr := clientStatsOrderByExpr(s.statsType)
	dbQueryStr := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
        SELECT 
            %[1]s AS url, 
            COUNT(*) AS pv,
            COUNT(DISTINCT l.ip_id) AS uv
        FROM "%[2]s_nginx_logs" l
        %[4]s
        WHERE l.pageview_flag = 1 AND l.timestamp >= ? AND l.timestamp < ?%[5]s
        GROUP BY %[3]s
        ORDER BY %[6]s
        LIMIT ?`,
		selectExpr, query.WebsiteID, groupExpr, joinClause, extraCondition, orderByExpr))

	args := []interface{}{startTime.Unix(), endTime.Unix()}
	if urlFilter != "" {
		args = append(args, "%"+urlFilter+"%")
	}
	args = append(args, limit)
	rows, err := s.repo.GetDB().Query(dbQueryStr, args...)
	if err != nil {
		return result, fmt.Errorf("查询客户端统计失败: %v", err)
	}
	defer rows.Close()

	totalPV := 0
	totalUV := 0

	for rows.Next() {
		var url string
		var pv, uv int
		if err := rows.Scan(&url, &pv, &uv); err != nil {
			return result, fmt.Errorf("解析客户端统计结果失败: %v", err)
		}
		result.Key = append(result.Key, url)
		result.PV = append(result.PV, pv)
		result.UV = append(result.UV, uv)
		totalPV += pv
		totalUV += uv
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历客户端统计结果失败: %v", err)
	}

	if totalPV > 0 && totalUV > 0 {
		for i := range result.PV {
			result.PVPercent = append(
				result.PVPercent, int(
					math.Round(float64(result.PV[i])/float64(totalPV)*100)))
			result.UVPercent = append(
				result.UVPercent, int(
					math.Round(float64(result.UV[i])/float64(totalUV)*100)))
		}
	}

	return result, nil

}

func buildInternalRefererCondition(domains []string, refererColumn string) string {
	conditions := make([]string, 0, len(domains))
	for _, raw := range domains {
		domain := normalizeDomain(raw)
		if domain == "" {
			continue
		}
		domain = strings.ReplaceAll(domain, "'", "''")
		conditions = append(conditions,
			fmt.Sprintf(
				"%[1]s LIKE 'http%%://%[2]s/%%' OR %[1]s LIKE 'http%%://%[2]s' OR %[1]s LIKE 'http%%://%[2]s:%%'",
				refererColumn, domain,
			),
		)
	}
	if len(conditions) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
}

func normalizeDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err == nil && parsed.Host != "" {
			return strings.TrimSuffix(parsed.Host, "/")
		}
	}
	raw = strings.TrimPrefix(raw, "//")
	raw = strings.TrimSuffix(raw, "/")
	return raw
}

func buildRefererSourceCondition(sourceKind string, refererColumn string) string {
	directCondition := fmt.Sprintf("(%s = '-' OR %s = '')", refererColumn, refererColumn)
	searchCondition := buildSearchEngineRefererCondition(refererColumn)

	switch sourceKind {
	case "search":
		return searchCondition
	case "direct":
		return directCondition
	case "external":
		if searchCondition == "" {
			return fmt.Sprintf("NOT %s", directCondition)
		}
		return fmt.Sprintf("(NOT %s AND NOT %s)", directCondition, searchCondition)
	default:
		return ""
	}
}

func buildSearchEngineRefererCondition(refererColumn string) string {
	patterns := []string{
		"baidu.",
		"google.",
		"bing.",
		"sogou.",
		"360.",
		"so.com",
		"yahoo.",
		"duckduckgo.",
		"yandex.",
	}

	conditions := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		safePattern := strings.ReplaceAll(strings.ToLower(pattern), "'", "''")
		conditions = append(conditions, fmt.Sprintf("LOWER(%s) LIKE '%%%s%%'", refererColumn, safePattern))
	}
	if len(conditions) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
}
