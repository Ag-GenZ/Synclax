---
outline: deep
---

# Quickstart（第一层：把它跑起来并可观测）

这一页面向“使用者/运维/前端联调”，目标只有一个：**在本地把 Synclax 跑起来，并且保证 API Server 永远有一个稳定可用的健康检查；同时你能启动/停止 Symphony，并通过 Snapshot 为 Web UI 提供实时状态。**

## 你会得到什么

Synclax = Anclax 应用 + Symphony 模块。

启动后你会有两类 HTTP 服务：

1. **主 API Server（Anclax + Fiber）**
   - 对外：`http://localhost:2910/api/v1/...`（开发环境默认）
   - 这个是你 Web UI、OpenAPI Client 以及外部系统应该依赖的入口
2. **Symphony Debug HTTP Server（可选，仅本机 127.0.0.1）**
   - 用于调试 orchestrator：`http://127.0.0.1:<port>/snapshot`
   - 注意：这是给开发者本机看的，不建议给浏览器/公网当作正式 API

::: tip
如果你要做 Web UI，优先使用主 API Server 的 `/api/v1/symphony/snapshot`，不要依赖 debug server。
:::

## 前置条件

- Docker + Docker Compose
- （可选）`jq`：用于格式化输出（没有也不影响）

## 1) 启动开发环境（Docker Compose）

在仓库根目录执行：

```bash
docker compose up
```

它会启动：

- API Server：映射到宿主机 `2910`
- Postgres：映射到宿主机 `5432`

Compose 里设置的关键环境变量（可在 `docker-compose.yaml` 看到）：

- `MYAPP_ANCLAX_PORT=2910`
- `MYAPP_ANCLAX_PG_DSN=postgres://postgres:postgres@db:5432/postgres`

## 2) 验证“主 API Server”健康

这是 Web UI 的第一依赖点：只要服务活着，这个 endpoint 就应该稳定返回 200。

```bash
curl -sS http://localhost:2910/api/v1/health | jq .
```

没有 `jq` 的话：

```bash
curl -sS http://localhost:2910/api/v1/health
```

典型响应（字段可能有 `null`）：

```json
{
  "status": "ok",
  "symphony_running": false,
  "symphony_workflow_path": "/abs/path/to/WORKFLOW.md",
  "symphony_last_error": null
}
```

你可以把它当作：

- **liveness**：API 进程是否可用
- **readiness（弱）**：DB 是否可用不由此 endpoint 强保证（这页先不展开到 DB readiness 设计）

## 3) 准备 WORKFLOW.md（Symphony 的核心配置）

Symphony 通过 `WORKFLOW.md` 读取：

- Linear tracker 配置（要跑起来必须有）
- 轮询间隔、workspace 根目录、hook 脚本
- Codex app-server 命令与超时
- （可选）Symphony debug HTTP server 端口

最简单方式：复制示例文件。

```bash
cp WORKFLOW.md.example WORKFLOW.md
```

然后你至少需要配置（在 `WORKFLOW.md` YAML front matter 中）：

- `tracker.project_slug`
- `tracker.api_key`（推荐直接写 `$LINEAR_API_KEY`）

并在环境变量里提供：

```bash
export LINEAR_API_KEY='你的 Linear API Key'
```

::: warning
如果缺少 `tracker.api_key` 或 `tracker.project_slug`，Symphony 启动后会在 dispatch preflight 阶段拒绝派发任务；你会在 Snapshot 中看到它一直空跑。
:::

## 4) 启动 Symphony（orchestrator）

通过主 API Server 启动（推荐路径）：

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/start | jq .
```

也可以传 JSON body 覆盖部分参数：

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/start \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_path": "./WORKFLOW.md",
    "http_port": 8089
  }' | jq .
```

说明：

- `workflow_path`：Symphony 使用的 `WORKFLOW.md` 路径；服务端会尽量转成绝对路径
- `http_port`：Symphony debug server 端口（绑定 `127.0.0.1`）

::: tip
`http_port` 可以设置为 `-1`：强制关闭 debug server（即使 `WORKFLOW.md` 里配置了 `server.port`）。
这对“你想保证只有主 API 提供对外接口”的场景很有用。
:::

如果你看到 `409 Conflict`，通常意味着：

- Symphony 已经在跑了
- 你尝试用不同的 `workflow_path` 或 `http_port` 再次 start（为了避免配置热切换导致歧义，当前实现会拒绝）

## 5) 读取 Snapshot（Web UI 最关键的数据源）

这个 endpoint 为 Web UI 提供“当前 orchestrator 状态的可视化数据”。

```bash
curl -sS http://localhost:2910/api/v1/symphony/snapshot | jq .
```

你会得到一个对象（概念上）：

- `running`: 当前正在跑的 issue 列表（每个 entry 有 phase、workspace_path、live 信息等）
- `retrying`: 重试队列（due_at、attempt、delay_type、error）
- `codex_totals`: 累计 token 与耗时统计
- `rate_limits`: 最近一次观测到的 rate limit 信息（best-effort）

### phase（阶段）怎么理解

UI 通常只需要把 phase 当作一个枚举字符串展示，例如：

- `PreparingWorkspace`
- `LaunchingAgentProcess`
- `InitializingSession`
- `BuildingPrompt`
- `StreamingTurn`
- `Finishing`
- `Stalled` / `CanceledByReconciliation`（异常/中断类）

::: details 想要更“UI 友好”的状态？
后续第二层（Web UI）文档会建议你在 UI 内部把 phase 映射成更稳定的状态机：

- `idle` / `running` / `retrying` / `blocked` / `error`

然后把 phase 作为 debug 文本显示即可。
:::

## 6) 停止 Symphony

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/stop | jq .
```

停止后再看 health：

```bash
curl -sS http://localhost:2910/api/v1/health | jq .
```

## 7)（可选）验证示例 endpoint：Counter

这是示例 API（不影响 Symphony 主功能）：

```bash
curl -sS http://localhost:2910/api/v1/counter | jq .
curl -sS -X POST http://localhost:2910/api/v1/counter
```

## 常见问题（第一层快速定位）

### Q1：`/api/v1/health` 访问不到

优先检查：

- `docker compose up` 是否还在运行
- 是否端口被占用：`2910`
- 容器日志里是否报错（DB 连接、迁移等）

### Q2：Symphony start 成功，但 snapshot 一直没有 running

常见原因：

- Linear 配置不完整：`LINEAR_API_KEY`、`tracker.project_slug`
- Issue 的 state 不在 `tracker.active_states`
- Issue state 已经在 `terminal_states`
- Issue 是 `Todo` 且存在未完成 blocker（当前实现会跳过）

### Q3：我到底该用哪个 Snapshot？

建议：

- Web UI → 用 `GET /api/v1/symphony/snapshot`
- 本机 debug → 可用 Symphony debug server 的 `/snapshot`（如果你开启了它）

