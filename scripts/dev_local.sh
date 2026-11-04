#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV_CONFIG="$ROOT_DIR/configs/nginxpulse_config.dev.json"
VERSION="${VERSION:-$(git -C "$ROOT_DIR" describe --tags --abbrev=0 2>/dev/null || echo "dev")}"
BUILD_TIME="${BUILD_TIME:-$(date "+%Y-%m-%d %H:%M:%S")}"
GIT_COMMIT="${GIT_COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short=7 HEAD 2>/dev/null || echo "unknown")}"
LDFLAGS="-s -w -X 'github.com/likaia/nginxpulse/internal/version.Version=${VERSION}' -X 'github.com/likaia/nginxpulse/internal/version.BuildTime=${BUILD_TIME}' -X 'github.com/likaia/nginxpulse/internal/version.GitCommit=${GIT_COMMIT}'"

backend_pid=""
backend_container=""
frontend_pid=""
mobile_frontend_pid=""
pg_started_by_us=0

PG_CONTAINER="${PG_CONTAINER:-nginxpulse-postgres}"
PG_VOLUME="${PG_VOLUME:-nginxpulse_pgdata}"
PG_HOST="${PG_HOST:-127.0.0.1}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-nginxpulse}"
PG_PASSWORD="${PG_PASSWORD:-nginxpulse}"
PG_DB="${PG_DB:-nginxpulse}"
USE_DOCKER_PG="${USE_DOCKER_PG:-auto}"
DB_DSN="${DB_DSN:-}"
DEFAULT_DSN="postgres://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${PG_DB}?sslmode=disable"
FORCE_SETUP_UI="${FORCE_SETUP_UI:-}"
FORCE_EMPTY_CONFIG="${FORCE_EMPTY_CONFIG:-}"
DOCKER_GO_IMAGE="${DOCKER_GO_IMAGE:-golang:1.24}"
BACKEND_RUNTIME="${BACKEND_RUNTIME:-auto}"
BACKEND_CONTAINER_NAME="${BACKEND_CONTAINER_NAME:-nginxpulse-dev-backend}"

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

ensure_cmd() {
  local cmd="$1"
  if ! have_cmd "$cmd"; then
    echo "$cmd not found in PATH. Please install it and retry." >&2
    exit 1
  fi
}

should_use_local_go() {
  if [[ "$BACKEND_RUNTIME" == "go" ]]; then
    return 0
  fi
  if [[ "$BACKEND_RUNTIME" == "docker" ]]; then
    return 1
  fi
  have_cmd go
}

ensure_backend_runtime() {
  if should_use_local_go; then
    return 0
  fi

  if have_cmd docker; then
    echo "go not found in PATH, falling back to Docker backend (${DOCKER_GO_IMAGE})."
    return 0
  fi

  cat >&2 <<'EOF'
Neither Go nor Docker is available.
- Install Go and re-run: pnpm run dev:local
- Or install Docker Desktop and let dev:local run the backend in a container
EOF
  exit 1
}

ensure_go_deps() {
  if [[ ! -f "$ROOT_DIR/go.sum" ]]; then
    echo "go.sum missing, running go mod tidy..."
    (cd "$ROOT_DIR" && GOFLAGS="-mod=mod" go mod tidy)
  fi
}

ensure_config() {
  local config_path="$DEV_CONFIG"
  if [[ ! -f "$config_path" ]]; then
    local base_config="$ROOT_DIR/configs/nginxpulse_config.json"
    if [[ ! -f "$base_config" ]]; then
      echo "configs/nginxpulse_config.json not found. Please create it first." >&2
      exit 1
    fi
    cp "$base_config" "$config_path"
    echo "Created configs/nginxpulse_config.dev.json from nginxpulse_config.json"
    echo "Edit configs/nginxpulse_config.dev.json and re-run." >&2
    exit 1
  fi
}

