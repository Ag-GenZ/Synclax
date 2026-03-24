import Image from "next/image";
import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import {
  ArrowRight,
  Blocks,
  BookOpenText,
  DatabaseZap,
  Eye,
  FileCode2,
  MonitorSmartphone,
  PauseCircle,
  Repeat2,
  Sparkles,
  Workflow,
  Wrench,
} from "lucide-react";
import { HomeLayout } from "fumadocs-ui/layouts/home";

const stats = [
  { value: "100%", label: "less time reading logs" },
  { value: "12s", label: "to start a task" },
  { value: "99%", label: "reproducible workspace" },
];

const ticker = [
  "可控",
  "可观测",
  "可重复",
  "OpenAPI-first",
  "Issue → PR",
  "Agent Runtime",
  "Workspace Isolation",
  "Snapshot",
  "Symphony",
];

const manifesto = [
  "自动化不应该是一次性脚本。",
  "Issue 是意图；执行是服务。",
  "你应该能暂停、继续、复盘任何任务。",
  "UI 不是装饰，它是系统状态的投影。",
];

const workflow = [
  {
    step: "[01 / 04]",
    title: "Issue 触发",
    description: "从 GitHub、Linear 或任意 Tracker 接收任务信号，解析为可执行运行上下文。",
  },
  {
    step: "[02 / 04]",
    title: "Codex 规划",
    description: "Symphony Codex 拆解任务目标，推导执行路径，选择合适的 Agent 策略与工具组合。",
  },
  {
    step: "[03 / 04]",
    title: "隔离执行",
    description: "每个任务在独立 Workspace 里运行：写代码、执行测试、提交变更，状态可持久化。",
  },
  {
    step: "[04 / 04]",
    title: "结果回报",
    description: "PR 自动创建，Snapshot 投影到 UI，执行历史可回溯，系统明确告诉你发生了什么。",
  },
];

const sections: Array<{
  title: string;
  href: string;
  description: string;
  icon: LucideIcon;
}> = [
  {
    title: "Getting Started",
    href: "/docs/getting-started/overview",
    description: "Overview、Quickstart、Configuration、Troubleshooting。",
    icon: BookOpenText,
  },
  {
    title: "Concepts",
    href: "/docs/concepts/architecture",
    description: "哲学理念、系统架构、Symphony 编排模型。",
    icon: Blocks,
  },
  {
    title: "Console",
    href: "/docs/console/overview",
    description: "运行态、快照模型与界面投影方式。",
    icon: MonitorSmartphone,
  },
  {
    title: "Developer",
    href: "/docs/developer/index",
    description: "代码地图、扩展方式、Codegen、集成与排障。",
    icon: Wrench,
  },
  {
    title: "Reference",
    href: "/docs/reference/index",
    description: "Workflow、HTTP API、Spec、Diagrams。",
    icon: DatabaseZap,
  },
];

const principles: Array<{
  title: string;
  description: string;
  icon: LucideIcon;
  wide?: boolean;
}> = [
  {
    title: "可观测",
    description: "运行中 / 历史 / 用量全部可视化。不读日志，看 UI。",
    icon: Eye,
  },
  {
    title: "可控",
    description: "Start / Stop 是系统能力，不是调试技巧。",
    icon: PauseCircle,
  },
  {
    title: "可重复",
    description: "Workspace 隔离，状态持久，能继续跑、能复盘。",
    icon: Repeat2,
  },
  {
    title: "契约先行",
    description: "OpenAPI-first，UI 与后台通过稳定契约协作。",
    icon: FileCode2,
  },
  {
    title: "Symphony Orchestrator",
    description:
      "多 Agent、可并发、带优先级队列的编排引擎，把复杂任务拆成可管理的执行单元，每个单元都有明确输入输出与状态边界。",
    icon: Workflow,
    wide: true,
  },
];

