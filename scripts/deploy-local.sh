#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/tmp/local-deploy"
BIN_DIR="$RUNTIME_DIR/bin"
PID_DIR="$RUNTIME_DIR/pids"
LOG_DIR="$ROOT_DIR/log/local-deploy"
WORKSPACE_DIR="$RUNTIME_DIR/workspaces"
ENV_FILE="$RUNTIME_DIR/synclax.env"
WORKFLOW_FILE="$ROOT_DIR/WORKFLOW.local.md"
APP_CONFIG_FILE="$ROOT_DIR/app.yaml"

POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-synclax-local-postgres}"
POSTGRES_VOLUME="${POSTGRES_VOLUME:-synclax-local-postgres-data}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:16-alpine}"

API_PORT="${API_PORT:-2910}"
WEB_PORT="${WEB_PORT:-3000}"
PG_PORT="${PG_PORT:-}"
DEBUG_PORT="${DEBUG_PORT:-8089}"

DB_NAME="${DB_NAME:-synclax}"
DB_USER="${DB_USER:-synclax}"
DB_PASSWORD="${DB_PASSWORD:-synclax}"
DB_DSN=""

API_PID_FILE="$PID_DIR/api.pid"
WEB_PID_FILE="$PID_DIR/web.pid"
API_LOG_FILE="$LOG_DIR/api.log"
WEB_LOG_FILE="$LOG_DIR/web.log"
DEPLOY_INFO_FILE="$RUNTIME_DIR/deploy-info.txt"

mkdir -p "$BIN_DIR" "$PID_DIR" "$LOG_DIR" "$WORKSPACE_DIR"

require_brew() {
  if ! command -v brew >/dev/null 2>&1; then
    echo "Homebrew is required to auto-install missing dependencies." >&2
    echo "Install Homebrew first: https://brew.sh/" >&2
    exit 1
  fi
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

ensure_brew_formula() {
  local cmd_name="$1"
  local formula="$2"
  if command -v "$cmd_name" >/dev/null 2>&1; then
    return 0
  fi
  require_brew
  echo "Installing missing dependency with Homebrew: $formula"
  brew install "$formula"
}

ensure_brew_cask() {
  local cmd_name="$1"
  local cask="$2"
  if command -v "$cmd_name" >/dev/null 2>&1; then
    return 0
  fi
  require_brew
  echo "Installing missing dependency with Homebrew cask: $cask"
  brew install --cask "$cask"
}

wait_for_docker() {
  for _ in $(seq 1 90); do
    if docker info >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "Docker is installed but the daemon is not ready." >&2
  echo "Start Docker Desktop (or your Docker daemon) and rerun the script." >&2
  exit 1
}

ensure_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    if [[ "$(uname -s)" == "Darwin" ]]; then
      ensure_brew_cask docker docker
    else
      ensure_brew_formula docker docker
    fi
  fi

  if docker info >/dev/null 2>&1; then
    return 0
  fi

  if [[ "$(uname -s)" == "Darwin" ]]; then
    open -ga Docker >/dev/null 2>&1 || true
  fi
  wait_for_docker
}

install_missing_dependencies() {
  ensure_docker
  ensure_brew_formula git git
  ensure_brew_formula go go
  ensure_brew_formula node node
  ensure_brew_formula pnpm pnpm
}

is_pid_running() {
  local pid_file="$1"
  if [[ ! -f "$pid_file" ]]; then
    return 1
  fi
  local pid
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  [[ -n "$pid" ]] || return 1
  kill -0 "$pid" >/dev/null 2>&1
}

stop_pid_file() {
  local pid_file="$1"
  if ! is_pid_running "$pid_file"; then
    rm -f "$pid_file"
    return 0
  fi

  local pid
  pid="$(cat "$pid_file")"
  kill "$pid" >/dev/null 2>&1 || true

  for _ in $(seq 1 20); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      break
    fi
    sleep 0.5
  done

  if kill -0 "$pid" >/dev/null 2>&1; then
    kill -9 "$pid" >/dev/null 2>&1 || true
  fi

  rm -f "$pid_file"
}

