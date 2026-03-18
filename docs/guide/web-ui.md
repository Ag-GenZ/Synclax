---
outline: deep
---

# Web UI（产品/前端集成指南）

这一页面向“使用者/联调”：目标是让你用现成的 **Web Console** 来操作与观察 Synclax：

- 能看到服务是否在线（连接状态）
- 能一键启动/停止后台编排（Symphony）
- 能持续展示运行中任务、阶段（phase）、重试队列与用量

::: tip
如果你想“开发/集成一个自己的 UI”，请看 Developer（Advanced）里的 [Web UI Integration](../developer/web-ui-integration)。
本页不需要你记住任何 API URL。
:::

## 1) 启动 Console

在仓库根目录，确保后端已启动（见 Quickstart），然后执行：

```bash
pnpm -C web dev
```

打开浏览器访问：

```bash
http://localhost:3000
```

## 2) Console 里你应该关注哪些信息

### 连接状态（Health）

- Console 首先要能判断“服务是否在线”
- 当连接失败时，优先检查：后端是否启动、端口是否被占用、以及 WORKFLOW 配置是否可读

### 控制（Start / Stop）

- Start：启动后台编排（Symphony）
- Stop：停止后台编排（用于暂停/排障/降噪）

### 运行时（Snapshot 投影）

Console 的核心价值是把运行态“讲清楚”：

- `running`：当前在跑哪些 Issue（以及当前阶段）
- `retrying`：哪些任务在等待下一次重试（为什么重试、什么时候再来）
- `completed`：最近发生过什么（用于“刚刚发生了什么”的理解）
- `totals`：累计用量（帮助你评估成本与吞吐）
