---
outline: deep
---

# 配置（第一层：能跑 + 能联调）

Synclax 的配置分两层：

1. **应用层配置**：`app.yaml` + 环境变量 `MYAPP_...`（影响 Anclax 服务与 Synclax 集成）
2. **Symphony 工作流**：`WORKFLOW.md`（影响 orchestrator 的 tracker/codex/workspace/polling 等行为）

这页只讲“第一层需要的部分”，目标是让你在本地或开发环境里**稳定跑起来并能联调 UI**。

## A. 应用层配置：`app.yaml` 与 `MYAPP_...`

Synclax 使用 Anclax 的配置加载逻辑：

- 如果当前工作目录存在 `app.yaml`，会读取它（YAML）
- 所有 `MYAPP_` 开头的环境变量会覆盖 `app.yaml` 的同名配置

> 映射规则：环境变量使用 `_` 表示层级（会被转成嵌套 key）。
> 例如 `MYAPP_ANCLAX_PORT=2910` 会映射为 `anclax.port: 2910`。

### 1) 最小可用：API Server + DB

在开发环境里最核心的是 `anclax` 配置（端口 + PG DSN）。

`app.yaml` 示例：

```yaml
anclax:
  port: 2910
  pg:
    dsn: postgres://postgres:postgres@localhost:5432/synclax
```

环境变量示例（推荐用于本地/CI）：

```bash
export MYAPP_ANCLAX_PORT=2910
export MYAPP_ANCLAX_PG_DSN='postgres://postgres:postgres@localhost:5432/synclax'
```

::: tip
`docker-compose.yaml` 已经为容器内设置了 `MYAPP_ANCLAX_PORT` 与 `MYAPP_ANCLAX_PG_DSN`，你通常不需要再手动设置。
:::

### 2) Synclax 集成：让 API Server 能控制 Symphony

Synclax 在应用层增加了 `symphony` 配置块，支持配置多个 workflow：

```yaml
symphony:
  workflow_paths:
    - ./WORKFLOW.md
  # 或单个：
  # workflow_path: ./WORKFLOW.md
  # http_port: 8089   # 可选：开启 Symphony debug server（仅 127.0.0.1）
```

等价环境变量：

```bash
export MYAPP_SYMPHONY_HTTP_PORT=8089
```

字段含义：

- `workflow_paths`： Symphony 加载的 `WORKFLOW.md` 路径列表（支持多 workflow 并发）
- `workflow_path`：单个 workflow 的简写，与 `workflow_paths: [./WORKFLOW.md]` 等价
- `http_port`：Symphony debug HTTP server 端口（可选）

::: warning
Web UI 不建议依赖 debug server。UI 应该通过主 API Server 的 `/api/v1/symphony/snapshot` 获取状态。
:::

### 3) SSH 安全配置

Symphony 通过 SSH 在宿主机上运行 Codex。**不要**把整个 `~/.ssh` 挂进容器，过盅导致所有私钥暴露。

请使用**专用密钥 + 命令限制**：

```bash
# 1. 生成专用密钥
ssh-keygen -t ed25519 -f ~/.ssh/synclax_codex -C "synclax-codex" -N ""

# 2. 把公钥加到 authorized_keys，限制只能执行 codex app-server
echo 'command="codex app-server",no-port-forwarding,no-X11-forwarding,no-agent-forwarding' \
  $(cat ~/.ssh/synclax_codex.pub) >> ~/.ssh/authorized_keys

# 3. 验证（必须不提示密码）
ssh -i ~/.ssh/synclax_codex -o BatchMode=yes localhost echo ok
```

然后将 Compose 文件里的 `${HOME}/.ssh:/root/.ssh:ro` 改为：

```yaml
volumes:
  - ${HOME}/.ssh/synclax_codex:/root/.ssh/id_ed25519:ro
  - ${HOME}/.ssh/synclax_codex.pub:/root/.ssh/id_ed25519.pub:ro
```

即使容器被攻击者控制，也只能运行 `codex app-server`，无法获得完整 shell。

## B. Symphony 工作流：`WORKFLOW.md`

`WORKFLOW.md` 是 Symphony 的“真配置”，文件结构为：

```md
---
# YAML front matter（配置）
---

Prompt template（Liquid）
```

你可以先使用仓库自带模板：

```bash
cp WORKFLOW.md.example WORKFLOW.md
```

### 最小可用字段（跑起来必须有）

要让 Symphony 真正能派发任务（从 Linear 拉 issue 并运行 agent），你至少需要：

- `tracker.project_slug`
- `tracker.api_key`（推荐写 `$LINEAR_API_KEY`）
- `codex.command`（默认是 `codex app-server`，不填也会有默认值，但你要确保运行环境里存在该命令）

并设置：

```bash
export LINEAR_API_KEY='你的 Linear API Key'
```

::: tip
更完整的字段解释与模板绑定变量见：`/reference/workflow`（后续我们也会把它完全中文化扩写）。
:::

## C. 覆盖策略：UI/联调时怎么切换 Workflow 与端口

主 API Server 暴露了：

- `POST /api/v1/symphony/start`

它允许你在启动时传 JSON 覆盖：

- `workflow_path`
- `http_port`

示例：

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/start \
  -H "Content-Type: application/json" \
  -d '{"workflow_path":"./WORKFLOW.md","http_port":8089}'
```

重要语义：

- `http_port: -1`：强制关闭 Symphony debug server（即使 `WORKFLOW.md` 设置了 `server.port`）
- 如果 Symphony 已经在运行，再传“不同的” `workflow_path` 或 `http_port`，当前实现会返回 `409 Conflict`（避免运行中热切换造成歧义）

