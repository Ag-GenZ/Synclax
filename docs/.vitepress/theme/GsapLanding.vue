<script setup lang="ts">
import { onMounted, onUnmounted, nextTick, ref } from 'vue'
import { gsap } from 'gsap'
import { ScrollTrigger } from 'gsap/ScrollTrigger'
import PipelineCard from './PipelineCard.vue'

const container = ref<HTMLElement | null>(null)
let ctx: gsap.Context | undefined
let _onMove: ((e: MouseEvent) => void) | undefined

onMounted(() => {
  gsap.registerPlugin(ScrollTrigger)

  nextTick(() => {
    ctx = gsap.context(() => {

      // ── 1. Hero chars entrance ───────────────────────────────────
      const chars = gsap.utils.toArray<HTMLElement>('.giant-char')
      const intro = gsap.timeline({ defaults: { ease: 'power4.out' } })
      intro
        .from('.nav-bar', { y: -40, autoAlpha: 0, duration: 0.7 })
        .from('.hero-kicker', { y: 24, autoAlpha: 0, duration: 0.55 }, '-=0.25')
        .from(chars, { y: '105%', duration: 0.9, stagger: 0.04 }, '-=0.3')
        .from('.hero-sub', { y: 22, autoAlpha: 0, duration: 0.6 }, '-=0.55')
        .from('.hero-cta-row > *', { y: 18, autoAlpha: 0, duration: 0.5, stagger: 0.1 }, '-=0.4')

      // ── 2. Ticker tape loop ──────────────────────────────────────
      const tape = document.querySelector('.tape-inner') as HTMLElement | null
      if (tape) {
        const clone = tape.cloneNode(true) as HTMLElement
        tape.parentElement!.appendChild(clone)
        const w = tape.offsetWidth
        gsap.fromTo(
          [tape, clone],
          { x: 0 },
          {
            x: -w,
            duration: 28,
            ease: 'none',
            repeat: -1,
            modifiers: { x: gsap.utils.unitize(gsap.utils.wrap(-w, 0)) },
          },
        )
      }

      // ── 3. Manifesto lines slide in ──────────────────────────────
      gsap.from('.manifesto-line', {
        x: -55,
        autoAlpha: 0,
        duration: 0.75,
        stagger: 0.16,
        ease: 'power3.out',
        scrollTrigger: { trigger: '.manifesto-section', start: 'top 72%' },
      })

      // ── 4. Draw lines scale from left ────────────────────────────
      gsap.utils.toArray<HTMLElement>('.draw-line').forEach((line) => {
        gsap.from(line, {
          scaleX: 0,
          transformOrigin: 'left center',
          duration: 1.1,
          ease: 'power3.inOut',
          scrollTrigger: { trigger: line, start: 'top 92%' },
        })
      })

        // ── 5. Stats counter ─────────────────────────────────────────
        ;[
          { sel: '.stat-n-0', end: 100 },
          { sel: '.stat-n-1', end: 12 },
          { sel: '.stat-n-2', end: 99 },
        ].forEach(({ sel, end }) => {
          const el = document.querySelector(sel) as HTMLElement | null
          if (!el) return
          const obj = { val: 0 }
          gsap.to(obj, {
            val: end,
            duration: 1.5,
            ease: 'power2.out',
            scrollTrigger: { trigger: '.stats-section', start: 'top 76%' },
            onUpdate: () => { el.textContent = Math.round(obj.val).toString() },
          })
        })

      // ── 6. Horizontal pin scroll ─────────────────────────────────
      const hTrack = document.querySelector('.h-track') as HTMLElement | null
      if (hTrack) {
        const slides = gsap.utils.toArray<HTMLElement>('.h-slide')
        const getTotalMove = () => hTrack.scrollWidth - window.innerWidth

        gsap.to(hTrack, {
          x: () => -getTotalMove(),
          ease: 'none',
          scrollTrigger: {
            trigger: '.h-scroll-section',
            pin: true,
            scrub: 0.8,
            start: 'top top',
            end: () => `+=${getTotalMove()}`,
            invalidateOnRefresh: true,
          },
        })

        slides.forEach((slide, i) => {
          ScrollTrigger.create({
            trigger: '.h-scroll-section',
            start: () => `top+=${(getTotalMove() / slides.length) * i} top`,
            end: () => `top+=${(getTotalMove() / slides.length) * (i + 1)} top`,
            scrub: true,
            onUpdate: (self) => {
              const fill = slide.querySelector('.slide-progress-fill') as HTMLElement | null
              if (fill) fill.style.width = `${self.progress * 100}%`
            },
          })
        })
      }

      // ── 7. Phrase mask reveal ────────────────────────────────────
      gsap.utils.toArray<HTMLElement>('.phrase-line').forEach((el) => {
        const inner = el.querySelector('.phrase-inner')
        if (!inner) return
        gsap.from(inner, {
          y: '104%',
          duration: 0.9,
          ease: 'power4.out',
          scrollTrigger: { trigger: el, start: 'top 88%', toggleActions: 'play none none reverse' },
        })
      })

      // ── 8. Grid items stagger ────────────────────────────────────
      ScrollTrigger.batch('.grid-item', {
        onEnter: (els) =>
          gsap.from(els, { y: 48, autoAlpha: 0, duration: 0.72, stagger: 0.1, ease: 'power3.out', clearProps: 'all' }),
        start: 'top 90%',
        once: true,
      })

      // ── A. Hero parallax on scroll ──────────────────────────────
      gsap.to('.hero-kicker', {
        y: -28, ease: 'none',
        scrollTrigger: { trigger: '.hero-section', start: 'top top', end: 'bottom top', scrub: 1 },
      })
      gsap.to('.hero-giant', {
        y: -52, ease: 'none',
        scrollTrigger: { trigger: '.hero-section', start: 'top top', end: 'bottom top', scrub: 1 },
      })
      gsap.to('.hero-bottom', {
        y: -16, ease: 'none',
        scrollTrigger: { trigger: '.hero-section', start: 'top top', end: 'bottom top', scrub: 1 },
      })

      // ── B. Slash clip-path reveal ────────────────────────────────
      gsap.fromTo('.slash-dark',
        { clipPath: 'inset(0 100% 0 0)' },
        {
          clipPath: 'inset(0 0% 0 0)',
          ease: 'none',
          scrollTrigger: {
            trigger: '.slash-section',
            start: 'top 75%',
            end: 'bottom 25%',
            scrub: 0.5,
          },
        },
      )

      // ── C. Grid 3D tilt on hover ─────────────────────────────────
      gsap.utils.toArray<HTMLElement>('.grid-item').forEach((card) => {
        card.addEventListener('mousemove', (e: Event) => {
          const me = e as MouseEvent
          const r = card.getBoundingClientRect()
          const xr = (me.clientX - r.left) / r.width - 0.5
          const yr = (me.clientY - r.top) / r.height - 0.5
          gsap.to(card, { rotateY: xr * 10, rotateX: -yr * 10, transformPerspective: 900, ease: 'power2.out', duration: 0.35 })
        })
        card.addEventListener('mouseleave', () => {
          gsap.to(card, { rotateY: 0, rotateX: 0, duration: 0.6, ease: 'power3.out' })
        })
      })

      // ── D. Custom cursor ─────────────────────────────────────────
      const dot = document.querySelector('.cursor-dot') as HTMLElement | null
      const ring = document.querySelector('.cursor-ring') as HTMLElement | null
      if (dot && ring && !('ontouchstart' in window)) {
        gsap.set([dot, ring], { autoAlpha: 1 })
        const xDot = gsap.quickTo(dot, 'x', { duration: 0.12, ease: 'power3' })
        const yDot = gsap.quickTo(dot, 'y', { duration: 0.12, ease: 'power3' })
        const xRing = gsap.quickTo(ring, 'x', { duration: 0.55, ease: 'power3' })
        const yRing = gsap.quickTo(ring, 'y', { duration: 0.55, ease: 'power3' })
        _onMove = (e: MouseEvent) => {
          xDot(e.clientX - 4)
          yDot(e.clientY - 4)
          xRing(e.clientX - 14)
          yRing(e.clientY - 14)
        }
        window.addEventListener('mousemove', _onMove)
        document.querySelectorAll('a, .grid-item, .nav-action').forEach(el => {
          el.addEventListener('mouseenter', () => gsap.to(ring, { scale: 2.6, duration: 0.3, ease: 'power2.out' }))
          el.addEventListener('mouseleave', () => gsap.to(ring, { scale: 1, duration: 0.3, ease: 'power2.out' }))
        })
      }

      // ── 9. CTA big text reveal ───────────────────────────────────
      gsap.from('.cta-word', {
        y: '105%',
        duration: 1,
        ease: 'power4.out',
        stagger: 0.07,
        scrollTrigger: { trigger: '.cta-big', start: 'top 80%' },
      })

      // ── 10. Nav fill on scroll ───────────────────────────────────
      ScrollTrigger.create({
        start: 'top -40',
        end: 'max',
        onToggle: (self) => {
          gsap.to('.nav-bar', {
            borderBottomColor: self.isActive ? 'rgba(0,0,0,0.1)' : 'transparent',
            backgroundColor: self.isActive ? '#f8f8f6' : 'transparent',
            duration: 0.4,
          })
        },
      })

      ScrollTrigger.refresh()
    }, container.value!)
  })
})

