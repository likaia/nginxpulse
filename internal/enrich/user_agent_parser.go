package enrich

import (
	"strings"

	"github.com/mileusna/useragent"
)

// ParseUserAgent 解析 User-Agent 字符串
func ParseUserAgent(uaString string) (browser, os, device string) {
	userAgent := useragent.Parse(uaString)

	if userAgent.Bot {
		return "蜘蛛", "蜘蛛", "蜘蛛"
	}

	browser = userAgent.Name
	if browser == "" {
		browser = "未知浏览器"
	}

	os = userAgent.OS
	if os == "" {
		os = "未知操作系统"
	}

	uaLower := strings.ToLower(uaString)
	// Apple native clients do not always identify their device or operating system.
	if strings.Contains(uaLower, "cfnetwork/") && strings.Contains(uaLower, "darwin/") {
		device = "苹果设备"
	} else if userAgent.Mobile {
		device = "手机"
	} else if userAgent.Tablet {
		device = "平板"
	} else if userAgent.Desktop {
		device = "桌面设备"
	} else {
		device = "其他设备"
	}

	return browser, os, device
}
