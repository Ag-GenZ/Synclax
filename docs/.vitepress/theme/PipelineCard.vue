<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { gsap } from 'gsap'

const root = ref<HTMLElement | null>(null)
let ctx: gsap.Context | undefined

onMounted(() => {
  ctx = gsap.context(() => {
    const el = root.value!
    gsap.set(el, { autoAlpha: 0 })

    const mm = gsap.matchMedia()

    mm.add('(prefers-reduced-motion: no-preference)', () => {
      // ─── refs ──────────────────────────────────────────────────
      const time = el.querySelector<HTMLElement>('.ft-time')!
      const ruleTop = el.querySelector<HTMLElement>('.rule-top')!
      const ruleBot = el.querySelector<HTMLElement>('.rule-bot')!

      // content layer refs
      const kicker = el.querySelector<HTMLElement>('.c-kicker')!
      const lineA = el.querySelector<HTMLElement>('.c-line-a')!
      const lineB = el.querySelector<HTMLElement>('.c-line-b')!
      const sub = el.querySelector<HTMLElement>('.c-sub')!
      const listEl = el.querySelector<HTMLElement>('.c-list')!
      const listItems = gsap.utils.toArray<HTMLElement>('.c-list-item', el)
      const bigNum = el.querySelector<HTMLElement>('.c-bignum')!
      const bigTag = el.querySelector<HTMLElement>('.c-bigtag')!
      const codeBlk = el.querySelector<HTMLElement>('.c-code')!
      const codeLines = gsap.utils.toArray<HTMLElement>('.c-code-line', el)
      const diffStat = el.querySelector<HTMLElement>('.c-diff')!
      const counter = el.querySelector<HTMLElement>('.c-counter')!
      const counterSub = el.querySelector<HTMLElement>('.c-counter-sub')!
      const barWrap = el.querySelector<HTMLElement>('.c-bar-wrap')!
      const barFill = el.querySelector<HTMLElement>('.c-bar-fill')!
      const result = el.querySelector<HTMLElement>('.c-result')!
      const resultSub = el.querySelector<HTMLElement>('.c-result-sub')!
      const statusDot = el.querySelector<HTMLElement>('.c-status-dot')!

      // ─── hide all content immediately before entrance ──────────
      const allContent = [kicker, lineA, lineB, sub, listEl, bigNum, bigTag,
        codeBlk, diffStat, counter, counterSub, barWrap, result, resultSub, statusDot]
      gsap.set(allContent, { autoAlpha: 0 })
      gsap.set(listItems, { autoAlpha: 0 })
      gsap.set(codeLines, { autoAlpha: 1 })
      codeLines.forEach(l => { l.textContent = '' })
      gsap.set(barFill, { width: '0%' })

      // ─── entrance ──────────────────────────────────────────────
      const enter = gsap.timeline({ delay: 1.4 })
      enter
        .fromTo(el, { y: 24, autoAlpha: 0 }, { y: 0, autoAlpha: 1, duration: 0.85, ease: 'power3.out' })
        .from([ruleTop, ruleBot],
          { scaleX: 0, transformOrigin: 'left center', duration: 0.65, ease: 'power3.inOut', stagger: 0.1 },
          '-=0.4')
        .from('.ft', { y: 8, autoAlpha: 0, duration: 0.3, ease: 'power2.out' }, '-=0.2')

      // ─── float ─────────────────────────────────────────────────
      gsap.to(el, { y: '-=5', duration: 4.8, repeat: -1, yoyo: true, ease: 'sine.inOut', delay: 3 })

      // ─── typewriter util ───────────────────────────────────────
      const tw = (elem: HTMLElement, text: string, dur: number): gsap.core.Tween => {
        elem.textContent = ''
        const o = { n: 0 }
        return gsap.to(o, {
          n: text.length, duration: dur, ease: 'none',
          onUpdate() { elem.textContent = text.slice(0, Math.round(o.n)) },
        })
      }

      // ─── rule flash ────────────────────────────────────────────
      const flash = (rule: HTMLElement) =>
        gsap.timeline()
          .to(rule, { scaleX: 0, transformOrigin: 'right center', duration: 0.14, ease: 'power2.in' })
          .set(rule, { transformOrigin: 'left center' })
          .to(rule, { scaleX: 1, duration: 0.38, ease: 'expo.out' })

      // ─── elapsed timer ─────────────────────────────────────────
      let elapsedTween: gsap.core.Tween | null = null
      const startElapsed = (dur: number) => {
        const o = { t: 0 }
        elapsedTween?.kill()
        elapsedTween = gsap.to(o, {
          t: dur, duration: dur, ease: 'none',
          onUpdate() { time.textContent = o.t.toFixed(1) + ' s' },
        })
      }

      // ══════════════════════════════════════════════════════════════
      //  THE FLOW  — one master timeline, never pauses, loops forever
      //  Elements are always on screen, morphing from one state to next
      // ══════════════════════════════════════════════════════════════
      const buildFlow = (): gsap.core.Timeline => {
        // hard-set everything invisible at start
        const allContent = [kicker, lineA, lineB, sub, listEl, bigNum, bigTag,
          codeBlk, diffStat, counter, counterSub, barWrap, result, resultSub, statusDot]
        allContent.forEach(e => gsap.set(e, { autoAlpha: 0, y: 0, x: 0, scale: 1, rotation: 0 }))
        listItems.forEach(e => gsap.set(e, { autoAlpha: 0, x: 0 }))
        codeLines.forEach(e => { gsap.set(e, { autoAlpha: 1 }); e.textContent = '' })
        gsap.set(barFill, { width: '0%' })
        time.textContent = '0.0 s'
        startElapsed(22)

        const tl = gsap.timeline()

        // ── 0.0 s  ─  kicker "// incoming" fades in ──────────────
        tl.to(kicker, { autoAlpha: 1, y: 0, duration: 0.5, ease: 'power3.out' }, 0.3)
        tl.add(tw(kicker, '// incoming', 0.55), 0.32)

        // ── 0.6 s  ─  "Issue" slides up big ──────────────────────
        tl.fromTo(lineA,
          { autoAlpha: 0, y: 20 },
          { autoAlpha: 1, y: 0, duration: 0.6, ease: 'power3.out' }, 0.65)

        // ── 0.9 s  ─  "#2847" slides up, lighter ─────────────────
        tl.fromTo(lineB,
          { autoAlpha: 0, y: 18 },
          { autoAlpha: 1, y: 0, duration: 0.55, ease: 'power3.out' }, 0.9)

        // ── 1.1 s  ─  branch types in below ──────────────────────
        tl.set(sub, { autoAlpha: 1 }, 1.15)
        tl.add(tw(sub, 'feature/add-oauth → workspace', 0.75), 1.17)

        // ── 2.4 s  ─  rule flash; Issue/branch morph into plan ───
        tl.add(flash(ruleTop), 2.5)
        tl.to([lineA, lineB], { y: -22, autoAlpha: 0, duration: 0.42, ease: 'power3.in', stagger: 0.06 }, 2.55)
        tl.to(sub, { autoAlpha: 0, y: -10, duration: 0.3, ease: 'power2.in' }, 2.6)
        tl.add(tw(kicker, '// strategy', 0.45), 2.6)

        // plan items stagger in from left—feels like a list materializing
        listItems.forEach((item, i) => {
          tl.fromTo(item,
            { autoAlpha: 0, x: -20 },
            { autoAlpha: 1, x: 0, duration: 0.4, ease: 'power3.out' },
            2.85 + i * 0.13)
        })
        tl.to(listEl, { autoAlpha: 1, duration: 0 }, 2.84)

        // ── 4.2 s  ─  plan collapses; [w-03] pops in center ──────
        tl.add(flash(ruleTop), 4.3)
        tl.to(listEl, { autoAlpha: 0, scale: 0.9, duration: 0.4, ease: 'power3.in' }, 4.35)
        tl.add(tw(kicker, '// spawn', 0.4), 4.4)
        tl.fromTo(bigNum,
          { autoAlpha: 0, scale: 0.3, rotation: -8 },
          { autoAlpha: 1, scale: 1, rotation: 0, duration: 0.7, ease: 'back.out(2.2)' }, 4.7)
        tl.fromTo(bigTag,
          { autoAlpha: 0, y: 10 },
          { autoAlpha: 1, y: 0, duration: 0.4, ease: 'power2.out' }, 5.15)

        // ── 5.8 s  ─  worker shrinks to corner; files type in ─────
        tl.add(flash(ruleTop), 5.9)
        tl.add(tw(kicker, '// writing', 0.4), 5.95)
        // worker drifts to a smaller ghost in top-right
        tl.to(bigNum, { scale: 0.38, x: 80, y: -50, autoAlpha: 0.18, duration: 0.6, ease: 'power3.inOut' }, 5.95)
        tl.to(bigTag, { autoAlpha: 0, duration: 0.3 }, 5.95)

        // code block appears, each line typed sequentially
        tl.set(codeBlk, { autoAlpha: 1 }, 6.2)
        const filenames = ['src/auth/oauth.go', 'src/auth/token.go', 'src/auth/middleware.go']
        codeLines.forEach((line, i) => {
          tl.add(tw(line, filenames[i], 0.5), 6.3 + i * 0.62)
        })

        // diff stat wipes in
        tl.fromTo(diffStat,
          { autoAlpha: 0, clipPath: 'inset(0 100% 0 0)' },
          { autoAlpha: 1, clipPath: 'inset(0 0% 0 0)', duration: 0.55, ease: 'power3.inOut' }, 8.1)

        // ── 9.0 s  ─  files slide away; counter rolls up ──────────
        tl.add(flash(ruleTop), 9.1)
        tl.add(tw(kicker, '// test suite', 0.45), 9.15)
        tl.to(bigNum, { autoAlpha: 0, duration: 0.3 }, 9.1)
        tl.to(codeBlk, { autoAlpha: 0, y: -12, duration: 0.4, ease: 'power3.in' }, 9.15)
        tl.to(diffStat, { autoAlpha: 0, duration: 0.28 }, 9.2)

        // counter rises from 0
        tl.fromTo(counter,
          { autoAlpha: 0, y: 18 },
          { autoAlpha: 1, y: 0, duration: 0.45, ease: 'power3.out' }, 9.5)
        const cObj = { v: 0 }
        tl.to(cObj, {
          v: 47, duration: 1.4, ease: 'power2.out',
          onUpdate() { counter.textContent = Math.round(cObj.v) + ' / 51' },
        }, 9.55)
        tl.fromTo(counterSub,
          { autoAlpha: 0, x: -10 },
          { autoAlpha: 1, x: 0, duration: 0.38, ease: 'power3.out' }, 9.72)

        // bar sweeps across
        tl.set(barWrap, { autoAlpha: 1 }, 9.8)
        tl.to(barFill, { width: '92%', duration: 1.4, ease: 'power2.out' }, 9.85)

        // ── 12.5 s  ─  tests fade; PR result rises ────────────────
        tl.add(flash(ruleTop), 12.6)
        tl.add(tw(kicker, '// result', 0.4), 12.65)
        tl.to([counter, counterSub], { y: -18, autoAlpha: 0, duration: 0.38, ease: 'power3.in', stagger: 0.05 }, 12.65)
        tl.to(barWrap, { autoAlpha: 0, duration: 0.3 }, 12.7)

        // status dot pulses in
        tl.fromTo(statusDot,
          { autoAlpha: 0, scale: 0 },
          { autoAlpha: 1, scale: 1, duration: 0.5, ease: 'back.out(3)' }, 12.9)

        // "PR #248" rises from where counter was — same spatial position = visual continuity
        tl.fromTo(result,
          { autoAlpha: 0, y: 22 },
          { autoAlpha: 1, y: 0, duration: 0.6, ease: 'power3.out' }, 13.05)

        tl.set(resultSub, { autoAlpha: 1 }, 13.5)
        tl.add(tw(resultSub, '→ ready for review', 0.85), 13.52)

        // ── 16.5 s  ─  everything fades out softly, loop restarts ─
        tl.to(allContent, {
          autoAlpha: 0, duration: 0.7, ease: 'power2.in', stagger: 0.04,
        }, 16.5)

        return tl
      }

      const loop = () => {
        const tl = buildFlow()
        tl.eventCallback('onComplete', () => { gsap.delayedCall(0.8, loop) })
      }

      enter.eventCallback('onComplete', () => { gsap.delayedCall(0.3, loop) })
    })

    mm.add('(prefers-reduced-motion: reduce)', () => {
      gsap.set(el, { autoAlpha: 1 })
    })
  }, root.value!)
})

