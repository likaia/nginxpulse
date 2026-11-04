#!/bin/sh
set -e

DATA_DIR="${DATA_DIR:-/app/var/nginxpulse_data}"
PGDATA="${PGDATA:-/app/var/pgdata}"
CONFIG_DIR="${CONFIG_DIR:-/app/configs}"
TMPDIR="${TMPDIR:-${DATA_DIR}/tmp}"
POSTGRES_USER="${POSTGRES_USER:-nginxpulse}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-nginxpulse}"
POSTGRES_DB="${POSTGRES_DB:-nginxpulse}"
POSTGRES_ADMIN_USER="${POSTGRES_ADMIN_USER:-nginxpulse_admin}"
POSTGRES_SOCKET_DIR="${POSTGRES_SOCKET_DIR:-/run/postgresql}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_LISTEN="${POSTGRES_LISTEN:-127.0.0.1}"
POSTGRES_CONNECT_HOST="${POSTGRES_CONNECT_HOST:-127.0.0.1}"
POSTGRES_STARTUP_TIMEOUT="${POSTGRES_STARTUP_TIMEOUT:-120}"
POSTGRES_STOP_TIMEOUT="${POSTGRES_STOP_TIMEOUT:-60}"

APP_UID="${PUID:-}"
APP_GID="${PGID:-}"
APP_USER="nginxpulse"
APP_GROUP="nginxpulse"
USE_EMBEDDED_PG=1
if [ -n "${DB_DSN:-}" ]; then
  USE_EMBEDDED_PG=0
fi

if [ -n "$APP_GID" ]; then
  EXISTING_GROUP="$(awk -F: -v gid="$APP_GID" '$3==gid{print $1; exit}' /etc/group)"
  if [ -z "$EXISTING_GROUP" ]; then
    addgroup -S -g "$APP_GID" appgroup
    APP_GROUP="appgroup"
  else
    APP_GROUP="$EXISTING_GROUP"
  fi
fi

if [ -n "$APP_UID" ]; then
  EXISTING_USER="$(awk -F: -v uid="$APP_UID" '$3==uid{print $1; exit}' /etc/passwd)"
  if [ -z "$EXISTING_USER" ]; then
    adduser -S -D -H -u "$APP_UID" -G "$APP_GROUP" appuser
    APP_USER="appuser"
  else
    APP_USER="$EXISTING_USER"
  fi
fi

export TMPDIR
mkdir -p "$DATA_DIR" "$PGDATA" "$TMPDIR" "$CONFIG_DIR"

is_truthy() {
  case "$(printf '%s' "$1" | tr 'A-Z' 'a-z')" in
    1|true|yes|on)
      return 0
      ;;
  esac
  return 1
}

coerce_positive_int() {
  local value="$1"
  local fallback="$2"
  case "$value" in
    ''|*[!0-9]*|0)
      printf '%s' "$fallback"
      ;;
    *)
      printf '%s' "$value"
      ;;
  esac
}

POSTGRES_STARTUP_TIMEOUT="$(coerce_positive_int "$POSTGRES_STARTUP_TIMEOUT" 120)"
POSTGRES_STOP_TIMEOUT="$(coerce_positive_int "$POSTGRES_STOP_TIMEOUT" 60)"

has_env_config_source() {
  [ -n "${CONFIG_JSON:-}" ] || [ -n "${WEBSITES:-}" ]
}

is_setup_mode() {
  if is_truthy "${FORCE_SETUP_UI:-}"; then
    return 0
  fi
  if is_truthy "${FORCE_EMPTY_CONFIG:-}"; then
    return 0
  fi
  if has_env_config_source; then
    return 1
  fi
  if [ -f "$CONFIG_DIR/nginxpulse_config.json" ]; then
    return 1
  fi
  return 0
}

extract_db_dsn_from_config() {
  local raw=""
  if [ -n "${CONFIG_JSON:-}" ]; then
    raw="$(printf '%s' "$CONFIG_JSON" | tr '\n' ' ' | awk -F'"' '/"dsn"[[:space:]]*:/ {print $4; exit}')"
  elif [ -f "$CONFIG_DIR/nginxpulse_config.json" ]; then
    raw="$(awk -F'"' '/"dsn"[[:space:]]*:/ {print $4; exit}' "$CONFIG_DIR/nginxpulse_config.json")"
  fi
  printf '%s' "$raw"
}

