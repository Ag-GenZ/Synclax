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

---

# 文档分层（从“使用”到“开发”）

你说得对：目前的文档更像“索引”，还不够“能用”。从这次开始我会按层次扩展，避免把不同受众的信息混在一起。

下面是建议的文档分层（后续我会按层逐步补齐）：

## 第一层：使用者/运维视角（先写这一层）

目标：**把服务跑起来、确认健康、启动/停止 Symphony、用 Snapshot 驱动 UI**。

包含内容：

- 部署/启动方式（本地/Compose）
- 健康检查与端口说明
- Symphony 的启动/停止与状态观测（`/symphony/snapshot`）
- 最小可用配置（`WORKFLOW.md`、环境变量）
- 常见故障排查（从现象 → 定位）

入口：

- [Quickstart](./quickstart)

## 第二层：产品/前端视角（Web UI 集成）

目标：**基于 OpenAPI 生成客户端，构建 UI 的数据流与状态机**。

包含内容（计划）：

- 以 `api/v1.yaml` 为唯一真相
- openapi-generator 常用生成方式与注意事项
- UI 轮询策略、页面状态、错误处理与重试
- 安全边界（哪些 endpoint 需要鉴权）

入口（待扩写）：

- [Web UI](./web-ui)

## 第三层：开发者视角（架构与扩展）

目标：**读懂模块、修改/扩展功能、保持与 Anclax 的生成体系一致**。

包含内容（计划）：

- 代码结构/依赖关系
- orchestrator 关键流程与并发模型
- tracker/codex/workspace 的边界与接口
- 生成代码约束（不要手改 `pkg/zgen/*`）

入口（已写但后续会加深）：

- [Architecture](../developer/architecture)
- [Codegen](../developer/codegen)

## 第四层：贡献/发布（规范化）

目标：**贡献者如何开发、测试、发布**（计划）。
