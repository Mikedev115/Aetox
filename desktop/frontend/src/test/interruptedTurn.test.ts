// A turn the app's own closing ended (§219). Go writes one marker for it —
// in front of the cancel when the close had time to stop the turn, alone when
// the next launch found the question still waiting — and the chat has to draw
// both the same way: not as a Stop the user never pressed, not as an error,
// and with the retry chip in the ordinary colour.
import { describe, it, expect, beforeEach } from 'vitest'
import { cockpit, restoreTranscript } from '../lib/stores/cockpit.svelte'
import type { main } from '../../wailsjs/go/models'

const row = (m: Partial<main.SessionMessage>): main.SessionMessage =>
  ({ role: 'user', text: '', time: '10:10', ...m }) as main.SessionMessage

beforeEach(() => {
  cockpit.chat = []
})

describe('a turn the app was closed under', () => {
  it('reads as interrupted — not stopped, not an error — when the close had time to write it', () => {
    const out = restoreTranscript([
      row({ role: 'user', text: 'ไล่บั๊กให้ที' }),
      row({
        role: 'agent', text: 'กำลังไล่ดูโค้ดให้ครับ พบว่า',
        errorText: 'aetox: app closed mid-turn: Post "https://provider/chat": context canceled',
      }),
    ])
    const last = out.at(-1)
    expect(last?.text).toContain('กำลังไล่ดูโค้ดให้ครับ พบว่า')
    expect(last?.text).toContain('แอปถูกปิดระหว่างที่กำลังทำงานอยู่')
    expect(last?.text).not.toContain('หยุดการทำงานแล้ว')
    expect(last?.text).not.toContain('เกิดข้อผิดพลาด')
    expect(last?.failed).toBe(true)
    expect(last?.stopped).toBe(true)
    expect(last?.failedText).toBe('ไล่บั๊กให้ที')
  })

  it('reads the same when the next launch closed the question', () => {
    const out = restoreTranscript([
      row({ role: 'user', text: 'ทำต่อให้ที' }),
      row({ role: 'agent', text: '', errorText: 'aetox: app closed mid-turn' }),
    ])
    const last = out.at(-1)
    expect(last?.text).toBe('แอปถูกปิดระหว่างที่กำลังทำงานอยู่ งานที่ทำไปก่อนหน้ายังอยู่ในแชตนี้ กดลองใหม่เพื่อทำต่อได้เลยครับ')
    expect(last?.failed).toBe(true)
    expect(last?.stopped).toBe(true)
    expect(last?.failedText).toBe('ทำต่อให้ที')
  })

  it('leaves a plain Stop worded as a Stop', () => {
    const out = restoreTranscript([
      row({ role: 'user', text: 'เริ่มงานใหญ่' }),
      row({ role: 'agent', text: '', errorText: 'context canceled' }),
    ])
    expect(out.at(-1)?.text).toBe('หยุดการทำงานแล้ว')
    expect(out.at(-1)?.stopped).toBe(true)
  })
})
