// The two pure readings GitLogPane makes of a commit list, kept out of the
// component so a test can hold them still: what kind of commit a subject
// says it is, and which rows share a day.

export type CommitType = 'feat' | 'fix' | 'refactor' | 'docs' | 'test' | 'other'

/** The kinds that get a colour of their own, in chip order. */
export const TYPES: CommitType[] = ['feat', 'fix', 'refactor', 'docs', 'test']
export const CHIP_TYPES: CommitType[] = [...TYPES, 'other']

export type ParsedSubject = {
  type: CommitType
  /** The word the author actually wrote before the colon — `chore`, `release` — or '' when there was none. */
  word: string
  scope: string
  /** The subject with its prefix removed. */
  text: string
}

const PREFIX = /^([a-zA-Z]+)(?:\(([^)]*)\))?!?:\s*(.*)$/

/** The Conventional Commits prefix, when there is one. Five kinds get a
 * colour of their own; everything else — chore, style, ci, release, a subject
 * with no prefix at all — is "other", but the row still shows the word the
 * author wrote rather than a category nobody typed. */
export function parseSubject(subject: string): ParsedSubject {
  const m = PREFIX.exec(subject)
  if (!m) return { type: 'other', word: '', scope: '', text: subject }
  const word = m[1].toLowerCase()
  const type = (TYPES as string[]).includes(word) ? (word as CommitType) : 'other'
  return { type, word, scope: m[2] ?? '', text: m[3] }
}

/** Cut a newest-first list into runs that share a local calendar day. A day
 * is the reader's day, not UTC's: a commit at 00:30 belongs to the night it
 * was made in. */
export function groupByDay<R extends { date: Date }>(rows: R[]): { key: string; date: Date; rows: R[] }[] {
  const out: { key: string; date: Date; rows: R[] }[] = []
  for (const r of rows) {
    const key = `${r.date.getFullYear()}-${r.date.getMonth()}-${r.date.getDate()}`
    const last = out[out.length - 1]
    if (last && last.key === key) last.rows.push(r)
    else out.push({ key, date: r.date, rows: [r] })
  }
  return out
}
