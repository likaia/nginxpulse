package ingest

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/likaia/nginxpulse/internal/alertpush"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/enrich"
	"github.com/likaia/nginxpulse/internal/ingest/dedup"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

var (
	defaultNginxLogRegex        = `^(?P<ip>\S+) - (?P<user>\S+) \[(?P<time>[^\]]+)\] "(?P<method>\S+) (?P<url>[^"]+) HTTP/\d(?:\.\d+)?" (?P<status>\d+) (?P<bytes>\d+|-) "(?P<referer>[^"]*)" "(?P<ua>[^"]*)"(?: (?P<trace_first>\S+) (?P<trace_second>\S+)(?: (?P<upstream_response_time>[^ ]+(?:,\s*[^ ]+)*))?(?: (?P<upstream_addr>\S+))?(?: (?P<host>\S+))?(?: (?P<request_id>\S+))?)?(?: .*)?$`
	defaultApacheLogRegex       = `^(?P<ip>\S+) (?P<ident>\S+) (?P<user>\S+) \[(?P<time>[^\]]+)\] "(?P<request>[^"]*)" (?P<status>\d{3}) (?P<bytes>\d+|-) "(?P<referer>[^"]*)" "(?P<ua>[^"]*)"`
	defaultTraefikLogRegex      = `^(?P<ip>\S+) (?P<ident>\S+) (?P<user>\S+) \[(?P<time>[^\]]+)\] "(?P<request>[^"]*)" (?P<status>\d{3}) (?P<bytes>\d+|-) "(?P<referer>[^"]*)" "(?P<ua>[^"]*)" (?P<req_count>\d+) "(?P<router>[^"]*)" "(?P<server_url>[^"]*)" (?P<duration_ms>[0-9.]+)ms`
	defaultEnvoyLogRegex        = `^\[(?P<time>[^\]]+)\] "(?P<request>[^"]*)" (?P<status>\d{3}) (?P<response_flags>\S+) (?P<bytes_received>\d+) (?P<bytes>\d+) (?P<duration>\d+) (?P<upstream_time>\S+) "(?P<ip>[^"]*)" "(?P<ua>[^"]*)" "(?P<request_id>[^"]*)" "(?P<authority>[^"]*)" "(?P<upstream_host>[^"]*)"`
	defaultHAProxyLogRegex      = `^(?:\w{3}\s+\d+\s+\d+:\d+:\d+\s+\S+\s+\S+\[\d+\]:\s+)?(?P<ip>\S+):\d+\s+\[(?P<time>[^\]]+)\]\s+\S+\s+\S+\s+-?\d+/-?\d+/-?\d+/-?\d+/-?\d+\s+(?P<status>\d{3})\s+(?P<bytes>\d+|-)\s+\S+\s+\S+\s+\S+\s+-?\d+/-?\d+/-?\d+/-?\d+/-?\d+\s+-?\d+/-?\d+(?:\s+(?:\{[^\}]*\}|-)){0,2}\s+\"(?P<request>[^\"]*)\"`
	defaultNginxIngressLogRegex = `^(?P<ip>\S+) - (?P<user>\S+) \[(?P<time>[^\]]+)\] "(?P<request>[^"]*)" (?P<status>\d{3}) (?P<bytes>\d+|-) "(?P<referer>[^"]*)" "(?P<ua>[^"]*)" (?P<request_length>\d+) (?P<request_time>[0-9.]+) \[(?P<proxy_upstream_name>[^\]]*)\] \[(?P<proxy_alternative_upstream_name>[^\]]*)\] (?P<upstream_addr>[^ ]+(?:,\s*[^ ]+)*) (?P<upstream_response_length>[^ ]+(?:,\s*[^ ]+)*) (?P<upstream_response_time>[^ ]+(?:,\s*[^ ]+)*) (?P<upstream_status>[^ ]+(?:,\s*[^ ]+)*) (?P<req_id>\S+)`
	defaultIISW3CLogRegex       = `^(?P<time>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+\S+\s+(?P<method>\S+)\s+(?P<url>\S+)\s+(?P<query>\S+)\s+\S+\s+\S+\s+(?P<ip>\S+)\s+(?P<ua>\S+)\s+(?P<referer>\S+)\s+(?P<status>\d{3})(?:\s+\S+){3}\s*$`
	defaultNPMLogRegex          = `^\[(?P<time>[^\]]+)\] - (?P<status>\d+) (?P<upstream_status>\d+) - (?P<method>\S+) (?P<scheme>\S+) (?P<host>\S+) "(?P<path>[^"]+)" \[Client (?P<ip>[^\]]+)\] \[Length (?P<bytes>\d+)\] \[Gzip (?P<gzip>[^\]]+)\] \[Sent-to (?P<upstream>[^\]]+)\] "(?P<ua>[^"]+)" "(?P<referer>[^"]*)"`
	defaultSafeLineWAFLogRegex  = `^(?P<ip>\S+)\s*(?:\|\s*|-\s+)(?P<user>\S+)\s*(?:\|\s*|\[\s*)(?P<time>[^\]|]+?)(?:\]\s+|\s*\|\s*)"(?P<host>[^"]*)"\s*(?:\|\s*|\s+)"(?P<request>[^"]*)"\s*(?:\|\s*|\s+)(?P<status>\d{3})\s*(?:\|\s*|\s+)(?P<bytes>\d+|-)\s*(?:\|\s*|\s+)"(?P<referer>[^"]*)"\s*(?:\|\s*|\s+)"(?P<ua>[^"]*)"(?:\s*(?:\|\s*|\s+)"(?P<http_x_forwarded_for>[^"]*)")?$`
	defaultZoraxyLogRegex       = `^\[?(?P<time>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?)\]?\s+\[router:(?P<router>[^\]]+)\]\s+\[origin:(?P<host>[^\]]*)\]\s+\[client:\s*(?P<ip>[^\]]+)\]\s+\[useragent:\s*(?P<ua>[^\]]*)\]\s+(?P<method>\S+)\s+(?P<url>\S+)\s+(?P<status>\d{3})\s*$`
	lastCleanupDate             = ""
	parsingMu                   sync.RWMutex
	parsingMode                 parseMode
	parsingWebsiteID            string
)

