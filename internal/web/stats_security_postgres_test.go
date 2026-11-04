package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/likaia/nginxpulse/internal/analytics"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/store"
)

func TestStatsSecurityPostgres(t *testing.T) {
	dsn := os.Getenv("NGINXPULSE_SECURITY_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("run scripts/test_security.sh for isolated PostgreSQL regression tests")
	}
	for _, tc := range []struct{ name, dsn string }{
		{"ordinary account", dsn},
		{"existing superuser account", os.Getenv("NGINXPULSE_SECURITY_TEST_PG_ADMIN_DSN")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.dsn == "" {
				t.Skip("set NGINXPULSE_SECURITY_TEST_PG_ADMIN_DSN to test external superuser compatibility")
			}
			testStatsSecurityPostgres(t, tc.dsn)
		})
	}
}

func testStatsSecurityPostgres(t *testing.T, dsn string) {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") || u.Path != "/nginxpulse_security_test" {
		t.Fatal("security tests require localhost/nginxpulse_security_test")
	}
	t.Setenv("CONFIG_JSON", `{"websites":[],"system":{"taskInterval":"1m"}}`)
	cfg := config.ReadConfig()
	savedDB := cfg.Database
	cfg.Database = config.DatabaseConfig{Driver: "postgres", DSN: dsn}
	t.Cleanup(func() { cfg.Database = savedDB })
	id, _ := config.RegisterRuntimeWebsite(config.WebsiteConfig{Name: t.Name()})
	t.Cleanup(func() { config.UnregisterRuntimeWebsite(id) })
	repo, err := store.NewRepository()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	if err := repo.Init(); err != nil {
		t.Fatal(err)
	}
	if err := repo.BatchInsertLogsForWebsite(id, []store.NginxLogRecord{{
		IP: "192.0.2.103", Timestamp: time.Now().Add(-time.Minute), PageviewFlag: 1,
		Method: "GET", Url: "/security-test", Status: 200, BytesSent: 103,
		Referer: "-", UserBrowser: "Chrome", UserOs: "Linux", UserDevice: "Desktop",
		DomesticLocation: "浙江·杭州", GlobalLocation: "中国",
	}}); err != nil {
		t.Fatal(err)
	}
	factory := analytics.NewStatsFactory(repo)
	router := gin.New()
	SetupRoutes(router, factory, nil)
	request := func(statsType, websiteID, locationType string) *httptest.ResponseRecorder {
		params := url.Values{"id": {websiteID}, "timeRange": {"7d"}, "limit": {"5"}, "locationType": {locationType}}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/stats/"+statsType+"?"+params.Encode(), nil))
		return response
	}
	for _, locationType := range []string{"domestic", "city", "global"} {
		response := request("location", id, locationType)
		var body analytics.ClientStats
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body.Key) != 1 || body.PV[0] != 1 {
			t.Fatalf("normal %s stats failed: %d %s", locationType, response.Code, response.Body.String())
		}
	}
	response := request("overall", id, "")
	var overall analytics.OverallStats
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &overall) != nil || overall.PV != 1 || overall.Traffic != 103 {
		t.Fatalf("normal overall stats failed: %d %s", response.Code, response.Body.String())
	}
	for _, tc := range []struct{ statsType, websiteID, locationType string }{
		{"overall", id + `_agg_daily" JOIN (SELECT 1) z ON 1=CAST('SQLI_PROBE' AS int) --`, ""},
		{"location", id, "global||'|'||current_user"},
	} {
		response := request(tc.statsType, tc.websiteID, tc.locationType)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("injection not rejected: %d %s", response.Code, response.Body.String())
		}
	}
	// Force a real database error and ensure details remain in server logs only.
	if _, err := repo.GetDB().Exec(`DROP TABLE "` + id + `_agg_daily"`); err != nil {
		t.Fatal(err)
	}
	factory.ClearCache()
	response = request("overall", id, "")
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), id) || strings.Contains(response.Body.String(), "SQLSTATE") {
		t.Fatalf("database error leaked or wrong status: %d %s", response.Code, response.Body.String())
	}
}