onUnmounted(() => {
  ctx?.revert()
  if (_onMove) window.removeEventListener('mousemove', _onMove)
})
</script>

<template>
  <div ref="container" class="sl-landing">
    <!-- Custom cursor -->
    <div class="cursor-dot" aria-hidden="true"></div>
    <div class="cursor-ring" aria-hidden="true"></div>

    <!-- NAV -->
    <nav class="nav-bar">
      <div class="nav-inner">
        <a href="/" class="nav-wordmark">SYNCLAX</a>
        <div class="nav-mid">
          <a href="/guide/overview">Docs</a>
          <a href="/guide/architecture">Architecture</a>
          <a href="/developer/developer-guide">Developer</a>
        </div>
        <a href="/guide/quickstart" class="nav-action">Start →</a>
      </div>
    </nav>

    <!-- HERO -->
    <section class="hero-section">
      <div class="hero-inner">
        <div class="hero-left">
          <p class="hero-kicker">Issue-Driven Agent Automation</p>
          <h1 class="hero-giant" aria-label="把 Issue 变成执行">
            <span class="title-clip"><span class="giant-char">把</span></span>
            <span class="title-clip title-clip--i1"><span class="giant-char">Issue</span></span>
            <span class="title-clip title-clip--i2"><span class="giant-char">变成</span></span>
            <span class="title-clip title-clip--i3"><span class="giant-char">执行</span></span>
          </h1>
          <div class="hero-bottom">
            <p class="hero-sub">
              Synclax 把 Tracker 里的 Issue 接入 Agent 运行时，<br />
              让自动化像服务一样：可启停、可观测、可重复。
            </p>
          </div>
        </div>

        <div class="hero-right" aria-hidden="true">
          <PipelineCard />
        </div>
      </div>
    </section>

    <!-- TICKER -->
    <div class="ticker-wrap" aria-hidden="true">
      <div class="ticker-track">
        <div class="tape-inner">
          <span>可控</span><span class="sep">×</span>
          <span>可观测</span><span class="sep">×</span>
          <span>可重复</span><span class="sep">×</span>
          <span>OpenAPI-first</span><span class="sep">×</span>
          <span>Issue → PR</span><span class="sep">×</span>
          <span>Agent Runtime</span><span class="sep">×</span>
          <span>Workspace Isolation</span><span class="sep">×</span>
          <span>Snapshot</span><span class="sep">×</span>
          <span>Symphony</span><span class="sep">×</span>
          <span>可控</span><span class="sep">×</span>
          <span>可观测</span><span class="sep">×</span>
          <span>可重复</span><span class="sep">×</span>
          <span>OpenAPI-first</span><span class="sep">×</span>
          <span>Issue → PR</span><span class="sep">×</span>
          <span>Agent Runtime</span><span class="sep">×</span>
          <span>Workspace Isolation</span><span class="sep">×</span>
          <span>Snapshot</span><span class="sep">×</span>
          <span>Symphony</span><span class="sep">×</span>
        </div>
      </div>
    </div>

    <!-- TICKER INV -->
    <div class="ticker-wrap ticker-wrap--inv" aria-hidden="true">
      <div class="tape-inv">
        <span>可控</span><span class="sep">×</span>
        <span>可观测</span><span class="sep">×</span>
        <span>可重复</span><span class="sep">×</span>
        <span>OpenAPI-first</span><span class="sep">×</span>
        <span>Issue → PR</span><span class="sep">×</span>
        <span>Agent Runtime</span><span class="sep">×</span>
        <span>Workspace Isolation</span><span class="sep">×</span>
        <span>Symphony</span><span class="sep">×</span>
        <span>可控</span><span class="sep">×</span>
        <span>可观测</span><span class="sep">×</span>
        <span>可重复</span><span class="sep">×</span>
        <span>OpenAPI-first</span><span class="sep">×</span>
        <span>Issue → PR</span><span class="sep">×</span>
        <span>Agent Runtime</span><span class="sep">×</span>
        <span>Workspace Isolation</span><span class="sep">×</span>
        <span>Symphony</span><span class="sep">×</span>
      </div>
    </div>

    <!-- MANIFESTO -->
    <section class="manifesto-section">
      <div class="manifesto-inner">
        <div class="manifesto-line">
          <span class="ml-num">01</span>
          <span class="ml-text">自动化不应该是一次性脚本。</span>
        </div>
        <div class="draw-line mf-div"></div>
        <div class="manifesto-line">
          <span class="ml-num">02</span>
          <span class="ml-text">Issue 是意图；执行是服务。</span>
        </div>
        <div class="draw-line mf-div"></div>
        <div class="manifesto-line">
          <span class="ml-num">03</span>
          <span class="ml-text">你应该能暂停、继续、复盘任何任务。</span>
        </div>
        <div class="draw-line mf-div"></div>
        <div class="manifesto-line">
          <span class="ml-num">04</span>
          <span class="ml-text">UI 不是装饰，它是系统状态的投影。</span>
        </div>
      </div>
    </section>

    <!-- STATS -->
    <section class="stats-section">
      <div class="draw-line"></div>
      <div class="stats-inner">
        <div class="stat-item">
          <div class="stat-number"><span class="stat-n-0">0</span><span class="stat-unit">%</span></div>
          <p class="stat-label">less time reading logs</p>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <div class="stat-number"><span class="stat-n-1">0</span><span class="stat-unit">s</span></div>
          <p class="stat-label">to start a task</p>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <div class="stat-number"><span class="stat-n-2">0</span><span class="stat-unit">%</span></div>
          <p class="stat-label">reproducible workspace</p>
        </div>
      </div>
      <div class="draw-line"></div>
    </section>

    <!-- HORIZONTAL SCROLL (4 slides) -->
    <div class="h-scroll-section">
      <div class="h-track">
        <div class="h-slide">
          <div class="slide-tag">[01 / 04]</div>
          <h2 class="slide-title">Issue<br />触发</h2>
          <p class="slide-body">从 GitHub、Linear 或任意 Tracker 接收任务信号，解析为可执行运行上下文，无需人工干预。</p>
          <div class="slide-prog">
            <div class="slide-progress-fill"></div>
          </div>
        </div>
        <div class="h-slide h-slide--inv">
          <div class="slide-tag">[02 / 04]</div>
          <h2 class="slide-title">Codex<br />规划</h2>
          <p class="slide-body">Symphony Codex 拆解任务目标，推导执行路径，选择合适的 Agent 策略与工具组合。每一步都可审计。</p>
          <div class="slide-prog">
            <div class="slide-progress-fill"></div>
          </div>
        </div>
        <div class="h-slide">
          <div class="slide-tag">[03 / 04]</div>
          <h2 class="slide-title">隔离<br />执行</h2>
          <p class="slide-body">每个任务在独立 Workspace 里运行：写代码、执行测试、提交变更。随时可中止，状态持久化。</p>
          <div class="slide-prog">
            <div class="slide-progress-fill"></div>
          </div>
        </div>
        <div class="h-slide h-slide--inv">
          <div class="slide-tag">[04 / 04]</div>
          <h2 class="slide-title">结果<br />回报</h2>
          <p class="slide-body">PR 自动创建，Snapshot 投影到 UI，执行历史可回溯。系统告诉你发生了什么，以及为什么。</p>
          <div class="slide-prog">
            <div class="slide-progress-fill"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- PHRASE -->
    <section class="phrase-section">
      <div class="draw-line"></div>
      <div class="phrase-block">
        <div class="phrase-line">
          <div class="phrase-inner">你写 Issue，</div>
        </div>
        <div class="phrase-line">
          <div class="phrase-inner phrase-inner--outline">Synclax 写代码。</div>
        </div>
      </div>
      <div class="draw-line"></div>
    </section>

    <!-- SLASH REVEAL -->
    <div class="slash-section">
      <div class="slash-light">
        <span class="slash-big">自动化</span>
        <span class="slash-caption">一切可重复的工作</span>
      </div>
      <div class="slash-dark">
        <span class="slash-big">自动化</span>
        <span class="slash-caption">一切可重复的工作</span>
      </div>
    </div>

    <!-- GRID -->
    <section class="grid-section">
      <div class="grid-inner">
        <div class="grid-item">
          <div class="gi-icon" aria-hidden="true">
            <!-- ICON PLACEHOLDER: eye -->
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
              stroke-linecap="round" stroke-linejoin="round">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
              <circle cx="12" cy="12" r="3" />
            </svg>
          </div>
          <h3 class="gi-title">可观测</h3>
          <p class="gi-body">运行中 / 历史 / 用量全部可视化。不读日志，看 UI。</p>
        </div>
        <div class="grid-item">
          <div class="gi-icon" aria-hidden="true">
            <!-- ICON PLACEHOLDER: power -->
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
              stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 2v10" />
              <path d="M18.4 6.6a9 9 0 1 1-12.8 0" />
            </svg>
          </div>
          <h3 class="gi-title">可控</h3>
          <p class="gi-body">Start / Stop 是系统能力，不是调试技巧。</p>
        </div>
        <div class="grid-item">
          <div class="gi-icon" aria-hidden="true">
            <!-- ICON PLACEHOLDER: layers -->
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
              stroke-linecap="round" stroke-linejoin="round">
              <polygon points="12 2 22 8.5 12 15 2 8.5 12 2" />
              <polyline points="2 15 12 21.5 22 15" />
              <polyline points="2 11.5 12 18 22 11.5" />
            </svg>
          </div>
          <h3 class="gi-title">可重复</h3>
          <p class="gi-body">Workspace 隔离，状态持久，能继续跑、能复盘。</p>
        </div>
        <div class="grid-item">
          <div class="gi-icon" aria-hidden="true">
            <!-- ICON PLACEHOLDER: file / contract -->
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
              stroke-linecap="round" stroke-linejoin="round">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
              <line x1="9" y1="13" x2="15" y2="13" />
              <line x1="9" y1="17" x2="13" y2="17" />
            </svg>
          </div>
          <h3 class="gi-title">契约先行</h3>
          <p class="gi-body">OpenAPI-first，UI 与后台通过稳定契约协作。</p>
        </div>
        <div class="grid-item gi-wide">
          <div class="gi-icon" aria-hidden="true">
            <!-- ICON PLACEHOLDER: zap / lightning -->
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
              stroke-linecap="round" stroke-linejoin="round">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
            </svg>
          </div>
          <h3 class="gi-title">Symphony Orchestrator</h3>
          <p class="gi-body">多 Agent、可并发、带优先级队列的编排引擎。把复杂任务分解为可管理的执行单元，每个单元都有明确的输入输出与状态边界。</p>
        </div>
      </div>
    </section>

    <!-- BIG CTA -->
    <section class="cta-section">
      <div class="draw-line"></div>
      <div class="cta-inner">
        <h2 class="cta-big">
          <span class="cta-line"><span class="cta-word">5 分钟</span></span>
          <span class="cta-line"><span class="cta-word">跑起第一个</span></span>
          <span class="cta-line cta-line--flex">
            <span class="cta-word">Agent 任务</span>
            <span class="cta-word cta-word--stroke">→</span>
          </span>
        </h2>
        <div class="cta-actions">
          <a href="/guide/quickstart" class="btn-dark btn-xl">Quickstart</a>
          <a href="/developer/developer-guide" class="btn-text btn-xl-text">Developer Guide →</a>
        </div>
      </div>
      <div class="draw-line"></div>
    </section>

    <!-- FOOTER -->
    <footer class="site-footer">
      <div class="footer-inner">
        <span class="footer-brand">SYNCLAX</span>
        <nav class="footer-nav">
          <a href="/guide/overview">Guide</a>
          <a href="/reference/http-api">API</a>
          <a href="/developer/developer-guide">Developer</a>
        </nav>
        <span class="footer-copy">MIT License</span>
      </div>
    </footer>

  </div>
