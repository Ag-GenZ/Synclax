# Synclax

**Synclax** = [**Symphony**](#symphony) + [**Anclax**](#anclax)

Turn tracker issues into controllable AI agent executions — with full observability, automatic lifecycle management, and an OpenAPI-first backend.

It's an open-source implementation of [openai/symphony](https://github.com/openai/symphony), featuring a Go-based API server (Anclax) and a React real-time dashboard, providing another solution for orchestrating LLM agents against your project management tools.

> **Status**: Work in progress

## Overview

Synclax is a self-hosted system that bridges project tracking (e.g. Linear) with autonomous AI agents. It polls your tracker for open issues, spins up isolated agent workspaces, runs an AI agent (Codex) against each issue, and exposes the whole thing through a REST API and a real-time web dashboard.

```
┌─────────────────────────────────┐
│     Web Dashboard (React)       │
│  Running · Retrying · Completed │
└────────────┬────────────────────┘
             │ HTTP /api/v1
┌────────────▼────────────────────┐
│   Anclax (Go / Fiber)           │
│   OpenAPI handlers · PostgreSQL │
│   Async tasks · Auth            │
└────────────┬────────────────────┘
             │ starts / manages
┌────────────▼────────────────────┐
│   Symphony Orchestrator         │
│   Polls tracker → launches      │
│   Codex agents → tracks state   │
└─────────────────────────────────┘
```



## Components

### Symphony

Symphony is the orchestrator engine. It:

- **Polls** a tracker (Linear) for issues matching configured states
- **Creates workspaces** on disk for each issue attempt
- **Launches Codex** (AI agent app-server) with per-issue context
- **Manages lifecycle**: turns, retries, exponential backoff, max concurrency
- **Tracks stats**: token usage, execution time, phase transitions
- **Reloads** `WORKFLOW.md` at runtime without a restart

The entire orchestrator behavior is defined in a single `WORKFLOW.md` file (YAML front matter + Liquid-templated prompt). Symphony reads this file to know which tracker to poll, how to configure workspaces, how many agents to run concurrently, and what prompt to send.

### Anclax

Anclax is the OpenAPI-first Go backend framework powering the HTTP API. `api/v1.yaml` is the single source of truth — handlers, types, and database queries are code-generated from it.

Key capabilities:
- Fiber-based HTTP server with OpenAPI validation
- PostgreSQL via `pgx` + `sqlc` for type-safe queries
- Async task queue with retry and cron support
- Wire-based dependency injection
- Built-in database migrations



## Quick Start

### Prerequisites

- Docker
- Go
- Node.js + pnpm
- Git
- A [Linear](https://linear.app) account + API key
- [Codex](https://github.com/openai/codex) installed on the machine that will execute agents

### Local one-command deploy

For local development on a machine that already has Docker, Go, Node.js,
pnpm, Git, and Codex installed, the fastest path is:

```bash
./scripts/deploy-local.sh
```

This script will:

- install missing `Docker`, `Go`, `Node.js`, `pnpm`, and `Git` with
  Homebrew when available
- start an isolated local PostgreSQL container
- generate a managed `app.yaml`
- generate `WORKFLOW.local.md`
- build and start the Go API on `http://localhost:2910`
- build and start the web dashboard on `http://localhost:3000`

Open:

- Dashboard: `http://localhost:3000`
- API health: `http://localhost:2910/api/v1/health`

If you have not configured Linear yet, edit:

```bash
tmp/local-deploy/synclax.env
```

and set:

```env
LINEAR_API_KEY=your_linear_api_key
LINEAR_PROJECT_SLUG=your_linear_project_slug
```

Then rerun:

```bash
./scripts/deploy-local.sh
```

To stop the local services:

```bash
./scripts/stop-local.sh
```

### Deployment modes

| Compose file | What it runs | Use when |
|---|---|---|
| `docker-compose.yaml` | Anclax API + PostgreSQL + Symphony (dev, hot-reload) | Local development |
| `docker-compose.full.yaml` | Anclax API + PostgreSQL + Symphony (production build) | Self-hosted production |
| `docker-compose.prod.yaml` | Symphony standalone only (no DB) | Lightweight / embedded dashboard only |

### Standalone Symphony (lightweight)

Runs the Symphony orchestrator + an embedded dashboard in a single container, executing Codex on a remote machine over SSH.

> The container uses `ssh` in `BatchMode=yes` — the SSH target must be reachable without password prompts.

1. Configure `WORKFLOW.md` (see below), set `server.port: 8089` and `worker.ssh_hosts`.
2. Set up a **dedicated SSH key** on the host (see [SSH setup](#ssh-setup)).
3. Start:

```bash
LINEAR_API_KEY=lin_api_xxx docker compose -f docker-compose.prod.yaml up -d
```

The dashboard is served at `/` and APIs at `/api/v1`.

### 1. SSH setup

Symphony SSH tunnels into the host to run Codex. Use a **dedicated key** to avoid exposing your main SSH identity to the container.

```bash
# Generate a dedicated keypair
ssh-keygen -t ed25519 -f ~/.ssh/synclax_codex -C "synclax-codex" -N ""

# Authorize it on the SSH target (restrict to codex app-server only)
echo 'command="codex app-server",no-port-forwarding,no-X11-forwarding,no-agent-forwarding' \
  $(cat ~/.ssh/synclax_codex.pub) >> ~/.ssh/authorized_keys

# Test (must succeed without a password prompt)
ssh -i ~/.ssh/synclax_codex -o BatchMode=yes localhost echo ok
```

Then update the compose volume mounts to use the dedicated key instead of the whole `~/.ssh` directory:

```yaml
volumes:
  - ${HOME}/.ssh/synclax_codex:/root/.ssh/id_ed25519:ro
  - ${HOME}/.ssh/synclax_codex.pub:/root/.ssh/id_ed25519.pub:ro
```

> **Why not mount the entire `~/.ssh`?** The container would have access to all your private keys. If the container is compromised, an attacker gains full SSH access to every host your keys are authorized on. A restricted dedicated key limits the blast radius to `codex app-server` only.

### 2. Configure

```bash
cp WORKFLOW.md.example WORKFLOW.md
```

Edit `WORKFLOW.md` YAML front matter — minimum required fields:

```yaml
tracker:
  kind: linear
  api_key: $LINEAR_API_KEY        # or set env var
  project_slug: your-project-id
  active_states: ["Todo", "In Progress"]
  terminal_states: ["Done", "Closed"]

worker:
  ssh_hosts:
    - "user@host.docker.internal:22"  # host machine from inside Docker (macOS)

agent:
  max_concurrent_agents: 3
  max_turns: 10
```

### 3. Start

**Development (hot-reload):**
```bash
LINEAR_API_KEY=lin_api_xxx docker compose up
```

**Production (full stack):**
```bash
LINEAR_API_KEY=lin_api_xxx docker compose -f docker-compose.full.yaml up -d
```

Both start the API server on port `2910` with a PostgreSQL database.

### 4. Use the API

```bash
# Check health
curl http://localhost:2910/api/v1/health

# Start Symphony
curl -X POST http://localhost:2910/api/v1/symphony/start

# Watch what's happening
curl http://localhost:2910/api/v1/symphony/snapshot

# Stop Symphony
curl -X POST http://localhost:2910/api/v1/symphony/stop
```

### 5. Web Dashboard

```bash
cd web
pnpm install
pnpm dev --port 3000   # opens at http://localhost:3000
```

The dashboard shows running agents, retrying issues, completed work, and token usage in real time.



## Configuration

### Environment Variables

| Variable | Description | Default |
||||
| `MYAPP_ANCLAX_PORT` | API server port | `2910` |
| `MYAPP_ANCLAX_PG_DSN` | PostgreSQL DSN | — |
| `MYAPP_SYMPHONY_HTTP_PORT` | Symphony debug HTTP port | — |
| `LINEAR_API_KEY` | Linear API key (if not inlined in WORKFLOW.md) | — |

### App Config (`app.yaml`)

```yaml
anclax:
  port: 2910
  pg:
    dsn: postgres://postgres:postgres@localhost:5432/synclax

symphony:
  workflow_paths:
    - ./WORKFLOW.md
  http_port: 8089
```



## API Reference

| Method | Path | Description |
||||
| `GET` | `/api/v1/health` | Liveness and readiness check |
| `GET` | `/api/v1/symphony/snapshot` | Orchestrator state: running, retrying, completed, stats |
| `POST` | `/api/v1/symphony/start` | Start the orchestrator |
| `POST` | `/api/v1/symphony/stop` | Stop the orchestrator |

The full spec is at [`api/v1.yaml`](api/v1.yaml).



## Development

### Project Structure

```
api/          OpenAPI specs (v1.yaml, tasks.yaml)
cmd/          Entry points (main server, standalone symphony)
pkg/
  symphony/   Orchestrator core (agent, tracker, workspace, codex, ...)
  handler/    HTTP handler implementations
  config/     Configuration loading
  zcore/      App bootstrap, database models
  zgen/       Generated code (apigen, querier, taskgen)
sql/          Migrations and sqlc query definitions
web/          React dashboard (TanStack Start + Tailwind)
docs/         VitePress documentation site
wire/         Wire dependency injection config
```

### Developer Docs

- Developer Guide（扩展点：provider / tracker）：`docs/developer/developer-guide.md`
- WORKFLOW.md Reference（配置契约）：`docs/reference/workflow.md`

### Regenerate Code

After modifying `api/v1.yaml`, `wire/wire.go`, or SQL files:

```bash
anclax gen   # regenerates OpenAPI types, DB querier, Wire graph
```

### Tests

```bash
make ut      # unit tests with race detector and coverage
make test    # e2e tests
```

### Hot Reload (Docker)

```bash
make dev     # docker compose up
make reload  # restart the dev container
make db      # open psql shell
```



## How It Works

1. **Symphony polls** the configured tracker on `polling.interval_ms` intervals.
2. For each unhandled issue, it **creates a workspace** (cloned repo or empty dir) and runs any configured `hooks`.
3. It **launches a Codex app-server** process and sends the issue as a Liquid-rendered prompt.
4. Symphony **runs turns** (tool calls + responses) until the agent finishes or `max_turns` is reached.
5. On failure, the attempt is **retried with exponential backoff**. On success, the issue is marked complete.
6. The Anclax API layer streams the current state via `/symphony/snapshot`, which the web dashboard polls.
7. `WORKFLOW.md` changes are **watched and reloaded** at runtime — no restart needed.



## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, Fiber, pgx, sqlc, Wire, golang-migrate |
| Orchestrator | Custom (Symphony), Liquid templates, JSON-RPC (Codex) |
| Frontend | React 19, TanStack Router/Query, Tailwind CSS 4, Vite |
| Database | PostgreSQL |
| API | OpenAPI 3.0 (oapi-codegen) |
| Infrastructure | Docker Compose |