wait_for_url() {
  local url="$1"
  local label="$2"
  for _ in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "$label did not become ready: $url" >&2
  exit 1
}

is_port_available() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    ! lsof -n -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    return
  fi
  ! nc -z 127.0.0.1 "$port" >/dev/null 2>&1
}

choose_postgres_port() {
  if docker ps -a --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER"; then
    PG_PORT="$(docker inspect --format '{{with (index (index .NetworkSettings.Ports "5432/tcp") 0)}}{{.HostPort}}{{end}}' "$POSTGRES_CONTAINER" 2>/dev/null || true)"
  fi

  if [[ -n "$PG_PORT" ]]; then
    if ! is_port_available "$PG_PORT"; then
      if ! docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER"; then
        echo "postgres port is already in use: $PG_PORT" >&2
        exit 1
      fi
    fi
  else
    for candidate in 55432 55433 55434 55435 55436 55437; do
      if is_port_available "$candidate"; then
        PG_PORT="$candidate"
        break
      fi
    done
  fi

  if [[ -z "$PG_PORT" ]]; then
    echo "could not find an open local port for postgres" >&2
    exit 1
  fi

  DB_DSN="postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:${PG_PORT}/${DB_NAME}?sslmode=disable"
}

write_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    return 0
  fi

  cat >"$ENV_FILE" <<EOF
LINEAR_API_KEY=
LINEAR_PROJECT_SLUG=
EOF
}

write_workflow_file() {
  local codex_cmd
  codex_cmd="$(command -v codex 2>/dev/null || true)"
  if [[ -z "$codex_cmd" ]]; then
    codex_cmd="codex"
  fi

  cat >"$WORKFLOW_FILE" <<EOF
---
tracker:
  kind: linear
  endpoint: https://api.linear.app/graphql
  api_key: \$LINEAR_API_KEY
  project_slug: \$LINEAR_PROJECT_SLUG
  active_states:
    - Todo
    - In Progress
    - Rework
    - Merging
  terminal_states:
    - Done
    - Closed
    - Cancelled
    - Canceled
    - Duplicate

logging:
  file: $LOG_DIR/symphony.log
  max_size_mb: 10
  max_backups: 5
  max_age_days: 7
  compress: false

polling:
  interval_ms: 5000

workspace:
  root: $WORKSPACE_DIR

hooks:
  after_create: |
    /usr/bin/tar --exclude .git --exclude node_modules --exclude web/node_modules --exclude tmp --exclude log -C $ROOT_DIR -cf - . | /usr/bin/tar -xf -
  before_run: |
    if command -v pnpm >/dev/null 2>&1 && [ -f web/package.json ]; then
      cd web
      pnpm install --frozen-lockfile
    fi
  timeout_ms: 120000

agent:
  max_concurrent_agents: 3
  max_turns: 12

codex:
  command: $codex_cmd app-server

server:
  port: $DEBUG_PORT
---

You are working on Linear ticket {{ issue.identifier }}.

Current state: {{ issue.state }}
Title: {{ issue.title }}
URL: {{ issue.url }}

Description:
{% if issue.description %}
{{ issue.description }}
{% else %}
No description provided.
{% endif %}

Rules:
1. Work only inside the provided repository copy.
2. Complete the requested implementation, validate it, and report final results briefly.
3. If required credentials or external access are missing, stop and report the blocker clearly.
EOF
}

write_app_config_file() {
  if [[ -f "$APP_CONFIG_FILE" ]] && ! grep -Fq "managed-by: scripts/deploy-local.sh" "$APP_CONFIG_FILE"; then
    echo "refusing to overwrite existing app.yaml that is not managed by deploy-local.sh" >&2
    exit 1
  fi

  cat >"$APP_CONFIG_FILE" <<EOF
# managed-by: scripts/deploy-local.sh
anclax:
  port: $API_PORT
  pg:
    dsn: $DB_DSN

symphony:
  workflow_path: $WORKFLOW_FILE
  http_port: $DEBUG_PORT
EOF
}