load_local_env_files() {
  local env_file=""
  local env_files=(
    "$ROOT_DIR/.env"
    "$ROOT_DIR/configs/.env"
    "$ROOT_DIR/.env.local"
    "$ROOT_DIR/configs/.env.local"
  )

  for env_file in "${env_files[@]}"; do
    if [[ -f "$env_file" ]]; then
      set -a
      # shellcheck disable=SC1090
      source "$env_file"
      set +a
    fi
  done
}

validate_dev_config_env() {
  local placeholder=""
  local var_name=""
  local missing=()

  while IFS= read -r placeholder; do
    [[ -z "$placeholder" ]] && continue
    if [[ "$placeholder" == *":-"* ]]; then
      continue
    fi
    var_name="${placeholder#\$\{}"
    var_name="${var_name%\}}"
    if [[ -z "${!var_name:-}" ]]; then
      missing+=("$var_name")
    fi
  done < <(rg -o '\$\{[A-Za-z_][A-Za-z0-9_]*(?::-([^}]*))?\}' "$DEV_CONFIG" | sort -u)

  if [[ "${#missing[@]}" -gt 0 ]]; then
    echo "Missing environment variables required by configs/nginxpulse_config.dev.json:" >&2
    printf '  - %s\n' "${missing[@]}" >&2
    echo "Set them in configs/.env or your shell environment, then retry." >&2
    exit 1
  fi
}

ensure_frontend_deps() {
  ensure_node_deps "$ROOT_DIR/webapp" "frontend" "vite"
}

ensure_mobile_frontend_deps() {
  ensure_node_deps "$ROOT_DIR/webapp_mobile" "mobile frontend" "vite"
}

ensure_node_deps() {
  local dir="$1"
  local label="$2"
  local bin_name="${3:-}"
  local install_needed=0
  if [[ ! -d "$dir/node_modules" ]]; then
    install_needed=1
  elif [[ -n "$bin_name" && ! -x "$dir/node_modules/.bin/$bin_name" ]]; then
    install_needed=1
  elif [[ "$dir/package.json" -nt "$dir/node_modules" ]]; then
    install_needed=1
  elif [[ -f "$dir/pnpm-lock.yaml" && "$dir/pnpm-lock.yaml" -nt "$dir/node_modules" ]]; then
    install_needed=1
  fi

  if [[ "$install_needed" -eq 1 ]]; then
    echo "Installing ${label} dependencies..."
    if [[ -f "$dir/pnpm-lock.yaml" ]]; then
      (cd "$dir" && pnpm install --frozen-lockfile) || (cd "$dir" && pnpm install)
    else
      (cd "$dir" && pnpm install)
    fi
  fi
}

is_truthy() {
  case "$1" in
    1|true|TRUE|yes|YES|on|ON)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

should_use_docker_pg() {
  if [[ -n "$DB_DSN" ]]; then
    return 1
  fi
  if [[ -z "$USE_DOCKER_PG" || "$USE_DOCKER_PG" == "auto" ]]; then
    return 0
  fi
  if is_truthy "$USE_DOCKER_PG"; then
    return 0
  fi
  return 1
}

