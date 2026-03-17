---
outline: deep
---

# Troubleshooting

## API server not responding

- Confirm docker containers are running: `docker compose ps`
- Check the exposed port (dev default): `http://localhost:2910`
- Verify health endpoint: `GET /api/v1/health`

## Symphony won’t start

`POST /api/v1/symphony/start` may fail if:

- `WORKFLOW.md` is missing or invalid YAML front matter
- Linear config is missing `tracker.api_key` / `tracker.project_slug`
- `codex.command` is empty

Start with a known-good config:

- Copy `WORKFLOW.md.example` → `WORKFLOW.md`
- Set `LINEAR_API_KEY` and `tracker.project_slug`

## Symphony running but no work is dispatched

Common causes:

- Issue state not in `tracker.active_states`
- Issue state is in `tracker.terminal_states`
- Issue is `Todo` but has non-terminal blockers (Symphony will skip it)
- `agent.max_concurrent_agents` is too low and all slots are busy

Use:

- `GET /api/v1/symphony/snapshot` to inspect `running` and `retrying`

## Debug HTTP server disabled unexpectedly

Symphony debug HTTP server is enabled by:

- `server.port` in `WORKFLOW.md`

But can be overridden by Synclax:

- `POST /api/v1/symphony/start` with `http_port: -1` forces it off

## Codex app-server issues

Symptoms:

- `turn timeout`
- `app-server exited`
- `user input required`

Notes:

- Turn timeout is controlled by `codex.turn_timeout_ms` in `WORKFLOW.md`
- If the app-server asks for user input, the current implementation stops the turn with an error (intended for unattended orchestration)

## Codegen drift

If builds break after changing the OpenAPI spec:

1. Run `anclax gen`
2. Update `pkg/handler/handler.go` to satisfy the new `apigen.ServerInterface`
3. Run `go test ./...`