const defaultNginxTimeLayout = "02/Jan/2006:15:04:05 -0700"
const defaultHAProxyTimeLayout = "02/Jan/2006:15:04:05.000"
const defaultIISTimeLayout = "2006-01-02 15:04:05"
const defaultZoraxyTimeLayout = "2006-01-02 15:04:05.000000"

const (
	parseTypeRegex     = "regex"
	parseTypeCaddyJSON = "caddy_json"
)

const (
	recentLogWindowDays   = 7
	recentScanChunkSize   = 256 * 1024
	defaultParseBatchSize = 100
)

var (
	ipAliases            = []string{"ip", "remote_addr", "real_client_ip", "client_ip", "http_x_forwarded_for"}
	timeAliases          = []string{"time", "time_local", "time_iso8601"}
	methodAliases        = []string{"method", "request_method"}
	urlAliases           = []string{"url", "request_uri", "uri", "path"}
	queryAliases         = []string{"query", "args", "query_string", "cs_uri_query"}
	statusAliases        = []string{"status"}
	bytesAliases         = []string{"bytes", "body_bytes_sent", "bytes_sent"}
	refererAliases       = []string{"referer", "http_referer"}
	userAgentAliases     = []string{"ua", "user_agent", "http_user_agent"}
	requestAliases       = []string{"request", "request_line"}
	requestLengthAliases = []string{"request_length", "bytes_received"}
	requestTimeAliases   = []string{"request_time_ms", "request_time_msec", "request_time", "duration_ms", "duration"}
	upstreamTimeAliases  = []string{"upstream_response_time", "upstream_time"}
	upstreamAddrAliases  = []string{"upstream_addr", "upstream", "upstream_host"}
	hostAliases          = []string{"host", "http_host", "server_name", "authority"}
	requestIDAliases     = []string{"request_id", "req_id", "x_request_id"}
)

