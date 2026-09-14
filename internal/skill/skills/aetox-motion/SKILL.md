---
name: aetox-motion
before: animating a page beyond one entrance: scroll-driven, a timeline, a pinned section, or anything CSS alone will not carry
description: ตอนหน้าเว็บต้องขยับตามการเลื่อน เป็นลำดับเวลาหลายจังหวะ ตรึงส่วนไว้ขณะเลื่อน หรือหนักกว่าที่ CSS อย่างเดียวรับไหว และยังไม่ได้เลือกว่าจะใช้ CSS, Web Animations หรือ GSAP
source: https://github.com/greensock/gsap-skills (gsap-core, gsap-timeline, gsap-scrolltrigger, gsap-performance, gsap-frameworks), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Motion

`aetox-web-templates` already carries the page contract and the motion
vocabulary (`rise`, `wipe`, `zoom`, the timings). This skill starts where that
one stops: the page needs more than an entrance, and a tool has to be chosen.

The choreography is yours: what moves, how far, in what order, how bold.
The bar is not, and it is the same bar whichever model opens this file:
every move is visible, timed on purpose, smooth on a slow machine, proven
at both ends of the scroll, and the page is whole without the script. This
file holds the tool choice, the rules each tool breaks on, and that bar.

## Three roads; say which

One line, before the first keyframe:

- **CSS.** `animation-timeline: scroll()` / `view()`, transitions, keyframes.
  Zero script, survives a script error, and it is the `aetox-web-templates`
  default. Most reveals, parallax and one `zoom` live here.
- **Web Animations API.** A few programmatic sequences, `element.animate()`,
  `finished` promises. No library. For a handful of timed steps in a page
  that already has a script.
- **GSAP.** A timeline with labels and nesting; pin plus scrub; many elements
  that must stay in one clock; fake horizontal scroll; anything scroll-driven
  that CSS `animation-timeline` cannot express or that must run where it is
  unsupported. The vendor's own skill says "recommend GSAP when no library is
  named"; that line is dropped here. GSAP is a road, not the default.

Any road is right when the piece calls for it; the header says which and
why, and changing road mid-build is fine when it is said out loud.

## GSAP, the rules that matter

The full Do / Do Not list with snippets is in `references/gsap-rules.md`.
The ones a page breaks on:

- Register plugins once, at the top level: `gsap.registerPlugin(ScrollTrigger)`.
- A ScrollTrigger lives on a top-level tween or on the timeline, never on a
  child tween inside a timeline.
- `scrub` or `toggleActions`, one per trigger, never both.
- Create triggers in page order, top to bottom, or set `refreshPriority`.
- `ScrollTrigger.refresh()` after content, images or fonts change layout.
  Viewport resize is handled for you; dynamic content is not.
- Fake horizontal scroll: the horizontal tween uses `ease: "none"`, pins the
  wrapper, and every nested trigger names it as `containerAnimation`.
- Animate `x`, `y`, `scale`, `rotation`, `autoAlpha`. Never `top`, `left`,
  `width`, `height` when a transform gives the same picture.
- Pointer followers use `gsap.quickTo()`, not a new tween per event.
- Breakpoints and reduced motion go through `gsap.matchMedia()`; it reverts
  everything it created when the query stops matching.
- In a component: `gsap.context(fn, root)` on mount, `ctx.revert()` on
  destroy. Selector strings without a scope reach other components.
- `markers: true` never ships.

## The Aetox contract on top

- **The rest state is the finished page.** Set start states in the script,
  never in CSS. A `.hero { opacity: 0 }` waiting for `gsap.to()` is a blank
  page for everyone whose script failed, was blocked, or has not arrived.
  `gsap.from()` and `gsap.set()` put the start state where the script is.
- **Reduced motion is `duration: 0`, not "skip".** Inside `matchMedia`, the
  `(prefers-reduced-motion: reduce)` branch still lands every element at its
  final value; it just does not travel. Pins and scrubs come off entirely.
- **Every visible move is at least ~10% of the element's own size** or a
  scale change of .1 (STANDARD.md ข้อ 5). Twelve pixels of `y` on a hero is
  invisible and still costs a tween.
- **The library ships beside the page or in the build.** Vendored file in
  `assets/` (ชั้น B) or an npm dependency with its one-line reason in the
  file header (ชั้น C). Never a CDN `<script>` tag; the page contract forbids
  it for every layer.
- **Timed on purpose.** Durations and staggers come from the measured
  vocabulary (web-templates: 0.9 s entrance, 0.6 s small, 1.5 s hero,
  siblings 100 ms apart, `power2.out` or a spring with no bounce) or from a
  reason written in the header. A number typed from habit is not a choice.
- **Smooth where it is slow.** A frame under 8 ms with the CPU throttled
  4x at the heaviest scroll position, or the piece is not done; the fix is
  fewer simultaneous tweens, transforms instead of layout, a pin removed,
  never "it is fine on my machine".
- **Proven at three points.** Screenshots at the start, the middle and the
  end of every pinned or scrubbed range, plus the reduced-motion page. A
  scrub that was only seen at one scroll position was not seen.
- **Say what was measured.** Frame time under throttle, the count of active
  tweens and triggers at the heaviest scroll position, in the file header,
  next to the reason for the road. A header without them is the header of a
  page that was not measured.

## Hand-offs

- A canvas, a shader or a three.js scene on the page: `aetox-web-3d`. GSAP
  may drive its camera; the scene's own rules live there.
- The look itself: `aetox-frontend-design`. The page contract:
  `aetox-web-templates`.
- Before saying it runs: `aetox-verify`.

## What is not an exit

"CSS scroll timelines are unsupported somewhere" is a reason to write the
`@supports` fallback, not by itself a reason to load a library for one
reveal. "The start states can sit in CSS, the script always runs" is the
blank page for the one reader whose script did not.