onUnmounted(() => ctx?.revert())
</script>

<template>
  <div ref="root" class="stage">

    <!-- header: just label + elapsed -->
    <div class="hd">
      <span class="hd-caption">// running</span>
      <span class="hd-rule-spacer"></span>
      <span class="ft-time hd-time">0.0 s</span>
    </div>

    <div class="rule rule-top"></div>

    <!-- single content layer — all elements coexist, GSAP controls visibility -->
    <div class="vp">

      <!-- kicker — rewritten by typewriter each phase -->
      <div class="c-kicker mono"></div>

      <!-- Issue title (phases 1) -->
      <div class="c-line-a big"></div>
      <div class="c-line-b big dim"></div>

      <!-- branch subtitle -->
      <div class="c-sub mono"></div>

      <!-- plan list (phase 2) -->
      <ul class="c-list">
        <li class="c-list-item"><span class="list-k">a.</span><span>Analyze scope</span></li>
        <li class="c-list-item"><span class="list-k">b.</span><span>Select toolchain</span></li>
        <li class="c-list-item"><span class="list-k">c.</span><span>Map dependencies</span></li>
      </ul>

      <!-- worker (phase 3) -->
      <div class="c-bignum">[w-03]</div>
      <div class="c-bigtag mono">agent ready</div>

      <!-- code (phase 4) -->
      <div class="c-code">
        <div class="c-code-line file"></div>
        <div class="c-code-line file"></div>
        <div class="c-code-line file"></div>
      </div>
      <div class="c-diff mono">+214 −31</div>

      <!-- test (phase 5) -->
      <div class="c-counter">0 / 51</div>
      <div class="c-counter-sub mono">passing</div>
      <div class="c-bar-wrap">
        <div class="c-bar">
          <div class="c-bar-fill"></div>
        </div>
      </div>

      <!-- result (phase 6) -->
      <div class="c-status-dot"></div>
      <div class="c-result big">PR #248</div>
      <div class="c-result-sub mono"></div>

    </div>

    <div class="rule rule-bot"></div>

    <div class="ft">
      <span class="ft-branch">feature/add-oauth</span>
      <span class="ft-time">0.0 s</span>
    </div>

  </div>