extract_dsn_host() {
  local dsn="$1"
  local host=""
  if printf '%s' "$dsn" | grep -qi 'host='; then
    host="$(printf '%s' "$dsn" | sed -n 's/.*host=\\([^ ]*\\).*/\\1/p' | head -n 1)"
  fi
  if [ -z "$host" ]; then
    host="$(printf '%s' "$dsn" | sed -n 's#^[A-Za-z0-9+.-]*://##p' | sed 's#^[^@]*@##' | sed 's#[/?].*$##' | sed 's#:.*$##' | head -n 1)"
  fi
  host="$(printf '%s' "$host" | sed 's/^\\[//; s/\\]$//')"
  printf '%s' "$host"
}

extract_dsn_port() {
  local dsn="$1"
  local port=""
  if printf '%s' "$dsn" | grep -qi 'port='; then
    port="$(printf '%s' "$dsn" | sed -n 's/.*port=\\([0-9][0-9]*\\).*/\\1/p' | head -n 1)"
  fi
  if [ -z "$port" ]; then
    port="$(printf '%s' "$dsn" | sed -n 's#^[A-Za-z0-9+.-]*://##p' | sed 's#^[^@]*@##' | sed 's#[/?].*$##' | sed -n 's#.*:\\([0-9][0-9]*\\)$#\\1#p' | head -n 1)"
  fi
  if [ -z "$port" ]; then
    port="5432"
  fi
  printf '%s' "$port"
}

is_loopback_host() {
  case "$1" in
    ""|localhost|127.0.0.1|::1)
      return 0
      ;;
  esac
  return 1
}

is_external_dsn() {
  local host
  host="$(extract_dsn_host "$1")"
  if [ -z "$host" ]; then
    return 1
  fi
  if is_loopback_host "$host"; then
    return 1
  fi
  return 0
}

if [ "$USE_EMBEDDED_PG" = "1" ]; then
  CONFIG_DSN="$(extract_db_dsn_from_config)"
  if [ -n "$CONFIG_DSN" ] && is_external_dsn "$CONFIG_DSN"; then
    USE_EMBEDDED_PG=0
  fi
fi

if is_setup_mode; then
  USE_EMBEDDED_PG=0
fi

if [ "$USE_EMBEDDED_PG" = "1" ]; then
  EFFECTIVE_DSN="${DB_DSN:-$CONFIG_DSN}"
  if [ -n "$EFFECTIVE_DSN" ] && command -v pg_isready >/dev/null 2>&1; then
    DSN_HOST="$(extract_dsn_host "$EFFECTIVE_DSN")"
    DSN_PORT="$(extract_dsn_port "$EFFECTIVE_DSN")"
    if [ -n "$DSN_HOST" ]; then
      if pg_isready -h "$DSN_HOST" -p "$DSN_PORT" >/dev/null 2>&1; then
        pg_ready=0
      else
        pg_ready=$?
      fi
      if [ "$pg_ready" -eq 0 ] || [ "$pg_ready" -eq 1 ]; then
        USE_EMBEDDED_PG=0
      fi
    fi
  fi
fi

EFFECTIVE_DSN="${DB_DSN:-$CONFIG_DSN}"
if [ "$USE_EMBEDDED_PG" = "1" ]; then
  echo "nginxpulse: use external postgres: no (using embedded postgres)"
elif [ -n "$EFFECTIVE_DSN" ]; then
  EFFECTIVE_DB_HOST="$(extract_dsn_host "$EFFECTIVE_DSN")"
  if [ -n "$EFFECTIVE_DB_HOST" ]; then
    echo "nginxpulse: use external postgres: yes (host=$EFFECTIVE_DB_HOST)"
  else
    echo "nginxpulse: use external postgres: yes"
  fi
else
  echo "nginxpulse: use external postgres: no (setup mode; database not configured yet)"
fi

normalize_web_base_path() {
  local value
  value="$(printf '%s' "$1" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//; s#^/*##; s#/*$##')"
  if [ -z "$value" ]; then
    printf ''
    return 0
  fi
  if printf '%s' "$value" | grep -q '/'; then
    printf ''
    return 0
  fi
  if ! printf '%s' "$value" | grep -Eq '^[A-Za-z0-9_-]+$'; then
    printf ''
    return 0
  fi
  case "$(printf '%s' "$value" | tr 'A-Z' 'a-z')" in
    api|m|assets|favicon.svg|brand-mark|brand-mark.svg|app-config.js|health)
      printf ''
      return 0
      ;;
  esac
  printf '%s' "$value"
}

