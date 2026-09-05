// A reply arrives in bursts and is shown at a steady rate.
//
// Owner, 5 ก.ย., pointing at a glm reply mid-stream: "บางทีโมเดลสตรีมแบบติดๆ
// ขัดๆ". Nothing about the answer was wrong — it was the rhythm. Two separate
// things made it, and this file answers both.
//
// The first is that the text does not arrive evenly. opencode-go hands a glm
// stream over in lumps: silence for a third of a second, then two hundred
// characters at once. Painted as it lands, the JITTER OF THE NETWORK BECOMES
// THE RHYTHM OF THE PAGE, which is a rhythm nobody designed.
//
// The second is that the answer was painted per chunk rather than per frame,
// and a paint is not cheap there: renderStreamingMarkdown runs the whole answer
// back through marked, DOMPurify and highlight.js, and morphInto parses the
// result. A burst of fifty chunks paid that fifty times to show forty-nine
// states no eye was ever going to see, and the frames it cost were the visible
// stutter.
//
// So arrival and display are pulled apart. `arrived` is the truth as the engine
// tells it; `shown` is how much of it has been let out, and it only moves on an
// animation frame. One paint per frame at most — the forty-nine invisible ones
// are simply not drawn, which is why this is faster than what it replaced even
// though it looks like more work.
//
// HOW FAST IS THE INTERESTING PART, and the first version got it wrong. It
// spread every lump over a fixed 140ms, which is shorter than the pause between
// lumps: the text sprinted for 140ms, stopped for two hundred more, and
// sprinted again. Smoother inside each lump, same stop-start overall —
// "เปลี่ยนนิดนึง", which is what a fix aimed at the symptom earns.
//
// The pause IS the budget. A burst that arrives every 350ms has 350ms to be
// read before the next one lands, so that is what it is given: the pacer
// measures the gaps between bursts and drains into the gap it expects. Held
// there the backlog stays about one burst deep and the letters never stop
// coming — the wire keeps bursting and the page never learns of it.
//
// The rate itself is `lag / time left`, which needs no words-per-minute to
// tune: drained at that rate the backlog reaches zero exactly at the deadline,
// which makes the rate constant. A provider that already streams evenly
// (DeepSeek) builds no backlog and is left alone by the same arithmetic.
//
// What this costs is the end of a reply landing one gap late — the final burst
// still plays out after the model has stopped talking. That is the whole price,
// it is paid once, and it buys back every burst before it.
import { morphInto } from './morph'
import { renderStreamingMarkdown } from './markdown'

// The window a burst may be spread over, in ms: the floor and ceiling on what
// the measured gap is allowed to ask for.
//
// The floor keeps a dense stream from being drained in a blink, which would be
// the old per-chunk behaviour reached by another route. The ceiling is a
// promise about the end of a reply — a model that pauses two seconds
// mid-thought must not buy itself a two-second tail after its last word.
const MIN_WINDOW_MS = 120
const MAX_WINDOW_MS = 400

// A pause longer than this ended a burst. Below it the events are one lump
// being handed over in pieces — opencode-go delivers ten in as many
// milliseconds — and folding those in would measure the shape of a single burst
// rather than the space between two.
const BURST_GAP_MS = 50

// How far a newly measured gap moves the average. Low enough that one slow
// round trip does not stretch the window for the rest of the reply.
const GAP_WEIGHT = 0.3

// The shortest window the rate may be computed against. Past its deadline — a
// backgrounded webview stops calling rAF for seconds at a time — `lag / time
// left` would divide by zero or by a negative; this makes it "release the
// backlog now", which is also the right answer for a window nobody watched.
const FRAME_MS = 16

type Draw = (text: string) => void

export type PacedStream = {
  /** The live text as the store holds it — the whole answer so far, not a chunk. */
  text: string
  /** Called after each paint, so a pinned transcript can follow text that now
   *  grows on frames the store knows nothing about. */
  onPaint?: () => void
}

class Pacer {
  private arrived = ''
  private shown = 0
  private slice = ''
  private frame = 0
  private last = 0
  private deadline = 0
  private lastFeed = 0
  private gap = MIN_WINDOW_MS
  private started = false

  constructor(private draw: Draw, private onPaint?: () => void) {}

