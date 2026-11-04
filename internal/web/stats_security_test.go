package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/likaia/nginxpulse/internal/analytics"
	"github.com/likaia/nginxpulse/internal/config"
)

func TestStatsHTTPRejectsInjectionBeforeDatabaseAccess(t *testing.T) {
	t.Setenv("CONFIG_JSON", `{"websites":[],"system":{"taskInterval":"1m"}}`)
	id, _ := config.RegisterRuntimeWebsite(config.WebsiteConfig{Name: t.Name()})
	t.Cleanup(func() { config.UnregisterRuntimeWebsite(id) })
	// No database: any request reaching SQL execution would fail this test.
	router := gin.New()
	SetupRoutes(router, analytics.NewStatsFactory(nil), nil)
	for _, tc := range []struct {
		statsType, websiteID, locationType string
	}{
		{"overall", id + `_agg_daily" JOIN (SELECT 1) z ON 1=CAST('probe' AS int) --`, "global"},
		{"logs", id + `"; SELECT 1 --`, "global"},
		{"location", id, "global||'|'||current_user"},
		{"location", id, "global,(SELECT current_database())"},
	} {
		t.Run(tc.statsType+"/"+tc.locationType+"/"+tc.websiteID, func(t *testing.T) {
			params := url.Values{
				"id": {tc.websiteID}, "locationType": {tc.locationType}, "timeRange": {"7d"},
				"limit": {"5"}, "page": {"1"}, "pageSize": {"10"},
				"sortField": {"timestamp"}, "sortOrder": {"desc"},
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/stats/"+tc.statsType+"?"+params.Encode(), nil))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s, want 400", response.Code, response.Body.String())
			}
		})
	}
}
