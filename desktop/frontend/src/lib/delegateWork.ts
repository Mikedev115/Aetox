// What a delegate is doing, and what it did — read off the rows it left behind.
//
// The card that draws a delegation used to say three things: who it is, what it
// was told to do, and how many tools it ran. All three are true and none of
// them is the work. Owner, over a card wearing a fresh 26px portrait: *"ทั้งที่
// มีอวตารแล้วทำไมยังเงียบแบบเดิมอยู่"* — the word was เงียบ, silent, not ugly.
// A card is silent when the only thing on it that changes is a clock.
//
// So this file answers the two questions the card had no way to ask:
//
//   - **what is it doing right now**, which is the one line that makes a card
//     look alive at a glance (§105.5 learned this for the tray and the
//     transcript's card never got it), and
//   - **what did it come back with**, which is what a finished card should say
//     instead of falling silent the moment the work is over.
//
// Both live here rather than in either component because the tray's card and
// the transcript's are deliberately ONE card at two moments (§105.6), and a
// rule about what counts as "a file it read" must not be able to differ between
// them.
import type { ToolStep } from './types'

/** The tools that read a named file, by the name the engine puts at the head of
 *  a row's label (`[tool.name, tool.subject].join(' ')`).
 *
 *  A list rather than a stamp on the row, because there is no stamp to read: a
 *  write is marked by the engine (`added`/`removed`/`diff` — see `wrote`), and
 *  a read is not. `count` is the nearest thing and it is not the same thing —
 *  it is "how much came back in the tool's own unit", so a grep that found 40
 *  matches across the repo would be counted here as forty files anybody read.
 *
 *  Unknown names count as neither, deliberately — the same fallback rule
 *  workerFace and agentFace's PROP already follow: a name this build does not
 *  know draws nothing rather than a guess. The cost of forgetting to add a
 *  tool here is a chip that undercounts; the cost of guessing is a card that
 *  states a number nobody can check. */
export const READ_TOOLS = new Set(['read', 'pdf_read', 'media_read', 'github_read_file'])

/** The tool half of a row's label. */
const toolOf = (s: ToolStep) => s.label.split(' ')[0]

/** The subject half — the path, the URL, the thing the call named. Empty when
 *  the call named nothing, which is why the tallies below count subjects and
 *  not rows: reading one file four times is one file, and a card that called it
 *  four would be inflating the only numbers on it. */
const subjectOf = (s: ToolStep) => {
  const at = s.label.indexOf(' ')
  return at < 0 ? '' : s.label.slice(at + 1).trim()
}

/** Whether a row changed a file, asked of the engine rather than of a name.
 *  `added`/`removed` are "the line counts of a write or edit, zero elsewhere"
 *  and `diff` is "empty on every tool that writes no file" — three stamps for
 *  one fact, and any of them is enough. A tool added next year is counted here
 *  without anybody editing this file. */
const wrote = (s: ToolStep) => Boolean(s.added || s.removed || s.diff)

/** Tool calls only. Narration and thinking ride in the same list and are not
 *  work — the same rule `ownTools` and `toolCount` already follow, for the same
 *  reason: counting sentences inflates every number a card shows. */
const tools = (steps: ToolStep[]) => steps.filter((s) => !s.kind)

/** Files touched, told apart by which way. Two numbers rather than one because
 *  they answer different questions — a delegate that read nine files and
 *  changed none went and looked, and one that changed three went and did. */
export type Tally = { read: number; wrote: number }

export function tally(steps: ToolStep[]): Tally {
  const readers = new Set<string>()
  const writers = new Set<string>()
  for (const s of tools(steps)) {
    const subject = subjectOf(s)
    if (!subject) continue
    if (wrote(s)) writers.add(subject)
    else if (READ_TOOLS.has(toolOf(s))) readers.add(subject)
  }
  return { read: readers.size, wrote: writers.size }
}

/** The row a card should be showing right now.
 *
 *  The newest one still running, and the newest one there is when nothing is —
 *  which is not the same row and matters at exactly the moment it changes: a
 *  delegate between calls has no running row at all, and a card that blanked
 *  for that second would flicker on every step boundary. So the last finished
 *  row holds the line until the next one starts. */
export function currentStep(steps: ToolStep[]): ToolStep | undefined {
  const rows = tools(steps)
  for (let i = rows.length - 1; i >= 0; i--) if (rows[i].state === 'run') return rows[i]
  return rows[rows.length - 1]
}