  feed(text: string, onPaint?: () => void): void {
    this.onPaint = onPaint
    // The first sight of a block is shown whole, never typed out. It is not
    // always one chunk's worth: a session parked and switched back to arrives
    // with everything it had streamed so far, and replaying that letter by
    // letter would be a story about the network, told late.
    if (!this.started) {
      this.started = true
      this.lastFeed = performance.now()
      this.snap(text)
      return
    }
    // Not a continuation: the engine's authoritative delivery of the finished
    // reply, a round it erased, a different session's text. Pacing can only add
    // letters, so anything that would take one back is taken whole.
    if (!this.continues(text)) {
      this.snap(text)
      return
    }
    if (text === this.arrived) return
    this.arrived = text
    this.deadline = performance.now() + this.window()
    this.run()
  }

  stop(): void {
    if (this.frame !== 0) cancelAnimationFrame(this.frame)
    this.frame = 0
  }

  /** How long this burst gets, measured from the ones before it. */
  private window(): number {
    const now = performance.now()
    const since = now - this.lastFeed
    this.lastFeed = now
    if (since > BURST_GAP_MS) this.gap = this.gap * (1 - GAP_WEIGHT) + since * GAP_WEIGHT
    return Math.min(Math.max(this.gap, MIN_WINDOW_MS), MAX_WINDOW_MS)
  }

  private snap(text: string): void {
    this.stop()
    this.arrived = text
    this.shown = text.length
    this.paint()
  }

  private continues(text: string): boolean {
    if (text.startsWith(this.arrived)) return true
    // Rewritten, but still a continuation if what is ALREADY on screen survives
    // it — only the buffered tail may be thrown away.
    return text.startsWith(this.arrived.slice(0, Math.floor(this.shown)))
  }

  private paint(): void {
    const slice = this.arrived.slice(0, Math.floor(this.shown))
    // A frame that releases half a character has nothing new to draw. Caught
    // here rather than inside the drawer so the markdown pass is skipped too,
    // which is the expensive half.
    if (slice === this.slice) return
    this.slice = slice
    this.draw(slice)
    this.onPaint?.()
  }

  private tick = (now: number): void => {
    const dt = now - this.last
    this.last = now
    const lag = this.arrived.length - this.shown
    const left = Math.max(this.deadline - now, FRAME_MS)
    this.shown = Math.min(this.arrived.length, this.shown + (lag / left) * dt)
    this.paint()
    // Caught up: stop asking for frames. A reply spends most of its life
    // waiting on a tool, and a loop that idles through that keeps the
    // compositor awake for nothing.
    this.frame = this.shown < this.arrived.length ? requestAnimationFrame(this.tick) : 0
  }

  private run(): void {
    if (this.frame !== 0) return
    this.last = performance.now()
    this.frame = requestAnimationFrame(this.tick)
  }
}

function paced(param: PacedStream, draw: Draw) {
  const pacer = new Pacer(draw, param.onPaint)
  pacer.feed(param.text, param.onPaint)
  return {
    update: (next: PacedStream) => pacer.feed(next.text, next.onPaint),
    // Nothing is flushed on the way out. The element only leaves when the turn
    // ends, and what replaces it is the finished text drawn in full.
    destroy: () => pacer.stop(),
  }
}

/** Svelte action for a streaming answer: `use:pacedStream={{ text, onPaint }}`.
 *  Replaces the plain morph action that stood here — the markdown render moved
 *  inside, because doing it in the template is doing it once per chunk. */
export function pacedStream(node: HTMLElement, param: PacedStream) {
  // The rendered markup, kept so a slice that heals to the same HTML — a bare
  // `*` appended and then dropped again — never reaches morphInto.
  let painted = ''
  return paced(param, (text) => {
    const html = renderStreamingMarkdown(text)
    if (html === painted) return
    painted = html
    morphInto(node, html)
  })
}

/** The same pacing for text drawn as text: `use:pacedText={{ text, onPaint }}`.
 *
 *  Reasoning is the one that needs it. It is the longest-running thing on
 *  screen with a thinking model, it is where the owner still saw the stutter
 *  after the answer was paced ("ตอนคิดก็กระตุกเหมือนเดิม แต่ตอนตอบเปลี่ยนนิด
 *  นึง"), and being cheap to draw never made it any smoother: the jitter is in
 *  the arrivals, and a cheap paint of a lump is still a lump. */
export function pacedText(node: HTMLElement, param: PacedStream) {
  return paced(param, (text) => {
    node.textContent = text
  })
}