</template>

<style scoped>
/* ── Base ──────────────────────────────────────────────────────── */
.sl-landing {
  --fg: #0a0a0a;
  --bg: #f8f8f6;
  --muted: rgba(10, 10, 10, 0.42);
  --border: rgba(10, 10, 10, 0.1);
  --mono: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;

  background: var(--bg);
  color: var(--fg);
  font-family: -apple-system, 'Inter', 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  overflow-x: hidden;
}

.sl-landing *,
.sl-landing *::before,
.sl-landing *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

.sl-landing a {
  color: inherit;
  text-decoration: none;
}

/* ── Nav ───────────────────────────────────────────────────────── */
.nav-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 200;
  background: transparent;
  border-bottom: 1px solid transparent;
  transition: background 0.35s, border-color 0.35s;
}

.nav-inner {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 2.5rem;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 3rem;
}

.nav-wordmark {
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.22em;
}

.nav-mid {
  display: flex;
  gap: 2.2rem;
  flex: 1;
}

.nav-mid a {
  font-size: 0.83rem;
  color: var(--muted);
  transition: color 0.18s;
}

.nav-mid a:hover {
  color: var(--fg);
}

.nav-action {
  margin-left: auto;
  font-size: 0.82rem;
  font-weight: 650;
  border: 1px solid var(--fg);
  padding: 0.36rem 1rem;
  border-radius: 3px;
  transition: background 0.18s, color 0.18s;
}

