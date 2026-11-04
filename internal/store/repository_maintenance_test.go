package store

import (
	"context"
	"strings"
	"testing"
)

func TestOrphanDimsUsesAntiJoinAndQuotesNames(t *testing.T) {
	query := orphanDimsSQL(`site"_dim_ip`, "site_nginx_logs", "ip_id", []string{`NOT EXISTS (SELECT 1 FROM "site_sessions" s WHERE s.ip_id=d.id)`})
	if strings.Contains(query, "NOT IN") || !strings.Contains(query, `DELETE FROM "site""_dim_ip" AS d`) || strings.Count(query, "NOT EXISTS") != 2 {
		t.Fatalf("unsafe orphan cleanup query: %s", query)
	}
}

func TestMaintenanceSingleFlight(t *testing.T) {
	r := &Repository{}
	r.cleanupMu.Lock()
	defer r.cleanupMu.Unlock()
	if err := r.CleanOldLogsContext(context.Background()); err != ErrMaintenanceBusy {
		t.Fatalf("got %v", err)
	}
}
