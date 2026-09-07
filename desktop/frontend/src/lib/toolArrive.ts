/** Rows that land in the same frame, drawn one after another.
 *
 *  The agent groups its read-only calls and runs them together
 *  (internal/cognitive/agent.go, "running %d read-only tool calls together"),
 *  so five searches are announced in one frame and five rows appear in one
 *  frame. Nothing is wrong with that — it is what actually happened — but
 *  painting it as it lands hands the page the wire's rhythm, and the wire has
 *  no rhythm. streamPace.ts already wrote this down for text: *"the jitter of
 *  the network becomes the rhythm of the page, which is a rhythm nobody
 *  designed"*. It was never applied to the rows themselves. Owner, 7 ก.ย.:
 *  *"ต่อให้ตู้มมาทีเดียว หน้าบ้านก็ควรสมูธครับ"*.
 *
 *  So a batch is dealt out instead of dropped. This is the same technique the
 *  search card already uses one level down (`--i` on .search-hit, 40ms apart),
 *  and for the same stated reason: the difference between reading a list and
 *  being handed a block.
 *
 *  What it does NOT do is claim the calls happened one after another. They did
 *  not, they happened at once, and the row's own clock still says so — every
 *  one of them counts from the same instant and they finish together. The
 *  stagger is the LIST appearing, which is a fact about the screen, not about
 *  the work.
 *
 *  An action for the reason toolGlide, toolWindow and toolFocus are actions:
 *  what it needs is the set of rows added in one batch, which only the DOM
 *  knows. The observer is attached after the first render, so a turn read back
 *  from the database — where every row is already there — is untouched. A
 *  record does not deal itself out.
 */

/** The beat between one row and the next. Half the search card's 40ms would be
 *  imperceptible at row height; twice it would be the list making the reader
 *  wait. */
const STEP_MS = 55

/** Past this the delay stops growing. Eight rows is 440ms, which is already the
 *  longest any arrival in this app is allowed to hold the screen; a batch of
 *  twenty must not be eleven hundred. */
const CAP = 7

export function toolArrive(node: HTMLElement) {
  const deal = (records: MutationRecord[]) => {
    let i = 0
    for (const record of records) {
      for (const added of record.addedNodes) {
        if (!(added instanceof HTMLElement)) continue
        // Rows and narration only. A search card is added in the same batch as
        // the row that ran it and belongs to that row's beat, not to one of its
        // own — it is already dealing its own results out inside itself.
        if (!added.classList.contains('tool-step') && !added.classList.contains('tool-note')) continue
        if (i > 0) added.style.setProperty('--i', String(Math.min(i, CAP)))
        i += 1
      }
    }
  }
  const rows = new MutationObserver(deal)
  rows.observe(node, { childList: true })
  return {
    destroy() {
      rows.disconnect()
    },
  }
}

export const TOOL_ARRIVE_STEP_MS = STEP_MS