.nav-action:hover {
  background: var(--fg);
  color: var(--bg);
}

/* ── Hero ──────────────────────────────────────────────────────── */
.hero-section {
  height: 100svh;
  display: flex;
  flex-direction: column;
  padding: 0 2.5rem 4rem;
  max-width: 1280px;
  margin: 0 auto;
  width: 100%;
  overflow: hidden;
}

.hero-inner {
  flex: 1;
  display: flex;
  align-items: stretch;
  gap: 4rem;
  padding-top: 56px;
}

.hero-left {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}

.hero-right {
  width: 480px;
  flex-shrink: 0;
  display: flex;
  align-items: stretch;
  will-change: transform, opacity;
}

.hero-kicker {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--muted);
  margin-bottom: 2rem;
  font-family: var(--mono);
}

.hero-giant {
  line-height: 0.9;
  letter-spacing: -0.045em;
  font-size: clamp(3.5rem, 8.5vw, 11rem);
  font-weight: 900;
  display: flex;
  flex-direction: column;
  gap: 0.04em;
  margin-bottom: 3rem;
}

.title-clip {
  overflow: hidden;
  display: block;
}

.title-clip--i1 {
  padding-left: 0.18em;
}

.title-clip--i2 {
  padding-left: 0.36em;
}

.title-clip--i3 {
  padding-left: 0.54em;
}

