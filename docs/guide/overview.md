---
outline: deep
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

