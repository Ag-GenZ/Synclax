---
outline: deep
---

# Architecture

Synclax combines two major parts:

1. **Anclax application** (HTTP API + tasks + DB model)
2. **Symphony orchestrator** (tracker → workspace → Codex)

## High-level diagram

```text
          ┌───────────────────────────┐
          │        Web UI             │
          │  (OpenAPI client)         │
          └───────────┬───────────────┘
                      │ HTTP (/api/v1)
                      ▼
┌────────────────────────────────────────────────┐
│               Anclax + Fiber                   │
│  - OpenAPI handlers (pkg/handler)              │
│  - Task handlers (pkg/asynctask)               │
│  - DB model/sqlc (pkg/zcore/model + zgen)      │
│  - Symphony control API                        │
└───────────────┬────────────────────────────────┘
                │ starts/stops
                ▼
┌────────────────────────────────────────────────┐
│                Symphony Orchestrator           │
│  - runtime.Manager (reload WORKFLOW.md)        │
│  - tracker.Client (Linear)                     │
│  - workspace.Manager (+ hooks)                 │
│  - codex.AppServer (Codex app-server protocol) │
└────────────────────────────────────────────────┘
```

## Code map

### Anclax side

- OpenAPI spec: `api/v1.yaml`
- Generated OpenAPI types/handlers: `pkg/zgen/apigen/spec_gen.go` (generated)
- HTTP handlers implementation: `pkg/handler/handler.go`
- Wire graph: `wire/wire.go` + `wire/wire_gen.go` (generated)
- App bootstrap: `cmd/main.go`

### Symphony side

- Workflow parsing + watcher: `pkg/symphony/workflow/*`
- Effective config (defaults/validation): `pkg/symphony/config/config.go`
- Prompt rendering (Liquid): `pkg/symphony/template/renderer.go`
- Runtime reload manager: `pkg/symphony/runtime/*`
- Orchestrator loop + scheduler: `pkg/symphony/orchestrator/orchestrator.go`
- Tracker adapter: `pkg/symphony/tracker/linear/*`
- Workspace manager + hooks: `pkg/symphony/workspace/manager.go`
- Codex app-server RPC client: `pkg/symphony/codex/appserver.go`
- Orchestrator lifecycle wrapper: `pkg/symphony/control/manager.go`

## Data flow details

### 1) Load workflow and runtime

- `runtime.Manager` watches `WORKFLOW.md` and keeps the last known-good runtime.
- If the initial workflow is invalid, startup fails.

### 2) Poll and dispatch

- Orchestrator polls tracker candidates every `polling.interval_ms`.
- It only dispatches issues in `tracker.active_states` and not in `terminal_states`.
- For `Todo` issues, all blockers must be terminal (otherwise it won’t dispatch).

### 3) Run an attempt

Per issue attempt:

- Create workspace (with optional hooks)
- Start a Codex app-server session rooted in the workspace
- Run up to `agent.max_turns` turns
- Refresh issue state from tracker between turns

### 4) Retry

- On success: schedule “continuation” retry if issue still active
- On error: schedule exponential backoff retry (capped)