var ErrParsingInProgress = errors.New("日志解析中，请稍后重试")

// 解析结果
type ParserResult struct {
	WebName      string
	WebID        string
	TotalEntries int
	Duration     time.Duration
	Success      bool
	Error        error
}

type LogScanState struct {
	Files             map[string]FileState   `json:"files"` // 每个文件的状态
	Targets           map[string]TargetState `json:"targets,omitempty"`
	ParsedHourBuckets map[int64]bool         `json:"parsed_hour_buckets,omitempty"`
	ParsedMinTs       int64                  `json:"parsed_min_ts,omitempty"`
	ParsedMaxTs       int64                  `json:"parsed_max_ts,omitempty"`
	LogMinTs          int64                  `json:"log_min_ts,omitempty"`
	LogMaxTs          int64                  `json:"log_max_ts,omitempty"`
	RecentCutoffTs    int64                  `json:"recent_cutoff_ts,omitempty"`
	BackfillPending   bool                   `json:"backfill_pending,omitempty"`
	InitialParsed     bool                   `json:"initial_parsed,omitempty"`
}

type FileState struct {
	LastOffset     int64 `json:"last_offset"`
	LastSize       int64 `json:"last_size"`
	RecentOffset   int64 `json:"recent_offset,omitempty"`
	BackfillOffset int64 `json:"backfill_offset,omitempty"`
	BackfillEnd    int64 `json:"backfill_end,omitempty"`
	BackfillDone   bool  `json:"backfill_done,omitempty"`
	FirstTimestamp int64 `json:"first_ts,omitempty"`
	LastTimestamp  int64 `json:"last_ts,omitempty"`
	ParsedMinTs    int64 `json:"parsed_min_ts,omitempty"`
	ParsedMaxTs    int64 `json:"parsed_max_ts,omitempty"`
	RecentCutoffTs int64 `json:"recent_cutoff_ts,omitempty"`
}

type TargetState struct {
	LastOffset     int64  `json:"last_offset"`
	LastSize       int64  `json:"last_size"`
	LastModTime    int64  `json:"last_mtime,omitempty"`
	LastETag       string `json:"last_etag,omitempty"`
	RecentOffset   int64  `json:"recent_offset,omitempty"`
	BackfillOffset int64  `json:"backfill_offset,omitempty"`
	BackfillEnd    int64  `json:"backfill_end,omitempty"`
	BackfillDone   bool   `json:"backfill_done,omitempty"`
	FirstTimestamp int64  `json:"first_ts,omitempty"`
	LastTimestamp  int64  `json:"last_ts,omitempty"`
	ParsedMinTs    int64  `json:"parsed_min_ts,omitempty"`
	ParsedMaxTs    int64  `json:"parsed_max_ts,omitempty"`
	RecentCutoffTs int64  `json:"recent_cutoff_ts,omitempty"`
}

type parseMode int

const (
	parseModeNone parseMode = iota
	parseModeForeground
	parseModeBackfill
)

type parseWindow struct {
	minTs int64
	maxTs int64
}

func (w parseWindow) allows(ts int64) bool {
	if w.minTs > 0 && ts < w.minTs {
		return false
	}
	if w.maxTs > 0 && ts >= w.maxTs {
		return false
	}
	return true
}

type logLineParser struct {
	regex      *regexp.Regexp
	indexMap   map[string]int
	timeLayout string
	source     string
	parseType  string
}

type LogParser struct {
	repo              *store.Repository
	statePath         string
	states            map[string]LogScanState // 各网站的扫描状态，以网站ID为键
	demoMode          bool
	retentionDays     int
	parseBatchSize    int
	ipGeoCacheLimit   int
	lineParsers       map[string]*logLineParser // key: websiteID or websiteID:sourceID
	dedup             *dedup.Cache
	whitelistMatchers map[string]*enrich.WhitelistMatcher
	alertDispatcher   *alertpush.Dispatcher
}