start_postgres() {
  if is_truthy "$FORCE_SETUP_UI" || is_truthy "$FORCE_EMPTY_CONFIG"; then
    echo "FORCE_SETUP_UI/FORCE_EMPTY_CONFIG enabled, skipping PostgreSQL startup."
    return 0
  fi
  if ! should_use_docker_pg; then
    return 0
  fi

  ensure_cmd docker

  if docker ps -a --format '{{.Names}}' | grep -qx "$PG_CONTAINER"; then
    if docker ps --format '{{.Names}}' | grep -qx "$PG_CONTAINER"; then
      echo "PostgreSQL container already running: ${PG_CONTAINER}"
    else
      echo "Starting PostgreSQL container: ${PG_CONTAINER}"
      docker start "$PG_CONTAINER" >/dev/null
      pg_started_by_us=1
    fi
  else
    echo "Creating PostgreSQL container: ${PG_CONTAINER}"
    docker run -d --name "$PG_CONTAINER" \
      -e POSTGRES_USER="$PG_USER" \
      -e POSTGRES_PASSWORD="$PG_PASSWORD" \
      -e POSTGRES_DB="$PG_DB" \
      -p "${PG_PORT}:5432" \
      -v "${PG_VOLUME}:/var/lib/postgresql/data" \
      postgres:16 >/dev/null
    pg_started_by_us=1
  fi

  echo "Waiting for PostgreSQL to be ready..."
  for _ in {1..30}; do
    if docker exec "$PG_CONTAINER" pg_isready -U "$PG_USER" -d "$PG_DB" >/dev/null 2>&1; then
      # Official postgres images bootstrap POSTGRES_USER as superuser. Apply the
      # same role split as the embedded image, including upgrades of dev volumes.
      docker cp "$ROOT_DIR/scripts/embedded_postgres.sh" "$PG_CONTAINER:/tmp/nginxpulse-embedded-postgres.sh" >/dev/null
      docker cp "$ROOT_DIR/scripts/postgres-app-role.sql" "$PG_CONTAINER:/tmp/nginxpulse-app-role.sql" >/dev/null
      docker cp "$ROOT_DIR/scripts/postgres-app-owner.sql" "$PG_CONTAINER:/tmp/nginxpulse-app-owner.sql" >/dev/null
      docker exec -i -e POSTGRES_USER="$PG_USER" -e POSTGRES_PASSWORD="$PG_PASSWORD" \
        -e POSTGRES_DB="$PG_DB" "$PG_CONTAINER" sh -se <<'SH'
APP_USER=postgres
APP_GROUP=postgres
POSTGRES_ADMIN_USER=nginxpulse_admin
POSTGRES_SOCKET_DIR=/var/run/postgresql
POSTGRES_PORT=5432
POSTGRES_APP_ROLE_SQL=/tmp/nginxpulse-app-role.sql
POSTGRES_APP_OWNER_SQL=/tmp/nginxpulse-app-owner.sql
su-exec() { shift; "$@"; }
. /tmp/nginxpulse-embedded-postgres.sh
ensure_database
SH
      echo "PostgreSQL is ready at ${PG_HOST}:${PG_PORT} (${PG_DB})."
      DB_DSN="$DEFAULT_DSN"
      export DB_DSN
      return 0
    fi
    sleep 1
  done

  echo "PostgreSQL did not become ready in time." >&2
  exit 1
}

start_backend() {
  echo "Starting backend on http://localhost:8089"
  if should_use_local_go; then
    start_backend_with_go
  else
    start_backend_with_docker
  fi
}

rewrite_loopback_for_docker() {
  local value="$1"
  value="${value//127.0.0.1/host.docker.internal}"
  value="${value//localhost/host.docker.internal}"
  printf '%s' "$value"
}

start_backend_with_go() {
  if is_truthy "$FORCE_SETUP_UI" || is_truthy "$FORCE_EMPTY_CONFIG"; then
    local force_setup_ui="0"
    if is_truthy "$FORCE_SETUP_UI"; then
      force_setup_ui="1"
    fi
    local force_empty_config="0"
    if is_truthy "$FORCE_EMPTY_CONFIG"; then
      force_empty_config="1"
    fi
    (cd "$ROOT_DIR" && FORCE_SETUP_UI="$force_setup_ui" FORCE_EMPTY_CONFIG="$force_empty_config" SERVER_PORT=":8089" exec go run -ldflags="${LDFLAGS}" ./cmd/nginxpulse/main.go) &
  else
    if [[ -n "$DB_DSN" ]]; then
      echo "Using DB_DSN from environment or local docker."
    fi
    (cd "$ROOT_DIR" && CONFIG_JSON="$(cat "$DEV_CONFIG")" SERVER_PORT=":8089" DB_DSN="$DB_DSN" exec go run -ldflags="${LDFLAGS}" ./cmd/nginxpulse/main.go) &
  fi
  backend_pid=$!
  sleep 1
  if ! kill -0 "$backend_pid" >/dev/null 2>&1; then
    echo "Backend failed to start. Check if :8089 is already in use." >&2
    exit 1
  fi
}

