// The one folding transition in the app.
//
// Everything else here animates in CSS, and that is still the rule — but CSS
// cannot animate a *removal*: by the time the rule would apply, Svelte has
// already taken the element out of the document. A delegation finishing is
// exactly that case. The card was there, with its beam and its step list, and
// then it was not, in one frame, with nothing saying where it went (owner,
// 15 ส.ค.: "ตอนเอเจนมันทำงานเสร็จ ผมอยากให้ค่อย ๆ พับลง").
//
// Wrapped rather than used directly at each site for two reasons: the duration
// is one decision, not one per card, and `prefers-reduced-motion` has to be
// honoured — svelte/transition does not consult it, and the CSS half of the app
// already does (style.css).
import { slide } from 'svelte/transition'
import { cubicOut, cubicInOut } from 'svelte/easing'

// Read per call rather than once at module load: the setting can change while
// the app is open, and a value captured at startup would need a restart to take
// effect. Guarded for the test environment, where matchMedia does not exist.
//
// Exported for the one caller that DRIVES a fold instead of declaring it: the
// beat Chat.svelte holds before shutting a phase is the same decision as the
// fold's own length, and asked to stop moving the app must not answer with a
// pause instead of a movement.
export function motionStill(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

const FOLD_MS = 240

/** Folds an element open and shut by its own height. `transition:fold`.
 *
 * `delay` is for one job and it is worth naming: staging an arrival in more
 * than one beat. A delegation card is drawn the instant the model hands work
 * over — that part is not allowed to wait — but what it was TOLD arrives a beat
 * behind the person it was told to, because that is the order the event happens
 * in and the order a reader takes it in (owner, 7 ก.ย.). It rides here rather
 * than on a second timer so the reduced-motion guard covers it too: asked to
 * stop moving, the app must not answer with a pause instead. */
export function fold(
  node: Element,
  { duration = FOLD_MS, delay = 0 }: { duration?: number; delay?: number } = {},
) {
  const still = motionStill()
  return slide(node, {
    duration: still ? 0 : duration,
    delay: still ? 0 : delay,
    easing: cubicOut,
  })
}

/** How long a stretch of work takes to fold away. */
export const SETTLE_MS = 520

/** The fold a stretch of work does when its last call comes back.
 * `transition:settle`.
 *
 * A separate transition from `fold` and not a parameter of it, because every
 * one of its three differences is the same decision taken again: this movement
 * is allowed to be *seen*, where a fold is a report that gets out of the way.
 *
 *  - **Twice the time.** The owner asked for นุ่มนวล four times before this was
 *    right, and 240ms was the last thing still answering him with a flick.
 *  - **Eased at both ends.** `fold` uses cubicOut, which on an OUTRO hangs and
 *    then drops — the shape reads as a decision made late. cubicInOut starts
 *    slow, moves, and arrives slowly, which is the whole of what "gently"
 *    means to an eye.
 *  - **It fades as it goes.** The height alone clips the list against the row
 *    below it, and a clip is a cut however long you take over it. Fading it out
 *    on the way down is what turns the last rows into something that LEFT
 *    rather than something that was trimmed off.
 *
 * `gap` is the container's, not the node's: a flex parent holds its whole gap
 * for as long as the child exists, so a block folding to nothing inside one
 * still owns its share of the row until the frame it is removed — the last few
 * pixels of the gentlest possible fold land as a jump. The margin cancels it
 * and unwinds alongside the height, the same way unroll cancels the composer's.
 */
export function settle(
  node: Element,
  { duration = SETTLE_MS, gap = 0 }: { duration?: number; gap?: number } = {},
) {
  // Measured before the transition starts, which for an outro is while the
  // block is still at full size — the one moment its real height can be read.
  const height = (node as HTMLElement).offsetHeight
  return {
    duration: motionStill() ? 0 : duration,
    easing: cubicInOut,
    css: (t: number) =>
      `overflow:hidden; height:${t * height}px; opacity:${t}; margin-bottom:${(t - 1) * gap}px;`,
  }
}

/** Opens an attachment from its top edge and shuts it the same way.
 * `transition:unroll`.
 *
 * The height IS the reveal: the card clips its own contents, so the picture and
 * the name arrive at full size and full sharpness from the first frame. That is
 * the whole reason this is not a scale — a card scaled up from 55% renders its
 * 11px filename at fractional sizes on the way, and the eye reads that as a
 * blurry picture rather than as movement (owner rejected exactly that, 25 ส.ค.,
 * and picked this one: "ม่านเปิดลง").
 *
 * What `slide` could not do is the composer's `gap:10px`. A gap belongs to the
 * container and appears the instant the child exists, so the whole box jumped
 * 10px before the card had drawn a single pixel — the jolt was still there no
 * matter how gently the card itself arrived. The negative margin cancels it and
 * unwinds with the height, so the space and the card are one movement rather
 * than two events a frame apart.
 */
export function unroll(
  node: Element,
  // 560ms is the owner's, picked at 2x on the live comparison and not a typo:
  // this is the slowest movement in the app on purpose. Every other transition
  // here reports something the app did, and gets out of the way; this one is
  // the app acknowledging something the user did, and it is the only frame in
  // which they can check that the right picture went in.
  { duration = 560, gap = 10 }: { duration?: number; gap?: number } = {},
) {
  // Measured before the transition starts, which for an outro is while the card
  // is still at full size — the one moment its real height can be read.
  const height = (node as HTMLElement).offsetHeight
  return {
    duration: motionStill() ? 0 : duration,
    easing: cubicOut,
    css: (t: number) => `overflow:hidden; height:${t * height}px; margin-bottom:${(t - 1) * gap}px;`,
  }
}
