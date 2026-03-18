---
outline: deep
---

# WORKFLOW.md Reference

`WORKFLOW.md` is Symphony’s primary configuration and prompt template file.

File structure:

```md
---
# YAML front matter config
---

Prompt template (Liquid)
```

If the file does not start with `---`, the entire file is treated as the prompt template and the config is empty.

## Front matter schema

The config is parsed as a YAML map. The effective config is built in `pkg/symphony/config/config.go` and defaults are applied for missing fields.

### `tracker`

```yaml
tracker:
  kind: linear
  endpoint: https://api.linear.app/graphql
  api_key: $LINEAR_API_KEY
  project_slug: your-project-slugId
  active_states: ["Todo", "In Progress"]
  terminal_states: ["Closed", "Cancelled", "Canceled", "Duplicate", "Done"]
  page_size: 50
  timeout_ms: 30000
```

Notes:

- Only `kind: linear` is supported right now.
- `api_key` may use `$ENV_VAR` expansion.

## Dynamic tools (Codex app-server)

During Codex `app-server` sessions, Symphony injects a small set of **dynamic tools** that the
agent can call.

Currently supported:

- `linear_graphql`: execute raw Linear GraphQL queries/mutations using Symphony’s configured
  Linear auth.

Requirements:

- `tracker.kind: linear`
- `tracker.endpoint` points to Linear GraphQL (default is `https://api.linear.app/graphql`)
- `tracker.api_key` is set (or `$LINEAR_API_KEY` is available in the environment)

This is what the repo-level `.codex/skills/linear` skill expects.

### `polling`

```yaml
polling:
  interval_ms: 30000
```

### `workspace`

```yaml
workspace:
  root: ~/.cache/symphony_workspaces
```

Notes:

- `~` is expanded to the user home directory.

### `hooks`

Hook scripts are executed with `bash -lc <script>` in the workspace directory.

```yaml
hooks:
  after_create: |
    git clone git@github.com:your-org/your-repo.git .
  before_run: |
    make gen
  after_run: |
    echo "done"
  before_remove: |
    echo "cleanup"
  timeout_ms: 60000
```

Behavior:

- `after_create` runs only when the workspace is first created; failures are fatal.
- `before_run` runs before Codex starts; failures are fatal.
- `after_run` runs after an attempt; failures are logged and ignored.
- `before_remove` runs before deleting the workspace; failures are logged and ignored.

### `agent`

```yaml
agent:
  max_concurrent_agents: 3
  max_turns: 10
  max_retry_backoff_ms: 300000
  max_concurrent_agents_by_state:
    In Progress: 2
```

Notes:

- `max_concurrent_agents_by_state` keys are normalized to lowercase internally.

### `codex`

```yaml
codex:
  command: codex app-server
  read_timeout_ms: 5000
  turn_timeout_ms: 3600000
  stall_timeout_ms: 300000
  # approval_policy: ...
  # thread_sandbox: ...
  # turn_sandbox_policy: ...
```

Notes:

- `stall_timeout_ms <= 0` disables stall detection.
- `approval_policy`, `thread_sandbox`, `turn_sandbox_policy` are passed through to the Codex app-server protocol as raw JSON/YAML values.

### `server`

```yaml
server:
  port: 8089
```

Enables Symphony’s internal debug HTTP server bound to `127.0.0.1:<port>`.

Debug endpoints:

- `GET /healthz`
- `GET /snapshot` (raw snapshot JSON, for quick inspection)
- `GET /api/v1/state` (snapshot + metadata)
- `POST /api/v1/refresh` (force a poll+dispatch cycle, best-effort)

### `logging`

Optional rotating log file sink for Symphony (process-wide `log` output).

```yaml
logging:
  file: ./log/symphony.log
  max_size_mb: 10
  max_backups: 5
  max_age_days: 0
  compress: false
```

Notes:

- If `logging.file` is omitted/blank, logs stay on stderr/stdout (default).
- Relative paths are resolved from the process working directory.

## Prompt template (Liquid)

The body of `WORKFLOW.md` is rendered with Liquid (`github.com/osteele/liquid`) using **strict variables**.

Available bindings:

- `attempt`: integer attempt number or `null`
- `issue`: an object with keys:
  - `id`, `identifier`, `title`, `state`
  - `description`, `priority`, `branch_name`, `url`
  - `labels`: array of strings
  - `blocked_by`: array of `{ id, identifier, state }`
  - `created_at`, `updated_at` (RFC3339 strings)

Example:

```md
You are working on a Linear issue.

- Title: {{ issue.title }}
- Identifier: {{ issue.identifier }}
- URL: {{ issue.url }}
- Attempt: {{ attempt }}

Follow this repo's contribution guidelines and WORKFLOW policy.
```

::: warning
Because strict variables are enabled, missing variables cause template render errors. Keep the template compatible with the provided bindings.
:::