start_backend_with_docker() {
  local force_setup_ui="0"
  local force_empty_config="0"
  local config_json=""
  local backend_dsn="$DB_DSN"
  local docker_args=(
    run --rm
    --name "$BACKEND_CONTAINER_NAME"
    --add-host host.docker.internal:host-gateway
    -p 8089:8089
    -v "$ROOT_DIR":/src
    -w /src
    -e SERVER_PORT=:8089
    -e LDFLAGS="$LDFLAGS"
  )

  if is_truthy "$FORCE_SETUP_UI"; then
    force_setup_ui="1"
  fi
  if is_truthy "$FORCE_EMPTY_CONFIG"; then
    force_empty_config="1"
  fi

  if [[ -n "$backend_dsn" ]]; then
    backend_dsn="$(rewrite_loopback_for_docker "$backend_dsn")"
  fi

  docker rm -f "$BACKEND_CONTAINER_NAME" >/dev/null 2>&1 || true

  if is_truthy "$FORCE_SETUP_UI" || is_truthy "$FORCE_EMPTY_CONFIG"; then
    docker_args+=(
      -e FORCE_SETUP_UI="$force_setup_ui"
      -e FORCE_EMPTY_CONFIG="$force_empty_config"
    )
  else
    config_json="$(cat "$DEV_CONFIG")"
    docker_args+=(
      -e CONFIG_JSON="$config_json"
      -e DB_DSN="$backend_dsn"
    )
  fi

  docker "${docker_args[@]}" \
    "$DOCKER_GO_IMAGE" \
    /bin/sh -lc 'go mod download >/dev/null && exec go run -ldflags="$LDFLAGS" ./cmd/nginxpulse/main.go' &
  backend_pid=$!
  backend_container="$BACKEND_CONTAINER_NAME"
  sleep 3
  if ! docker ps --format '{{.Names}}' | grep -qx "$BACKEND_CONTAINER_NAME"; then
    echo "Backend container failed to start. Check docker logs ${BACKEND_CONTAINER_NAME}." >&2
    exit 1
  fi
}

start_frontend() {
  echo "Starting frontend on http://localhost:8088"
  (cd "$ROOT_DIR/webapp" && pnpm exec vite --host 0.0.0.0 --port 8088) &
  frontend_pid=$!
}

start_mobile_frontend() {
  echo "Starting mobile frontend on http://localhost:8087 (LAN: http://<your-ip>:8087)"
  (cd "$ROOT_DIR/webapp_mobile" && pnpm exec vite --host 0.0.0.0 --port 8087) &
  mobile_frontend_pid=$!
}

cleanup() {
  if [[ -n "$frontend_pid" ]]; then
    kill "$frontend_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "$mobile_frontend_pid" ]]; then
    kill "$mobile_frontend_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "$backend_pid" ]]; then
    if have_cmd pkill; then
      pkill -TERM -P "$backend_pid" >/dev/null 2>&1 || true
    fi
    kill "$backend_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "$backend_container" ]]; then
    docker rm -f "$backend_container" >/dev/null 2>&1 || true
  fi
  if [[ "$pg_started_by_us" -eq 1 ]]; then
    docker stop "$PG_CONTAINER" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

ensure_cmd node
ensure_cmd pnpm
ensure_backend_runtime
if should_use_local_go; then
  ensure_go_deps
fi
if ! is_truthy "$FORCE_SETUP_UI" && ! is_truthy "$FORCE_EMPTY_CONFIG"; then
  ensure_config
  load_local_env_files
  validate_dev_config_env
else
  echo "FORCE_SETUP_UI enabled, skipping config file checks."
fi
ensure_frontend_deps
ensure_mobile_frontend_deps

start_postgres
start_backend
start_frontend
start_mobile_frontend

wait
