#!/bin/sh
# Sourced by entrypoint.sh; functions are also exercised against isolated PG.

init_postgres() {
  if [ -s "$PGDATA/PG_VERSION" ]; then
    return 0
  fi

  echo "nginxpulse: initializing postgres data dir at $PGDATA"
  if [ "$POSTGRES_ADMIN_USER" = "$POSTGRES_USER" ]; then
    echo "nginxpulse: POSTGRES_ADMIN_USER must differ from POSTGRES_USER" >&2
    return 1
  fi
  # The administrator has no password and is used only over the local socket.
  su-exec "$APP_USER:$APP_GROUP" initdb -D "$PGDATA" \
    --username="$POSTGRES_ADMIN_USER" \
    --auth-host=md5 \
    --auth-local=trust >/dev/null
}

start_postgres() {
  if [ "$(id -u)" = "0" ]; then
    mkdir -p "$POSTGRES_SOCKET_DIR"
    chown "$APP_USER:$APP_GROUP" "$POSTGRES_SOCKET_DIR" 2>/dev/null || true
    chmod 775 "$POSTGRES_SOCKET_DIR" 2>/dev/null || true
  fi
  su-exec "$APP_USER:$APP_GROUP" postgres -D "$PGDATA" \
    -p "$POSTGRES_PORT" \
    -c listen_addresses="$POSTGRES_LISTEN" -c unix_socket_directories="$POSTGRES_SOCKET_DIR" &
  pg_pid=$!

  waited=0
  while [ "$waited" -lt "$POSTGRES_STARTUP_TIMEOUT" ]; do
    if su-exec "$APP_USER:$APP_GROUP" pg_isready -h "$POSTGRES_CONNECT_HOST" -p "$POSTGRES_PORT" >/dev/null 2>&1; then
      return 0
    fi
    if ! kill -0 "$pg_pid" >/dev/null 2>&1; then
      return 1
    fi
    sleep 1
    waited=$((waited + 1))
  done
  return 1
}

# Always use the embedded cluster's Unix socket for administrative work.
# -X and ON_ERROR_STOP prevent local psql settings or partial failures from
# silently leaving the application privileged.
embedded_psql() {
  role="$1"
  shift
  su-exec "$APP_USER:$APP_GROUP" psql -X -v ON_ERROR_STOP=1 \
    -h "$POSTGRES_SOCKET_DIR" -p "$POSTGRES_PORT" -U "$role" -d postgres "$@"
}

ensure_database() {
  if [ "$POSTGRES_ADMIN_USER" = "$POSTGRES_USER" ]; then
    echo "nginxpulse: POSTGRES_ADMIN_USER must differ from POSTGRES_USER" >&2
    return 1
  fi

  if ! embedded_psql "$POSTGRES_ADMIN_USER" -tAc 'SELECT 1' >/dev/null 2>&1; then
    # PostgreSQL forbids demoting its bootstrap role (OID 10). Rename that
    # legacy login to the administrator, then recreate the application login.
    # A temporary local-only admin makes renaming the old session user possible.
    embedded_psql "$POSTGRES_USER" -v migration_user="${POSTGRES_ADMIN_USER}_migration" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN SUPERUSER PASSWORD NULL', :'migration_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'migration_user')
\gexec
SQL
    embedded_psql "${POSTGRES_ADMIN_USER}_migration" \
      -v app_user="$POSTGRES_USER" -v admin_user="$POSTGRES_ADMIN_USER" <<'SQL'
BEGIN;
ALTER ROLE :"app_user" RENAME TO :"admin_user";
ALTER ROLE :"admin_user" PASSWORD NULL;
CREATE ROLE :"app_user" LOGIN;
COMMIT;
SQL
  fi

  embedded_psql "$POSTGRES_ADMIN_USER" \
    -v app_user="$POSTGRES_USER" -v app_password="$POSTGRES_PASSWORD" \
    -v app_db="$POSTGRES_DB" -f "${POSTGRES_APP_ROLE_SQL:-/app/postgres-app-role.sql}"

  embedded_psql "$POSTGRES_ADMIN_USER" -d "$POSTGRES_DB" \
    -v app_user="$POSTGRES_USER" -v admin_user="$POSTGRES_ADMIN_USER" \
    -f "${POSTGRES_APP_OWNER_SQL:-/app/postgres-app-owner.sql}"

  embedded_psql "$POSTGRES_ADMIN_USER" -v migration_user="${POSTGRES_ADMIN_USER}_migration" <<'SQL'
DROP ROLE IF EXISTS :"migration_user";
SQL
}
