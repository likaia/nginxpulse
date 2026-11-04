#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bash "$ROOT_DIR/scripts/test_postgres_roles.sh"

PG_TEST_IMAGE="${PG_TEST_IMAGE:-postgres:16}"
container_id="$(docker run -d --rm \
  -e POSTGRES_USER=security_admin -e POSTGRES_PASSWORD=security-admin-test-only \
  -e POSTGRES_DB=nginxpulse_security_test -p 127.0.0.1::5432 "$PG_TEST_IMAGE")"
trap 'docker rm -fv "$container_id" >/dev/null' EXIT
for attempt in {1..30}; do
  if docker exec "$container_id" pg_isready -h 127.0.0.1 -U security_admin -d nginxpulse_security_test >/dev/null 2>&1; then
    break
  fi
  if [[ "$attempt" == 30 ]]; then
    echo "Isolated PostgreSQL did not become ready" >&2
    exit 1
  fi
  sleep 1
done
docker exec -i "$container_id" psql -X -v ON_ERROR_STOP=1 -U security_admin -d postgres \
  -v app_user=security_app -v app_password=security-app-test-only \
  -v app_db=nginxpulse_security_test < "$ROOT_DIR/scripts/postgres-app-role.sql"
docker exec "$container_id" createdb -U security_admin -O security_app nginxpulse_cleanup_test
pg_port="$(docker port "$container_id" 5432/tcp)"
pg_port="${pg_port##*:}"
cd "$ROOT_DIR"
NGINXPULSE_SECURITY_TEST_PG_DSN="postgres://security_app:security-app-test-only@127.0.0.1:${pg_port}/nginxpulse_security_test?sslmode=disable" \
NGINXPULSE_SECURITY_TEST_PG_ADMIN_DSN="postgres://security_admin:security-admin-test-only@127.0.0.1:${pg_port}/nginxpulse_security_test?sslmode=disable" \
NGINXPULSE_TEST_PG_DSN="postgres://security_app:security-app-test-only@127.0.0.1:${pg_port}/nginxpulse_cleanup_test?sslmode=disable" \
  go test -count=1 ./...