</template>

<style scoped>
.stage {
  --fg: #0a0a0a;
  --bg: #f8f8f6;
  --muted: rgba(10, 10, 10, 0.42);
  --border: rgba(10, 10, 10, 0.1);
  --mono: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;

  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  visibility: hidden;
  will-change: transform, opacity;
}

/* ─── header ──────────────────────────────────────── */
.hd {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  padding-bottom: 0.85rem;
  flex-shrink: 0;
}

.hd-caption {
  font-size: 0.65rem;
  font-weight: 600;
  font-family: var(--mono);
  color: var(--muted);
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.hd-rule-spacer {
  flex: 1;
}

.hd-time {
  font-size: 0.65rem;
  font-family: var(--mono);
  color: var(--muted);
}

/* ─── rule ────────────────────────────────────────── */
.rule {
  height: 1px;
  background: var(--border);
  flex-shrink: 0;
  will-change: transform;
}

/* ─── viewport ────────────────────────────────────── */
.vp {
  flex: 1;
  min-height: 0;
  position: relative;
  overflow: hidden;
  margin: 1.2rem 0;
}

/* All content children sit in the same space, GSAP controls them */
.vp>* {
  position: absolute;
  left: 0;
  will-change: transform, opacity;
  opacity: 0;
  visibility: hidden;
}

/* ─── vertical rhythm inside vp ──────────────────── */
/* kicker always at top */
.c-kicker {
  top: 0;
}

/* main title block — vertically centered */
.c-line-a {
  top: 30%;
  transform: translateY(-100%);
}

.c-line-b {
  top: 30%;
  transform: translateY(10%);
}

.c-sub {
  top: calc(30% + 3.6rem + 0.6rem);
}

/* plan list — centered */
.c-list {
  top: 50%;
  transform: translateY(-50%);
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  padding: 0;
  margin: 0;
  width: 100%;
}

/* worker — centered */
.c-bignum {
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  white-space: nowrap;
}

.c-bigtag {
  top: calc(50% + 2.8rem);
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
}

/* code block */
.c-code {
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 0.32rem;
  width: 100%;
}

.c-diff {
  bottom: 16%;
}

/* test counter */
.c-counter {
  top: 38%;
  transform: translateY(-50%);
}

.c-counter-sub {
  top: calc(38% + 2.8rem);
}

.c-bar-wrap {
  bottom: 22%;
  left: 0;
  right: 0;
}

/* result */
.c-status-dot {
  top: 25%;
  left: 50%;
  transform: translateX(-50%);
}

.c-result {
  top: 38%;
  left: 50%;
  transform: translate(-50%, -50%);
  white-space: nowrap;
}

.c-result-sub {
  top: calc(38% + 2.8rem);
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
}

/* ─── typography ──────────────────────────────────── */
.big {
  font-size: clamp(2.6rem, 4vw, 4rem);
  font-weight: 900;
  color: var(--fg);
  letter-spacing: -0.04em;
  line-height: 1;
}

.dim {
  color: var(--muted);
}

.mono {
  font-size: 0.8rem;
  font-family: var(--mono);
  color: var(--muted);
  letter-spacing: 0.01em;
}

/* plan list */
.c-list-item {
  display: flex;
  align-items: baseline;
  gap: 0.55rem;
  font-size: 0.98rem;
  font-weight: 500;
  color: var(--fg);
}

.list-k {
  font-size: 0.68rem;
  font-family: var(--mono);
  color: var(--muted);
  font-weight: 700;
  flex-shrink: 0;
}

/* worker bracket */
.c-bignum {
  font-size: clamp(3.2rem, 5.5vw, 5rem);
  font-weight: 900;
  font-family: var(--mono);
  color: var(--fg);
  letter-spacing: -0.02em;
  line-height: 1;
}

/* code file lines */
.file {
  font-size: 0.82rem;
  font-family: var(--mono);
  color: var(--fg);
  letter-spacing: 0.01em;
  min-height: 1.2em;
}

/* test counter big number */
.c-counter {
  font-size: clamp(3rem, 5vw, 4.8rem);
  font-weight: 900;
  font-family: var(--mono);
  color: var(--fg);
  letter-spacing: -0.04em;
  line-height: 1;
}

/* progress bar */
.c-bar {
  height: 2px;
  background: var(--border);
  overflow: hidden;
}

.c-bar-fill {
  height: 100%;
  width: 0%;
  background: var(--fg);
}

/* status dot */
.c-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--fg);
  will-change: transform, opacity;
}

/* result PR number — same size as Issue for visual rhyme */
.c-result {
  font-size: clamp(2.6rem, 4vw, 4rem);
  font-weight: 900;
  color: var(--fg);
  letter-spacing: -0.04em;
  line-height: 1;
}

/* ─── footer ─────────────────────────────────────── */
.ft {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding-top: 0.85rem;
  flex-shrink: 0;
}

.ft-branch,
.ft-time {
  font-size: 0.62rem;
  font-family: var(--mono);
  color: var(--muted);
  letter-spacing: 0.02em;
}
</style>