.giant-char {
  display: block;
  will-change: transform;
}

.hero-bottom {
  display: flex;
  align-items: flex-end;
  gap: 5rem;
}

.hero-sub {
  font-size: 1rem;
  line-height: 1.76;
  color: var(--muted);
  max-width: 400px;
  flex: 1;
}

.hero-cta-row {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  flex-shrink: 0;
}

/* ── Buttons ───────────────────────────────────────────────────── */
.btn-dark {
  display: inline-block;
  background: var(--fg);
  color: var(--bg);
  padding: 0.72rem 1.7rem;
  border-radius: 3px;
  font-size: 0.88rem;
  font-weight: 650;
  transition: opacity 0.18s, transform 0.15s;
}

.btn-dark:hover {
  opacity: 0.75;
  transform: translateY(-1px);
}

.btn-text {
  font-size: 0.88rem;
  font-weight: 550;
  color: var(--muted);
  transition: color 0.18s;
}

.btn-text:hover {
  color: var(--fg);
}

.btn-xl {
  font-size: 1rem;
  padding: 0.85rem 2.1rem;
}

.btn-xl-text {
  font-size: 1rem;
}

/* ── Ticker ────────────────────────────────────────────────────── */
.ticker-wrap {
  overflow: hidden;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  padding: 0.8rem 0;
}