extract_web_base_path() {
  local raw=""
  if [ -n "${WEB_BASE_PATH:-}" ]; then
    raw="$WEB_BASE_PATH"
  elif [ -n "${CONFIG_JSON:-}" ]; then
    raw="$(printf '%s' "$CONFIG_JSON" | tr '\n' ' ' | sed -n 's/.*\"webBasePath\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p' | head -n 1)"
  elif [ -f "$CONFIG_DIR/nginxpulse_config.json" ]; then
    raw="$(sed -n 's/.*\"webBasePath\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p' "$CONFIG_DIR/nginxpulse_config.json" | head -n 1)"
  fi
  normalize_web_base_path "$raw"
}

write_app_config() {
  local base_path="$1"
  local prefix=""
  if [ -n "$base_path" ]; then
    prefix="/$base_path"
  fi
  printf 'window.__NGINXPULSE_BASE_PATH__ = "%s";\n' "$prefix" > /usr/share/nginx/html/app-config.js
  printf 'window.__NGINXPULSE_BASE_PATH__ = "%s";\n' "$prefix" > /usr/share/nginx/html/m/app-config.js
}

write_nginx_conf() {
  local base_path="$1"
  local conf="/etc/nginx/conf.d/default.conf"
  if [ -z "$base_path" ]; then
    cat > "$conf" <<'EOF'
server {
  listen 8088;
  server_name _;
  absolute_redirect off;
  port_in_redirect off;

  root /usr/share/nginx/html;
  index index.html;

  location = /app-config.js {
    add_header Cache-Control "no-store";
    try_files $uri =404;
  }

  location /api/ {
    proxy_pass http://127.0.0.1:8089;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }

  location = /m {
    return 302 /m/;
  }

  location /m/ {
    try_files $uri $uri/ /m/index.html;
  }

  location / {
    try_files $uri $uri/ /index.html;
  }
}
EOF
    return 0
  fi

  cat > "$conf" <<EOF
server {
  listen 8088;
  server_name _;
  absolute_redirect off;
  port_in_redirect off;

  root /usr/share/nginx/html;
  index index.html;

  location = /app-config.js {
    add_header Cache-Control "no-store";
    try_files \$uri =404;
  }

  location = /m/app-config.js {
    add_header Cache-Control "no-store";
    try_files \$uri =404;
  }

  location = /favicon.svg {
    try_files \$uri =404;
  }

  location = /brand-mark.svg {
    try_files \$uri =404;
  }

  location = /m/favicon.svg {
    try_files \$uri =404;
  }

  location = /m/brand-mark.svg {
    try_files \$uri =404;
  }

  location = /index.html {
    try_files \$uri =404;
  }

  location = /m/index.html {
    try_files \$uri =404;
  }

  location /assets/ {
    try_files \$uri =404;
  }

  location /m/assets/ {
    try_files \$uri =404;
  }

  location = /$base_path {
    return 302 /$base_path/;
  }

  location = /$base_path/m {
    return 302 /$base_path/m/;
  }

  location /$base_path/api/ {
    proxy_pass http://127.0.0.1:8089;
    proxy_http_version 1.1;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }

  location = /$base_path/api {
    return 308 /$base_path/api/;
  }

  location /$base_path/m/ {
    rewrite ^/$base_path/m/(.*)$ /m/\$1 break;
    try_files \$uri /m/index.html;
  }

  location /$base_path/ {
    rewrite ^/$base_path/(.*)$ /\$1 break;
    try_files \$uri /index.html;
  }

  location / {
    return 404;
  }
}
EOF
}

is_mount_point() {
  awk -v target="$1" '$2==target {found=1} END {exit found?0:1}' /proc/mounts
}

if ! is_mount_point "$DATA_DIR"; then
  echo "nginxpulse: $DATA_DIR is not a mounted volume. Please bind-mount a host directory to $DATA_DIR." >&2
  exit 1
fi
if [ "$USE_EMBEDDED_PG" = "1" ]; then
  if ! is_mount_point "$PGDATA"; then
    echo "nginxpulse: $PGDATA is not a mounted volume. Please bind-mount a host directory to $PGDATA." >&2
    exit 1
  fi
fi

if [ "$(id -u)" = "0" ]; then
  if ! su-exec "$APP_USER:$APP_GROUP" sh -lc "touch '$DATA_DIR/.write_test' && rm -f '$DATA_DIR/.write_test'" >/dev/null 2>&1; then
    chown -R "$APP_USER:$APP_GROUP" "$DATA_DIR" 2>/dev/null || true
  fi
fi

if ! su-exec "$APP_USER:$APP_GROUP" sh -lc "touch '$DATA_DIR/.write_test' && rm -f '$DATA_DIR/.write_test'" >/dev/null 2>&1; then
  echo "nginxpulse: $DATA_DIR is not writable; file logging may fail and will fall back to stdout" >&2
fi

# Ensure CONFIG_DIR is writable for saving configuration
if [ "$(id -u)" = "0" ]; then
  if ! su-exec "$APP_USER:$APP_GROUP" sh -lc "touch '$CONFIG_DIR/.write_test' && rm -f '$CONFIG_DIR/.write_test'" >/dev/null 2>&1; then
    chown -R "$APP_USER:$APP_GROUP" "$CONFIG_DIR" 2>/dev/null || true
  fi
fi

if ! su-exec "$APP_USER:$APP_GROUP" sh -lc "touch '$CONFIG_DIR/.write_test' && rm -f '$CONFIG_DIR/.write_test'" >/dev/null 2>&1; then
  echo "nginxpulse: $CONFIG_DIR is not writable; configuration saving may fail" >&2
fi

if [ "$USE_EMBEDDED_PG" = "1" ]; then
  if [ "$(id -u)" = "0" ]; then
    if ! su-exec "$APP_USER:$APP_GROUP" sh -lc "touch '$PGDATA/.write_test' && rm -f '$PGDATA/.write_test'" >/dev/null 2>&1; then
      chown -R "$APP_USER:$APP_GROUP" "$PGDATA" 2>/dev/null || true
    fi
  fi

  if ! su-exec "$APP_USER:$APP_GROUP" sh -lc "touch '$PGDATA/.write_test' && rm -f '$PGDATA/.write_test'" >/dev/null 2>&1; then
    echo "nginxpulse: $PGDATA is not writable; postgres may fail to start" >&2
  fi
fi

. /app/embedded_postgres.sh

if [ "$USE_EMBEDDED_PG" = "1" ] && [ -z "${DB_DSN:-}" ]; then
  export DB_DRIVER="postgres"
  export DB_DSN="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_CONNECT_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
fi

if [ "$USE_EMBEDDED_PG" = "1" ]; then
  init_postgres
  if ! start_postgres; then
    echo "nginxpulse: postgres did not become ready within ${POSTGRES_STARTUP_TIMEOUT}s" >&2
    exit 1
  fi
  ensure_database
fi

if command -v nginx >/dev/null 2>&1; then
  WEB_BASE_PATH_VALUE="$(extract_web_base_path)"
  write_app_config "$WEB_BASE_PATH_VALUE"
  write_nginx_conf "$WEB_BASE_PATH_VALUE"

  su-exec "$APP_USER:$APP_GROUP" /app/nginxpulse "$@" &
  backend_pid=$!
  nginx -g 'daemon off;' &
  nginx_pid=$!
  shutting_down=0

  stop_process() {
    local pid="$1"
    if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
      kill -TERM "$pid" >/dev/null 2>&1 || true
    fi
  }

  wait_process() {
    local pid="$1"
    if [ -n "$pid" ]; then
      wait "$pid" >/dev/null 2>&1 || true
    fi
  }

  stop_embedded_postgres() {
    if [ "$USE_EMBEDDED_PG" != "1" ]; then
      return 0
    fi
    if [ -s "$PGDATA/postmaster.pid" ] && command -v pg_ctl >/dev/null 2>&1; then
      su-exec "$APP_USER:$APP_GROUP" pg_ctl -D "$PGDATA" -m fast -w -t "$POSTGRES_STOP_TIMEOUT" stop >/dev/null 2>&1 || true
    fi
    if [ -n "${pg_pid:-}" ] && kill -0 "$pg_pid" >/dev/null 2>&1; then
      kill -TERM "$pg_pid" >/dev/null 2>&1 || true
    fi
  }

  shutdown() {
    if [ "$shutting_down" -eq 1 ]; then
      return 0
    fi
    shutting_down=1

    stop_process "${backend_pid:-}"
    stop_process "${nginx_pid:-}"
    stop_embedded_postgres

    wait_process "${backend_pid:-}"
    wait_process "${nginx_pid:-}"
    wait_process "${pg_pid:-}"
  }

  handle_signal() {
    shutdown
    exit 0
  }

  trap handle_signal INT TERM

  while :; do
    if [ -n "${pg_pid:-}" ] && ! kill -0 "$pg_pid" >/dev/null 2>&1; then
      shutdown
      exit 1
    fi
    if [ -n "${backend_pid:-}" ] && ! kill -0 "$backend_pid" >/dev/null 2>&1; then
      shutdown
      exit 1
    fi
    if [ -n "${nginx_pid:-}" ] && ! kill -0 "$nginx_pid" >/dev/null 2>&1; then
      shutdown
      exit 1
    fi
    sleep 1
  done
fi

exec su-exec "$APP_USER:$APP_GROUP" /app/nginxpulse "$@"
