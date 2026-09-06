// What a delegate has cost, worded once for both cards that say it.
//
// The tray's card (BackgroundWork.svelte) and the transcript's (Chat.svelte,
// subagentTimeline) are deliberately ONE card at two moments (§105.6): a
// delegation that finished inside its turn and one still running afterwards are
// the same event, and two visual languages for that taught the user they were
// two different things.
//
// These three lines lived in the tray alone, and the transcript's card said
// nothing about spend at all — which is the drift §105.6 warned about, arrived
// at by omission rather than by editing. They live here now so neither card
// owns them: the next person who changes where a number rounds, or fixes the
// "0 · 0" guard, cannot fix it for one card and leave the other saying
// something else about the same worker.
import { t } from './i18n.svelte'

/** The spend a card can draw. Structural rather than `BackgroundTask`, because
 *  the transcript's card holds a register entry that may be missing entirely —
 *  a turn read back from the database has no register left to ask. */
export type Spend = {
  tokensIn: number
  tokensOut: number
  cachedIn: number
  cacheReported: boolean
}

/** Thousands as "12.3k". Past four figures the eye reads a count as a shape
 *  rather than as a number, and the shape is what it is checking. */
export function compact(n: number): string {
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k'
  return String(n)
}

/** Drawn only once there is something to draw: a delegate that has not finished
 *  its first round has spent nothing, and "เข้า 0 · ออก 0" reads as a
 *  measurement rather than as the absence of one. */
export const hasSpend = (s?: Spend): boolean => Boolean(s && (s.tokensIn > 0 || s.tokensOut > 0))

/** Two numbers rather than one, because they fail differently and the brake is
 *  a different decision for each (§163.3): input climbing is a transcript being
 *  re-sent every round, output climbing is a model that will not stop writing. */
export const spendLabel = (s: Spend): string =>
  t('bgw.spend', { in: compact(s.tokensIn), out: compact(s.tokensOut) })

/** The cache line is appended only when the provider actually accounts for one.
 *  Without that test a local runtime, which reports nothing, would be described
 *  as having hit the cache zero times, which it never claimed. */
export const spendTitle = (s: Spend): string => {
  const head = t('bgw.spendTitle', { in: String(s.tokensIn), out: String(s.tokensOut) })
  if (!s.cacheReported || s.cachedIn <= 0) return head
  return head + '\n' + t('bgw.spendCached', { n: compact(s.cachedIn) })
}