// NewLogParser 创建新的日志解析器
func NewLogParser(userRepoPtr *store.Repository) *LogParser {
	statePath := filepath.Join(config.DataDir, "nginx_scan_state.json")
	cfg := config.ReadConfig()
	retentionDays := cfg.System.LogRetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}
	parseBatchSize := cfg.System.ParseBatchSize
	if parseBatchSize <= 0 {
		parseBatchSize = defaultParseBatchSize
	}
	ipGeoCacheLimit := cfg.System.IPGeoCacheLimit
	if ipGeoCacheLimit <= 0 {
		ipGeoCacheLimit = 1000000
	}
	parser := &LogParser{
		repo:              userRepoPtr,
		statePath:         statePath,
		states:            make(map[string]LogScanState),
		demoMode:          cfg.System.DemoMode,
		retentionDays:     retentionDays,
		parseBatchSize:    parseBatchSize,
		ipGeoCacheLimit:   ipGeoCacheLimit,
		lineParsers:       make(map[string]*logLineParser),
		dedup:             dedup.NewCache(100000, 10*time.Minute),
		whitelistMatchers: make(map[string]*enrich.WhitelistMatcher),
		alertDispatcher:   alertpush.NewDispatcher(cfg.System.AlertPush),
	}
	for _, websiteID := range config.GetAllWebsiteIDs() {
		if site, ok := config.GetWebsiteByID(websiteID); ok {
			if matcher := enrich.NewWhitelistMatcher(site.Whitelist); matcher != nil {
				parser.whitelistMatchers[websiteID] = matcher
			}
		}
	}
	parser.loadState()
	parser.resetStateIfEmptyDB()
	enrich.InitPVFilters()
	return parser
}

// loadState 加载上次扫描状态
func (p *LogParser) loadState() {
	data, err := os.ReadFile(p.statePath)
	if os.IsNotExist(err) {
		// 状态文件不存在，创建空状态映射
		p.states = make(map[string]LogScanState)
		return
	}

	if err != nil {
		logrus.Errorf("无法读取扫描状态文件: %v", err)
		p.notifyFileIO("", p.statePath, "读取扫描状态文件", err)
		p.states = make(map[string]LogScanState)
		return
	}

	if err := json.Unmarshal(data, &p.states); err != nil {
		logrus.Errorf("解析扫描状态失败: %v", err)
		p.notifyFileIO("", p.statePath, "解析扫描状态文件", err)
		p.states = make(map[string]LogScanState)
	}

	for websiteID, state := range p.states {
		if state.Files == nil {
			state.Files = make(map[string]FileState)
		}
		normalizedFiles := make(map[string]FileState, len(state.Files))
		for path, fileState := range state.Files {
			normalizedFiles[normalizeLogPath(path)] = fileState
		}
		state.Files = normalizedFiles
		if state.Targets == nil {
			state.Targets = make(map[string]TargetState)
		}
		if state.ParsedHourBuckets == nil {
			state.ParsedHourBuckets = make(map[int64]bool)
		}
		p.states[websiteID] = state
		p.refreshWebsiteRanges(websiteID)
	}
}

// updateState 更新并保存状态
func (p *LogParser) updateState() {
	data, err := json.Marshal(p.states)
	if err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
		p.notifyFileIO("", p.statePath, "序列化扫描状态文件", err)
		return
	}

	tmpPath := p.statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
		p.notifyFileIO("", p.statePath, "写入扫描状态文件", err)
		return
	}
	if err := os.Rename(tmpPath, p.statePath); err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
		p.notifyFileIO("", p.statePath, "保存扫描状态文件", err)
	}
}

