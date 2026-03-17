---
outline: deep
---

# Web UI Integration

The recommended way to build a web UI for Synclax is:

1. Treat `api/v1.yaml` as the **source of truth**
2. Generate your API client using OpenAPI tooling
3. Use the `health` + `symphony/*` endpoints to drive UI state

## OpenAPI spec

- Spec path: `api/v1.yaml`
- Base URL in dev (docker compose): `http://localhost:2910/api/v1`

::: tip
Many generators prefer absolute server URLs. If your generator doesn’t like the relative `servers: /api/v1`, configure the base URL in the client runtime (recommended), or pass a CLI/server override supported by your generator.
:::

## Generate a client (example: openapi-generator)

The exact command depends on your target language. Here are a couple common examples:

### TypeScript (fetch)

```bash
openapi-generator-cli generate \
  -i api/v1.yaml \
  -g typescript-fetch \
  -o ./ui/src/gen/api
```

### TypeScript (axios)

```bash
openapi-generator-cli generate \
  -i api/v1.yaml \
  -g typescript-axios \
  -o ./ui/src/gen/api
```

## UI flow (recommended)

### 1) Check health

Call `GET /health` on app load to confirm the API is reachable.

### 2) Start Symphony

Call `POST /symphony/start` when the user presses “Start”, or automatically when the UI loads.

If you want to allow selecting a workflow file in the UI:

- `workflow_path`: path to a `WORKFLOW.md` file (server must be able to read it)
- `http_port`: optional debug server port (or `-1` to disable)

### 3) Poll snapshot

Call `GET /symphony/snapshot` periodically (e.g. 1–3s) to render:

- Running entries and their phases
- Retry queue entries
- Codex usage totals/rate limits

### 4) Stop Symphony

Call `POST /symphony/stop` to stop background work.

## Security note

The Symphony control endpoints are currently marked `security: []` in `api/v1.yaml` (no auth required). For deployments beyond localhost, you should add authentication/authorization before exposing these endpoints publicly.