.ticker-track {
  display: flex;
  white-space: nowrap;
  overflow: hidden;
}

.tape-inner {
  display: inline-flex;
  align-items: center;
  gap: 1.6rem;
  padding-right: 1.6rem;
  will-change: transform;
}

.tape-inner span {
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  font-family: var(--mono);
  color: var(--fg);
}

.tape-inner .sep {
  color: var(--muted);
  font-size: 0.6rem;
}

/* ── Manifesto ─────────────────────────────────────────────────── */
.manifesto-section {
  max-width: 1280px;
  margin: 0 auto;
  padding: 10rem 2.5rem;
}

.manifesto-inner {
  display: flex;
  flex-direction: column;
}

.mf-div {
  height: 1px;
  background: var(--border);
  border: none;
}

.manifesto-line {
  display: flex;
  align-items: baseline;
  gap: 2.5rem;
  padding: 1.8rem 0;
}

.ml-num {
  font-size: 0.7rem;
  font-weight: 700;
  font-family: var(--mono);
  color: var(--muted);
  letter-spacing: 0.06em;
  flex-shrink: 0;
  width: 2rem;
}

.ml-text {
  font-size: clamp(1.4rem, 2.8vw, 2.4rem);
  font-weight: 750;
  letter-spacing: -0.03em;
  line-height: 1.15;
}

/* ── Stats ─────────────────────────────────────────────────────── */
.stats-section {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 2.5rem;
}

.draw-line {
  height: 1px;
  background: var(--border);
  width: 100%;
  will-change: transform;
}

.stats-inner {
  display: flex;
  align-items: stretch;
  padding: 4.5rem 0;
}

.stat-item {
  flex: 1;
  padding: 0 3rem;
}

.stat-item:first-child {
  padding-left: 0;
}

.stat-item:last-child {
  padding-right: 0;
}

.stat-divider {
  width: 1px;
  background: var(--border);
  flex-shrink: 0;
}

.stat-number {
  font-size: clamp(4rem, 8vw, 7rem);
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  font-family: var(--mono);
  margin-bottom: 0.55rem;
  display: flex;
  align-items: baseline;
  gap: 0.06em;
}

