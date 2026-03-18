---
outline: deep
---

# Overview（先理解它能帮你做什么）

Synclax 是一个“可控的后台编排器”：

- 输入：Tracker（当前是 Linear）里的 Issue
- 输出：可被 UI 消费的运行时状态（`/health` + `/symphony/snapshot`），以及稳定的 workspace 执行环境
- 控制：通过 API 明确 start/stop，而不是靠人肉跑脚本

如果你只想快速落地，优先记住这三个事实：

1. **唯一真相是 OpenAPI**：`api/v1.yaml`（UI 建议用生成 client）
2. **运行时状态看 Snapshot**：`GET /api/v1/symphony/snapshot`
3. **行为由 WORKFLOW.md 驱动**：轮询、并发、workspace、hooks、prompt、超时都在这里

## 最短路径（按你的角色）

- 我只想把它跑起来并可观测 → [Quickstart](./quickstart)
- 我需要知道怎么配 `WORKFLOW.md` / 环境变量 → [Configuration](./configuration)
- 我要做一个 Web UI，想要一套推荐的数据流 → [Web UI](./web-ui)
- 我想理解 orchestrator 的行为与 phase → [Symphony](./symphony)