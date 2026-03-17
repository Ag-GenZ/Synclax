---
outline: deep
---

# HTTP API Reference

All API endpoints are served under the base path: `/api/v1`.

The OpenAPI spec is `api/v1.yaml`.

## Health

### `GET /health`

Liveness/readiness endpoint for the API server plus Symphony status.

Example:

```bash
curl -sS http://localhost:2910/api/v1/health | jq .
```

Response (200):

- `status`: `"ok"`
- `symphony_running`: whether the orchestrator goroutine is running
- `symphony_workflow_path`: resolved workflow path (if configured)
- `symphony_last_error`: last orchestrator exit error, if any

## Symphony control

### `GET /symphony/snapshot`

Returns the current orchestrator snapshot for UI rendering.

```bash
curl -sS http://localhost:2910/api/v1/symphony/snapshot | jq .
```

This returns a JSON object with:

- `running`: array of running entries
- `retrying`: array of retry entries
- `codex_totals`: aggregated token/seconds counters
- `rate_limits`: last observed rate limit payload (best-effort)

### `POST /symphony/start`

Starts Symphony if not already running. Idempotent.

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/start | jq .
```

Optional JSON body:

```json
{
  "workflow_path": "./WORKFLOW.md",
  "http_port": 8089
}
```

Notes:

- `http_port: -1` disables Symphony’s debug HTTP server even if `WORKFLOW.md` sets `server.port`.
- If Symphony is already running and you pass a different `workflow_path` or `http_port`, the server returns `409 Conflict`.

### `POST /symphony/stop`

Stops Symphony if running. Idempotent.

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/stop | jq .
```

## Demo endpoint

### `GET /counter`

Returns the current counter state.

```bash
curl -sS http://localhost:2910/api/v1/counter | jq .
```

### `POST /counter`

Enqueues a task to increment the counter.

```bash
curl -sS -X POST http://localhost:2910/api/v1/counter
```

