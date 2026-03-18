---
outline: deep
---

# Developer Guide（Extending Synclax）

这份文档面向 **Synclax 代码库开发者**，目标是把“扩展点”讲清楚，做到：

- **人类友好**：按步骤做就能落地（包含命令、路径、检查点）。
- **LLM/Agent 友好**：可直接复制 Prompt / Checklist，让 Agent 自己按约定改代码。

> 术语提醒：本文的 **Provider** 指 `pkg/symphony/provider` 下的“执行引擎适配器”，不要和 Wire / DI 语境里的“provider set”混淆。

## 我们采用的“市面常见开发指南结构”

这份指南在写法上刻意对齐常见的扩展/插件/适配器类文档结构（便于人类与 LLM 扫描）：

1. **先讲边界与术语**：明确“你要实现的接口”与“你要改的注册点”。
2. **给最小闭环步骤**：Step 0/1/2… 每一步都能落盘到具体路径/命令。
3. **提供骨架模板**：直接 copy/paste 的代码框架与 checklist。
4. **写清设计约束**：哪些是稳定契约（`kind`、配置），哪些可演进（params、工具注入）。

## 你需要先理解的 3 个概念

### 1) Tracker（问题来源）

Tracker 负责从外部系统（例如 Linear）读取 issue，并把它们映射成内部统一的数据结构 `domain.Issue`。

- 接口：`pkg/symphony/tracker/tracker.go`
- 现有实现：`pkg/symphony/tracker/linear/*`
- 创建入口：`pkg/symphony/orchestrator/orchestrator.go` 的 `newTracker(...)`

### 2) Provider（执行引擎）

Provider 负责“跑一轮 turn”：启动 session、流式接收事件、最后给出结果（最后一条消息、token 用量等）。

- 接口：`pkg/symphony/provider/provider.go`
- 工厂：`pkg/symphony/provider/factory.go`（根据 `provider.kind` 选择实现）
- 现有实现：`pkg/symphony/provider/codex_adapter.go`（`kind: codex`）

### 3) WORKFLOW.md（契约与配置）

`WORKFLOW.md` 的 YAML front matter 决定：

- `tracker.kind` 用哪个 tracker
- `provider.kind` 用哪个 provider
- workspace/hooks/agent/codex 等运行参数

解析代码：`pkg/symphony/config/config.go`  
参考文档：`docs/reference/workflow.md`

## 设计原则（扩展时别破坏这些）

1. **`kind` 是对用户/配置的稳定契约**：一旦发布，尽量不要改含义（除非提供迁移/兼容）。
2. **适配器边界清晰**：外部 SDK 类型不要泄漏到上层（统一映射到 `domain.Issue` / provider 的结构）。
3. **可取消、可超时、可关闭**：所有网络调用尊重 `context.Context`；`Session.Close()` 要幂等。
4. **错误可分层归类**：Provider 建议包装为 `*provider.Error`（分类见 `pkg/symphony/provider/errors.go`）；Tracker 用 `*tracker.Error`。
5. **默认值与验证要一致**：新增 kind 后，记得同步更新 *配置默认/校验* 与 *文档 reference*。

---

## 手把手：新增 Provider

本节以“新增一个 `provider.kind: <your_kind>`”为目标，按最小闭环给出步骤。

### Step 0：决定 provider 的边界与依赖

先回答 4 个问题（这决定你的实现形态）：

1. session 是“本地子进程”（像 `codex app-server`）还是“远程 HTTP API”？
2. 需要哪些 auth/config（token、endpoint、timeout）？放在哪里（`provider:` stanza 还是复用现有的 `tracker.params`）？
3. 是否需要给 agent 注入“动态工具”（类似 `linear_graphql`）？如果需要，工具的鉴权从哪里来？
4. 失败/超时/退出如何归类（`provider.ErrTurnTimeout` / `provider.ErrProcessExit` 等）？

### Step 1：实现 Provider 接口

接口定义在 `pkg/symphony/provider/provider.go`：

- `StartSession(ctx, workspacePath) (Session, error)`
- `RunTurn(ctx, session, workspacePath, title, inputText, onEvent) (*TurnResult, error)`

