<script lang="ts">
  // The mascot, drawn rather than stored — the robot the owner's sheets
  // describe, in the frame the cartoon person (until 12 ก.ย. 2026) stood in.
  //
  // Same contract as that component on purpose: a name in, a face out, with
  // nothing read from disk. The hue comes off the name (coverHue) unless a
  // profile names one, the ear badge is an ICONS id the profile already
  // declares (`icon:`), the template the slots start from is a `role`
  // (roles.ts), the body's finish a `shell` (palette.ts), and what the mascot
  // is DOING arrives as `pose`, which is presence's to set and never a
  // profile's. Every part lives in parts.ts, every pose in poses.ts, every
  // movement in mascot.css; this file is the frame and the one place --t is
  // written.
  //
  // `look` turns on mouse-follow (lookAt.ts) — for the one mascot the user is
  // facing, never for a roster of tiles.
  import { coverHue } from '../coverHue'
  import { fade } from 'svelte/transition'
  import { resolveMascot, mascotSVG, lookRange, handVars, DETAIL_MIN_PX, type MascotOptions } from './rig'
  import { roleOptions } from './roles'
  import { lookAt } from './lookAt'
  import './mascot.css'

  let {
    name = undefined,
    role = undefined,
    icon = undefined,
    hue = undefined,
    accent = undefined,
    shell = undefined,
    top = undefined,
    badge = undefined,
    badgeR = undefined,
    face = undefined,
    prop = undefined,
    pose = 'idle',
    turn = undefined,
    size = 38,
    off = false,
    look = false,
    sway = false,
    hop = false,
    still = false,
    snap = false,
    glow = undefined,
  }: {
    name?: string
    /** A ROLE row id (roles.ts) — the template the slots start from. */
    role?: string
    /** An agent's `icon:` — worn on both ears over the role's badge. */
    icon?: string
    size?: number
    /** Drained of colour — an agent the assistant may not hand work to. */
    off?: boolean
    /** Degrees, overriding the pose's rest angle. */
    turn?: number
    /** Follow the pointer within the pose's look range. */
    look?: boolean
    /** Sway in place instead — for the one that sits on the screen. */
    sway?: boolean
    /** One small hop — set for the moment of a reaction. */
    hop?: boolean
    /** No motion at all — for a picker's cells and a roster's tiles, where
     *  twenty of these breathing together is a page that stutters. */
    still?: boolean
    /** Take `turn` as given, without easing towards it — for a caller that
     *  is already moving the head itself, every event, and for the one
     *  frame it needs to rewind a wound-up angle by a full circle unseen. */
    snap?: boolean
  } & Omit<MascotOptions, 'hue' | 'size'> & { hue?: number } = $props()

  // No name is the assistant, in the accent it was given or the mark's own
  // white and grey; a name is an agent, at full colour in the hue its name
  // gives it — the same rule the cartoon faces followed — unless its file
  // named an accent, which a persona handed to it would (a hue in degrees
  // still wins over both, as in rig.ts).
  const m = $derived(
    resolveMascot({ ...roleOptions(role, icon, { accent, shell, top, badge, badgeR, face, prop }), hue: hue ?? (name && !accent ? coverHue(name) : undefined), pose, size, glow }),
  )
  const inner = $derived(mascotSVG(m))
  const rest = $derived(turn ?? m.pose.turn)
  // A change of pose is a change of markup, and a redraw would snap. So the
  // drawings crossfade — the old one fades over the new for a moment — while
  // the head turn and the hand targets, which live on the root and outlive
  // the redraw, glide (mascot.css .settle). Each drawing carries the moment
  // it was made as --ms-phase, so its loops continue the last drawing's.
  const CROSS_MS = 240
  const reduced = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
  const cross = $derived(still || reduced ? 0 : CROSS_MS)
  const phase = $derived.by(() => {
    void inner
    return typeof performance === 'object' ? -(performance.now() / 1000) : 0
  })
  // Nobody blinks on the beat: the interval is taken off the hue, so two
  // mascots side by side never sync up.
  const blink = $derived(4.4 + (m.hue % 9) * 0.3)
</script>

<span
  class="mascot pose-{m.poseId}"
  class:off
  class:sway
  class:hop
  class:still
  class:lite={size < DETAIL_MIN_PX}
  class:settle={!look && !snap}
  style="--t:{rest}deg; {handVars(m)}; --ms-blink:{blink}s; width:{size}px; height:{size}px"
  use:lookAt={{ base: rest, range: lookRange(m.pose), on: look }}
  aria-hidden="true"
>
  <!-- Our own markup out of rig.ts, never anything a user typed: a profile's
       fields choose a part by id, they do not carry one. -->
  {#key inner}
    <svg viewBox="0 0 64 64" style="--ms-phase:{phase}s" transition:fade={{ duration: cross }}>{@html inner}</svg>
  {/key}
</span>

<style>
  /* The two drawings of a crossfade sit on top of each other. */
  .mascot :global(svg) { position: absolute; inset: 0; }
</style>
