---
outline: deep
---

# Code Generation (Anclax)

Synclax follows the Anclax workflow: specs and SQL are sources of truth, generated code is the contract between layers.

## What’s generated

Generation is controlled by `anclax.yaml`:

- OpenAPI → `pkg/zgen/apigen/*`
- Tasks → `pkg/zgen/taskgen/*`
- SQLC → `pkg/zgen/querier/*`
- Wire DI → `wire/wire_gen.go`

## When to run `anclax gen`

Run `anclax gen` after changing any of:

- `api/v1.yaml`
- `api/tasks.yaml`
- `sql/migrations/*`
- `sql/queries/*`
- `wire/wire.go`

```bash
anclax gen
```

::: warning
Don’t hand-edit generated files under `pkg/zgen/*` or `wire/wire_gen.go`. They will be overwritten.
:::

## OpenAPI conventions used here

- Use schemas in `components/schemas` and reference them (avoid inline schemas except simple arrays)
- Use clear `operationId` values (they map to generated handler method names)
- If an endpoint doesn’t require auth, set `security: []` explicitly

## Updating the API for the Web UI

The Web UI should rely on the OpenAPI-generated client and the stable endpoints:

- `GET /health`
- `GET /symphony/snapshot`
- `POST /symphony/start`
- `POST /symphony/stop`

After editing `api/v1.yaml`, regenerate and then implement any new `apigen.ServerInterface` methods in `pkg/handler/handler.go`.

## Regenerating Wire

Wire is regenerated as part of `anclax gen`. If you add a new provider:

1. Add it to `wire/wire.go`
2. Run `anclax gen`
3. Fix any build errors in `wire/wire_gen.go` consumers (usually handler constructor signatures)