func (p *LogParser) resetStateIfEmptyDB() {
	if p.demoMode {
		return
	}

	websiteIDs := config.GetAllWebsiteIDs()
	if len(websiteIDs) == 0 {
		return
	}

	hasState := false
	for _, id := range websiteIDs {
		state, ok := p.states[id]
		if !ok {
			continue
		}
		if len(state.Files) > 0 || len(state.Targets) > 0 || len(state.ParsedHourBuckets) > 0 ||
			state.ParsedMaxTs > 0 || state.LogMaxTs > 0 {
			hasState = true
			break
		}
	}
	if !hasState {
		return
	}

	for _, id := range websiteIDs {
		hasLogs, err := p.repo.HasLogs(id)
		if err != nil {
			logrus.WithError(err).Warnf("检测网站 %s 日志数据失败，跳过自动清理扫描状态", id)
			return
		}
		if hasLogs {
			return
		}
	}

	logrus.Warn("检测到扫描状态已存在但数据库日志为空，自动清理扫描状态以便重新解析")
	p.ResetScanState("")
}

func (p *LogParser) ensureWebsiteState(websiteID string) LogScanState {
	state, ok := p.states[websiteID]
	if !ok {
		state = LogScanState{
			Files:   make(map[string]FileState),
			Targets: make(map[string]TargetState),
		}
	}
	if state.Files == nil {
		state.Files = make(map[string]FileState)
	}
	if state.Targets == nil {
		state.Targets = make(map[string]TargetState)
	}
	if state.ParsedHourBuckets == nil {
		state.ParsedHourBuckets = make(map[int64]bool)
	}
	return state
}

func (p *LogParser) hasUnparsedWebsite(websiteIDs []string) bool {
	for _, id := range websiteIDs {
		state, ok := p.states[id]
		if !ok || !state.InitialParsed {
			return true
		}
	}
	return false
}

func (p *LogParser) markInitialParsed(websiteID string) {
	state := p.ensureWebsiteState(websiteID)
	if state.InitialParsed {
		return
	}
	state.InitialParsed = true
	p.states[websiteID] = state
}

func (p *LogParser) getFileState(websiteID, filePath string) (FileState, bool) {
	state, ok := p.states[websiteID]
	if !ok || state.Files == nil {
		return FileState{}, false
	}
	fileState, ok := state.Files[normalizeLogPath(filePath)]
	return fileState, ok
}

func (p *LogParser) setFileState(websiteID, filePath string, fileState FileState) {
	state := p.ensureWebsiteState(websiteID)
	state.Files[normalizeLogPath(filePath)] = fileState
	p.states[websiteID] = state
}

func (p *LogParser) deleteFileState(websiteID, filePath string) {
	state, ok := p.states[websiteID]
	if !ok || state.Files == nil {
		return
	}
	delete(state.Files, normalizeLogPath(filePath))
	p.states[websiteID] = state
}

func (p *LogParser) recordParsedHourBuckets(websiteID string, buckets map[int64]struct{}) {
	if len(buckets) == 0 {
		return
	}
	state := p.ensureWebsiteState(websiteID)
	if state.ParsedHourBuckets == nil {
		state.ParsedHourBuckets = make(map[int64]bool)
	}
	for bucket := range buckets {
		state.ParsedHourBuckets[bucket] = true
	}
	p.states[websiteID] = state
}

func (p *LogParser) getTargetState(websiteID, targetKey string) (TargetState, bool) {
	state, ok := p.states[websiteID]
	if !ok || state.Targets == nil {
		return TargetState{}, false
	}
	targetState, ok := state.Targets[targetKey]
	return targetState, ok
}

func (p *LogParser) setTargetState(websiteID, targetKey string, targetState TargetState) {
	state := p.ensureWebsiteState(websiteID)
	state.Targets[targetKey] = targetState
	p.states[websiteID] = state
}

func (p *LogParser) deleteTargetState(websiteID, targetKey string) {
	state, ok := p.states[websiteID]
	if !ok || state.Targets == nil {
		return
	}
	delete(state.Targets, targetKey)
	p.states[websiteID] = state
}