推荐落盘方式（与现有 `codex_adapter.go` 风格一致）：

- 在 `pkg/symphony/provider/` 下新增一个文件：`<your_kind>_adapter.go`
- 实现一个 `type <yourKind>Provider struct { ... }`
- 实现一个 `type <yourKind>Session struct { ... }`

最小骨架示例（只展示结构，不含具体实现细节）：

```go
package provider

import "context"

type yourKindProvider struct {
  // TODO: clients/options
}

func newYourKindProvider(/* deps */) Provider {
  return &yourKindProvider{}
}

type yourKindSession struct {
  id string
}

func (s *yourKindSession) SessionID() string { return s.id }
func (s *yourKindSession) PID() *int         { return nil } // 非进程型实现可返回 nil
func (s *yourKindSession) Close() error      { return nil } // 需要幂等

func (p *yourKindProvider) StartSession(ctx context.Context, workspacePath string) (Session, error) {
  // TODO
  return &yourKindSession{id: "..."}, nil
}

func (p *yourKindProvider) RunTurn(
  ctx context.Context,
  session Session,
  workspacePath, title, inputText string,
  onEvent func(event string, payload map[string]any),
) (*TurnResult, error) {
  // TODO: call onEvent("item/message", map[string]any{...}) etc if you support streaming
  return &TurnResult{LastMessage: "..."}, nil
}
```

### Step 2：在 provider 工厂注册 kind

编辑 `pkg/symphony/provider/factory.go`：

- 在 `switch cfg.Provider.Kind` 里新增你的 `case "<your_kind>"`.
- 构建 provider 所需依赖（client/options），返回 `Provider`.
- 如果你的 provider 需要额外配置，**同步扩展** `pkg/symphony/config/config.go` 的 `ProviderConfig` 与解析逻辑（见 Step 3）。

### Step 3：补齐配置解析 / 校验（必要时）

目前 `ProviderConfig` 只有：

```go
type ProviderConfig struct { Kind string }
```

如果你的 provider 需要额外参数，建议仿照 `TrackerConfig.Params` 的模式加入：

- `ProviderConfig.Params map[string]any`
- `FromWorkflowConfig(...)` 里读取 `provider:` 的 raw map（或新增 `provider.params:`）
- `resolveEnvironment(...)` 做 `$ENV_VAR` 替换（与 tracker 参数一致）
- `validate(...)` 支持新 kind，或给出缺参错误

> 设计建议：`provider.kind` 应尽量稳定；参数扩展尽量通过 `provider.params.*`（或 `provider.*` 的可选字段）向后兼容。

### Step 4：写一个最小 smoke test

参考现有测试：`pkg/symphony/orchestrator/orchestrator_provider_smoke_test.go`

目标是验证：

- 不填 `provider.kind` 的默认行为不变
- 显式指定 `provider.kind: <your_kind>` 时能成功 `applyRuntimeLocked(...)`（至少能初始化依赖）

### Step 5：更新文档 reference

当你真正引入新 provider kind 后，记得同步更新：

- `docs/reference/workflow.md`：新增/更新 `provider:` stanza 说明
- `docs/developer/developer-guide.md`（本文）：补充你的 provider 特定注意事项

---

## 手把手：新增 Tracker

Tracker 的目标是把外部系统的 issue 映射为 `domain.Issue`，并提供 3 个查询能力：

- `FetchCandidateIssues(ctx)`：按 WORKFLOW 里的条件拉取候选 issues（通常是 active states）
- `FetchIssuesByStates(ctx, stateNames)`：按状态名过滤拉取
- `FetchIssueStatesByIDs(ctx, issueIDs)`：刷新一批 issue 的 state（用于 turn 之间刷新）

### Step 0：确定 tracker.kind 与配置参数

- `tracker.kind` 是对配置暴露的稳定字符串（例如 `linear`）。
- tracker-specific 参数来自 `TrackerConfig.Params`（也就是 front matter `tracker:` 的“整张 map”）。

现有线性工具函数可复用：`pkg/symphony/tracker/linear/decoders.go` 里的 `StringParam(...)` 等。

### Step 1：新增一个 tracker 适配器包

推荐目录结构（与 linear 一致）：

