package ingest

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/likaia/nginxpulse/internal/config"
)

func isGzipFile(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".gz")
}

func skipReaderBytes(reader io.Reader, offset int64) error {
	if offset <= 0 {
		return nil
	}
	_, err := io.CopyN(io.Discard, reader, offset)
	return err
}

func (p *LogParser) getLineParser(websiteID string) (*logLineParser, error) {
	return p.getLineParserForSource(websiteID, "")
}

func (p *LogParser) getLineParserForSource(websiteID, sourceID string) (*logLineParser, error) {
	key := websiteID
	if sourceID != "" {
		key = websiteID + ":" + sourceID
	}
	if parser, ok := p.lineParsers[key]; ok {
		return parser, nil
	}

	website, ok := config.GetWebsiteByID(websiteID)
	if !ok {
		return nil, fmt.Errorf("未找到网站配置: %s", websiteID)
	}

	var sourceCfg *config.SourceConfig
	if sourceID != "" {
		for i := range website.Sources {
			if strings.TrimSpace(website.Sources[i].ID) == sourceID {
				sourceCfg = &website.Sources[i]
				break
			}
		}
	}

	parser, err := newLogLineParser(website, sourceCfg)
	if err != nil {
		return nil, err
	}

	p.lineParsers[key] = parser
	return parser, nil
}

func newLogLineParser(website config.WebsiteConfig, sourceCfg *config.SourceConfig) (*logLineParser, error) {
	logType := strings.ToLower(strings.TrimSpace(website.LogType))
	logFormat := website.LogFormat
	logRegex := website.LogRegex
	timeLayout := website.TimeLayout

	if sourceCfg != nil && sourceCfg.Parse != nil {
		parseOverride := sourceCfg.Parse
		if strings.TrimSpace(parseOverride.LogType) != "" {
			logType = strings.ToLower(strings.TrimSpace(parseOverride.LogType))
		}
		if strings.TrimSpace(parseOverride.LogFormat) != "" {
			logFormat = parseOverride.LogFormat
		}
		if strings.TrimSpace(parseOverride.LogRegex) != "" {
			logRegex = parseOverride.LogRegex
		}
		if strings.TrimSpace(parseOverride.TimeLayout) != "" {
			timeLayout = parseOverride.TimeLayout
		}
	}
	if logType == "" {
		logType = "nginx"
	}

	pattern := defaultNginxLogRegex
	source := "default"
	parseType := parseTypeRegex

	if strings.TrimSpace(logRegex) != "" {
		pattern = ensureAnchors(logRegex)
		source = "logRegex"
	} else if strings.TrimSpace(logFormat) != "" {
		compiled, err := buildRegexFromFormat(logFormat)
		if err != nil {
			return nil, err
		}
		pattern = compiled
		source = "logFormat"
	} else {
		switch logType {
		case "caddy":
			return &logLineParser{
				timeLayout: timeLayout,
				source:     "caddy",
				parseType:  parseTypeCaddyJSON,
			}, nil
		case "nginx":
			// default nginx pattern
		case "nginx-proxy-manager", "npm":
			pattern = defaultNPMLogRegex
			source = "nginx-proxy-manager"
		case "apache", "httpd", "apache-httpd":
			pattern = defaultApacheLogRegex
			source = "apache"
		case "tengine":
			pattern = defaultNginxLogRegex
			source = "tengine"
		case "traefik", "traefik-ingress":
			pattern = defaultTraefikLogRegex
			source = "traefik"
		case "envoy":
			pattern = defaultEnvoyLogRegex
			source = "envoy"
		case "nginx-ingress", "ingress-nginx":
			pattern = defaultNginxIngressLogRegex
			source = "nginx-ingress"
		case "haproxy", "haproxy-ingress":
			pattern = defaultHAProxyLogRegex
			source = "haproxy"
			if strings.TrimSpace(timeLayout) == "" {
				timeLayout = defaultHAProxyTimeLayout
			}
		case "iis", "iis-w3c", "w3c-iis":
			pattern = defaultIISW3CLogRegex
			source = "iis"
			if strings.TrimSpace(timeLayout) == "" {
				timeLayout = defaultIISTimeLayout
			}
		case "safeline", "safeline-waf", "raywaf", "ray-waf", "leichi", "leichi-waf":
			pattern = defaultSafeLineWAFLogRegex
			source = "safeline-waf"
		case "zoraxy", "zoraxy-router":
			pattern = defaultZoraxyLogRegex
			source = "zoraxy"
			if strings.TrimSpace(timeLayout) == "" {
				timeLayout = defaultZoraxyTimeLayout
			}
		default:
			return nil, fmt.Errorf("不支持的日志类型: %s", logType)
		}
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("日志格式正则无效 (%s): %w", source, err)
	}

	indexMap := make(map[string]int)
	for i, name := range regex.SubexpNames() {
		if name != "" {
			indexMap[name] = i
		}
	}

	if err := validateLogPattern(indexMap); err != nil {
		return nil, err
	}

	return &logLineParser{
		regex:      regex,
		indexMap:   indexMap,
		timeLayout: timeLayout,
		source:     source,
		parseType:  parseType,
	}, nil
}

func ensureAnchors(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return trimmed
	}
	if !strings.HasPrefix(trimmed, "^") {
		trimmed = "^" + trimmed
	}
	if !strings.HasSuffix(trimmed, "$") {
		trimmed = trimmed + "$"
	}
	return trimmed
}