.stat-unit {
  font-size: 0.38em;
  font-weight: 700;
  color: var(--muted);
}

.stat-label {
  font-size: 0.78rem;
  color: var(--muted);
  font-weight: 500;
  font-family: var(--mono);
}

/* ── Horizontal scroll ─────────────────────────────────────────── */
.h-scroll-section {
  overflow: hidden;
  height: 100vh;
}

.h-track {
  display: flex;
  height: 100%;
  will-change: transform;
}

.h-slide {
  width: 100vw;
  flex-shrink: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  padding: 0 2.5rem 5rem;
  border-right: 1px solid var(--border);
  position: relative;
  background: var(--bg);
}

.h-slide--inv {
  background: var(--fg);
  color: var(--bg);
}

.h-slide--inv .slide-body {
  color: rgba(248, 248, 246, 0.45);
}

.h-slide--inv .slide-tag {
  color: rgba(248, 248, 246, 0.28);
}

.h-slide--inv .slide-prog {
  background: rgba(255, 255, 255, 0.12);
}

.h-slide--inv .slide-progress-fill {
  background: #fff;
}

.slide-tag {
  position: absolute;
  top: 50%;
  right: 2.5rem;
  transform: translateY(-50%);
  font-size: 0.7rem;
  font-family: var(--mono);
  color: var(--muted);
  letter-spacing: 0.08em;
  writing-mode: vertical-rl;
}

.slide-title {
  font-size: clamp(4rem, 10vw, 9.5rem);
  font-weight: 900;
  letter-spacing: -0.045em;
  line-height: 0.92;
  margin-bottom: 2.5rem;
}

.slide-body {
  font-size: 0.97rem;
  line-height: 1.76;
  color: var(--muted);
  max-width: 480px;
  margin-bottom: 2.2rem;
}

.slide-prog {
  width: 100px;
  height: 2px;
  background: rgba(10, 10, 10, 0.1);
  border-radius: 999px;
  overflow: hidden;
}

.slide-progress-fill {
  height: 100%;
  width: 0%;
  background: var(--fg);
  border-radius: 999px;
}

/* ── Phrase ────────────────────────────────────────────────────── */
.phrase-section {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 2.5rem;
}

.phrase-block {
  padding: 8rem 0;
}

.phrase-line {
  overflow: hidden;
  display: block;
}

.phrase-inner {
  display: block;
  font-size: clamp(3.5rem, 9vw, 8.5rem);
  font-weight: 900;
  letter-spacing: -0.045em;
  line-height: 0.98;
  will-change: transform;
}

.phrase-inner--outline {
  -webkit-text-stroke: 2px var(--fg);
  color: transparent;
}

/* ── Grid ──────────────────────────────────────────────────────── */
.grid-section {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 2.5rem 8rem;
}

.grid-inner {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1px;
  background: var(--border);
  border: 1px solid var(--border);
}

.grid-item {
  background: var(--bg);
  padding: 2.4rem 2rem;
  will-change: transform;
  transform-style: preserve-3d;
  transition: box-shadow 0.3s ease;
}

.grid-item:hover {
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.08);
  z-index: 1;
}

.gi-wide {
  grid-column: span 2;
}

.gi-icon {
  color: var(--fg);
  margin-bottom: 1.2rem;
  opacity: 0.6;
}

.gi-title {
  font-size: 0.93rem;
  font-weight: 750;
  letter-spacing: -0.01em;
  margin-bottom: 0.6rem;
}

.gi-body {
  font-size: 0.83rem;
  color: var(--muted);
  line-height: 1.7;
}

/* ── Cursor ───────────────────────────────────────────────────── */
.cursor-dot,
.cursor-ring {
  position: fixed;
  top: 0;
  left: 0;
  pointer-events: none;
  z-index: 9999;
  visibility: hidden;
  opacity: 0;
}

.cursor-dot {
  width: 8px;
  height: 8px;
  background: var(--fg);
  border-radius: 50%;
}

.cursor-ring {
  width: 28px;
  height: 28px;
  border: 1.5px solid var(--fg);
  border-radius: 50%;
  mix-blend-mode: difference;
}

.sl-landing {
  cursor: none;
}

/* ── Slash reveal ──────────────────────────────────────────────── */
.slash-section {
  position: relative;
  overflow: hidden;
  padding: 8rem 2.5rem;
  max-width: 1280px;
  margin: 0 auto;
}

.slash-light,
.slash-dark {
  display: flex;
  align-items: baseline;
  gap: 3rem;
}

