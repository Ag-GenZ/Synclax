import Image from "next/image";
import Link from "next/link";
import { ArrowRight, Blocks, Gauge, GitBranchPlus, Telescope } from "lucide-react";
import { HomeLayout } from "fumadocs-ui/layouts/home";

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
          text: "Console",
          url: "/docs/console/overview",
        },
      ]}
    >
      <main className="home-shell">
        <section className="home-hero">
          <div className="home-panel">
            <div className="home-panel-inner">
              <div className="home-eyebrow">Fumadocs Migration Complete</div>
              <h1 className="home-title">让 Agent 编排像产品一样可控。</h1>
              <p className="home-lead">
                Synclax 把 Tracker 里的 Issue 变成一个可观察、可暂停、可恢复的执行系统。这套文档把
                Console、Concepts、Developer、Reference 全部统一进 Fumadocs，并补上结构化导航、图示和富
                MDX 组件。
              </p>
              <div className="home-actions">
                <Link className="home-primary" href="/docs">
                  打开文档
                </Link>
                <Link className="home-secondary" href="/docs/getting-started/quickstart">
                  先跑起来 <ArrowRight size={16} />
                </Link>
              </div>
              <div className="home-metrics">
                <div className="home-metric">
                  <div className="home-stat-value">6</div>
                  <div className="home-stat-label">主分区</div>
                </div>
                <div className="home-metric">
                  <div className="home-stat-value">14+</div>
                  <div className="home-stat-label">文档页面</div>
                </div>
                <div className="home-metric">
                  <div className="home-stat-value">Full</div>
                  <div className="home-stat-label">Cards / Tabs / Files / Mermaid</div>
                </div>
              </div>
            </div>
          </div>

          <aside className="home-aside">
            <Image
              alt="Synclax orchestration overview"
              height={640}
              src="/images/flow-hero.svg"
              width={1200}
            />
            <div className="home-aside-card">
              <h2>阅读顺序</h2>
              <p>先用 Start Here 跑通系统，再用 Concepts 建立认知模型，最后进入 Console / Developer / Reference。</p>
            </div>
          </aside>
        </section>

        <section className="home-grid">
          <Link className="home-card-link" href="/docs/getting-started/overview">
            <article className="home-section">
              <div className="home-eyebrow">
                <Gauge size={14} />
                Start Here
              </div>
              <h2 className="home-card-title">把系统跑起来并知道该看哪里</h2>
              <p>Overview、Quickstart、Configuration、Troubleshooting 四页覆盖安装、启动与最短排障路径。</p>
            </article>
          </Link>

          <Link className="home-card-link" href="/docs/concepts/architecture">
            <article className="home-section">
              <div className="home-eyebrow">
                <Blocks size={14} />
                Concepts
              </div>
              <h2 className="home-card-title">理解控制循环、架构和状态投影</h2>
              <p>用架构图、生命周期图和设计原则解释 Synclax 为什么这样组织，以及 UI 应该相信什么。</p>
            </article>
          </Link>

          <Link className="home-card-link" href="/docs/developer/index">
            <article className="home-section">
              <div className="home-eyebrow">
                <GitBranchPlus size={14} />
                Developer
              </div>
              <h2 className="home-card-title">给仓库开发者的代码地图与扩展指南</h2>
              <p>从 provider / tracker 扩展，到 codegen、Wire、OpenAPI 与 Web UI 集成，都按落盘路径组织。</p>
            </article>
          </Link>
        </section>

        <section className="home-section">
          <div className="home-eyebrow">
            <Telescope size={14} />
            文档特性
          </div>
          <h2>这次迁移补上的，不只是渲染器。</h2>
          <ul className="home-list">
            <li>Fumadocs App Router shell，统一 landing、docs layout、sidebar、TOC 与 page tabs。</li>
            <li>富 MDX 组件贯穿内容：Cards、Callout、Tabs、Files、Steps、ImageZoom、Mermaid。</li>
            <li>围绕 Synclax 的真实场景重组导航，不再只是把 Vitepress 页面机械搬运过去。</li>
          </ul>
        </section>
      </main>
    </HomeLayout>
  );
}
