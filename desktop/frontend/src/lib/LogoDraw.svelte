<script lang="ts">
  // The mark as a wait. Logo.svelte is the A as a filled shape and can only
  // pop in; a loading card needs something that keeps moving for as long as
  // the wait does, and a generic ring says nothing about whose wait it is.
  // This is the same A traced along its centreline as three strokes (the
  // outer loop, the inner Λ, the crossbar with its leg), so it can be DRAWN:
  // the pen writes it, the letter stands a moment, unwrites, and starts
  // again (13 ก.ย. 2026 — a turn between the two was tried and taken out
  // the same day: "ไม่เอาหมุนแล้ว วาดก็พอ"). Same viewBox and stroke
  // weight (40) as the filled mark, so the two are the same letter at any
  // size — checked against the fill path, not the icon PNG, whose padding
  // differs.
  //
  // pathLength="1" on every stroke so one dash pattern fits all three; the
  // gap is 1.1, a little longer than the path, so that at offset 1.05 no
  // round cap peeks out at either end.
  let { size = 26 }: { size?: number } = $props()
</script>

<svg
  class="logo-draw" width={size} height={size} viewBox="0 0 804 762"
  fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"
>
  <path class="p1" pathLength="1" d="M 42 630 L 262 130 C 300 70 350 39 416 39 C 480 39 528 72 552 143 L 637 347 C 646 380 635 405 600 418 L 330 425 C 290 435 270 460 259 483 L 165 690 C 152 728 62 742 42 692 C 34 672 34 650 42 630 Z" />
  <path class="p2" pathLength="1" d="M 300 365 L 383 183 C 395 162 410 162 420 183 L 517 404" />
  <path class="p3" pathLength="1" d="M 330 528 L 555 528 C 572 528 576 545 582 562 L 623 660 C 640 725 745 750 763 655 L 680 445" />
</svg>

<style>
  .logo-draw { color: var(--text-primary); flex-shrink: 0; overflow: visible; }
  .logo-draw path {
    stroke: currentColor; stroke-width: 40; stroke-linecap: round; stroke-linejoin: round;
    stroke-dasharray: 1 1.1; stroke-dashoffset: 1.05;
  }
  /* One clock for the three pens, so they stay a beat apart: draw 0–52%
     (each stroke starts as the one before it is well under way), the whole
     letter stands until 68%, unwrites 68–88%, a breath, and again. */
  .logo-draw .p1 { animation: logo-draw-1 3s linear infinite; }
  .logo-draw .p2 { animation: logo-draw-2 3s linear infinite; }
  .logo-draw .p3 { animation: logo-draw-3 3s linear infinite; }
  @keyframes logo-draw-1 {
    0%        { stroke-dashoffset: 1.05; }
    38%, 68%  { stroke-dashoffset: 0; }
    88%, 100% { stroke-dashoffset: -1.05; }
  }
  @keyframes logo-draw-2 {
    0%, 24%   { stroke-dashoffset: 1.05; }
    42%, 68%  { stroke-dashoffset: 0; }
    88%, 100% { stroke-dashoffset: -1.05; }
  }
  @keyframes logo-draw-3 {
    0%, 34%   { stroke-dashoffset: 1.05; }
    52%, 68%  { stroke-dashoffset: 0; }
    88%, 100% { stroke-dashoffset: -1.05; }
  }
  /* Motion off: the blanket rule in style.css stops the pen, which would
     leave the letter at its resting offset — unwritten, an empty box. Show
     it whole instead; the card's own words say that something is loading. */
  @media (prefers-reduced-motion: reduce) {
    .logo-draw path { stroke-dashoffset: 0; }
  }
</style>
