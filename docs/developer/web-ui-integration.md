---
outline: deep
---

# Web UI Integration（Advanced）

> 本页是“集成/开发者视角”的内容；如果你只是想使用现成的 Console，请优先读 Guide 里的 Web Console 页面。

这一页面向“产品 / 前端 / 联调”：目标是让你用最小成本做出一个能用的 Synclax 控制台：

- 能看到服务是否在线（health）
- 能一键启动/停止后台编排（start/stop）
- 能持续展示运行中任务、阶段（phase）、重试队列与用量（snapshot）

推荐原则：

1. 把 OpenAPI 当作唯一真相（客户端建议由 OpenAPI 生成）
2. UI 只依赖主 API Server（不要依赖 127.0.0.1 的 debug server）
3. 以 Snapshot 驱动 UI 状态机：running / retrying / idle / error

## OpenAPI 与 Base URL（开发环境示例）

- Spec 路径：`api/v1.yaml`
- 本地开发（docker compose）Base URL：`http://localhost:2910/api/v1`

## 生成客户端（示例：openapi-generator）

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

### 1) 连接状态：Health

App 启动时调用 `GET /health`：

- 成功 → 显示“服务在线”，并展示 `symphony_running`
- 失败 → 显示“无法连接服务”，并停止后续轮询（或进入重试）

### 2) 控制：Start / Stop

- “Start” → `POST /symphony/start`（幂等）
- “Stop” → `POST /symphony/stop`（幂等）

### 3) 运行时：Snapshot

以 `GET /symphony/snapshot` 作为 UI 轮询源（建议 `1s ~ 3s`）：

- `running[]`：当前运行中的 issue 与 `phase`
- `retrying[]`：重试队列
- `completed[]`：最近结束的 attempt 历史
- `codex_totals`：累计用量
- `rate_limits`：最近一次观测到的限流信息（best-effort）

## 安全提示（上线前必读）

目前 Symphony 控制类 endpoint 在 `api/v1.yaml` 中标记为 `security: []`（默认不鉴权）。
如果你的 UI 不只跑在 localhost，请先补齐鉴权/授权再对外暴露。