func buildRegexFromFormat(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return "", errors.New("logFormat 不能为空")
	}

	varPattern := regexp.MustCompile(`\$\w+`)
	locations := varPattern.FindAllStringIndex(format, -1)
	if len(locations) == 0 {
		return "", errors.New("logFormat 未包含任何变量")
	}

	var builder strings.Builder
	usedNames := make(map[string]bool)
	last := 0
	for _, loc := range locations {
		literal := format[last:loc[0]]
		builder.WriteString(regexp.QuoteMeta(literal))

		varName := format[loc[0]+1 : loc[1]]
		quoted := isQuotedTokenBoundary(literal, format[loc[1]:])
		builder.WriteString(tokenRegexForVar(varName, usedNames, quoted))
		last = loc[1]
	}
	builder.WriteString(regexp.QuoteMeta(format[last:]))

	return "^" + builder.String() + "$", nil
}

func tokenRegexForVar(name string, used map[string]bool, quoted bool) string {
	addGroup := func(group, pattern string) string {
		if used[group] {
			return pattern
		}
		used[group] = true
		return "(?P<" + group + ">" + pattern + ")"
	}

	commaListPattern := `[^,\s]+(?:,\s*[^,\s]+)*`
	optionalTokenPattern := `\S*`
	optionalQuotedPattern := `[^"]*`
	requiredTokenPattern := `\S+`
	requiredQuotedPattern := `[^"]+`
	if quoted {
		commaListPattern = `(?:[^,\s]+(?:,\s*[^,\s]+)*)?`
		optionalTokenPattern = optionalQuotedPattern
		requiredTokenPattern = requiredQuotedPattern
	}

	switch name {
	case "remote_addr":
		return addGroup("ip", requiredTokenPattern)
	case "real_client_ip":
		return addGroup("real_client_ip", requiredTokenPattern)
	case "http_x_forwarded_for":
		return addGroup("http_x_forwarded_for", commaListPattern)
	case "remote_user":
		return addGroup("user", optionalTokenPattern)
	case "time_local":
		return addGroup("time", `[^]]+`)
	case "time_iso8601":
		return addGroup("time", requiredTokenPattern)
	case "request":
		return addGroup("request", requiredTokenPattern)
	case "request_method":
		return addGroup("method", requiredTokenPattern)
	case "request_uri", "uri":
		return addGroup("url", requiredTokenPattern)
	case "args":
		return addGroup("args", optionalTokenPattern)
	case "query_string":
		return addGroup("query_string", optionalTokenPattern)
	case "status":
		return addGroup("status", `\d{3}`)
	case "body_bytes_sent", "bytes_sent":
		return addGroup("bytes", `\d+`)
	case "http_referer":
		return addGroup("referer", optionalTokenPattern)
	case "http_user_agent":
		return addGroup("ua", optionalTokenPattern)
	case "host":
		return addGroup("host", requiredTokenPattern)
	case "http_host":
		return addGroup("host", requiredTokenPattern)
	case "server_name":
		return addGroup("server_name", requiredTokenPattern)
	case "scheme":
		return addGroup("scheme", requiredTokenPattern)
	case "request_length":
		return addGroup("request_length", `\d+`)
	case "remote_port":
		return addGroup("remote_port", `\d+`)
	case "connection":
		return addGroup("connection", `\d+`)
	case "request_time":
		return addGroup("request_time", `\d+(?:\.\d+)?`)
	case "request_time_msec":
		return addGroup("request_time_msec", `\d+(?:\.\d+)?`)
	case "upstream_addr":
		return addGroup("upstream_addr", commaListPattern)
	case "upstream_status":
		return addGroup("upstream_status", commaListPattern)
	case "upstream_response_time":
		return addGroup("upstream_response_time", commaListPattern)
	case "upstream_connect_time":
		return addGroup("upstream_connect_time", commaListPattern)
	case "upstream_header_time":
		return addGroup("upstream_header_time", commaListPattern)
	case "request_id":
		return addGroup("request_id", requiredTokenPattern)
	default:
		return optionalTokenPattern
	}
}

func isQuotedTokenBoundary(prefix, suffix string) bool {
	prefixTrim := strings.TrimRight(prefix, " \t\r\n")
	if !strings.HasSuffix(prefixTrim, "\"") {
		return false
	}
	suffixTrim := strings.TrimLeft(suffix, " \t\r\n")
	return strings.HasPrefix(suffixTrim, "\"")
}

func validateLogPattern(indexMap map[string]int) error {
	if len(indexMap) == 0 {
		return errors.New("logRegex/logFormat 必须包含命名分组")
	}

	if !hasAnyField(indexMap, ipAliases) {
		return errors.New("日志格式缺少 IP 字段（ip/remote_addr）")
	}
	if !hasAnyField(indexMap, timeAliases) {
		return errors.New("日志格式缺少时间字段（time/time_local/time_iso8601）")
	}
	if !hasAnyField(indexMap, statusAliases) {
		return errors.New("日志格式缺少状态码字段（status）")
	}
	if !hasAnyField(indexMap, urlAliases) && !hasAnyField(indexMap, requestAliases) {
		return errors.New("日志格式缺少 URL 字段（url/request_uri 或 request）")
	}
	return nil
}
