---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Synclax"
  text: "把 Tracker 里的 Issue 变成可控的 Agent 执行"
  tagline: 让自动化像服务一样运行：可启停、可观测、可重复（并且能被 UI 解释清楚）。
  actions:
    - theme: brand
      text: Quickstart（先跑起来）
      link: /guide/quickstart
    - theme: alt
      text: Web Console（怎么用）
      link: /guide/web-ui
    - theme: alt
      text: 系统架构（先理解）
      link: /guide/architecture
    - theme: alt
      text: Developer Guide（扩展）
      link: /developer/developer-guide

features:
  - title: 可控（能启停、能解释）
    details: Start/Stop 是系统能力；状态通过 Snapshot 投影给 UI，而不是让人读日志。
  - title: 可观测（UI-first）
    details: 运行中/重试中/历史记录/用量等都可视化；“现在发生了什么”一眼能看懂。
  - title: 可重复（Workspace）
    details: 每个任务有稳定的工作目录与 hooks；能继续跑、能复盘、能回滚。
  - title: 契约先行（OpenAPI-first）
    details: UI 与后台通过稳定契约协作；集成方按契约接入，而不是靠口口相传的约定。
---
