package analytics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/store"
)

// StatsResult 统计结果的基础接口
type StatsResult interface {
	GetType() string
}

// StatsQuery 统计查询的通用参数
type StatsQuery struct {
	WebsiteID  string
	ExtraParam map[string]interface{}
}

// StatsManager 统计管理器接口
type StatsManager interface {
	Query(query StatsQuery) (StatsResult, error)
}

// StatsFactory 统计工厂，管理所有统计管理器
type StatsFactory struct {
	repo        *store.Repository
	managers    map[string]StatsManager
	cache       *StatsCache
	mu          sync.RWMutex
	cacheExpiry time.Duration
}

// NewStatsFactory 创建新的统计工厂
func NewStatsFactory(repo *store.Repository) *StatsFactory {
	cfg := config.ReadConfig()
	expiry := config.ParseInterval(cfg.System.TaskInterval, 5*time.Minute)

	factory := &StatsFactory{
		repo:        repo,
		managers:    make(map[string]StatsManager),
		cache:       NewStatsCache(),
		cacheExpiry: expiry,
	}

	factory.registerDefaultManagers()
	return factory
}

// registerDefaultManagers 注册默认的统计管理器
func (f *StatsFactory) registerDefaultManagers() {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 注册各种统计管理器
	f.managers["timeseries"] = NewTimeSeriesStatsManager(f.repo)
	f.managers["overall"] = NewOverallStatsManager(f.repo)
	f.managers["developer_daily"] = NewDeveloperDailyStatsManager(f.repo)

	f.managers["url"] = NewURLStatsManager(f.repo)
	f.managers["referer"] = NewrefererStatsManager(f.repo)
	f.managers["referer_ip"] = NewRefererIPStatsManager(f.repo)
	f.managers["referer_ip_batch"] = NewRefererIPBatchStatsManager(f.repo)

	f.managers["browser"] = NewBrowserStatsManager(f.repo)
	f.managers["os"] = NewOsStatsManager(f.repo)
	f.managers["device"] = NewDeviceStatsManager(f.repo)

	f.managers["location"] = NewLocationStatsManager(f.repo)

	f.managers["logs"] = NewLogsStatsManager(f.repo)
	f.managers["session"] = NewSessionsStatsManager(f.repo)
	f.managers["session_summary"] = NewSessionSummaryStatsManager(f.repo)
	f.managers["realtime"] = NewRealtimeStatsManager(f.repo)
}

// GetManager 获取指定类型的统计管理器
func (f *StatsFactory) GetManager(managerType string) (StatsManager, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	manager, exists := f.managers[managerType]
	return manager, exists
}

// ClearCache 清空统计缓存
func (f *StatsFactory) ClearCache() {
	f.cache.Clear()
}

func (f *StatsFactory) Repo() *store.Repository {
	return f.repo
}

