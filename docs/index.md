---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Synclax"
  text: "Symphony + Anclax = Synclax"
  tagline: Run a tracker-driven orchestrator behind an OpenAPI-first Anclax API.
  actions:
    - theme: brand
      text: Quickstart
      link: /guide/quickstart
    - theme: alt
      text: HTTP API
      link: /reference/http-api

features:
  - title: OpenAPI-first
    details: Edit `api/v1.yaml`, run `anclax gen`, and generate both server handlers and clients.
  - title: Symphony control plane
    details: Start/stop Symphony and poll `/symphony/snapshot` to power a web UI.
  - title: Hot-reload workflows
    details: Update `WORKFLOW.md` to change polling, hooks, and prompt templates at runtime.
---
