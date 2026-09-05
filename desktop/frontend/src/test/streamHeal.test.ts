// Half-written emphasis, drawn as if it were finished.
//
// A model writes bold in three frames: `**`, then `**คุณอยากได้`, then the
// closer. Drawn as written, the middle frame is two literal asterisks followed
// by plain text, and the closer re-flows the line into bold — letters move,
// line breaks move, and the reader reports a stutter that no dropped frame can
// explain. The owner's screenshot of it (5 ก.ย.) is what these pin.
//
// Only the streaming path heals. A finished message is not half-written, and
// asterisks that survived to the end of a turn are the model's, not a stream's.
import { describe, it, expect } from 'vitest'
import { renderMarkdown, renderStreamingMarkdown } from '../lib/markdown'

const text = (html: string) => {
  const el = document.createElement('div')
  el.innerHTML = html
  return (el.textContent ?? '').trim()
}

describe('healing a half-written line', () => {
  it('draws unclosed bold as bold, not as asterisks', () => {
    const html = renderStreamingMarkdown('ตามสกิลแล้ว **ห้ามค้นหาก่อนถาม')
    expect(html).toContain('<strong>ห้ามค้นหาก่อนถาม</strong>')
    expect(text(html)).not.toContain('*')
  })

  it('draws unclosed italics and inline code the same way', () => {
    expect(renderStreamingMarkdown('งาน *ของคุณเอง')).toContain('<em>ของคุณเอง</em>')
    expect(renderStreamingMarkdown('เรียกที่ `applyAgentChunk')).toContain('<code>applyAgentChunk</code>')
  })

  it('shows nothing for a marker that has nothing to mark yet', () => {
    // Closed rather than dropped, `**` would heal to `****` — four literal
    // asterisks, a worse flash than the one being fixed.
    expect(text(renderStreamingMarkdown('ข้อแรกคือ **'))).toBe('ข้อแรกคือ')
    expect(text(renderStreamingMarkdown('ข้อแรกคือ `'))).toBe('ข้อแรกคือ')
  })

  it('leaves the paragraphs above the live one alone', () => {
    const html = renderStreamingMarkdown('**หัวข้อ** จบแล้ว\n\nกำลังเขียน **ต่อ')
    expect(html).toContain('<strong>หัวข้อ</strong>')
    expect(html).toContain('<strong>ต่อ</strong>')
  })

  it('never writes into an open code fence', () => {
    // Every marker inside a fence is literal text, and a healed one would be a
    // character the user copies out of the block.
    const html = renderStreamingMarkdown('```py\nx = a ** b\nprint("*")')
    expect(text(html)).toContain('a ** b')
    expect(text(html)).toContain('print("*")')
  })

  it('closes bold that opened before the fence did not', () => {
    // A closed fence is settled; the prose after it is what is being written.
    const html = renderStreamingMarkdown('```py\nx = 1\n```\n\nแปลว่า **ค่านี้')
    expect(html).toContain('<strong>ค่านี้</strong>')
  })

  it('does not heal a finished message', () => {
    // renderMarkdown is what a turn that ended draws through. Asterisks that
    // lived to the end of a reply are the model's own text.
    expect(text(renderMarkdown('เขียนค้างไว้ **แบบนี้'))).toContain('**')
  })
})