// QueryStats 通过指定类型的管理器查询统计数据
func (f *StatsFactory) QueryStats(managerType string, query StatsQuery) (StatsResult, error) {
	if err := validateWebsiteID(query.WebsiteID); err != nil {
		return nil, err
	}
	// 获取对应的管理器
	manager, exists := f.GetManager(managerType)
	if !exists {
		return nil, fmt.Errorf("未找到统计管理器: %s", managerType)
	}

	if f.shouldCache(managerType) {
		// 构建缓存键
		cacheKey := f.buildCacheKey(managerType, query)

		// 尝试从缓存获取
		if cachedResult, ok := f.cache.Get(cacheKey, f.cacheExpiry); ok {
			return cachedResult.(StatsResult), nil
		}

		// 执行查询
		result, err := manager.Query(query)
		if err != nil {
			return nil, err
		}

		// 缓存结果
		f.cache.Set(cacheKey, result)

		return result, nil
	}

	// 执行查询
	result, err := manager.Query(query)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (f *StatsFactory) shouldCache(managerType string) bool {
	switch managerType {
	case "logs", "realtime", "session":
		return false
	default:
		return true
	}
}

// buildCacheKey 构建缓存键
func (f *StatsFactory) buildCacheKey(managerType string, query StatsQuery) string {
	// 基础键：统计类型-网站ID
	key := fmt.Sprintf("%s-%s", managerType, query.WebsiteID)

	// 拼接所有额外参数
	if query.ExtraParam != nil {
		keys := make([]string, 0, len(query.ExtraParam))
		for paramKey := range query.ExtraParam {
			keys = append(keys, paramKey)
		}
		sort.Strings(keys)
		for _, paramKey := range keys {
			paramValue := query.ExtraParam[paramKey]
			switch v := paramValue.(type) {
			case string:
				key = fmt.Sprintf("%s-%s:%s", key, paramKey, v)
			case int:
				key = fmt.Sprintf("%s-%s:%d", key, paramKey, v)
			case float64:
				key = fmt.Sprintf("%s-%s:%f", key, paramKey, v)
			case bool:
				key = fmt.Sprintf("%s-%s:%t", key, paramKey, v)
			case time.Time:
				key = fmt.Sprintf("%s-%s:%d", key, paramKey, v.Unix())
			default:
				key = fmt.Sprintf("%s-%s:%v", key, paramKey, v)
			}
		}
	}

	return key
}

// BuildQueryFromRequest 根据请求参数构建查询对象
func (f *StatsFactory) BuildQueryFromRequest(
	statsType string, params map[string]string) (StatsQuery, error) {

	query := StatsQuery{
		WebsiteID:  "",
		ExtraParam: make(map[string]interface{}),
	}

	// 定义每种统计类型需要的参数
	requiredParams := map[string]map[string]string{
		"timeseries":       {"id": "string", "timeRange": "string", "viewType": "string"},
		"overall":          {"id": "string", "timeRange": "string"},
		"developer_daily":  {"id": "string", "timeRange": "string"},
		"url":              {"id": "string", "timeRange": "string", "limit": "int"},
		"referer":          {"id": "string", "timeRange": "string", "limit": "int"},
		"referer_ip":       {"id": "string", "timeRange": "string", "limit": "int"},
		"referer_ip_batch": {"id": "string", "timeRange": "string", "limit": "int"},
		"browser":          {"id": "string", "timeRange": "string", "limit": "int"},
		"os":               {"id": "string", "timeRange": "string", "limit": "int"},
		"device":           {"id": "string", "timeRange": "string", "limit": "int"},
		"location":         {"id": "string", "timeRange": "string", "limit": "int", "locationType": "enum:domestic,city,global"},
		"logs":             {"id": "string", "page": "int", "pageSize": "int", "sortField": "string", "sortOrder": "enum:asc,desc"},
		"session":          {"id": "string", "page": "int", "pageSize": "int"},
		"session_summary":  {"id": "string", "timeRange": "string"},
		"realtime":         {"id": "string"},
	}

	// 检查是否支持的统计类型
	paramDefs, exists := requiredParams[statsType]
	if !exists {
		return query, fmt.Errorf("不支持的统计类型: %s", statsType)
	}

	// 获取网站ID
	websiteID, err := getRequiredString(params, "id")
	if err != nil {
		return query, err
	}
	if err := validateWebsiteID(websiteID); err != nil {
		return query, err
	}
	query.WebsiteID = websiteID

	// 处理其他参数
	for paramName, paramType := range paramDefs {
		// 跳过已处理的id参数
		if paramName == "id" {
			continue
		}

		switch {
		case paramType == "string":
			value, err := getRequiredString(params, paramName)
			if err != nil {
				return query, err
			}
			query.ExtraParam[paramName] = value

		case paramType == "int":
			value, err := getRequiredInt(params, paramName, 1)
			if err != nil {
				return query, err
			}
			query.ExtraParam[paramName] = value

		case strings.HasPrefix(paramType, "enum:"):
			// 处理枚举类型，如 "enum:asc,desc"
			allowedValues := strings.Split(strings.TrimPrefix(paramType, "enum:"), ",")
			value, err := getRequiredStringEnum(params, paramName, allowedValues)
			if err != nil {
				return query, err
			}
			query.ExtraParam[paramName] = value
		}
	}

	// 处理特殊可选参数
	if statsType == "logs" {
		if filter, ok := params["filter"]; ok && filter != "" {
			query.ExtraParam["filter"] = filter
		}
		if timeRange, ok := params["timeRange"]; ok && timeRange != "" {
			query.ExtraParam["timeRange"] = timeRange
		}
		if timeStart, ok := params["timeStart"]; ok && timeStart != "" {
			query.ExtraParam["timeStart"] = timeStart
		}
		if timeEnd, ok := params["timeEnd"]; ok && timeEnd != "" {
			query.ExtraParam["timeEnd"] = timeEnd
		}
		if statusCodeRaw, ok := params["statusCode"]; ok && statusCodeRaw != "" {
			value, err := strconv.Atoi(statusCodeRaw)
			if err != nil {
				return query, fmt.Errorf("statusCode 参数格式错误")
			}
			query.ExtraParam["statusCode"] = value
		}
		if statusClass, ok := params["statusClass"]; ok && statusClass != "" {
			valid := map[string]bool{
				"2xx": true,
				"3xx": true,
				"4xx": true,
				"5xx": true,
			}
			if !valid[statusClass] {
				return query, fmt.Errorf("statusClass 参数无效")
			}
			query.ExtraParam["statusClass"] = statusClass
		}
		if excludeInternalRaw, ok := params["excludeInternal"]; ok && excludeInternalRaw != "" {
			switch strings.ToLower(excludeInternalRaw) {
			case "true", "1":
				query.ExtraParam["excludeInternal"] = true
			case "false", "0":
				query.ExtraParam["excludeInternal"] = false
			default:
				return query, fmt.Errorf("excludeInternal 参数无效")
			}
		}
		if excludeSpiderRaw, ok := params["excludeSpider"]; ok && excludeSpiderRaw != "" {
			switch strings.ToLower(excludeSpiderRaw) {
			case "true", "1":
				query.ExtraParam["excludeSpider"] = true
			case "false", "0":
				query.ExtraParam["excludeSpider"] = false
			default:
				return query, fmt.Errorf("excludeSpider 参数无效")
			}
		}
		if excludeForeignRaw, ok := params["excludeForeign"]; ok && excludeForeignRaw != "" {
			switch strings.ToLower(excludeForeignRaw) {
			case "true", "1":
				query.ExtraParam["excludeForeign"] = true
			case "false", "0":
				query.ExtraParam["excludeForeign"] = false
			default:
				return query, fmt.Errorf("excludeForeign 参数无效")
			}
		}
		if ipFilter, ok := params["ipFilter"]; ok && ipFilter != "" {
			query.ExtraParam["ipFilter"] = ipFilter
		}
		if locationFilter, ok := params["locationFilter"]; ok && locationFilter != "" {
			query.ExtraParam["locationFilter"] = locationFilter
		}
		if urlFilter, ok := params["urlFilter"]; ok && urlFilter != "" {
			query.ExtraParam["urlFilter"] = urlFilter
		}
		if pageviewOnlyRaw, ok := params["pageviewOnly"]; ok && pageviewOnlyRaw != "" {
			switch strings.ToLower(pageviewOnlyRaw) {
			case "true", "1":
				query.ExtraParam["pageviewOnly"] = true
			case "false", "0":
				query.ExtraParam["pageviewOnly"] = false
			default:
				return query, fmt.Errorf("pageviewOnly 参数无效")
			}
		}
		if newVisitor, ok := params["newVisitor"]; ok && newVisitor != "" {
			valid := map[string]bool{
				"all":       true,
				"new":       true,
				"returning": true,
			}
			if !valid[newVisitor] {
				return query, fmt.Errorf("newVisitor 参数无效")
			}
			query.ExtraParam["newVisitor"] = newVisitor
		}
		if distinctIpRaw, ok := params["distinctIp"]; ok && distinctIpRaw != "" {
			switch strings.ToLower(distinctIpRaw) {
			case "true", "1":
				query.ExtraParam["distinctIp"] = true
			case "false", "0":
				query.ExtraParam["distinctIp"] = false
			default:
				return query, fmt.Errorf("distinctIp 参数无效")
			}
		}
	}
	if statsType == "session" {
		if timeRange, ok := params["timeRange"]; ok && timeRange != "" {
			query.ExtraParam["timeRange"] = timeRange
		}
		if timeStart, ok := params["timeStart"]; ok && timeStart != "" {
			query.ExtraParam["timeStart"] = timeStart
		}
		if timeEnd, ok := params["timeEnd"]; ok && timeEnd != "" {
			query.ExtraParam["timeEnd"] = timeEnd
		}
		if ipFilter, ok := params["ipFilter"]; ok && ipFilter != "" {
			query.ExtraParam["ipFilter"] = ipFilter
		}
		if deviceFilter, ok := params["deviceFilter"]; ok && deviceFilter != "" {
			query.ExtraParam["deviceFilter"] = deviceFilter
		}
		if browserFilter, ok := params["browserFilter"]; ok && browserFilter != "" {
			query.ExtraParam["browserFilter"] = browserFilter
		}
		if osFilter, ok := params["osFilter"]; ok && osFilter != "" {
			query.ExtraParam["osFilter"] = osFilter
		}
	}
	if statsType == "overall" {
		if entryLimit, ok := params["entryLimit"]; ok && entryLimit != "" {
			value, err := getRequiredInt(params, "entryLimit", 1)
			if err != nil {
				return query, err
			}
			query.ExtraParam["entryLimit"] = value
		}
	}
	if statsTypeSupportsURLFilter(statsType) {
		if urlFilter, ok := params["urlFilter"]; ok && strings.TrimSpace(urlFilter) != "" {
			query.ExtraParam["urlFilter"] = strings.TrimSpace(urlFilter)
		}
	}
	if statsType == "realtime" {
		if windowRaw, ok := params["window"]; ok && windowRaw != "" {
			value, err := strconv.Atoi(windowRaw)
			if err != nil {
				return query, fmt.Errorf("window 参数格式错误")
			}
			query.ExtraParam["window"] = value
		}
	}
	if statsType == "referer_ip" {
		if sourceKind, ok := params["sourceKind"]; ok && sourceKind != "" {
			valid := map[string]bool{
				"all":      true,
				"search":   true,
				"direct":   true,
				"external": true,
			}
			if !valid[sourceKind] {
				return query, fmt.Errorf("sourceKind 参数无效")
			}
			query.ExtraParam["sourceKind"] = sourceKind
		}
	}

	return query, nil
}

func statsTypeSupportsURLFilter(statsType string) bool {
	switch statsType {
	case "timeseries",
		"overall",
		"developer_daily",
		"url",
		"referer",
		"referer_ip",
		"referer_ip_batch",
		"browser",
		"os",
		"device",
		"location",
		"session_summary":
		return true
	default:
		return false
	}
}

// Website IDs become SQL identifier prefixes. Only accept IDs issued by config,
// including runtime-discovered sites (whose collision suffix can contain '_').
func validateWebsiteID(id string) error {
	if _, ok := config.GetWebsiteByID(id); !ok {
		return fmt.Errorf("站点不存在")
	}
	return nil
}

// getRequiredInt 获取并验证必须的整数参数
func getRequiredInt(params map[string]string, key string, minValue int) (int, error) {
	if valueStr, ok := params[key]; ok && valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil && value >= minValue {
			return value, nil
		}
		return 0, fmt.Errorf("%s 参数无效，必须为大于等于 %d 的整数", key, minValue)
	}
	return 0, fmt.Errorf("缺少必要参数: %s", key)
}

// getRequiredString 获取并验证必须的字符串参数
func getRequiredString(params map[string]string, key string) (string, error) {
	if value, ok := params[key]; ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("缺少必要参数: %s", key)
}

// getRequiredStringEnum 获取并验证必须的字符串参数，且值必须在允许列表中
func getRequiredStringEnum(params map[string]string, key string, allowedValues []string) (string, error) {
	value, err := getRequiredString(params, key)
	if err != nil {
		return "", err
	}

	for _, allowed := range allowedValues {
		if value == allowed {
			return value, nil
		}
	}

	return "", fmt.Errorf("%s 参数无效，必须为以下值之一: %v", key, allowedValues)
}
