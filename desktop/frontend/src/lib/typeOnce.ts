/** The brief, written out once, at the moment somebody is handed it.
 *
 * NOT the pacer. `pacedText` was the obvious reach and it is the wrong tool:
 * its first rule is that the first sight of a block is shown WHOLE, never typed
 * (streamPace.ts, `feed`), because that text arrived over a network in bursts
 * and replaying the buffer letter by letter is "a story about the network, told
 * late". That rule is right and this is not the case it is about. A brief is
 * not streaming — it exists complete the instant the delegation starts. What is
 * being staged here is the HANDOVER, which genuinely is happening now.
 *
 * So: a fixed one-shot, and only ever for a delegation that started in front of
 * the reader. A card read back out of the database has its brief the moment it
 * is drawn, because nothing is being handed to anybody — the same line
 * .reasoning-body.live draws between a thing in motion and the record of it.
 *
 * `on` going false finishes it immediately rather than cancelling it. That is
 * the whole safety of the effect: the delegate is already working while this
 * types, so the caller drops `on` the instant the first tool row lands and the
 * brief snaps whole. The animation may never be the reason the screen is behind
 * the work.
 */
export type TypeOnce = { text: string; on: boolean }

// 18ms a character is a shade quicker than the eye reads, which is the point:
// this is meant to be watched finishing, not read as it goes — the text is
// there to be read once it lands. The ceiling matters more than the rate. A
// brief can be a paragraph, and no arrival animation in this app is allowed to
// hold the screen for longer than it takes to say what happened.
const MS_PER_CHAR = 18
const MIN_MS = 400
const MAX_MS = 2200

/** The beat between the person arriving and the work they were given.
 *
 * The card itself may not wait — a delegation appearing IS the model starting
 * one, and holding that back would be the window lying about when it happened.
 * What can wait is the brief, and the reason it should is that these are two
 * events and not one: somebody was hired, and then they were told what to do.
 * Arriving together, the card lands as a paragraph-sized block dropped into the
 * transcript in a single frame, and there is no moment at which the reader is
 * looking at the person rather than at the wall of text beside them.
 *
 * Exported because .bgw-told's own slide has to use the same number — the block
 * opening and the text starting are one movement, and two constants a frame
 * apart is how they stop being one. */
export const HANDOVER_LEAD_MS = 260

function reduced(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

export function typeOnce(node: HTMLElement, param: TypeOnce) {
  let frame = 0
  let timer: ReturnType<typeof setTimeout> | 0 = 0
  let text = param.text

  const stop = () => {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
    if (timer) clearTimeout(timer)
    timer = 0
  }

  const whole = () => {
    stop()
    node.textContent = text
  }

  const run = () => {
    const total = Math.min(Math.max(text.length * MS_PER_CHAR, MIN_MS), MAX_MS)
    const start = performance.now()
    const step = (now: number) => {
      const t = Math.min((now - start) / total, 1)
      node.textContent = text.slice(0, Math.round(t * text.length))
      frame = t < 1 ? requestAnimationFrame(step) : 0
    }
    frame = requestAnimationFrame(step)
  }

  if (param.on && !reduced() && text) {
    // Empty for the lead, which is the point of it: the card is on screen with
    // its portrait and its name, and nothing else, for as long as it takes to
    // register that somebody has been hired.
    node.textContent = ''
    timer = setTimeout(() => {
      timer = 0
      run()
    }, HANDOVER_LEAD_MS)
  } else whole()

  return {
    update(next: TypeOnce) {
      // A brief is written once and never edited, so a changed string means a
      // different delegation in a reused element. Taken whole either way: the
      // one thing this must never do is start over on a card the reader has
      // already read.
      const changed = next.text !== text
      text = next.text
      if (changed || !next.on) whole()
    },
    destroy() {
      stop()
    },
  }
}
