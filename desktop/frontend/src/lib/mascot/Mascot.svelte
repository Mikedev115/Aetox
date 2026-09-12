<script lang="ts">
  // The mascot, drawn rather than stored — the robot the owner's sheets
  // describe, in the same frame AgentFace.svelte gives the cartoon person.
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
  import { resolveMascot, mascotSVG, lookRange, BRAND_HUE, type MascotOptions } from './rig'
  import { roleOptions } from './roles'
  import { lookAt } from './lookAt'
  import './mascot.css'

  let {
    name = undefined,
    role = undefined,
    icon = undefined,
    hue = undefined,
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
  } & Omit<MascotOptions, 'hue' | 'size'> & { hue?: number } = $props()

  // No name is the assistant, in the brand's own hue; a name is an agent, in
  // the hue its name gives it — the same rule the cartoon faces follow.
  const m = $derived(
    resolveMascot({ ...roleOptions(role, icon, { shell, top, badge, badgeR, face, prop }), hue: hue ?? (name ? coverHue(name) : BRAND_HUE), pose, size }),
  )
  const inner = $derived(mascotSVG(m))
  const rest = $derived(turn ?? m.pose.turn)
  // Nobody blinks on the beat: the interval is taken off the name, the same
  // way AgentFace does it, so two mascots side by side never sync up.
  const blink = $derived(4.4 + (m.hue % 9) * 0.3)
</script>

<span
  class="mascot pose-{m.poseId}"
  class:off
  class:sway
  class:hop
  class:settle={!look}
  style="--t:{rest}deg; --ms-blink:{blink}s; width:{size}px; height:{size}px"
  use:lookAt={{ base: rest, range: lookRange(m.pose), on: look }}
  aria-hidden="true"
>
  <!-- Our own markup out of rig.ts, never anything a user typed: a profile's
       fields choose a part by id, they do not carry one. -->
  <svg viewBox="0 0 64 64">{@html inner}</svg>
</span>
