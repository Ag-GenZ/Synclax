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

- Docker and Docker Compose
- A [Linear](https://linear.app) account + API key
- [Codex](https://github.com/openai/codex) installed and on `$PATH`

### 1. Configure

```bash
cp WORKFLOW.md.example WORKFLOW.md
```

Edit `WORKFLOW.md` and fill in the YAML front matter:

```yaml

tracker:
  kind: linear
  api_key: $LINEAR_API_KEY        # or set env var
  project_slug: your-project-id
  active_states: ["Todo", "In Progress"]
  terminal_states: ["Done", "Closed"]

agent:
  max_concurrent_agents: 3
  max_turns: 10

codex:
  command: codex app-server

You are working on the following Linear issue: {{ issue.title }}
...
```

### 2. Start

```bash
docker compose up
```

This starts the API server on port `2910` and a PostgreSQL database.

### 3. Use the API

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

### 4. Web Dashboard

```bash
cd web
pnpm install
pnpm run dev   # opens at http://localhost:3000
```

The dashboard shows running agents, retrying issues, completed work, and token usage in real time.



## Configuration

### Environment Variables

| Variable | Description | Default |
||||
| `MYAPP_ANCLAX_PORT` | API server port | `2910` |
| `MYAPP_ANCLAX_PG_DSN` | PostgreSQL DSN | — |
| `MYAPP_SYMPHONY_WORKFLOW_PATH` | Path to `WORKFLOW.md` | `./WORKFLOW.md` |
| `MYAPP_SYMPHONY_HTTP_PORT` | Symphony debug HTTP port | — |
| `LINEAR_API_KEY` | Linear API key (if not inlined in WORKFLOW.md) | — |

### App Config (`app.yaml`)

```yaml
anclax:
  port: 2910
  pg:
    dsn: postgres://postgres:postgres@localhost:5432/postgres

symphony:
  workflow_path: ./WORKFLOW.md
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
