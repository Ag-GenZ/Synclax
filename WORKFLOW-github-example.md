---
tracker:
  kind: github
  endpoint: https://api.github.com/graphql
  token: $GITHUB_TOKEN
  project_owner: your-org-or-user
  project_number: 1
  repository: your-org-or-user/your-repo
  state_field: Status
  active_states: ["Todo", "In Progress"]
  terminal_states: ["Done", "Closed", "Cancelled", "Canceled", "Duplicate"]

polling:
  interval_ms: 30000

workspace:
  root: ~/.cache/symphony_workspaces

hooks:
  # after_create: |
  #   git clone git@github.com:your-org/your-repo.git .
  # before_run: |
  #   make gen
  timeout_ms: 60000

agent:
  max_concurrent_agents: 3
  max_turns: 10
  max_retry_backoff_ms: 300000
  max_concurrent_agents_by_state:
    In Progress: 2

worker:
  # Optional. When set, Symphony will run workspaces/hooks/Codex over SSH on the selected host.
  # Formats: "user@host:port", "host:port", "user@host", "host", "[::1]:2222"
  ssh_hosts:
    - "user@host.docker.internal:22"
  # Optional. 0 => auto (max_concurrent_agents / len(ssh_hosts))
  max_concurrent_agents_per_host: 0

codex:
  command: codex app-server
  read_timeout_ms: 5000
  turn_timeout_ms: 3600000
  stall_timeout_ms: 300000

server:
  port: 8089
---

You are working on a GitHub Project v2 issue.

- Title: {{ issue.title }}
- Identifier: {{ issue.identifier }}
- URL: {{ issue.url }}
- Attempt: {{ attempt }}

Follow this repo's contribution guidelines and WORKFLOW policy.