func (p *LogParser) refreshWebsiteRanges(websiteID string) {
	state, ok := p.states[websiteID]
	if !ok || (state.Files == nil && state.Targets == nil) {
		return
	}

	var logMin, logMax int64
	var parsedMin, parsedMax int64
	var recentCutoff int64
	backfillPending := false
	var backfillTotalBytes int64
	var backfillProcessedBytes int64

	applyState := func(firstTs, lastTs, parsedMinTs, parsedMaxTs, recentCutoffTs int64, backfillDone bool, backfillEnd, backfillOffset int64) {
		if firstTs > 0 {
			if logMin == 0 || firstTs < logMin {
				logMin = firstTs
			}
		}
		if lastTs > 0 {
			if logMax == 0 || lastTs > logMax {
				logMax = lastTs
			}
		}
		if parsedMinTs > 0 {
			if parsedMin == 0 || parsedMinTs < parsedMin {
				parsedMin = parsedMinTs
			}
		}
		if parsedMaxTs > 0 {
			if parsedMax == 0 || parsedMaxTs > parsedMax {
				parsedMax = parsedMaxTs
			}
		}
		if recentCutoffTs > 0 {
			if recentCutoff == 0 || recentCutoffTs < recentCutoff {
				recentCutoff = recentCutoffTs
			}
		}
		if !backfillDone {
			if backfillEnd > backfillOffset || backfillEnd == 0 {
				backfillPending = true
			}
		}
	}
	accumulateBackfill := func(done bool, backfillEnd, backfillOffset, lastSize int64) {
		total, processed := computeBackfillBytes(done, backfillEnd, backfillOffset, lastSize)
		backfillTotalBytes += total
		backfillProcessedBytes += processed
	}

	for _, fileState := range state.Files {
		applyState(
			fileState.FirstTimestamp,
			fileState.LastTimestamp,
			fileState.ParsedMinTs,
			fileState.ParsedMaxTs,
			fileState.RecentCutoffTs,
			fileState.BackfillDone,
			fileState.BackfillEnd,
			fileState.BackfillOffset,
		)
		accumulateBackfill(fileState.BackfillDone, fileState.BackfillEnd, fileState.BackfillOffset, fileState.LastSize)
	}

	for _, targetState := range state.Targets {
		applyState(
			targetState.FirstTimestamp,
			targetState.LastTimestamp,
			targetState.ParsedMinTs,
			targetState.ParsedMaxTs,
			targetState.RecentCutoffTs,
			targetState.BackfillDone,
			targetState.BackfillEnd,
			targetState.BackfillOffset,
		)
		accumulateBackfill(targetState.BackfillDone, targetState.BackfillEnd, targetState.BackfillOffset, targetState.LastSize)
	}

	if logMin == 0 && parsedMin > 0 {
		logMin = parsedMin
	}
	if logMax == 0 && parsedMax > 0 {
		logMax = parsedMax
	}
	if parsedMin == 0 && recentCutoff > 0 {
		parsedMin = recentCutoff
	}

	state.LogMinTs = logMin
	state.LogMaxTs = logMax
	state.ParsedMinTs = parsedMin
	state.ParsedMaxTs = parsedMax
	state.RecentCutoffTs = recentCutoff
	state.BackfillPending = backfillPending
	p.states[websiteID] = state

	UpdateWebsiteParseStatus(websiteID, WebsiteParseStatus{
		LogMinTs:               logMin,
		LogMaxTs:               logMax,
		ParsedMinTs:            parsedMin,
		ParsedMaxTs:            parsedMax,
		RecentCutoffTs:         recentCutoff,
		BackfillPending:        backfillPending,
		BackfillTotalBytes:     backfillTotalBytes,
		BackfillProcessedBytes: backfillProcessedBytes,
		ParsedHourBuckets:      state.ParsedHourBuckets,
	})
}

func computeBackfillBytes(done bool, backfillEnd, backfillOffset, lastSize int64) (int64, int64) {
	if done {
		total := backfillEnd
		if total <= 0 {
			total = lastSize
		}
		if total <= 0 {
			return 0, 0
		}
		return total, total
	}
	if backfillEnd > 0 {
		processed := backfillOffset
		if processed < 0 {
			processed = 0
		}
		if processed > backfillEnd {
			processed = backfillEnd
		}
		return backfillEnd, processed
	}
	if lastSize > 0 {
		return lastSize, 0
	}
	return 0, 0
}

