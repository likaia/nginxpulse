package web

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/likaia/nginxpulse/internal/alertpush"
	"github.com/likaia/nginxpulse/internal/analytics"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/ingest"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/likaia/nginxpulse/internal/version"
	"github.com/sirupsen/logrus"
)

// 初始化Web路由
func SetupRoutes(
	router *gin.Engine,
	statsFactory *analytics.StatsFactory,
	logParser *ingest.LogParser) {

	// 获取所有网站列表
	router.GET("/api/websites", func(c *gin.Context) {
		websiteIDs := config.GetAllWebsiteIDs()

		websites := make([]map[string]string, 0, len(websiteIDs))
		for _, id := range websiteIDs {
			website, ok := config.GetWebsiteByID(id)
			if !ok {
				continue
			}

			websites = append(websites, map[string]string{
				"id":   id,
				"name": website.Name,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"websites": websites,
		})
	})

	router.GET("/api/status", func(c *gin.Context) {
		cfg := config.ReadConfig()
		migrationRequired := needsPGMigration()
		ipGeoPendingCount := int64(0)
		if logParser != nil {
			ipGeoPendingCount = logParser.GetIPGeoPendingCount()
		}
		c.JSON(http.StatusOK, gin.H{
			"log_parsing":                             ingest.IsIPParsing(),
			"log_parsing_website_id":                  ingest.GetParsingWebsiteID(),
			"log_parsing_stage":                       ingest.GetLogParsingStage(),
			"log_parsing_progress":                    ingest.GetIPParsingProgress(),
			"log_parsing_estimated_total_seconds":     ingest.GetIPParsingEstimatedTotalSeconds(),
			"log_parsing_estimated_remaining_seconds": ingest.GetIPParsingEstimatedRemainingSeconds(),
			"ip_geo_parsing":                          ingest.IsIPGeoParsing(),
			"ip_geo_pending":                          ipGeoPendingCount > 0,
			"ip_geo_progress":                         ingest.GetIPGeoParsingProgress(ipGeoPendingCount),
			"ip_geo_estimated_remaining_seconds":      ingest.GetIPGeoEstimatedRemainingSeconds(ipGeoPendingCount),
			"demo_mode":                               cfg.System.DemoMode,
			"access_key_expire_days":                  cfg.System.AccessKeyExpireDays,
			"mobile_pwa_enabled":                      cfg.System.MobilePWAEnabled,
			"language":                                config.NormalizeLanguage(cfg.System.Language),
			"version":                                 version.Version,
			"git_commit":                              version.GitCommit,
			"migration_required":                      migrationRequired,
			"setup_required":                          config.IsSetupMode(),
			"config_readonly":                         config.ConfigReadOnly(),
		})
	})

	registerServerStatusRoute(router)

	router.GET("/api/system/notifications", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持系统通知",
			})
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		unreadOnly := strings.EqualFold(c.DefaultQuery("unreadOnly", "false"), "true") ||
			strings.EqualFold(c.DefaultQuery("unread_only", "false"), "true")

		repo := statsFactory.Repo()
		notifications, hasMore, err := repo.ListSystemNotifications(page, pageSize, unreadOnly)
		if err != nil {
			logrus.WithError(err).Error("读取系统通知失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("读取系统通知失败: %v", err),
			})
			return
		}
		unreadCount, err := repo.GetSystemNotificationUnreadCount()
		if err != nil {
			logrus.WithError(err).Warn("读取未读通知数失败")
		}

		c.JSON(http.StatusOK, gin.H{
			"notifications": notifications,
			"has_more":      hasMore,
			"unread_count":  unreadCount,
		})
	})

	router.POST("/api/system/notifications/read", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持系统通知",
			})
			return
		}
		type readRequest struct {
			IDs []int64 `json:"ids"`
			All bool    `json:"all"`
		}
		var req readRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		repo := statsFactory.Repo()
		if req.All {
			if err := repo.MarkAllSystemNotificationsRead(); err != nil {
				logrus.WithError(err).Error("标记通知已读失败")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("标记通知已读失败: %v", err),
				})
				return
			}
		} else {
			if err := repo.MarkSystemNotificationsRead(req.IDs); err != nil {
				logrus.WithError(err).Error("标记通知已读失败")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("标记通知已读失败: %v", err),
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
		})
	})

	router.POST("/api/system/notifications/clear", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持系统通知",
			})
			return
		}
		type clearRequest struct {
			IDs []int64 `json:"ids"`
			All bool    `json:"all"`
		}
		var req clearRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		if !req.All && len(req.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "缺少清理目标",
			})
			return
		}
		repo := statsFactory.Repo()
		var (
			deleted int64
			err     error
		)
		if req.All {
			deleted, err = repo.DeleteAllSystemNotifications()
		} else {
			deleted, err = repo.DeleteSystemNotifications(req.IDs)
		}
		if err != nil {
			logrus.WithError(err).Error("清理系统通知失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("清理系统通知失败: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"deleted": deleted,
		})
	})

	router.GET("/api/config", func(c *gin.Context) {
		cfg, err := config.ReadRawConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("读取配置失败: %v", err),
			})
			return
		}
		defaultLogPath := ""
		if config.IsSetupMode() {
			defaultLogPath = config.SuggestDefaultLogPath()
			if strings.TrimSpace(cfg.Database.DSN) == "" {
				cfg.Database.DSN = buildEmbeddedPostgresDSN()
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"config":           cfg,
			"readonly":         config.ConfigReadOnly(),
			"setup_required":   config.IsSetupMode(),
			"default_log_path": defaultLogPath,
		})
	})

	router.POST("/api/config/validate", func(c *gin.Context) {
		cfg, err := bindConfigPayload(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		result := config.ValidateConfig(cfg, config.ValidateOptions{
			CheckPaths: true,
		})
		c.JSON(http.StatusOK, result)
	})

	router.POST("/api/config/save", func(c *gin.Context) {
		if config.ConfigReadOnly() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "配置来自环境变量，无法保存",
			})
			return
		}

		cfg, err := bindConfigPayload(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}

		result := config.ValidateConfig(cfg, config.ValidateOptions{
			CheckPaths: true,
		})
		if len(result.Errors) > 0 {
			c.JSON(http.StatusBadRequest, result)
			return
		}

		if err := config.WriteConfigFile(cfg); err != nil {
			logrus.WithError(err).Error("保存配置失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("保存配置失败: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":          true,
			"restart_required": true,
		})
	})

	router.POST("/api/system/restart", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
		})

		go func() {
			time.Sleep(200 * time.Millisecond)
			if proc, err := os.FindProcess(os.Getpid()); err == nil {
				_ = proc.Signal(syscall.SIGTERM)
			}
		}()
	})

	router.POST("/api/alert-push/test", func(c *gin.Context) {
		req := struct {
			AlertPush *config.AlertPushConfig `json:"alertPush"`
			Message   string                  `json:"message"`
			Channels  []string                `json:"channels"`
		}{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}

		pushCfg := req.AlertPush
		if pushCfg == nil {
			cfg := config.ReadConfig()
			pushCfg = cfg.System.AlertPush
		}
		dispatcher := alertpush.NewDispatcher(pushCfg)
		if dispatcher == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "告警推送未启用或配置为空",
			})
			return
		}

		message := strings.TrimSpace(req.Message)
		if message == "" {
			message = "这是一条来自 NginxPulse 的测试告警消息。"
		}

		entry := store.SystemNotification{
			Level:    "info",
			Category: "alert_push_test",
			Title:    "告警推送测试",
			Message:  message,
			Metadata: map[string]interface{}{
				"test": true,
				"time": time.Now().Format("2006-01-02 15:04:05"),
			},
		}

		results := dispatcher.SendWithResult(entry, req.Channels)
		tested := 0
		succeeded := 0
		for _, result := range results {
			tested++
			if result.Success {
				succeeded++
			}
		}
		if tested == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "没有可用的推送通道",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":   tested == succeeded,
			"tested":    tested,
			"succeeded": succeeded,
			"results":   results,
		})
	})

	router.GET("/api/version", func(c *gin.Context) {
		refresh := c.Query("refresh")
		updateInfo := version.GetUpdateInfo()
		if refresh == "1" || refresh == "true" {
			updateInfo = version.RefreshUpdateInfo()
		}
		c.JSON(http.StatusOK, gin.H{
			"version":            version.Version,
			"git_commit":         version.GitCommit,
			"update_available":   updateInfo.UpdateAvailable,
			"latest_version":     updateInfo.LatestVersion,
			"latest_release_url": updateInfo.LatestReleaseURL,
		})
	})

	router.POST("/api/logs/reparse", func(c *gin.Context) {
		if logParser == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持日志解析",
			})
			return
		}
		type reparseRequest struct {
			ID        string `json:"id"`
			Migration bool   `json:"migration"`
			Mode      string `json:"mode"`
			StartAt   string `json:"start_at"`
			EndAt     string `json:"end_at"`
		}

		var req reparseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}

		websiteID := strings.TrimSpace(req.ID)
		if websiteID != "" {
			if _, ok := config.GetWebsiteByID(websiteID); !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "站点不存在",
				})
				return
			}
		}

		mode := strings.ToLower(strings.TrimSpace(req.Mode))
		if mode == "" {
			mode = "full"
		}

		var err error
		switch mode {
		case "full":
			err = logParser.TriggerReparse(websiteID)
		case "range":
			if websiteID == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "按时间段重解析仅支持单个站点",
				})
				return
			}
			startAt, parseErr := parseReparseTime(req.StartAt)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("开始时间无效: %v", parseErr),
				})
				return
			}
			endAt, parseErr := parseReparseTime(req.EndAt)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("结束时间无效: %v", parseErr),
				})
				return
			}
			if !endAt.After(startAt) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "结束时间必须晚于开始时间",
				})
				return
			}
			err = logParser.TriggerRangeReparse(websiteID, startAt, endAt)
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "不支持的重解析模式",
			})
			return
		}

		if err != nil {
			if errors.Is(err, ingest.ErrParsingInProgress) {
				c.JSON(http.StatusConflict, gin.H{
					"error": err.Error(),
				})
				return
			}
			logrus.WithError(err).Error("触发重新解析失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("重新解析失败: %v", err),
			})
			return
		}

		if req.Migration {
			if err := markPGMigrationDone(); err != nil {
				logrus.WithError(err).Warn("记录迁移状态失败")
			}
		}

		statsFactory.ClearCache()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
		})
	})

	router.GET("/api/ip-geo/failures", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持失败记录",
			})
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
		websiteID := strings.TrimSpace(c.DefaultQuery("id", ""))
		reason := strings.TrimSpace(c.DefaultQuery("reason", ""))
		keyword := strings.TrimSpace(c.DefaultQuery("keyword", ""))

		repo := statsFactory.Repo()
		failures, hasMore, err := repo.ListIPGeoAPIFailuresFiltered(websiteID, reason, keyword, page, pageSize)
		if err != nil {
			logrus.WithError(err).Error("读取 IP 归属地失败记录失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("读取失败记录失败: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"failures": failures,
			"has_more": hasMore,
		})
	})

	router.GET("/api/ip-geo/failures/export", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持失败记录导出",
			})
			return
		}

		websiteID := strings.TrimSpace(c.DefaultQuery("id", ""))
		reason := strings.TrimSpace(c.DefaultQuery("reason", ""))
		keyword := strings.TrimSpace(c.DefaultQuery("keyword", ""))
		websiteLabel := "all"
		if websiteID != "" {
			if site, ok := config.GetWebsiteByID(websiteID); ok && strings.TrimSpace(site.Name) != "" {
				websiteLabel = site.Name
			} else {
				websiteLabel = websiteID
			}
		}

		repo := statsFactory.Repo()
		const pageSize = 2000
		page := 1

		var buffer strings.Builder
		writer := csv.NewWriter(&buffer)
		_ = writer.Write([]string{"website", "ip", "reason", "source", "error", "status_code", "created_at"})

		for {
			failures, hasMore, err := repo.ListIPGeoAPIFailuresFiltered(websiteID, reason, keyword, page, pageSize)
			if err != nil {
				logrus.WithError(err).Error("导出失败记录失败")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("导出失败记录失败: %v", err),
				})
				return
			}
			for _, entry := range failures {
				_ = writer.Write([]string{
					websiteLabel,
					entry.IP,
					entry.Reason,
					entry.Source,
					entry.Error,
					strconv.Itoa(entry.StatusCode),
					entry.CreatedAt.Format(time.RFC3339),
				})
			}
			if !hasMore {
				break
			}
			page++
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			logrus.WithError(err).Error("生成 CSV 失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "生成 CSV 失败",
			})
			return
		}

		filename := fmt.Sprintf("ip_geo_failures_%s.csv", time.Now().Format("20060102_150405"))
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.String(http.StatusOK, buffer.String())
	})

	router.POST("/api/ip-geo/failures/clear", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持失败记录",
			})
			return
		}
		type clearRequest struct {
			ID        string `json:"id"`
			WebsiteID string `json:"websiteId"`
			Reason    string `json:"reason"`
			Keyword   string `json:"keyword"`
		}
		var req clearRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		websiteID := strings.TrimSpace(req.ID)
		if websiteID == "" {
			websiteID = strings.TrimSpace(req.WebsiteID)
		}
		reason := strings.TrimSpace(req.Reason)
		keyword := strings.TrimSpace(req.Keyword)
		repo := statsFactory.Repo()
		deleted, err := repo.ClearIPGeoAPIFailuresFiltered(websiteID, reason, keyword)
		if err != nil {
			logrus.WithError(err).Error("清理 IP 归属地失败记录失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("清理失败记录失败: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"deleted": deleted,
		})
	})

	router.GET("/api/ip-geo/override", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持 IP 归属地覆盖",
			})
			return
		}

		ip := strings.TrimSpace(c.DefaultQuery("ip", ""))
		if net.ParseIP(ip) == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "IP 参数无效",
			})
			return
		}

		repo := statsFactory.Repo()
		override, err := repo.GetIPGeoManualOverride(ip)
		if err != nil {
			logrus.WithError(err).Error("读取 IP 归属地人工覆盖失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("读取 IP 归属地人工覆盖失败: %v", err),
			})
			return
		}

		entry, overridden, note, err := repo.GetEffectiveIPGeo(ip, ingest.PendingLocationLabel())
		if err != nil {
			logrus.WithError(err).Error("读取 IP 归属地详情失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("读取 IP 归属地详情失败: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ip":         ip,
			"domestic":   entry.Domestic,
			"global":     entry.Global,
			"source":     entry.Source,
			"note":       note,
			"overridden": overridden,
			"override":   override,
		})
	})

	router.POST("/api/ip-geo/override", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持 IP 归属地覆盖",
			})
			return
		}

		type overrideRequest struct {
			IP       string `json:"ip"`
			Domestic string `json:"domestic"`
			Global   string `json:"global"`
			Note     string `json:"note"`
		}

		var req overrideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}

		ip := strings.TrimSpace(req.IP)
		domestic := strings.TrimSpace(req.Domestic)
		global := strings.TrimSpace(req.Global)
		note := strings.TrimSpace(req.Note)
		if net.ParseIP(ip) == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "IP 参数无效",
			})
			return
		}
		if domestic == "" || global == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "归属地不能为空",
			})
			return
		}

		repo := statsFactory.Repo()
		if err := repo.UpsertIPGeoManualOverride(store.IPGeoManualOverride{
			IP:       ip,
			Domestic: domestic,
			Global:   global,
			Note:     note,
		}); err != nil {
			logrus.WithError(err).Error("保存 IP 归属地人工覆盖失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("保存 IP 归属地人工覆盖失败: %v", err),
			})
			return
		}
		if err := repo.DeleteIPGeoPending([]string{ip}); err != nil {
			logrus.WithError(err).Warn("清理 IP 归属地待解析队列失败")
		}

		logsAffected, sessionsAffected, updatedWebsites, err := repo.ApplyIPGeoLocationForIP(ip, store.IPGeoCacheEntry{
			Domestic: domestic,
			Global:   global,
			Source:   "manual",
		})
		if err != nil {
			logrus.WithError(err).Error("回写 IP 归属地人工覆盖失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("回写 IP 归属地人工覆盖失败: %v", err),
			})
			return
		}

		statsFactory.ClearCache()
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"ip":                ip,
			"domestic":          domestic,
			"global":            global,
			"note":              note,
			"source":            "manual",
			"overridden":        true,
			"updated_websites":  updatedWebsites,
			"affected_logs":     logsAffected,
			"affected_sessions": sessionsAffected,
		})
	})

	router.DELETE("/api/ip-geo/override", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持 IP 归属地覆盖",
			})
			return
		}

		ip := strings.TrimSpace(c.DefaultQuery("ip", ""))
		if net.ParseIP(ip) == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "IP 参数无效",
			})
			return
		}

		repo := statsFactory.Repo()
		if err := repo.DeleteIPGeoManualOverride(ip); err != nil {
			logrus.WithError(err).Error("删除 IP 归属地人工覆盖失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("删除 IP 归属地人工覆盖失败: %v", err),
			})
			return
		}

		entry, _, _, err := repo.GetEffectiveIPGeo(ip, ingest.PendingLocationLabel())
		if err != nil {
			logrus.WithError(err).Error("读取 IP 归属地缓存失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("读取 IP 归属地缓存失败: %v", err),
			})
			return
		}

		if entry.Source == "pending" {
			if err := repo.MarkIPGeoPendingForIP(ip, ingest.PendingLocationLabel()); err != nil {
				logrus.WithError(err).Error("恢复 IP 归属地自动解析失败")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("恢复 IP 归属地自动解析失败: %v", err),
				})
				return
			}
			if logParser != nil {
				go logParser.ProcessPendingIPGeo(1)
			}
		}

		logsAffected, sessionsAffected, updatedWebsites, err := repo.ApplyIPGeoLocationForIP(ip, entry)
		if err != nil {
			logrus.WithError(err).Error("回写 IP 归属地自动结果失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("回写 IP 归属地自动结果失败: %v", err),
			})
			return
		}

		statsFactory.ClearCache()
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"ip":                ip,
			"domestic":          entry.Domestic,
			"global":            entry.Global,
			"source":            entry.Source,
			"overridden":        false,
			"updated_websites":  updatedWebsites,
			"affected_logs":     logsAffected,
			"affected_sessions": sessionsAffected,
		})
	})

	router.GET("/api/logs/export", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持日志导出",
			})
			return
		}

		params := map[string]string{
			"page":      "1",
			"pageSize":  fmt.Sprintf("%d", exportBatchSize),
			"sortField": "timestamp",
			"sortOrder": "desc",
		}
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
		normalizeLogsExportWebsiteID(params)

		query, err := statsFactory.BuildQueryFromRequest("logs", params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		if _, ok := config.GetWebsiteByID(query.WebsiteID); !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "站点不存在",
			})
			return
		}

		filename := fmt.Sprintf("nginxpulse_logs_%s.xlsx", time.Now().Format("20060102_150405"))
		c.Header("Content-Type", logsExportContentType)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)

		if err := exportLogsXLSX(c.Writer, statsFactory, query, c.Query("lang")); err != nil {
			logrus.WithError(err).Error("导出日志失败")
		}
	})

	router.POST("/api/logs/export", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持日志导出",
			})
			return
		}

		rawParams := make(map[string]any)
		if err := c.ShouldBindJSON(&rawParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		if len(rawParams) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "导出参数不能为空",
			})
			return
		}

		params := make(map[string]string, len(rawParams))
		for key, value := range rawParams {
			if value == nil {
				continue
			}
			params[key] = fmt.Sprint(value)
		}
		normalizeLogsExportWebsiteID(params)

		query, err := statsFactory.BuildQueryFromRequest("logs", params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		if _, ok := config.GetWebsiteByID(query.WebsiteID); !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "站点不存在",
			})
			return
		}

		lang := params["lang"]
		job, err := exportJobs.Create(statsFactory, query, lang, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"job_id":   job.ID,
			"status":   job.Status,
			"fileName": job.FileName,
		})
	})

	router.GET("/api/logs/export/status", func(c *gin.Context) {
		jobID := strings.TrimSpace(c.Query("id"))
		if jobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "任务 ID 不能为空",
			})
			return
		}
		job, ok := exportJobs.Get(jobID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "任务不存在",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id":         job.ID,
			"status":     job.Status,
			"processed":  job.Processed,
			"total":      job.Total,
			"fileName":   job.FileName,
			"error":      job.Error,
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
			"website_id": job.WebsiteID,
		})
	})

	router.GET("/api/logs/export/list", func(c *gin.Context) {
		websiteID := strings.TrimSpace(c.Query("website_id"))
		if websiteID == "" {
			websiteID = strings.TrimSpace(c.Query("websiteId"))
		}
		if websiteID == "" {
			websiteID = strings.TrimSpace(c.Query("id"))
		}
		page := 1
		pageSize := 20
		if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
			if parsed, err := strconv.Atoi(rawPage); err == nil && parsed > 0 {
				page = parsed
			}
		}
		if rawPageSize := strings.TrimSpace(c.Query("pageSize")); rawPageSize != "" {
			if parsed, err := strconv.Atoi(rawPageSize); err == nil && parsed > 0 {
				pageSize = parsed
			}
		}
		jobs, total := exportJobs.List(websiteID, page, pageSize)
		hasMore := page*pageSize < total
		c.JSON(http.StatusOK, gin.H{
			"jobs":     jobs,
			"total":    total,
			"has_more": hasMore,
		})
	})

	router.POST("/api/logs/export/cancel", func(c *gin.Context) {
		type cancelRequest struct {
			ID string `json:"id"`
		}
		var req cancelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		jobID := strings.TrimSpace(req.ID)
		if jobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "任务 ID 不能为空",
			})
			return
		}
		job, err := exportJobs.Cancel(jobID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id":     job.ID,
			"status": job.Status,
		})
	})

	router.POST("/api/logs/export/retry", func(c *gin.Context) {
		type retryRequest struct {
			ID string `json:"id"`
		}
		var req retryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}
		jobID := strings.TrimSpace(req.ID)
		if jobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "任务 ID 不能为空",
			})
			return
		}
		params, ok := exportJobs.GetParams(jobID)
		if !ok || len(params) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "任务参数不存在",
			})
			return
		}
		normalizeLogsExportWebsiteID(params)
		query, err := statsFactory.BuildQueryFromRequest("logs", params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if _, ok := config.GetWebsiteByID(query.WebsiteID); !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "站点不存在",
			})
			return
		}
		lang := params["lang"]
		job, err := exportJobs.Create(statsFactory, query, lang, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"job_id":   job.ID,
			"status":   job.Status,
			"fileName": job.FileName,
		})
	})

	router.GET("/api/logs/export/download", func(c *gin.Context) {
		jobID := strings.TrimSpace(c.Query("id"))
		if jobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "任务 ID 不能为空",
			})
			return
		}
		job, ok := exportJobs.Get(jobID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "任务不存在",
			})
			return
		}
		websiteID := strings.TrimSpace(c.Query("website_id"))
		if websiteID == "" {
			websiteID = strings.TrimSpace(c.Query("websiteId"))
		}
		if websiteID != "" && job.WebsiteID != "" && job.WebsiteID != websiteID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "导出任务与当前站点不匹配",
			})
			return
		}
		if job.Status != logsExportSuccess {
			c.JSON(http.StatusConflict, gin.H{
				"error": "导出任务尚未完成",
			})
			return
		}
		if job.FilePath == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "导出文件不存在",
			})
			return
		}
		if _, err := os.Stat(job.FilePath); err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "导出文件不存在",
			})
			return
		}

		filename := job.FileName
		if filename == "" {
			filename = fmt.Sprintf("nginxpulse_logs_%s.xlsx", time.Now().Format("20060102_150405"))
		}
		c.Header("Content-Type", logsExportContentType)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Cache-Control", "no-store")
		c.File(job.FilePath)
	})

	router.POST("/api/ingest/logs", func(c *gin.Context) {
		if logParser == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持日志解析",
			})
			return
		}
		type ingestRequest struct {
			WebsiteID string   `json:"website_id"`
			SourceID  string   `json:"source_id"`
			Lines     []string `json:"lines"`
		}

		var req ingestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "请求参数错误",
			})
			return
		}

		websiteID := strings.TrimSpace(req.WebsiteID)
		if websiteID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "缺少站点ID",
			})
			return
		}
		if _, ok := config.GetWebsiteByID(websiteID); !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "站点不存在",
			})
			return
		}
		if len(req.Lines) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "日志内容为空",
			})
			return
		}

		accepted, deduped, err := logParser.IngestLines(websiteID, strings.TrimSpace(req.SourceID), req.Lines)
		if err != nil {
			logrus.WithError(err).Error("日志推送解析失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("解析失败: %v", err),
			})
			return
		}

		statsFactory.ClearCache()
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"accepted": accepted,
			"deduped":  deduped,
		})
	})

	// 查询接口
	router.GET("/api/stats/:type", func(c *gin.Context) {
		if statsFactory == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "初始化模式暂不支持统计查询",
			})
			return
		}
		statsType := c.Param("type")
		params := make(map[string]string)
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}

		query, err := statsFactory.BuildQueryFromRequest(statsType, params)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"stats_type": statsType,
				"params":     params,
			}).WithError(err).Warn("统计查询参数无效")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// 执行查询
		result, err := statsFactory.QueryStats(statsType, query)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"stats_type": statsType,
				"params":     params,
				"website_id": query.WebsiteID,
			}).WithError(err).Error("查询统计数据失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "统计查询失败，请查看服务端日志",
			})
			return
		}

		c.JSON(http.StatusOK, result)
	})

}

func parseReparseTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("不能为空")
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("不支持的时间格式")
}

func bindConfigPayload(c *gin.Context) (*config.Config, error) {
	payload := struct {
		Config config.Config `json:"config"`
	}{
		Config: config.DefaultConfig(),
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		return nil, err
	}
	return &payload.Config, nil
}

func migrationMarkerPath() string {
	return filepath.Join(config.DataDir, "pg_migration_done")
}

func sqliteDataPath() string {
	return filepath.Join(config.DataDir, "nginxpulse.db")
}

func needsPGMigration() bool {
	if _, err := os.Stat(migrationMarkerPath()); err == nil {
		return false
	}
	if _, err := os.Stat(sqliteDataPath()); err == nil {
		return true
	}
	return false
}

func markPGMigrationDone() error {
	if err := os.WriteFile(migrationMarkerPath(), []byte("ok\n"), 0644); err != nil {
		return err
	}
	if err := os.Remove(sqliteDataPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func buildEmbeddedPostgresDSN() string {
	user := envWithDefault("POSTGRES_USER", "nginxpulse")
	password := envWithDefault("POSTGRES_PASSWORD", "nginxpulse")
	host := envWithDefault("POSTGRES_CONNECT_HOST", "127.0.0.1")
	port := envWithDefault("POSTGRES_PORT", "5432")
	database := envWithDefault("POSTGRES_DB", "nginxpulse")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, database,
	)
}

func envWithDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
