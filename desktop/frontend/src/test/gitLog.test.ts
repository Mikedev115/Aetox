import { describe, expect, it } from 'vitest'
import { parseSubject, groupByDay } from '../lib/workbench/gitLog'

describe('parseSubject', () => {
  it('reads the Conventional Commits prefix, scope and text', () => {
    expect(parseSubject('feat(git): ห้องไทมไลน์')).toEqual({ type: 'feat', word: 'feat', scope: 'git', text: 'ห้องไทมไลน์' })
    expect(parseSubject('fix: one')).toEqual({ type: 'fix', word: 'fix', scope: '', text: 'one' })
    expect(parseSubject('refactor(desktop)!: breaking')).toMatchObject({ type: 'refactor', scope: 'desktop', text: 'breaking' })
  })

  it('keeps the word the author wrote when the kind has no colour of its own', () => {
    // chore, release, style … are "other" for the chips, but the row still
    // shows what was typed rather than a category nobody wrote.
    expect(parseSubject('chore: bump')).toEqual({ type: 'other', word: 'chore', scope: '', text: 'bump' })
    expect(parseSubject('release: v1.5.28')).toMatchObject({ type: 'other', word: 'release' })
    expect(parseSubject('Docs: caps')).toMatchObject({ type: 'docs', word: 'docs' })
  })

  it('leaves a subject with no prefix whole', () => {
    expect(parseSubject('Merge branch main')).toEqual({ type: 'other', word: '', scope: '', text: 'Merge branch main' })
    expect(parseSubject('§248 สถานะ: not a prefix')).toMatchObject({ type: 'other', word: '', text: '§248 สถานะ: not a prefix' })
    // A URL-ish subject is not "http" of scope nothing.
    expect(parseSubject('http://x')).toMatchObject({ word: 'http', text: '//x' })
  })
})

describe('groupByDay', () => {
  const at = (iso: string) => ({ date: new Date(iso), iso })

  it('cuts a newest-first list into runs that share a local day', () => {
    const rows = [at('2026-09-12T14:00:00'), at('2026-09-12T01:00:00'), at('2026-09-11T23:59:00'), at('2026-09-09T10:00:00')]
    const days = groupByDay(rows)
    expect(days.map((d) => d.rows.length)).toEqual([2, 1, 1])
    expect(days.map((d) => d.key)).toEqual(['2026-8-12', '2026-8-11', '2026-8-9'])
    // The day carries the first row's date, which is what the label is drawn from.
    expect(days[0].date).toBe(rows[0].date)
  })

  it('is empty for nothing', () => {
    expect(groupByDay([])).toEqual([])
  })
})