.slash-dark {
  position: absolute;
  inset: 0;
  padding: 8rem 2.5rem;
  background: var(--fg);
  color: var(--bg);
  clip-path: inset(0 100% 0 0);
  will-change: clip-path;
}

.slash-big {
  font-size: clamp(5rem, 14vw, 14rem);
  font-weight: 900;
  letter-spacing: -0.045em;
  line-height: 1;
  display: block;
}

.slash-caption {
  font-size: clamp(1rem, 2vw, 1.6rem);
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--muted);
  align-self: flex-end;
  padding-bottom: 0.18em;
}

.slash-dark .slash-caption {
  color: rgba(248, 248, 246, 0.48);
}

/* ── Inverted ticker ───────────────────────────────────────────── */
.ticker-wrap--inv {
  background: var(--fg);
  border-color: transparent;
}

.tape-inv {
  display: inline-flex;
  align-items: center;
  gap: 1.6rem;
  padding-right: 1.6rem;
  white-space: nowrap;
  animation: ticker-rev 26s linear infinite;
}

.tape-inv span {
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  font-family: var(--mono);
  color: var(--bg);
}

.tape-inv .sep {
  color: rgba(248, 248, 246, 0.28);
  font-size: 0.6rem;
}

@keyframes ticker-rev {
  0% {
    transform: translateX(-50%);
  }

  100% {
    transform: translateX(0);
  }
}

/* ── CTA ───────────────────────────────────────────────────────── */
.cta-section {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 2.5rem;
}

.cta-inner {
  padding: 8rem 0;
  display: flex;
  flex-direction: column;
  gap: 4rem;
}

.cta-big {
  display: flex;
  flex-direction: column;
}

.cta-line {
  display: block;
  overflow: hidden;
  line-height: 0.94;
}

.cta-line--flex {
  display: flex;
  gap: 0.25em;
  overflow: hidden;
}

.cta-word {
  display: block;
  font-size: clamp(4.5rem, 11vw, 10.5rem);
  font-weight: 900;
  letter-spacing: -0.045em;
  will-change: transform;
}

.cta-word--stroke {
  -webkit-text-stroke: 3px var(--fg);
  color: transparent;
}

.cta-actions {
  display: flex;
  align-items: center;
  gap: 2.5rem;
}

/* ── Footer ────────────────────────────────────────────────────── */
.site-footer {
  border-top: 1px solid var(--border);
  padding: 1.5rem 2.5rem;
}

.footer-inner {
  max-width: 1280px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  gap: 3rem;
}

.footer-brand {
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.22em;
}

.footer-nav {
  display: flex;
  gap: 2rem;
  flex: 1;
}

.footer-nav a {
  font-size: 0.8rem;
  color: var(--muted);
  transition: color 0.18s;
}

.footer-nav a:hover {
  color: var(--fg);
}

.footer-copy {
  font-size: 0.72rem;
  color: var(--muted);
  font-family: var(--mono);
  margin-left: auto;
}

/* ── Responsive ────────────────────────────────────────────────── */
@media (max-width: 960px) {
  .hero-inner {
    flex-direction: column;
    padding-top: 72px;
    gap: 2rem;
  }

  .hero-right {
    display: none;
  }

  .hero-bottom {
    flex-direction: column;
    align-items: flex-start;
    gap: 2rem;
  }

  .nav-mid {
    display: none;
  }

  .stats-inner {
    flex-direction: column;
  }

  .stat-item {
    padding: 2.8rem 0;
    border-bottom: 1px solid var(--border);
  }

  .stat-divider {
    display: none;
  }

  .grid-inner {
    grid-template-columns: 1fr 1fr;
  }

  .gi-wide {
    grid-column: span 2;
  }
}

@media (max-height: 780px) {
  .hero-giant {
    font-size: clamp(2.8rem, 7vw, 9rem);
  }

  .hero-section {
    padding-bottom: 2.8rem;
  }

  .hero-kicker {
    margin-bottom: 1.2rem;
  }

  .hero-giant {
    margin-bottom: 1.8rem;
  }
}

@media (max-width: 600px) {

  .hero-section,
  .manifesto-section,
  .stats-section,
  .phrase-section,
  .grid-section,
  .cta-section {
    padding-left: 1.4rem;
    padding-right: 1.4rem;
  }

  .grid-inner {
    grid-template-columns: 1fr;
  }

  .gi-wide {
    grid-column: span 1;
  }

  .h-slide {
    padding: 0 1.4rem 3.5rem;
  }

  .footer-inner {
    flex-wrap: wrap;
    gap: 1.2rem;
  }

  .footer-copy {
    margin-left: 0;
  }

  .cta-line--flex {
    flex-direction: column;
  }
}
</style>
