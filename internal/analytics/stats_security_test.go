package analytics

import (
	"strings"
	"testing"

	"github.com/likaia/nginxpulse/internal/config"
)

func securityTestFactory(t *testing.T) (*StatsFactory, string) {
	t.Helper()
	t.Setenv("CONFIG_JSON", `{"websites":[],"system":{"taskInterval":"1m"}}`)
	id, created := config.RegisterRuntimeWebsite(config.WebsiteConfig{Name: t.Name()})
	if !created {
		t.Fatal("test website already exists")
	}
	t.Cleanup(func() { config.UnregisterRuntimeWebsite(id) })
	return NewStatsFactory(nil), id
}

func securityTestParams(id string) map[string]string {
	return map[string]string{
		"id": id, "timeRange": "7d", "viewType": "pv", "limit": "5",
		"page": "1", "pageSize": "10", "sortField": "timestamp",
		"sortOrder": "desc", "locationType": "global",
	}
}

func TestStatsRequestRejectsSQLInjection(t *testing.T) {
	factory, id := securityTestFactory(t)
	for statsType := range factory.managers {
		for _, injectedID := range []string{
			id + `_agg_daily" JOIN (SELECT 1) z ON 1=CAST('probe' AS int) --`,
			id + `"; SELECT 1; --`,
			id + "' OR '1'='1", "unregistered_site", "", " " + id,
		} {
			t.Run(statsType+"/"+injectedID, func(t *testing.T) {
				if _, err := factory.BuildQueryFromRequest(statsType, securityTestParams(injectedID)); err == nil {
					t.Fatal("untrusted website ID accepted")
				}
			})
		}
	}
	for _, locationType := range []string{
		"global||'|'||current_user", "global,(SELECT current_database())", "global; SELECT 1 --", "GLOBAL", "",
	} {
		t.Run("location/"+locationType, func(t *testing.T) {
			params := securityTestParams(id)
			params["locationType"] = locationType
			if _, err := factory.BuildQueryFromRequest("location", params); err == nil {
				t.Fatal("untrusted location expression accepted")
			}
		})
	}
}

func TestStatsFactoryRejectsUnregisteredQueries(t *testing.T) {
	factory, id := securityTestFactory(t)
	for statsType := range factory.managers {
		for _, untrustedID := range []string{"", "unregistered_site", id + `" JOIN (SELECT 1) z ON true --`} {
			if _, err := factory.QueryStats(statsType, StatsQuery{WebsiteID: untrustedID}); err == nil {
				t.Fatalf("%s: untrusted website ID reached query execution", statsType)
			}
		}
	}
}

func TestStatsRequestAcceptsRegisteredSitesAndLocationTypes(t *testing.T) {
	factory, id := securityTestFactory(t)
	for statsType := range factory.managers {
		if _, err := factory.BuildQueryFromRequest(statsType, securityTestParams(id)); err != nil {
			t.Errorf("%s rejected registered website: %v", statsType, err)
		}
	}
	for _, locationType := range []string{"domestic", "city", "global"} {
		params := securityTestParams(id)
		params["locationType"] = locationType
		if _, err := factory.BuildQueryFromRequest("location", params); err != nil {
			t.Errorf("%s rejected: %v", locationType, err)
		}
	}

	// These two names hash to the same base ID; discovery adds a '_1' suffix.
	first, created := config.RegisterRuntimeWebsite(config.WebsiteConfig{Name: "issue103-site-134"})
	if !created {
		t.Fatal("collision fixture already exists")
	}
	t.Cleanup(func() { config.UnregisterRuntimeWebsite(first) })
	second, created := config.RegisterRuntimeWebsite(config.WebsiteConfig{Name: "issue103-site-406"})
	if !created || !strings.HasPrefix(second, first+"_") {
		t.Fatalf("expected a runtime collision suffix, got %q and %q", first, second)
	}
	t.Cleanup(func() { config.UnregisterRuntimeWebsite(second) })
	if _, err := factory.BuildQueryFromRequest("overall", securityTestParams(second)); err != nil {
		t.Fatalf("runtime website rejected: %v", err)
	}
	config.UnregisterRuntimeWebsite(second)
	if _, err := factory.QueryStats("overall", StatsQuery{WebsiteID: second}); err == nil {
		t.Fatal("removed website accepted")
	}
}

func TestLocationManagerRejectsExpressionsBeforeDatabaseAccess(t *testing.T) {
	manager := NewLocationStatsManager(nil)
	for _, locationType := range []interface{}{"global||'|'||current_user", "unknown", "", nil, 1} {
		if _, err := manager.Query(StatsQuery{ExtraParam: map[string]interface{}{"locationType": locationType}}); err == nil {
			t.Errorf("locationType %v accepted", locationType)
		}
	}
}
