# GSAP rules, condensed from the vendor's skills

Source: greensock/gsap-skills (MIT), July 2026. GSAP 3.13+, every plugin free
since the Webflow acquisition; nothing needs a Club token or a private
registry. Only the rules a page actually breaks on are here; the vendor's
files carry the full API tables.

## Setup

```js
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
gsap.registerPlugin(ScrollTrigger);   // once, at app level
```

Vendored (ชั้น B): copy `gsap.min.js` and `ScrollTrigger.min.js` into
`assets/` and load them with plain `<script src="assets/...">` before the
page script. The globals are `gsap` and `ScrollTrigger`.

## Tweens

- `gsap.to()`, `gsap.from()`, `gsap.fromTo()`. Vars in camelCase.
- Transform aliases: `x`, `y`, `xPercent`, `yPercent`, `scale`, `rotation`.
  `autoAlpha` over `opacity` when the element must also stop taking clicks.
- Stacking two `from()` on one property of one target: the later one needs
  `immediateRender: false`, or the first is overwritten before it plays.
- Prefer a timeline to chaining `delay`s.
- Store the returned tween when you will pause, reverse or kill it.

## Timeline

```js
const tl = gsap.timeline({ defaults: { duration: 0.9, ease: "power2.out" } });
tl.from(".w-hero h1", { y: 50, autoAlpha: 0 })
  .from(".w-hero p",  { y: 20, autoAlpha: 0 }, "-=0.6")   // position parameter
  .addLabel("cta")
  .from(".w-hero a",  { y: 20, autoAlpha: 0 }, "cta");
```

- `defaults` on the timeline when children share duration or ease.
- Position parameter: `"-=0.6"`, `"<"` (with previous), a label.
- A timeline's duration is its children's; the constructor takes none.

## ScrollTrigger

```js
gsap.timeline({
  scrollTrigger: {
    trigger: ".w-story",
    start: "top top",
    end: "+=2000",
    pin: true,
    scrub: 1,          // seconds of catch-up; true = locked to scroll
  },
}).to(".w-story .a", { x: 100 }).to(".w-story .b", { y: 50 });
```

- On the timeline or a top-level tween only. Not on a child tween, not on
  a timeline nested in another timeline.
- `scrub` xor `toggleActions`. If both, scrub wins and the other is noise.
- `start` / `end`: `"triggerEdge viewportEdge"`, `"+=px"`, `"+=100%"`,
  `"max"`, or `clamp(top bottom)` to stay inside the page.
- `pin: true` pins the trigger; animate its children, not the pinned box.
  `pinSpacing: false` only when the layout handles the gap itself.
- Create top to bottom in page order. Async or out-of-order creation needs
  `refreshPriority` (lower refreshes first, first section = lowest).
- `ScrollTrigger.refresh()` after layout-changing content. Resize is
  automatic (debounced 200 ms).
- Many similar reveals: `ScrollTrigger.batch(".w-card", { onEnter: els =>
  gsap.to(els, { y: 0, autoAlpha: 1, stagger: 0.1, overwrite: true }) })`.
- Custom smooth scrolling library: `ScrollTrigger.scrollerProxy()` plus
  `lib.addListener(ScrollTrigger.update)`, or ScrollTrigger reads a stale
  position.

Fake horizontal scroll:

```js
const track = document.querySelector(".w-track");
const move = gsap.to(track, {
  x: () => -(track.scrollWidth - window.innerWidth),
  ease: "none",                                   // required
  scrollTrigger: {
    trigger: track, pin: track.parentNode,
    start: "top top", end: () => "+=" + track.scrollWidth,
    scrub: true, invalidateOnRefresh: true,
  },
});
gsap.from(".w-track .panel-3 h2", {
  y: 40, autoAlpha: 0,
  scrollTrigger: { containerAnimation: move, trigger: ".w-track .panel-3",
                   start: "left center" },
});
```

No pin and no snap on a trigger that uses `containerAnimation`.

## Responsive and reduced motion

```js
const mm = gsap.matchMedia();
mm.add({
  wide:   "(min-width: 800px)",
  reduce: "(prefers-reduced-motion: reduce)",
}, (ctx) => {
  const { wide, reduce } = ctx.conditions;
  gsap.from(".w-hero h1", { y: reduce ? 0 : 50, autoAlpha: 0,
                            duration: reduce ? 0 : 0.9 });
  if (!reduce) { /* pins, scrubs, loops */ }
  return () => {};   // extra cleanup, optional
});
```

Everything created inside the handler is reverted when the query stops
matching. Do not wrap a `gsap.context()` inside it; it already is one.
`mm.revert()` on unmount.

## Components (Svelte shown; Vue and React are the same shape)

```svelte
<script>
  import { onMount } from "svelte";
  import { gsap } from "gsap";
  let root;
  onMount(() => {
    const ctx = gsap.context(() => {
      gsap.from(".item", { autoAlpha: 0, stagger: 0.1 });   // scoped to root
    }, root);
    return () => ctx.revert();
  });
</script>
<div bind:this={root}> ... </div>
```

- Create after mount; the nodes do not exist earlier.
- Always pass the root as scope; bare selectors reach the whole page.
- `ctx.revert()` kills tweens and ScrollTriggers and removes inline styles.
- React: `useGSAP()` from `@gsap/react` does the same.

## Performance

- Transforms and opacity stay on the compositor; layout properties do not.
- `will-change: transform` on the elements that animate, not on everything.
- `gsap.quickTo()` for pointer followers and anything updated per event.
- `stagger` over many tweens with hand-set delays.
- Pin only what needs pinning; each pin promotes a layer.
- Kill or pause what is off-screen or on a page you left.

## Do not

- Leave `markers: true` in.
- Put a ScrollTrigger inside a nested timeline.
- Use `scrub` and `toggleActions` together.
- Use an ease other than `"none"` on a `containerAnimation` tween.
- Forget `refresh()` after images or fonts change the layout.
- Register plugins inside a component body that runs on every render.
- Animate `width` / `height` / `top` / `left` for movement.
