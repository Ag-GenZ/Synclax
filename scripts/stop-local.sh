#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/tmp/local-deploy"
PID_DIR="$RUNTIME_DIR/pids"

POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-synclax-local-postgres}"

stop_pid_file() {
  local pid_file="$1"
  if [[ ! -f "$pid_file" ]]; then
    return 0
  fi

  local pid
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  if [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
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
  fi

  rm -f "$pid_file"
}

stop_pid_file "$PID_DIR/api.pid"
stop_pid_file "$PID_DIR/web.pid"

if command -v pkill >/dev/null 2>&1; then
  pkill -f '/tmp/local-deploy/bin/anclax-local' >/dev/null 2>&1 || true
  pkill -f 'vite-plus-core/dist/vite/node/cli.js preview --host 0.0.0.0 --port 3000' >/dev/null 2>&1 || true
  pkill -f 'vite-plus/bin/vp preview -- --host 0.0.0.0 --port 3000' >/dev/null 2>&1 || true
fi

if command -v docker >/dev/null 2>&1; then
  if docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER"; then
    docker stop "$POSTGRES_CONTAINER" >/dev/null
  fi
fi

echo "Synclax local services stopped."
