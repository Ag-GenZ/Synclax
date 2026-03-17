---
outline: deep
---

# Symphony SPEC（系统性解析）

这页是对上游 `openai/symphony` 的 `SPEC.md` 的“结构化解读”，目的是回答两个问题：

1. **Symphony 到底是什么？**（它解决哪些操作性问题、边界在哪里）
2. **我应该怎么用？**（需要哪些输入、有哪些可观测输出、如何跟 UI/运维对接）

> 说明：此页以“规范”作为主线，但会在关键处补充 Synclax 当前实现的落点，帮助你把规范映射到代码与接口。

## 1) Symphony 是什么：问题定义与边界

SPEC 把 Symphony 定义为一个常驻服务（daemon-like），持续从 Issue Tracker 读取工作、为每个 issue 创建隔离的 workspace，然后在 workspace 中运行 coding agent。它要解决的核心是**可操作性（operability）**：让“跑 agent”从一次性脚本变成可重复、可并发、可观测的服务。 citeturn4view0

同时 SPEC 明确了边界：

- Symphony 是 **scheduler / runner / tracker reader**，不是“业务逻辑引擎”
- “写回 ticket 的动作”（例如改状态、评论、贴 PR 链接）通常由 agent 在工具环境里完成 citeturn4view0
- SPEC 不强制某一种 approval/sandbox 策略，要求实现方明确说明自己的信任与安全姿态 citeturn4view0

对你来说，这意味着：

- 你要把团队流程（例如什么时候改状态、什么时候停在 `Human Review`）写到 `WORKFLOW.md` 的 prompt 和工具使用策略里
- Symphony 主要提供“把 agent 跑起来并运营起来”的壳

## 2) 总体目标（为什么这些能力是“必须的”）

SPEC 的目标集合可以被理解为一套最小可运营特性：

- 固定 cadence 轮询并发派发（bounded concurrency）
- 单一权威的编排状态（running/retry/reconcile）
- 可重复的 per-issue workspace，且跨运行保留
- issue 状态变化时能停止不再 eligible 的运行
- transient failure 的指数退避重试
- 从 repo 的 `WORKFLOW.md` 加载运行时行为
- 暴露可观测性（至少结构化日志）
- **支持重启恢复且不依赖持久数据库** citeturn4view0

Synclax 的 UI 方案就是围绕这些“最小可运营”能力搭建：

- `/api/v1/health` 做稳定健康检查
- `/api/v1/symphony/snapshot` 做 UI 状态源
- `/api/v1/symphony/start|stop` 做控制面

## 3) 组件分层（把它当作一个可移植的架构）

SPEC 推荐把系统拆成这些组件（有利于移植与测试）： citeturn4view0

- `Workflow Loader`：读 `WORKFLOW.md`，解析 front matter + prompt body
- `Config Layer`：做 typed getter、默认值、env token、校验
- `Issue Tracker Client`：拉候选 issue、按 id 刷新状态、startup cleanup
- `Orchestrator`：轮询、内存态、派发/重试/停止/释放、metrics
- `Workspace Manager`：workspace 映射与 hooks、terminal cleanup
- `Agent Runner`：构建 prompt、拉起 app-server、流式回传事件
- `Status Surface`（可选）：人类可读状态面板/终端
- `Logging`：结构化日志

Synclax 的实现基本一一对应（代码入口见 `docs/developer/architecture`）。

## 4) 核心领域模型（UI 与可观测的“语义契约”）

SPEC 给了一个“规范化 issue”模型，以及 orchestration runtime 状态： citeturn4view0

你做 UI 时可以把它当作“稳定结构”理解：

- Issue：`id / identifier / title / description / priority / state / url / labels / blocked_by / created_at / updated_at`
  - `priority`：数值越小优先级越高
  - `labels`：建议归一为小写
  - `blocked_by`：每个 blocker ref 包含 `id/identifier/state/created_at/updated_at`
- Live session：thread/turn id、pid、最近 event、token usage、turn_count
- Retry entry：attempt、due_at、error
- Orchestrator runtime state：running map、claimed set、retry map、codex_totals、rate_limits

Synclax 的 `/api/v1/symphony/snapshot` 就是把这些东西序列化出来，供 UI 消费。

## 5) `WORKFLOW.md` 规范（Repo Contract）

### 5.1 发现与路径优先级

SPEC 的路径优先级：

1. 显式设置（例如 CLI 传入）
2. 默认：进程 cwd 下的 `WORKFLOW.md` citeturn4view0

### 5.2 文件格式

- 如果以 `---` 开头：解析 YAML front matter，遇到下一个 `---` 结束，然后剩余部分是 prompt body
- 如果没有 front matter：整文件就是 prompt body，config 视为空 map
- front matter 必须 decode 成 map，否则是错误 citeturn4view0

### 5.3 front matter 顶层 key

规范顶层 key：`tracker/polling/workspace/hooks/agent/codex`，未知 key 需要忽略以保持 forward compatible。 citeturn4view0

同时允许扩展 key，例如常见的 `server.port` 用于开启可选 HTTP server。 citeturn4view0

Synclax 在第一层里建议把“团队流程与运行时策略”尽量收敛在 `WORKFLOW.md` 中，这样它能随 repo 版本化演进。

### 5.4 Prompt 模板契约

SPEC 建议：

- 使用严格模板引擎（Liquid 语义足够）
- 未知变量/未知 filter 必须渲染失败 citeturn4view0

模板输入变量：

- `issue`（包含规范化字段）
- `attempt`（首次为 null，后续为整数） citeturn4view0

这就是为什么 UI/运维经常需要关注 template 错误：template 错误通常是“单次 attempt 失败”，而不是整个 orchestrator 失效（具体 gating 行为看实现策略）。

## 6) 调度、重试与可观测：你如何“运营它”

SPEC 的落点是可运营性，因此你需要有三个能力：

1. **看得到**：health、snapshot、日志
2. **控得住**：并发上限、按状态并发、退避上限
3. **可恢复**：重启后不需要 DB 也能继续工作（通过 tracker + workspace）

在 Synclax 的实践里：

- UI 主要看 `/api/v1/symphony/snapshot`
- 排查主要看日志 + snapshot 中的 `retrying` 和 `running.phase`
- 运行策略主要调 `WORKFLOW.md` 的 `polling/agent/codex/hooks`

## 7) “怎么用”的最短路径（对应第一层）

建议你按这个顺序上手：

1. 先让主 API Server 健康：`GET /api/v1/health`
2. 配好 `WORKFLOW.md` 的 Linear 与 Codex
3. `POST /api/v1/symphony/start`
4. UI 轮询 `GET /api/v1/symphony/snapshot` 并渲染
5. 需要时再启用 debug server（`server.port` 或 start body 的 `http_port`）

具体步骤见：

- `docs/guide/quickstart`
- `docs/guide/symphony`

