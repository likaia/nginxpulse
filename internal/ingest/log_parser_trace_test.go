package ingest

import (
	"regexp"
	"testing"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
)

func TestNginxIngressParserParsesTraceFields(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "nginx-ingress"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(nginx-ingress) error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	line := `203.0.113.10 - - [` + now.Format(defaultNginxTimeLayout) + `] "GET /orders?id=42 HTTP/2.0" 200 512 "-" "curl/8.0.1" 128 0.245 [backend] [alt] 10.0.0.2:8080 512 0.200 200 req-123`

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}

	if record.RequestLength != 128 {
		t.Fatalf("unexpected request_length: %d", record.RequestLength)
	}
	if record.RequestTimeMs != 245 {
		t.Fatalf("unexpected request_time_ms: %d", record.RequestTimeMs)
	}
	if record.UpstreamTimeMs != 200 {
		t.Fatalf("unexpected upstream_response_time_ms: %d", record.UpstreamTimeMs)
	}
	if record.UpstreamAddr != "10.0.0.2:8080" {
		t.Fatalf("unexpected upstream_addr: %q", record.UpstreamAddr)
	}
	if record.RequestID != "req-123" {
		t.Fatalf("unexpected request_id: %q", record.RequestID)
	}
}

func TestBuildRegexFromFormatSupportsRequestTimeAndRequestID(t *testing.T) {
	pattern, err := buildRegexFromFormat(`$remote_addr [$time_local] "$request" $status $body_bytes_sent $request_time $request_length $request_id`)
	if err != nil {
		t.Fatalf("buildRegexFromFormat error: %v", err)
	}
	parser := &logLineParser{
		regex:      mustCompileRegex(t, pattern),
		indexMap:   map[string]int{},
		timeLayout: defaultNginxTimeLayout,
		parseType:  parseTypeRegex,
	}
	for i, name := range parser.regex.SubexpNames() {
		if name != "" {
			parser.indexMap[name] = i
		}
	}

	now := time.Now().UTC().Truncate(time.Second)
	line := `203.0.113.10 [` + now.Format(defaultNginxTimeLayout) + `] "GET /health HTTP/1.1" 200 12 0.123 64 req-789`
	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}
	if record.RequestTimeMs != 123 {
		t.Fatalf("unexpected request_time_ms: %d", record.RequestTimeMs)
	}
	if record.RequestLength != 64 {
		t.Fatalf("unexpected request_length: %d", record.RequestLength)
	}
	if record.RequestID != "req-789" {
		t.Fatalf("unexpected request_id: %q", record.RequestID)
	}
}

func TestDefaultNginxParserParsesExtendedTraceSuffix(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "nginx"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(nginx) error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	line := `203.0.113.10 - - [` + now.Format(defaultNginxTimeLayout) + `] "GET / HTTP/2.0" 200 59302 "-" "Blackbox Exporter/0.24.0" 0.134 33 0.133 172.16.7.133:7004 www.qcb.cn req-456`

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}
	if record.RequestTimeMs != 134 {
		t.Fatalf("unexpected request_time_ms: %d", record.RequestTimeMs)
	}
	if record.RequestLength != 33 {
		t.Fatalf("unexpected request_length: %d", record.RequestLength)
	}
	if record.UpstreamTimeMs != 133 {
		t.Fatalf("unexpected upstream_response_time_ms: %d", record.UpstreamTimeMs)
	}
	if record.UpstreamAddr != "172.16.7.133:7004" {
		t.Fatalf("unexpected upstream_addr: %q", record.UpstreamAddr)
	}
	if record.Host != "www.qcb.cn" {
		t.Fatalf("unexpected host: %q", record.Host)
	}
	if record.RequestID != "req-456" {
		t.Fatalf("unexpected request_id: %q", record.RequestID)
	}
}

func TestNginxJSONLogFormatParsesEmptyUpstreamResponseTime(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{
		LogType:   "nginx",
		LogFormat: nginxJSONTraceLogFormat(),
	}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(nginx json logFormat) error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	line := `{"@timestamp":"` + now.Format(time.RFC3339) + `","host":"example.com","server_name":"example.com","scheme":"https","remote_addr":"203.0.113.10","request_method":"GET","request":"GET /index.html HTTP/1.1","status":200,"request_length":128,"body_bytes_sent":512,"request_time":0.012,"upstream_response_time":"","http_referer":"","http_user_agent":"curl/8.0.1","http_x_forwarded_for":""}`

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}
	if record.IP != "203.0.113.10" {
		t.Fatalf("unexpected ip: %q", record.IP)
	}
	if record.RequestTimeMs != 12 {
		t.Fatalf("unexpected request_time_ms: %d", record.RequestTimeMs)
	}
	if record.UpstreamTimeMs != 0 {
		t.Fatalf("unexpected upstream_response_time_ms: %d", record.UpstreamTimeMs)
	}
	if record.Host != "example.com" {
		t.Fatalf("unexpected host: %q", record.Host)
	}
}

func TestNginxJSONLogFormatParsesQuotedUpstreamResponseTime(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{
		LogType:   "nginx",
		LogFormat: nginxJSONTraceLogFormat(),
	}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(nginx json logFormat) error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	line := `{"@timestamp":"` + now.Format(time.RFC3339) + `","host":"example.com","server_name":"example.com","scheme":"https","remote_addr":"203.0.113.10","request_method":"GET","request":"GET /api HTTP/1.1","status":200,"request_length":256,"body_bytes_sent":1024,"request_time":0.245,"upstream_response_time":"0.133","http_referer":"https://example.com/","http_user_agent":"curl/8.0.1","http_x_forwarded_for":"198.51.100.1"}`

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}
	if record.RequestTimeMs != 245 {
		t.Fatalf("unexpected request_time_ms: %d", record.RequestTimeMs)
	}
	if record.UpstreamTimeMs != 133 {
		t.Fatalf("unexpected upstream_response_time_ms: %d", record.UpstreamTimeMs)
	}
}

func nginxJSONTraceLogFormat() string {
	return `{"@timestamp":"$time_iso8601","host":"$host","server_name":"$server_name","scheme":"$scheme","remote_addr":"$real_client_ip","request_method":"$request_method","request":"$request","status":$status,"request_length":$request_length,"body_bytes_sent":$body_bytes_sent,"request_time":$request_time,"upstream_response_time":"$upstream_response_time","http_referer":"$http_referer","http_user_agent":"$http_user_agent","http_x_forwarded_for":"$http_x_forwarded_for"}`
}

func mustCompileRegex(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	regex, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("regexp.Compile error: %v", err)
	}
	return regex
}
