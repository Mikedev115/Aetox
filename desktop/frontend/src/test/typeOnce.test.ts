// The brief being written out, and the two things that must never happen: a
// record replaying itself, and an animation holding the screen behind work that
// has already started.
import { describe, it, expect, vi } from 'vitest'
import { typeOnce, HANDOVER_LEAD_MS } from '../lib/typeOnce'

const el = () => document.createElement('div')
const BRIEF = 'กางข้อกล่าวอ้างออกทีละข้อ แล้วหาโค้ดที่ยืนยันหรือหักล้าง'

describe('the brief written out once', () => {
  // A card read back out of the database is not a handover, it is the record of
  // one. Same line .reasoning-body.live draws.
  it('is whole from the first frame when nothing is being handed over', () => {
    const node = el()
    typeOnce(node, { text: BRIEF, on: false })
    expect(node.textContent).toBe(BRIEF)
  })

  // The delegate is already working while this types. The instant the caller
  // says so — it drops `on` when the first tool row lands — the brief has to be
  // there in full, not somewhere in the middle of itself.
  it('snaps whole the moment it is told to stop', () => {
    const node = el()
    const action = typeOnce(node, { text: BRIEF, on: true })
    expect(node.textContent!.length).toBeLessThan(BRIEF.length)
    action.update!({ text: BRIEF, on: false })
    expect(node.textContent).toBe(BRIEF)
  })

  // Two events, not one: somebody was hired, and then they were told what to
  // do. For the length of the lead the card is a portrait and a name and
  // nothing else — which is the whole of what the stagger buys.
  it('holds the card at the portrait alone before it writes anything', () => {
    vi.useFakeTimers()
    try {
      const node = el()
      typeOnce(node, { text: BRIEF, on: true })
      expect(node.textContent).toBe('')
      vi.advanceTimersByTime(HANDOVER_LEAD_MS - 20)
      expect(node.textContent).toBe('')
    } finally {
      vi.useRealTimers()
    }
  })

  // A brief is written once and never edited, so a different string in the same
  // element is a different delegation. Starting over on a card the reader has
  // already read is the one behaviour this must not have.
  it('takes a changed brief whole rather than typing it again', () => {
    const node = el()
    const action = typeOnce(node, { text: BRIEF, on: true })
    action.update!({ text: 'งานใหม่คนละใบ', on: true })
    expect(node.textContent).toBe('งานใหม่คนละใบ')
  })

  it('writes nothing but the text it was given, and stops cleanly', () => {
    const node = el()
    const action = typeOnce(node, { text: '', on: true })
    expect(node.textContent).toBe('')
    action.destroy!()
  })

  it('is whole at once when motion is unwelcome', () => {
    const mm = vi.fn().mockReturnValue({ matches: true })
    const had = window.matchMedia
    window.matchMedia = mm as unknown as typeof window.matchMedia
    try {
      const node = el()
      typeOnce(node, { text: BRIEF, on: true })
      expect(node.textContent).toBe(BRIEF)
      expect(mm).toHaveBeenCalledWith('(prefers-reduced-motion: reduce)')
    } finally {
      window.matchMedia = had
    }
  })
})
