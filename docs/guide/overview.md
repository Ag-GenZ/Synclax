---
outline: deep
---

# Overview（先理解它能帮你做什么）

Synclax 是一个**可控的后台编排器（orchestrator）**：

- 输入：Tracker（当前是 Linear）里的 Issue
- 输出：可被 UI 消费的运行时状态（Snapshot），以及稳定的 workspace 执行环境
- 控制：明确的 start/stop（可启停），而不是靠人肉跑脚本

如果你只想快速落地，优先记住这三个事实（用户视角版本）：

1. **它是一个控制循环**：从 Tracker 拉任务 → 调度 → 执行 → 生成 UI 状态投影
2. **你主要通过 Web Console 使用它**：看状态、启停编排、理解“现在在做什么”
3. **行为由 WORKFLOW.md 驱动**：轮询、并发、workspace、hooks、prompt、超时都在这里

## 最短路径（按你的角色）

- 我只想把它跑起来并可观测 → [Quickstart](./quickstart)
- 我想先理解它的理念（为什么这样设计）→ [哲学理念](./philosophy)
- 我想理解系统怎么组成/怎么流动 → [系统架构](./architecture)
- 我需要知道怎么配 `WORKFLOW.md` / 环境变量 → [Configuration](./configuration)
- 我想理解 orchestrator 行为与 phase → [Symphony](./symphony)
