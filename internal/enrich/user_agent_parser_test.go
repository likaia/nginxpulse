package enrich

import "testing"

func TestParseUserAgentDevice(t *testing.T) {
	tests := []struct {
		name, ua, device string
	}{
		{"native app", "VirPulse/337 CFNetwork/3896.200.61 Darwin/27.2.0", "苹果设备"},
		{"widget", "VirPulseWidgets/337 CFNetwork/3896.200.61 Darwin/27.2.0", "苹果设备"},
		{"case insensitive", "App/1 cfnetwork/1 darwin/1", "苹果设备"},
		{"cfnetwork only", "App/1 CFNetwork/3896.200.61", "其他设备"},
		{"darwin only", "App/1 Darwin/27.2.0", "其他设备"},
		{"iphone browser", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", "手机"},
		{"ipad browser", "Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", "平板"},
		{"desktop browser", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36", "桌面设备"},
		{"bot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html) CFNetwork/1 Darwin/1", "蜘蛛"},
		{"empty", "", "其他设备"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, device := ParseUserAgent(tt.ua)
			if device != tt.device {
				t.Fatalf("device = %q, want %q", device, tt.device)
			}
		})
	}
}

func TestParseAppleUserAgentPreservesUnknownOS(t *testing.T) {
	browser, os, _ := ParseUserAgent("VirPulse/337 CFNetwork/3896.200.61 Darwin/27.2.0")
	if browser != "VirPulse" || os != "未知操作系统" {
		t.Fatalf("browser = %q, os = %q", browser, os)
	}
}
