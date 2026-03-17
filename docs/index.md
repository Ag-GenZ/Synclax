---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Synclax"
  text: "把 Tracker 里的 Issue 变成可控的 Agent 执行"
  tagline: Symphony（编排器）+ Anclax（OpenAPI-first API）= 可启动 / 可停止 / 可观测的后台自动化。
  actions:
    - theme: brand
      text: Quickstart（先跑起来）
      link: /guide/quickstart
    - theme: alt
      text: HTTP API（给 UI）
      link: /reference/http-api
    - theme: alt
      text: Web UI（集成指南）
      link: /guide/web-ui

features:
  - title: OpenAPI-first（对接成本低）
    details: 把 `api/v1.yaml` 当作唯一真相；跑 `anclax gen` 生成 server handler 与 client。
  - title: UI-ready Snapshot（可观测）
    details: 用 `/health` + `/symphony/snapshot` 直接驱动 UI：运行中/重试中/用量统计一目了然。
  - title: 可控的后台编排（可启停）
    details: 通过 `/symphony/start`、`/symphony/stop` 控制 orchestrator 生命周期；避免“脚本跑飞”。
  - title: WORKFLOW.md（配置即行为）
    details: 用一个文件定义 polling、workspace、hook、prompt 模板与超时；支持运行时 reload。
---