```
pkg/symphony/tracker/<your_kind>/
  client.go
  queries.go        # 如果需要
  decoders.go       # 如果需要
  client_test.go
```

并提供一个构造函数（保持与 orchestrator/newTracker 对齐）：

```go
package yourkind

import symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
import "github.com/wibus-wee/synclax/pkg/symphony/tracker"

func NewFromConfig(cfg symphonycfg.TrackerConfig) (tracker.Client, error) {
  // TODO
  return &client{/*...*/}, nil
}
```

### Step 2：实现 tracker.Client 接口

接口在 `pkg/symphony/tracker/tracker.go`。实现时建议：

- **分页与限流**：优先把过滤推给远端（避免拉全量再本地过滤）
- **超时**：使用 `cfg.Timeout`；每次请求用 `context.WithTimeout`
- **错误包装**：把 HTTP 状态码/远端错误分类到 `*tracker.Error`
- **映射稳定**：输出 `domain.Issue` 字段尽量完整（`Identifier/Title/State/URL/Labels/...`）

`domain.Issue` 定义在：`pkg/symphony/domain/issue.go`

### Step 3：在 orchestrator 注册 kind

需要改 2 处（缺一不可）：

1. `pkg/symphony/orchestrator/orchestrator.go` 的 `newTracker(...)`：
   - 在 `switch cfg.Kind` 新增 `case "<your_kind>": return yourkind.NewFromConfig(cfg)`
2. `pkg/symphony/tracker/tracker.go` 的 `Supported(...)`：
   - 新增 `case "<your_kind>": return true`

### Step 4：补齐 WORKFLOW reference 文档

更新 `docs/reference/workflow.md` 的 `tracker` 小节：

- 标注支持的 `kind` 列表
- 说明 `<your_kind>` 需要的参数（例如 `endpoint`/`api_key`/`project_slug` 等）

### Step 5：写测试

参考：

- `pkg/symphony/tracker/tracker_test.go`：`Supported(...)` 的单测
- `pkg/symphony/tracker/linear/client_test.go`：HTTP stub + 解析/分页行为

---

## Agent / LLM 友好：可复制 Prompt（新增 Provider / Tracker）

把下面这段完整复制给 Agent，让它按步骤实现并自检（把 `<...>` 替换成你的目标）。

```md
你在一个 Go 仓库里实现 Synclax 的扩展点：新增 <provider|tracker>。

约束：
- 只改与目标相关的文件；遵循现有代码风格与目录结构。
- 任何配置变更必须同步更新 docs/reference/workflow.md。
- 必须补充至少一个最小测试，且 `go test ./...` 通过。

任务：
1) 先用 ripgrep 定位现有实现（provider: pkg/symphony/provider/*；tracker: pkg/symphony/tracker/*；注册点：provider/factory.go、orchestrator/newTracker、tracker/Supported）。
2) 按现有实现复制出一份最小可用骨架：
   - provider: 实现 provider.Provider + provider.Session
   - tracker: 实现 tracker.Client，返回 domain.Issue
3) 在 switch/factory 中注册 kind，并补齐 config 校验（必要时扩展 ProviderConfig）。
4) 加最小 smoke test / client test，覆盖“配置能成功初始化 + 基本解析/调用路径”。
5) 更新 VitePress 文档：docs/developer/developer-guide.md（记录步骤与陷阱），docs/reference/workflow.md（配置 schema）。
6) 运行 `go test ./...`，确保全绿。

交付物：
- 代码改动（实现 + 注册点 + 测试）
- 文档改动（developer guide + workflow reference）
- 说明你改了哪些文件、测试怎么跑、为什么这样设计
```

## PR 自检 Checklist（建议复制到 PR 描述）

- [ ] `kind` 命名清晰、稳定（文档与代码一致）
- [ ] 适配器边界清晰：外部 SDK 类型未泄漏到上层
- [ ] respect `context.Context`（超时/取消生效）
- [ ] `Close()` 幂等，不会 panic
- [ ] 错误有分类（provider/tracker error）
- [ ] 文档 reference 已更新（`docs/reference/workflow.md`）
- [ ] 新增/更新测试覆盖关键路径
- [ ] `go test ./...` 通过
