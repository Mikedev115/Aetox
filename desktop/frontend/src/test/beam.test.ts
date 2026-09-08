// The running beam, checked as CSS text rather than as pixels.
//
// The structure this pins was found by a bug that was invisible in every unit
// test and obvious on screen: a streamed answer is drawn with {@html}, so the
// markup inside it is REPLACED on every token. An animation declared on the
// ring restarts with the element that carries it, so the light twitched in
// place instead of travelling ("ตอนวาดแผน อนิเมชั่นพังครับ เพราะมันสตรีมอ่ะ").
//
// The fix was structural — the clock runs on the carrier, the phase inherits
// down to whatever ring exists this frame — so the thing worth pinning is the
// structure, not the look. The transcript is no longer one of the carriers
// (see below), and the rule outlives the case that found it.
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'

const css = readFileSync('src/style.css', 'utf8')
const rule = (selector: string) =>
  css.slice(css.indexOf(selector)).slice(0, css.slice(css.indexOf(selector)).indexOf('}') + 1)

describe('the running beam', () => {
  it('runs its clock on the carrier, never on the ring', () => {
    expect(css).toContain('.wb.busy-glow { animation:beam-phase')
    // The ring reads the phase; if it ever animates it again, a streamed block
    // is back to restarting sixty times a second.
    const ring = rule('.wb.busy-glow::before {')
    expect(ring).toContain('var(--beam-phase)')
    expect(ring).not.toContain('animation:')
  })

  it('inherits the phase, or a ring built this frame starts from zero', () => {
    expect(css).toMatch(/@property --beam-phase\s*\{[^}]*inherits:\s*true/)
  })

  it('is worn only by what is still working', () => {
    expect(css).toContain('.wb.busy-glow::before')
    // A finished delegation is a record, and a record that glows asks to be
    // re-read. `.done`/`.err` must never pick this up.
    expect(css).not.toContain('.bgw-card.done::before')
  })

  // The chat lists gave it back on 8 ก.ย. A sidebar is a narrow column of rows,
  // so a light chasing round one outline pulls the eye off the list itself
  // every three seconds for as long as the turn runs (owner, over the running
  // app: "เอากรอบสีวิบวับออก"). The dot on the row says the same thing from the
  // place the eye already lands — and says the state the ring never could.
  it('is not worn by a chat row, which has a dot instead', () => {
    for (const selector of ['.sess-row.working::before', '.proj-group-sess.working::before',
      '.proj-chats li button.working::before']) {
      expect(css).not.toContain(selector)
    }
    // And the clock is off them too — a carrier animating --beam-phase for a
    // ring that no longer exists is a frame of work every 3s for nothing.
    expect(css).not.toContain('.sess-row.working,')
    // Two dots, two states, both still defined: green for the one still going,
    // amber for one that finished while the user was elsewhere.
    expect(css).toContain('.dot.green {')
    expect(css).toContain('.dot.amber {')
  })

  // The reply column lost it on 7 ก.ย. A plan, a drawing and a long fence used
  // to wear one while they were still being written, which was a light moving
  // inside the thing the user is reading — the one place in the app that is
  // prose rather than chrome. The waiting phrase below the message says the
  // same thing in words, and is on screen for as long as anything is arriving.
  it('is not worn by anything inside the transcript', () => {
    for (const selector of ['.plan-card.live', '.codeblock.live', '.drawing-box.live']) {
      expect(css).not.toContain(selector)
    }
    // And the clock is off the column too — a carrier animating --beam-phase
    // for rings that no longer exist is a frame of work every 3s for nothing.
    expect(css).not.toContain('.markdown-body:has(.live)')
  })

  // The delegation card gave the beam back. It was added when the card had no
  // other way to say "alive"; the portrait says it now — AgentFace's `work`
  // state puts a laptop in front of the person and has them type — and two
  // signals for one fact is one too many when four delegations run at once and
  // the transcript is behind four chasing lights.
  //
  // The other carriers keep it. None of them has a face to say it for them.
  it('is not worn by the delegation card, which has a face instead', () => {
    expect(css).not.toContain('.bgw-card.run::before')
    expect(css).not.toContain('.bgw-card::before')
    // And the clock is off it too — a carrier animating --beam-phase for a ring
    // that no longer exists is a frame of work every 3s for nothing.
    expect(css).not.toContain('.bgw-card.run,')
  })

  it('drops the motion but keeps the signal when motion is unwelcome', () => {
    const at = css.indexOf('.wb.busy-glow { animation:none; }')
    expect(at).toBeGreaterThan(-1)
    // The nearest at-rule above it is the guard, not some other block it drifted
    // into: stopping the clock unconditionally would leave a still gradient at
    // whatever angle it happened to be on.
    const before = css.slice(0, at)
    expect(before.slice(before.lastIndexOf('@media'))).toContain('prefers-reduced-motion')
    // Still lit, so which one is working is still answerable without motion.
    expect(css.slice(at, at + 500)).toContain('--interactive')
  })
})
