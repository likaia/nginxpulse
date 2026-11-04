package ingest

import (
	"fmt"
	"testing"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
)

func TestZoraxyDefaultParserParsesRouterLine(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "zoraxy"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(zoraxy) error: %v", err)
	}
	if parser.timeLayout != defaultZoraxyTimeLayout {
		t.Fatalf("unexpected zoraxy time layout: got %q want %q", parser.timeLayout, defaultZoraxyTimeLayout)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	line := "[" + now.Format(defaultZoraxyTimeLayout) + "] [router:host-http] [origin:app.example.com] [client: 203.0.113.10] [useragent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36] GET /dashboard?tab=logs 200"

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}

	if record.IP != "203.0.113.10" {
		t.Fatalf("unexpected ip: %q", record.IP)
	}
	if record.Method != "GET" {
		t.Fatalf("unexpected method: %q", record.Method)
	}
	if record.Url != "/dashboard?tab=logs" {
		t.Fatalf("unexpected url: %q", record.Url)
	}
	if record.Host != "app.example.com" {
		t.Fatalf("unexpected host: %q", record.Host)
	}
	if record.Status != 200 {
		t.Fatalf("unexpected status: %d", record.Status)
	}
	if record.Timestamp.UnixMicro() != now.UnixMicro() {
		t.Fatalf("unexpected timestamp: got %d want %d", record.Timestamp.UnixMicro(), now.UnixMicro())
	}
}

func TestZoraxyDefaultParserNormalizesClientWithPort(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "zoraxy-router"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(zoraxy-router) error: %v", err)
	}
	if parser.source != "zoraxy" {
		t.Fatalf("unexpected parser source: %q", parser.source)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	line := "[" + now.Format(defaultZoraxyTimeLayout) + "] [router:host-websocket] [origin:ws.example.com] [client: 198.51.100.20:53124] [useragent: curl/8.7.1] GET /socket 101"

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}
	if record.IP != "198.51.100.20" {
		t.Fatalf("unexpected normalized ip: %q", record.IP)
	}
	if record.Status != 101 {
		t.Fatalf("unexpected status: %d", record.Status)
	}
}

func TestZoraxyDefaultParserAcceptsMillisecondTimestamp(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "zoraxy"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(zoraxy) error: %v", err)
	}

	// 本用例验证毫秒精度，不应随着固定日期超过保留期而失败。
	stamp := time.Now().Truncate(time.Second).Format("2006-01-02 15:04:05") + ".123"
	line := fmt.Sprintf(`[%s] [router:redirect] [origin:app.example.com] [client: 203.0.113.10] [useragent: curl/8.7.1] GET /old 307`, stamp)
	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}
	if record.Timestamp.Nanosecond() != 123*int(time.Millisecond) {
		t.Fatalf("unexpected timestamp nanosecond: %d", record.Timestamp.Nanosecond())
	}
}

func TestZoraxyDefaultParserSkipsSystemLogLine(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "zoraxy"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(zoraxy) error: %v", err)
	}

	line := `[2026-04-24 10:00:00.000000] [database] [system:info] Using bolt as the database backend`
	p := &LogParser{retentionDays: 30}
	if _, err := p.parseRegexLogLine(parser, line); err == nil {
		t.Fatal("expected system log line to be rejected")
	}
}
