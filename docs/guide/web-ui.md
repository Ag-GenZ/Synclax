---
outline: deep
---

# Web UI（产品/前端集成指南）

这一页面向“产品 / 前端 / 联调”：目标是让你用**最小成本**做出一个能用的 Synclax 控制台：

- 能看到服务是否在线（health）
- 能一键启动/停止后台编排（start/stop）
- 能持续展示运行中任务、阶段（phase）、重试队列与用量（snapshot）

推荐原则（先记住这 3 条）：

1. **把 `api/v1.yaml` 当作唯一真相**（UI 客户端建议由 OpenAPI 生成）
2. **UI 只依赖主 API Server**：`/api/v1/*`（不要依赖 127.0.0.1 的 debug server）
3. **以 Snapshot 驱动 UI 状态机**：running / retrying / idle / error

## OpenAPI 与 Base URL

- Spec 路径：`api/v1.yaml`
- 本地开发（docker compose）Base URL：`http://localhost:2910/api/v1`

::: tip
`api/v1.yaml` 里 `servers: /api/v1` 是相对路径；如果你的生成器更偏好绝对 URL，建议在客户端运行时配置 base URL（更稳），或使用生成器的 server override 参数。
:::

## 生成客户端（示例：openapi-generator）

命令会随语言/框架不同而变化，下面给两个常见 TypeScript 示例：

### TypeScript（fetch）

```bash
openapi-generator-cli generate \
  -i api/v1.yaml \
  -g typescript-fetch \
  -o ./ui/src/gen/api
```

### TypeScript（axios）

```bash
openapi-generator-cli generate \
  -i api/v1.yaml \
  -g typescript-axios \
  -o ./ui/src/gen/api
```

## UI 数据流（推荐落地方案）

你可以把 UI 拆成 3 个“永远成立的模块”，方便产品与研发对齐：

### 1) 连接状态：Health

App 启动时调用 `GET /health`：

- 成功 → 显示“服务在线”，并展示 `symphony_running`
- 失败 → 显示“无法连接服务”，并停止后续轮询（或进入重试）

### 2) 控制：Start / Stop

按钮行为建议：

- “Start” → `POST /symphony/start`（幂等；重复点击应该安全）
- “Stop” → `POST /symphony/stop`（幂等）

如果你希望在 UI 里切换工作流文件（常用于联调/多环境）：

- `workflow_path`：`WORKFLOW.md` 路径（服务端必须能读到）
- `http_port`：可选 debug server 端口（或 `-1` 强制关闭）

### 3) 运行时：Snapshot（UI 的主数据源）

以 `GET /symphony/snapshot` 作为 UI 轮询源（建议 `1s ~ 3s`）：

- `running[]`：当前运行中的 issue 与 `phase`（进度条/状态标签）
- `retrying[]`：重试队列（显示 due_at、attempt、error）
- `codex_totals`：累计用量（产品/运营会关心）
- `rate_limits`：最近一次观测到的限流信息（best-effort）

::: tip
UI 里不要把 `phase` 当作“稳定产品状态”。更推荐你把它映射成更稳定的状态机：`idle / running / retrying / blocked / error`，然后把 `phase` 作为 debug 文本展示。
:::

## 安全提示（上线前必读）

目前 Symphony 控制类 endpoint 在 `api/v1.yaml` 中标记为 `security: []`（默认不鉴权）。

如果你的 UI 不只跑在 localhost，请先补齐鉴权/授权（例如：反向代理鉴权、API token、或在 OpenAPI 里加安全方案）再对外暴露。
