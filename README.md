# Synclax

Synclax = **Symphony（编排器）** + **Anclax（OpenAPI-first API Server）**。

它的目标很直接：把 Tracker（当前是 Linear）里的 Issue 变成“可控的后台执行”，并通过一套稳定的 HTTP API（`/health` + `/symphony/*`）把运行状态暴露给 Web UI。

## 适合谁 / 用在什么场景

- 你想把“LLM Agent 执行”变成一个**可启动/可停止/可观测**的后台服务，而不是手动跑脚本。
- 你希望 UI 能看到：当前有哪些任务在跑、跑到哪一步、失败为何重试、token 用量等（通过 `snapshot`）。
- 你需要每个 Issue 对应一个稳定 workspace，并用 hook 把 “clone repo / make gen / test” 这些准备动作自动化。

## 你会得到什么

- 一套 OpenAPI 定义的主 API（见 `api/v1.yaml`，开发环境 base URL：`http://localhost:2910/api/v1`）
- 对 UI 友好的状态接口：
  - `GET /health`
  - `GET /symphony/snapshot`
  - `POST /symphony/start`
  - `POST /symphony/stop`
- 一个核心工作流配置文件：`WORKFLOW.md`（tracker/codex/workspace/hook/polling 等都在这里）

## Quickstart（本地跑起来 + UI 可观测）

更完整的步骤在：`docs/guide/quickstart.md`。

1) 启动开发环境（Docker Compose）

```bash
docker compose up
```

2) 验证 API Server 健康

```bash
curl -sS http://localhost:2910/api/v1/health
```

3) 准备 `WORKFLOW.md`（Symphony 的核心配置）

```bash
cp WORKFLOW.md.example WORKFLOW.md
export LINEAR_API_KEY='你的 Linear API Key'
```

并在 `WORKFLOW.md` 的 YAML front matter 里填写 `tracker.project_slug`。

4) 启动 Symphony 并查看 Snapshot

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/start
curl -sS http://localhost:2910/api/v1/symphony/snapshot
```

5) 停止 Symphony

```bash
curl -sS -X POST http://localhost:2910/api/v1/symphony/stop
```

## 文档（VitePress）

在线阅读入口在 `docs/`，本地启动：

```bash
pnpm -C docs install
pnpm -C docs docs:dev
```

然后访问 VitePress 输出的本地地址（通常是 `http://localhost:5173`）。

## Repo 结构（面向“改哪里”）

- `api/`：OpenAPI spec（`api/v1.yaml`）
- `cmd/`：服务启动入口（`cmd/main.go`）
- `pkg/`：业务与 Symphony 实现（包含 `pkg/handler/*`、`pkg/symphony/*`）
- `sql/`：迁移与查询（sqlc）
- `wire/`：依赖注入（`wire_gen.go` 为生成产物）
- `docs/`：VitePress 文档站
- `web/`：Web UI（如有）

## 安全提示（很重要）

目前 Symphony 的控制类 endpoint 在 `api/v1.yaml` 里标记为 `security: []`（即默认不鉴权）。

如果你要部署到 localhost 以外，请先加好鉴权/鉴权边界，再对外暴露。