start_postgres() {
  if docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER"; then
    return 0
  fi

  if docker ps -a --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER"; then
    docker start "$POSTGRES_CONTAINER" >/dev/null
    return 0
  fi

  docker volume create "$POSTGRES_VOLUME" >/dev/null
  docker run -d \
    --name "$POSTGRES_CONTAINER" \
    --restart unless-stopped \
    -p "${PG_PORT}:5432" \
    -e "POSTGRES_DB=${DB_NAME}" \
    -e "POSTGRES_USER=${DB_USER}" \
    -e "POSTGRES_PASSWORD=${DB_PASSWORD}" \
    -v "${POSTGRES_VOLUME}:/var/lib/postgresql/data" \
    --health-cmd "pg_isready -U ${DB_USER} -d ${DB_NAME}" \
    --health-interval 5s \
    --health-timeout 5s \
    --health-retries 20 \
    "$POSTGRES_IMAGE" >/dev/null
}

wait_for_postgres() {
  for _ in $(seq 1 60); do
    local status
    status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$POSTGRES_CONTAINER" 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      return 0
    fi
    sleep 1
  done
  echo "postgres container did not become healthy" >&2
  docker logs "$POSTGRES_CONTAINER" | tail -100 >&2 || true
  exit 1
}

build_api_binary() {
  (cd "$ROOT_DIR" && go build -o "$BIN_DIR/anclax-local" ./cmd/main.go)
}

start_api() {
  stop_pid_file "$API_PID_FILE"

  (
    cd "$ROOT_DIR"
    set -a
    source "$ENV_FILE"
    set +a
    nohup "$BIN_DIR/anclax-local" >"$API_LOG_FILE" 2>&1 &
    echo $! >"$API_PID_FILE"
  )
}

build_web() {
  (
    cd "$ROOT_DIR/web"
    pnpm install --frozen-lockfile
    pnpm run build
  )
}

start_web() {
  stop_pid_file "$WEB_PID_FILE"

  (
    cd "$ROOT_DIR/web"
    nohup pnpm run preview -- --host 0.0.0.0 --port "$WEB_PORT" >"$WEB_LOG_FILE" 2>&1 &
    echo $! >"$WEB_PID_FILE"
  )
}

write_deploy_info() {
  cat >"$DEPLOY_INFO_FILE" <<EOF
Synclax local deployment

API: http://localhost:$API_PORT/api/v1/health
Web: http://localhost:$WEB_PORT
App config: $APP_CONFIG_FILE
Workflow: $WORKFLOW_FILE
Env file: $ENV_FILE
Postgres container: $POSTGRES_CONTAINER

To enable Symphony against your Linear project:
1. Edit $ENV_FILE
2. Set LINEAR_API_KEY and LINEAR_PROJECT_SLUG
3. Re-run scripts/deploy-local.sh
EOF
}

install_missing_dependencies
require_cmd docker
require_cmd curl
require_cmd go
require_cmd pnpm
require_cmd git
require_cmd tar

choose_postgres_port
write_env_file
write_workflow_file
write_app_config_file
start_postgres
wait_for_postgres
build_api_binary
start_api
wait_for_url "http://127.0.0.1:${API_PORT}/api/v1/health" "api"
build_web
start_web
wait_for_url "http://127.0.0.1:${WEB_PORT}" "web"
write_deploy_info

echo "Synclax is running."
echo "API: http://localhost:${API_PORT}/api/v1/health"
echo "Web: http://localhost:${WEB_PORT}"
echo "Workflow: ${WORKFLOW_FILE}"
echo "Env file: ${ENV_FILE}"