export default function HomePage() {
  return (
    <HomeLayout
      nav={{
        title: "Synclax Docs",
      }}
      githubUrl="https://github.com/wibus-wee/synclax"
      links={[
        {
          type: "main",
          text: "Docs",
          url: "/docs",
        },
        {
          type: "main",
          text: "Quickstart",
          url: "/docs/getting-started/quickstart",
        },
      ]}
    >
      <main className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-6 sm:px-6 lg:gap-8 lg:py-10">
        <section className="grid gap-6 lg:grid-cols-[minmax(0,1.15fr)_minmax(20rem,0.85fr)]">
          <div className="rounded-2xl border bg-fd-card p-6 sm:p-8">
            <p className="inline-flex items-center rounded-full border border-fd-primary/20 bg-fd-primary/10 px-3 py-1 text-sm font-medium text-fd-primary">
              Issue-Driven Agent Automation
            </p>
            <h1 className="mt-4 text-4xl font-semibold tracking-tight text-fd-foreground sm:text-5xl">
              把 Issue 变成执行
            </h1>
            <p className="mt-4 max-w-2xl text-base leading-7 text-fd-muted-foreground sm:text-lg">
              Synclax 把 Tracker 里的 Issue 接入 Agent 运行时，让自动化像服务一样：可启停、可观测、可重复。
              这套文档把 Console、Concepts、Developer、Reference 统一迁移进 Fumadocs，并保留原有的信息架构判断。
            </p>
            <div className="mt-6 flex flex-wrap gap-3">
              <Link
                className="inline-flex items-center gap-2 rounded-full bg-fd-primary px-4 py-2 text-sm font-medium text-fd-primary-foreground transition-colors hover:opacity-90"
                href="/docs/getting-started/quickstart"
              >
                Quickstart
                <ArrowRight size={16} />
              </Link>
              <Link
                className="inline-flex items-center gap-2 rounded-full border bg-fd-background px-4 py-2 text-sm font-medium text-fd-foreground transition-colors hover:bg-fd-accent/50"
                href="/docs"
              >
                查看文档分区
              </Link>
            </div>
            <dl className="mt-6 grid gap-3 sm:grid-cols-3">
              {stats.map((item) => (
                <div key={item.label} className="rounded-xl border bg-fd-background p-4">
                  <dt className="text-2xl font-semibold text-fd-foreground">{item.value}</dt>
                  <dd className="mt-1 text-sm text-fd-muted-foreground">{item.label}</dd>
                </div>
              ))}
            </dl>
          </div>

          <div className="flex flex-col gap-4">
            <div className="rounded-2xl border bg-fd-card p-4">
              <Image
                alt="Synclax orchestration overview"
                className="w-full rounded-xl border bg-fd-background"
                height={640}
                priority
                src="/images/flow-hero.svg"
                width={1200}
              />
            </div>
            <div className="rounded-2xl border bg-fd-card p-5">
              <div className="flex items-center gap-2 text-sm font-medium text-fd-foreground">
                <Sparkles className="size-4 text-fd-primary" />
                文档分区
              </div>
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                {sections.map((section) => {
                  const Icon = section.icon;

                  return (
                    <Link
                      key={section.href}
                      className="rounded-xl border bg-fd-background p-4 transition-colors hover:bg-fd-accent/50"
                      href={section.href}
                    >
                      <div className="flex items-center gap-2 text-sm font-medium text-fd-foreground">
                        <Icon className="size-4 text-fd-primary" />
                        {section.title}
                      </div>
                      <p className="mt-2 text-sm leading-6 text-fd-muted-foreground">{section.description}</p>
                    </Link>
                  );
                })}
              </div>
            </div>
          </div>
        </section>

        <section className="rounded-2xl border bg-fd-card p-4 sm:p-5">
          <div className="flex flex-wrap gap-2">
            {ticker.map((item) => (
              <span
                key={item}
                className="rounded-full border bg-fd-background px-3 py-1 text-sm text-fd-muted-foreground"
              >
                {item}
              </span>
            ))}
          </div>
        </section>

        <section className="rounded-2xl border bg-fd-card">
          <div className="divide-y">
            {manifesto.map((item, index) => (
              <div
                key={item}
                className="flex flex-col gap-2 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
              >
                <span className="text-sm font-medium text-fd-primary">0{index + 1}</span>
                <p className="text-base text-fd-foreground sm:text-lg">{item}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="grid gap-4 lg:grid-cols-4">
          {workflow.map((item) => (
            <article key={item.step} className="rounded-2xl border bg-fd-card p-5">
              <p className="text-sm font-medium text-fd-primary">{item.step}</p>
              <h2 className="mt-3 text-2xl font-semibold text-fd-foreground">{item.title}</h2>
              <p className="mt-3 text-sm leading-6 text-fd-muted-foreground">{item.description}</p>
            </article>
          ))}
        </section>

        <section className="rounded-2xl border bg-fd-card p-6 sm:p-8">
          <p className="text-sm font-medium text-fd-primary">产品判断</p>
          <div className="mt-3 space-y-1 text-3xl font-semibold tracking-tight text-fd-foreground sm:text-4xl">
            <p>你写 Issue，</p>
            <p>Synclax 写代码。</p>
          </div>
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          {principles.map((item) => {
            const Icon = item.icon;

            return (
              <article
                key={item.title}
                className={`rounded-2xl border bg-fd-card p-5 ${item.wide ? "lg:col-span-2" : ""}`}
              >
                <div className="flex items-center gap-2 text-sm font-medium text-fd-foreground">
                  <Icon className="size-4 text-fd-primary" />
                  {item.title}
                </div>
                <p className="mt-3 text-sm leading-6 text-fd-muted-foreground sm:text-base">{item.description}</p>
              </article>
            );
          })}
        </section>

        <section className="grid gap-6 rounded-2xl border bg-fd-card p-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-center">
          <div>
            <p className="text-sm font-medium text-fd-primary">Getting Started</p>
            <h2 className="mt-3 text-3xl font-semibold tracking-tight text-fd-foreground">
              5 分钟跑起第一个 Agent 任务
            </h2>
            <p className="mt-3 max-w-2xl text-base leading-7 text-fd-muted-foreground">
              从 Quickstart 跑通第一条执行链路，再进入 Developer Guide 看扩展边界与代码地图。
            </p>
            <div className="mt-6 flex flex-wrap gap-3">
              <Link
                className="inline-flex items-center gap-2 rounded-full bg-fd-primary px-4 py-2 text-sm font-medium text-fd-primary-foreground transition-colors hover:opacity-90"
                href="/docs/getting-started/quickstart"
              >
                Quickstart
              </Link>
              <Link
                className="inline-flex items-center gap-2 rounded-full border bg-fd-background px-4 py-2 text-sm font-medium text-fd-foreground transition-colors hover:bg-fd-accent/50"
                href="/docs/developer/index"
              >
                Developer Guide
                <ArrowRight size={16} />
              </Link>
            </div>
          </div>
          <div className="rounded-2xl border bg-fd-background p-4">
            <Image
              alt="Workspace execution grid"
              className="w-full rounded-xl border bg-fd-card"
              height={640}
              src="/images/workspace-grid.svg"
              width={1200}
            />
          </div>
        </section>
      </main>
    </HomeLayout>
  );
}
