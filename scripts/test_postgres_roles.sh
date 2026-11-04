#!/usr/bin/env bash
set -euo pipefail

# Disposable clusters only. No project config, existing containers or host PG.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PG_TEST_IMAGE="${PG_TEST_IMAGE:-postgres:16}"
for scenario in fresh legacy quoted; do
  echo "Testing embedded PostgreSQL roles: $scenario"
  docker run --rm -i --network none \
    -v "$ROOT_DIR/scripts:/work:ro" -e SCENARIO="$scenario" \
    --entrypoint bash "$PG_TEST_IMAGE" -seu <<'SH'
export PATH="/usr/lib/postgresql/16/bin:$PATH"
APP_USER=postgres
APP_GROUP=postgres
PGDATA=/tmp/role-test-data
POSTGRES_USER=nginxpulse
POSTGRES_PASSWORD="probe'password"
POSTGRES_DB=nginxpulse_role_test
POSTGRES_ADMIN_USER=nginxpulse_admin
POSTGRES_PORT=5432
POSTGRES_LISTEN=127.0.0.1
POSTGRES_CONNECT_HOST=127.0.0.1
POSTGRES_SOCKET_DIR=/tmp/role-test-socket
POSTGRES_STARTUP_TIMEOUT=30
POSTGRES_APP_ROLE_SQL=/work/postgres-app-role.sql
POSTGRES_APP_OWNER_SQL=/work/postgres-app-owner.sql
su-exec() { shift; gosu postgres "$@"; }
. /work/embedded_postgres.sh
mkdir -p "$PGDATA"
chown postgres:postgres "$PGDATA"

if [ "$SCENARIO" = quoted ]; then
  POSTGRES_USER='app"role'
  POSTGRES_DB="app'database"
fi
if [ "$SCENARIO" = legacy ]; then
  su-exec postgres initdb -D "$PGDATA" --username="$POSTGRES_USER" \
    --auth-host=md5 --auth-local=trust >/dev/null
else
  init_postgres
fi
start_postgres >/tmp/pg-test.log 2>&1
trap 'su-exec postgres pg_ctl -D "$PGDATA" -m immediate stop >/dev/null' EXIT

if [ "$SCENARIO" = legacy ]; then
  embedded_psql "$POSTGRES_USER" -v app_user="$POSTGRES_USER" -v app_db="$POSTGRES_DB" <<'SQL'
CREATE DATABASE :"app_db" OWNER :"app_user";
GRANT pg_read_server_files, pg_write_server_files, pg_execute_server_program TO :"app_user";
SQL
  su-exec postgres psql -X -v ON_ERROR_STOP=1 -h "$POSTGRES_SOCKET_DIR" \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB" <<'SQL'
CREATE TABLE preserved_data(id SERIAL PRIMARY KEY);
INSERT INTO preserved_data(id) VALUES (103);
CREATE TABLE preserved_partition(id int) PARTITION BY RANGE(id);
CREATE TABLE preserved_partition_old PARTITION OF preserved_partition FOR VALUES FROM (0) TO (1000);
INSERT INTO preserved_partition VALUES (103);
CREATE VIEW preserved_view AS SELECT id FROM preserved_data;
CREATE SCHEMA preserved_schema;
CREATE TABLE preserved_schema.existing_table(id int);
SQL
fi
ensure_database
embedded_psql "$POSTGRES_ADMIN_USER" -v app_user="$POSTGRES_USER" -v admin_user="$POSTGRES_ADMIN_USER" <<'SQL'
ALTER ROLE :"app_user" SUPERUSER CREATEDB CREATEROLE REPLICATION BYPASSRLS;
GRANT pg_read_server_files, pg_write_server_files, pg_execute_server_program, :"admin_user" TO :"app_user";
SQL
ensure_database # Restarts must be idempotent.

flags="$(embedded_psql "$POSTGRES_ADMIN_USER" -v app_user="$POSTGRES_USER" -tA <<'SQL'
SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls
FROM pg_roles WHERE rolname = :'app_user';
SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member WHERE r.rolname=:'app_user';
SQL
)"
[ "$flags" = "$(printf 'f\n0')" ]
[ "$(embedded_psql "$POSTGRES_ADMIN_USER" -tAc "SELECT rolpassword IS NULL FROM pg_authid WHERE rolname='nginxpulse_admin'")" = t ]

app_psql() {
  PGPASSWORD="$POSTGRES_PASSWORD" psql -X -v ON_ERROR_STOP=1 \
    -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"
}
app_psql <<'SQL'
CREATE TABLE partition_test (id int) PARTITION BY RANGE(id);
CREATE TABLE partition_test_p1 PARTITION OF partition_test FOR VALUES FROM (0) TO (1000);
CREATE INDEX partition_test_idx ON partition_test(id);
INSERT INTO partition_test VALUES (103);
UPDATE partition_test SET id=104 WHERE id=103;
DELETE FROM partition_test WHERE id=104;
DROP TABLE partition_test;
SQL
if [ "$SCENARIO" = legacy ]; then
  [ "$(app_psql -tAc 'SELECT id FROM preserved_data')" = 103 ]
  [ "$(app_psql -tAc 'SELECT id FROM preserved_partition')" = 103 ]
  [ "$(app_psql -tAc 'SELECT id FROM preserved_view')" = 103 ]
  app_psql <<'SQL'
INSERT INTO preserved_data DEFAULT VALUES;
ALTER TABLE preserved_data ADD COLUMN migrated boolean;
ALTER TABLE preserved_partition DETACH PARTITION preserved_partition_old;
CREATE TABLE preserved_schema.new_table(id int);
SQL
fi
for forbidden in \
  "SELECT pg_read_file('/etc/passwd')" \
  "SELECT lo_export(lo_from_bytea(0, 'probe'::bytea), '/tmp/forbidden-write')" \
  "COPY (SELECT 1) TO PROGRAM 'true'" \
  "SET ROLE nginxpulse_admin" \
  "CREATE ROLE forbidden_role" \
  "ALTER SYSTEM SET work_mem='4MB'"; do
  if app_psql -c "$forbidden" >/tmp/forbidden-result 2>&1; then
    echo "Application unexpectedly executed: $forbidden" >&2
    exit 1
  fi
  # Check a permission denial, rather than treating a SQL/connection error as success.
  grep -Eq 'permission denied|must be superuser|must have privileges' /tmp/forbidden-result
done
echo "PASS: $SCENARIO (non-superuser, data/DDL retained, server privileges denied)"
SH
done
