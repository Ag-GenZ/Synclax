---
outline: deep
---

# Quickstart（第一层：把它跑起来并可观测）

这一页面向“使用者/运维/联调”，目标只有一个：**在本地把 Synclax 跑起来，并能通过 Web Console 看到它在工作、并且能启停后台编排。**

## 成功标准（你应该看到什么）

做到下面 3 点，就说明“第一层”已经闭环（不需要你记住任何具体 API URL）：

1. API Server 启动后稳定在线（Console 能连接上）
2. 你能在 Console 里一键启动/停止后台编排（Symphony）
3. Console 能持续展示运行状态（running / retrying / 历史 / 用量 等）

## 你会得到什么

Synclax = Anclax 应用 + Symphony 模块。

启动后你会有两类服务角色：

1. **API Server（后台）**：负责对外提供“控制 + 状态投影”的统一入口（给 Console 用）
2. **Web Console（前台）**：你真正操作与观察系统的界面

::: tip
HTTP API 仍然存在，但它是“Console 的后端契约”。除非你在做二次集成，否则不需要从 Quickstart 开始就记住 URL。
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

## 2) 准备 WORKFLOW.md（Symphony 的核心配置）

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

## 3) 启动 Web Console（前端）

在另一个终端中执行：

```bash
pnpm -C web dev
```

然后打开浏览器访问（开发模式默认端口）：

```bash
http://localhost:3000
```

## 4) 用 Console 启动/停止编排（Symphony）

在 Console 中：

1. 看到服务连接正常（Health）
2. 点击 Start 启动 Symphony
3. 在运行列表里观察 phase、重试队列与历史记录
4. 需要暂停时点击 Stop

## （可选）Advanced：需要用 curl 验证吗？

如果你在排障时需要直接用 `curl` 验证后端 API，请看 [HTTP API Reference（Advanced）](../reference/http-api)。

## 常见问题（第一层快速定位）

### Q1：Console 连接不上后端

优先检查：

- `docker compose up` 是否还在运行
- 是否端口被占用：`2910`
- 容器日志里是否报错（DB 连接、迁移等）

### Q2：Start 成功，但一直没有任务在运行

常见原因：

- Linear 配置不完整：`LINEAR_API_KEY`、`tracker.project_slug`
- Issue 的 state 不在 `tracker.active_states`
- Issue state 已经在 `terminal_states`
- Issue 是 `Todo` 且存在未完成 blocker（当前实现会跳过）

### Q3：我应该相信 Console 展示的状态吗？

建议把 Console 展示的 Snapshot 当作“运行时投影”来理解：

- 它的目标是帮助你**解释现在发生了什么**（running / retrying / completed / totals）
- 它不是强一致审计日志；更偏向 UX 与运维观测

## 下一步（按你要做的事）

- 需要配置详解（WORKFLOW / 环境变量）→ [Configuration](./configuration)
- 需要理解 orchestrator 行为与 phase → [Symphony](./symphony)
- 想先理解理念与架构 → [哲学理念](./philosophy) / [系统架构](./architecture)