// CleanOldLogs 清理保留天数之前的日志数据
func (p *LogParser) CleanOldLogs() error {
	today := time.Now().Format("2006-01-02")
	currentHour := time.Now().Hour()

	shouldClean := lastCleanupDate == "" || (currentHour == 2 && lastCleanupDate != today)

	if !shouldClean {
		return nil
	}

	err := p.repo.CleanOldLogs()
	if err != nil {
		return err
	}

	lastCleanupDate = today

	return nil
}

// ScanNginxLogs 增量扫描Nginx日志文件
func (p *LogParser) ScanNginxLogs() []ParserResult {
	if p.demoMode {
		return []ParserResult{}
	}
	p.discoverAutoHostWebsites()
	websiteIDs := config.GetAllWebsiteIDs()
	stage := parseStagePeriodic
	if p.hasUnparsedWebsite(websiteIDs) {
		stage = parseStageInitial
	}
	if !startIPParsingWithStage(stage) {
		return []ParserResult{}
	}
	defer finishIPParsing()

	return p.scanNginxLogsInternal(websiteIDs)
}

// ScanNginxLogsForWebsite 扫描指定网站的日志文件
func (p *LogParser) ScanNginxLogsForWebsite(websiteID string) []ParserResult {
	if p.demoMode {
		return []ParserResult{}
	}
	stage := parseStagePeriodic
	if p.hasUnparsedWebsite([]string{websiteID}) {
		stage = parseStageInitial
	}
	if !startIPParsingWithStage(stage) {
		return []ParserResult{}
	}
	defer finishIPParsing()

	return p.scanNginxLogsInternal([]string{websiteID})
}

// ResetScanState 重置日志扫描状态
func (p *LogParser) ResetScanState(websiteID string) {
	if websiteID == "" {
		p.states = make(map[string]LogScanState)
		ResetWebsiteParseStatus("")
	} else {
		delete(p.states, websiteID)
		ResetWebsiteParseStatus(websiteID)
	}
	p.updateState()
}

// TriggerReparse 清空指定网站的日志并触发重新解析
func (p *LogParser) TriggerReparse(websiteID string) error {
	if p.demoMode {
		var err error
		if websiteID == "" {
			err = p.repo.ClearAllLogs()
		} else {
			err = p.repo.ClearLogsForWebsite(websiteID)
		}
		if err != nil {
			return err
		}
		p.ResetScanState(websiteID)
		return nil
	}

	if !startIPParsingWithStage(parseStageReparse) {
		return ErrParsingInProgress
	}

	var ids []string
	if websiteID == "" {
		ids = config.GetAllWebsiteIDs()
	} else {
		ids = []string{websiteID}
	}

	var err error
	if websiteID == "" {
		err = p.repo.ClearAllLogs()
	} else {
		err = p.repo.ClearLogsForWebsite(websiteID)
	}
	if err != nil {
		finishIPParsing()
		return err
	}

	p.ResetScanState(websiteID)

	go func() {
		defer finishIPParsing()
		p.scanNginxLogsInternal(ids)
	}()

	return nil
}

