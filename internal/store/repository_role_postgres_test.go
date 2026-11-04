package store

import (
	"database/sql"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestPostgresWarnsWithoutBlockingPrivilegedApplicationRoles(t *testing.T) {
	appDSN := os.Getenv("NGINXPULSE_SECURITY_TEST_PG_DSN")
	adminDSN := os.Getenv("NGINXPULSE_SECURITY_TEST_PG_ADMIN_DSN")
	if appDSN == "" || adminDSN == "" {
		t.Skip("run scripts/test_security.sh for isolated role validation tests")
	}
	for _, dsn := range []string{appDSN, adminDSN} {
		u, err := url.Parse(dsn)
		if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") || u.Path != "/nginxpulse_security_test" {
			t.Fatal("role tests require localhost/nginxpulse_security_test")
		}
	}
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	appURL, _ := url.Parse(appDSN)
	mustExec(t, admin, "CREATE ROLE security_role_probe LOGIN PASSWORD 'role-probe-test-only'")
	t.Cleanup(func() { mustExec(t, admin, "DROP ROLE security_role_probe") })
	appURL.User = url.UserPassword("security_role_probe", "role-probe-test-only")
	appDSN = appURL.String()
	appUser := pgx.Identifier{appURL.User.Username()}.Sanitize()
	savedHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	hook := logtest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(savedHooks) })
	assertConnection := func(t *testing.T, dsn string, wantWarning bool) {
		t.Helper()
		loginURL, _ := url.Parse(dsn)
		snapshot := func() string {
			t.Helper()
			var state string
			if err := admin.QueryRow(`SELECT row_to_json(r)::text || COALESCE((
                SELECT json_agg(m ORDER BY m.roleid, m.grantor)::text
                FROM pg_auth_members m WHERE m.member = r.oid
            ), '[]') FROM pg_roles r WHERE r.rolname = $1`, loginURL.User.Username()).Scan(&state); err != nil {
				t.Fatal(err)
			}
			return state
		}
		before := snapshot()
		hook.Reset()
		db, err := openPostgres(config.DatabaseConfig{Driver: "postgres", DSN: dsn})
		if err != nil {
			t.Fatalf("existing account blocked from connecting: %v", err)
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			t.Fatalf("connection unusable after privilege warning: %v", err)
		}
		if after := snapshot(); after != before {
			t.Fatal("startup changed the existing external database role or its memberships")
		}
		entries := hook.AllEntries()
		if !wantWarning {
			if len(entries) != 0 {
				t.Fatalf("ordinary role produced warnings: %v", entries)
			}
			return
		}
		if len(entries) != 1 || entries[0].Level != logrus.WarnLevel || entries[0].Data["security_event"] != "postgres_role_privileges" {
			t.Fatalf("missing privilege warning: %v", entries)
		}
	}
	t.Run("ordinary account", func(t *testing.T) { assertConnection(t, appDSN, false) })
	t.Run("existing superuser login", func(t *testing.T) { assertConnection(t, adminDSN, true) })
	t.Run("privileged login with ordinary current role", func(t *testing.T) {
		maskedURL, _ := url.Parse(adminDSN)
		options := maskedURL.Query()
		options.Set("options", "-c role=security_role_probe")
		maskedURL.RawQuery = options.Encode()
		assertConnection(t, maskedURL.String(), true)
	})
	for _, attribute := range []string{"SUPERUSER", "CREATEDB", "CREATEROLE", "REPLICATION", "BYPASSRLS"} {
		t.Run(attribute, func(t *testing.T) {
			mustExec(t, admin, "ALTER ROLE "+appUser+" "+attribute)
			t.Cleanup(func() { mustExec(t, admin, "ALTER ROLE "+appUser+" NO"+attribute) })
			assertConnection(t, appDSN, true)
		})
	}
	for _, role := range []string{"pg_read_server_files", "pg_write_server_files", "pg_execute_server_program"} {
		t.Run(role, func(t *testing.T) {
			mustExec(t, admin, "GRANT "+role+" TO "+appUser)
			t.Cleanup(func() { mustExec(t, admin, "REVOKE "+role+" FROM "+appUser) })
			assertConnection(t, appDSN, true)
		})
	}
	t.Run("indirect membership", func(t *testing.T) {
		mustExec(t, admin, "CREATE ROLE security_privilege_group NOLOGIN NOINHERIT")
		mustExec(t, admin, "GRANT pg_read_server_files TO security_privilege_group")
		mustExec(t, admin, "GRANT security_privilege_group TO "+appUser)
		t.Cleanup(func() {
			mustExec(t, admin, "REVOKE security_privilege_group FROM "+appUser)
			mustExec(t, admin, "DROP ROLE security_privilege_group")
		})
		assertConnection(t, appDSN, true)
	})
	t.Run("ordinary account after privilege removal", func(t *testing.T) { assertConnection(t, appDSN, false) })
	t.Run("privilege inspection failure remains nonfatal", func(t *testing.T) {
		db, err := sql.Open("pgx", appDSN)
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
		hook.Reset()
		warnPostgresRole(db)
		entries := hook.AllEntries()
		if len(entries) != 1 || entries[0].Level != logrus.WarnLevel || entries[0].Data["security_event"] != "postgres_role_check_failed" {
			t.Fatalf("missing nonfatal privilege inspection warning: %v", entries)
		}
	})
}
