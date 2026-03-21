import { withMermaid } from "vitepress-plugin-mermaid";
import type { UserConfig } from "vitepress";

const repository = process.env.GITHUB_REPOSITORY?.split("/")[1];
const isGitHubActions = process.env.GITHUB_ACTIONS === "true";
const isUserOrOrgSite = repository?.endsWith(".github.io");
const base = isGitHubActions && repository && !isUserOrOrgSite ? `/${repository}/` : "/";

export default withMermaid({
  base,
  lang: "zh-CN",
  title: "Synclax",
  description: "把 Tracker 里的 Issue 变成可控的 Agent 执行",
  cleanUrls: true,

  themeConfig: {
    nav: [
      { text: "Guide", link: "/guide/overview" },
      { text: "Philosophy", link: "/guide/philosophy" },
      { text: "Architecture", link: "/guide/architecture" },
      { text: "Developer", link: "/developer/developer-guide" },
      { text: "Console", link: "/guide/web-ui" },
      { text: "Reference", link: "/reference/workflow" },
    ],

    sidebar: {
      "/guide/": [
        {
          text: "开始使用",
          items: [
            { text: "Overview", link: "/guide/overview" },
            { text: "Quickstart", link: "/guide/quickstart" },
            { text: "Web Console", link: "/guide/web-ui" },
          ],
        },
        {
          text: "理解 Synclax",
          items: [
            { text: "哲学理念", link: "/guide/philosophy" },
            { text: "系统架构", link: "/guide/architecture" },
            { text: "Symphony（Orchestrator）", link: "/guide/symphony" },
          ],
        },
        {
          text: "配置与排障",
          items: [
            { text: "Configuration", link: "/guide/configuration" },
            { text: "Troubleshooting", link: "/guide/troubleshooting" },
          ],
        },
      ],
      "/reference/": [
        {
          text: "Reference",
          items: [
            { text: "WORKFLOW.md", link: "/reference/workflow" },
            { text: "Symphony Spec", link: "/reference/symphony-spec" },
            { text: "Mermaid Diagrams", link: "/reference/diagrams" },
            { text: "HTTP API（Advanced）", link: "/reference/http-api" },
          ],
        },
      ],
      "/developer/": [
        {
          text: "Developer（Advanced）",
          items: [
            { text: "Developer Guide（Extending）", link: "/developer/developer-guide" },
            { text: "Architecture（Code Map）", link: "/developer/architecture" },
            { text: "Web UI Integration", link: "/developer/web-ui-integration" },
            { text: "Codegen", link: "/developer/codegen" },
            { text: "Troubleshooting", link: "/developer/troubleshooting" },
          ],
        },
      ],
    },

    search: {
      provider: "local",
    },
  },
} satisfies UserConfig);