func (p *LogParser) scanNginxLogsInternal(websiteIDs []string) []ParserResult {
	setParsingTotalBytes(p.calculateTotalBytesToScan(websiteIDs))
	setParsingWebsiteID("")
	defer setParsingWebsiteID("")
	parserResults := make([]ParserResult, len(websiteIDs))

	for i, id := range websiteIDs {
		setParsingWebsiteID(id)
		startTime := time.Now()

		website, _ := config.GetWebsiteByID(id)
		parserResult := EmptyParserResult(website.Name, id)
		p.markInitialParsed(id)
		if len(website.Sources) > 0 {
			p.scanSources(id, website, &parserResult)
		} else {
			if _, err := p.getLineParser(id); err != nil {
				parserResult.Success = false
				parserResult.Error = err
				p.notifyLogParsing(id, "", "日志解析配置", err)
				parserResults[i] = parserResult
				continue
			}

			logPath := website.LogPath
			if strings.Contains(logPath, "*") {
				matches, err := filepath.Glob(logPath)
				if err != nil {
					errstr := "解析日志路径模式 " + logPath + " 失败: " + err.Error()
					parserResult.Success = false
					parserResult.Error = errors.New(errstr)
					p.notifyLogParsing(id, logPath, "解析日志路径模式", err)
				} else if len(matches) == 0 {
					errstr := "日志路径模式 " + logPath + " 未匹配到任何文件"
					parserResult.Success = false
					parserResult.Error = errors.New(errstr)
					p.notifyLogParsing(id, logPath, "日志路径未匹配到文件", errors.New(errstr))
				} else {
					for _, matchPath := range matches {
						p.scanSingleFile(id, matchPath, &parserResult)
					}
				}
			} else {
				p.scanSingleFile(id, logPath, &parserResult)
			}
		}

		p.refreshWebsiteRanges(id)
		p.updateState()
		parserResult.Duration = time.Since(startTime)
		parserResults[i] = parserResult
	}

	p.updateState()

	return parserResults
}

func (p *LogParser) calculateTotalBytesToScan(websiteIDs []string) int64 {
	var total int64

	for _, id := range websiteIDs {
		website, ok := config.GetWebsiteByID(id)
		if !ok {
			continue
		}

		logPath := website.LogPath
		if strings.Contains(logPath, "*") {
			matches, err := filepath.Glob(logPath)
			if err != nil {
				logrus.Warnf("解析日志路径模式 %s 失败: %v", logPath, err)
				continue
			}
			for _, matchPath := range matches {
				total += p.scanableBytes(id, matchPath)
			}
			continue
		}

		total += p.scanableBytes(id, logPath)
	}

	return total
}

func (p *LogParser) scanableBytes(websiteID, logPath string) int64 {
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		return 0
	}

	currentSize := fileInfo.Size()
	startOffset := p.determineStartOffset(websiteID, logPath, currentSize)
	if isGzipFile(logPath) {
		if startOffset < 0 {
			return 0
		}
		return currentSize
	}
	if currentSize <= startOffset {
		return 0
	}
	return currentSize - startOffset
}

func startIPParsingWithStage(stage parseStage) bool {
	parsingMu.Lock()
	defer parsingMu.Unlock()
	if parsingMode != parseModeNone {
		return false
	}
	parsingMode = parseModeForeground
	parsingWebsiteID = ""
	setParseStage(stage)
	resetParsingProgress()
	return true
}

func finishIPParsing() {
	parsingMu.Lock()
	if parsingMode == parseModeForeground {
		parsingMode = parseModeNone
	}
	parsingWebsiteID = ""
	parsingMu.Unlock()
	resetParseStage()
	finalizeParsingProgress()
}

func IsIPParsing() bool {
	parsingMu.RLock()
	defer parsingMu.RUnlock()
	return parsingMode == parseModeForeground
}

func GetParsingWebsiteID() string {
	parsingMu.RLock()
	defer parsingMu.RUnlock()
	if parsingMode != parseModeForeground {
		return ""
	}
	return parsingWebsiteID
}

func setParsingWebsiteID(websiteID string) {
	parsingMu.Lock()
	defer parsingMu.Unlock()
	if parsingMode != parseModeForeground {
		parsingWebsiteID = ""
		return
	}
	parsingWebsiteID = websiteID
}

func startBackfillParsing() bool {
	parsingMu.Lock()
	defer parsingMu.Unlock()
	if parsingMode != parseModeNone {
		return false
	}
	parsingMode = parseModeBackfill
	return true
}

func finishBackfillParsing() {
	parsingMu.Lock()
	if parsingMode == parseModeBackfill {
		parsingMode = parseModeNone
	}
	parsingMu.Unlock()
}

func IsBackfillParsing() bool {
	parsingMu.RLock()
	defer parsingMu.RUnlock()
	return parsingMode == parseModeBackfill
}

// scanSingleFile 扫描单个日志文件
