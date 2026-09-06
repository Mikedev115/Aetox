// A citation the model went and found, thrown away by the renderer.
//
// `[^1]: ที่มา` is a valid LINK DEFINITION to markdown — label `^1`, URL
// `ที่มา` — so a model writing an ordinary footnote had its source line
// swallowed whole and its `[^1]` turned into a live link to a made-up address.
// The reader was shown a blue `^1` that goes nowhere and never shown the
// source. Text disappearing is the bug; the numbering and the jump are the
// part that makes what is left readable.
import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../lib/markdown'

const render = (src: string) => {
  const host = document.createElement('div')
  host.innerHTML = renderMarkdown(src)
  return host
}

const CITED = 'Aetox เขียนด้วย Go[^1] และ Svelte[^2]\n\n[^1]: ภาษาฝั่งเครื่อง\n[^2]: ฝั่งหน้าจอ'

describe('a footnote in an answer', () => {
  it('keeps the source the model wrote', () => {
    const host = render(CITED)

    const notes = host.querySelectorAll('.fn-notes .fn-note')
    expect(notes.length).toBe(2)
    expect(notes[0].textContent).toContain('ภาษาฝั่งเครื่อง')
    expect(notes[1].textContent).toContain('ฝั่งหน้าจอ')
  })

  // The old behaviour, pinned so it cannot come back: the definition became a
  // link definition and the reference became an anchor pointing at its text.
  it('never turns a citation into a link', () => {
    const host = render(CITED)

    expect(host.querySelector('a')).toBeNull()
    expect(host.innerHTML).not.toContain('href')
  })

  it('numbers the markers by where they are read, not where they are defined', () => {
    const host = render('ก[^b] ข[^a]\n\n[^a]: หนึ่ง\n[^b]: สอง')

    const refs = Array.from(host.querySelectorAll('.fn-ref'), (r) => r.textContent)
    expect(refs).toEqual(['1', '2'])
    const marks = Array.from(host.querySelectorAll('.fn-mark'), (m) => m.textContent)
    expect(marks).toEqual(['1', '2'])
    // …and the note under marker 1 is the one that marker cited.
    expect(host.querySelector('.fn-note')?.textContent).toContain('สอง')
  })

  // Not an <a href="#fn-1">: an id is document-wide, every answer numbers from
  // 1, and a hash link in a webview navigates the app itself.
  it('pairs a marker to its note by label, not by id', () => {
    const host = render(CITED)

    expect(host.querySelector('.fn-ref')?.getAttribute('data-fn')).toBe('1')
    expect(host.querySelector('.fn-note')?.getAttribute('data-fn')).toBe('1')
    expect(host.querySelector('[id]')).toBeNull()
  })

  it('reads a label that is a word', () => {
    const host = render('อ้างอิง[^ที่มา]\n\n[^ที่มา]: เอกสาร')

    expect(host.querySelector('.fn-ref')?.textContent).toBe('1')
    expect(host.querySelector('.fn-note')?.textContent).toContain('เอกสาร')
  })

  it('joins a note that wrapped onto the next line', () => {
    const host = render('ก[^1]\n\n[^1]: บรรทัดแรก\n    บรรทัดต่อ')

    expect(host.querySelector('.fn-note')?.textContent).toContain('บรรทัดแรก บรรทัดต่อ')
  })

  it('renders the markdown inside a note', () => {
    const host = render('ก[^1]\n\n[^1]: ดู `main.go` ที่ **บรรทัด 12**')

    expect(host.querySelector('.fn-note code')?.textContent).toBe('main.go')
    expect(host.querySelector('.fn-note strong')?.textContent).toBe('บรรทัด 12')
  })

  // The whole point is that nothing the model wrote goes missing, so a note
  // nobody pointed at is printed too — with a dash, because it has no number.
  it('prints a note that nothing refers to', () => {
    const host = render('ไม่มีการอ้าง\n\n[^ลอย]: ยังอยากให้เห็น')

    expect(host.querySelector('.fn-note')?.textContent).toContain('ยังอยากให้เห็น')
    expect(host.querySelector('.fn-mark')?.textContent).toBe('—')
  })

  // A marker with nothing behind it keeps its number in the sentence; an empty
  // row under the answer would say a source exists.
  it('prints no row for a marker with no note', () => {
    const host = render('อ้างลอย[^9]')

    expect(host.querySelector('.fn-ref')?.textContent).toBe('1')
    expect(host.querySelector('.fn-notes')).toBeNull()
  })
})

describe('what is not a footnote', () => {
  it('leaves an ordinary link alone', () => {
    const host = render('[เว็บ](https://aetox.dev)')

    expect(host.querySelector('a')?.getAttribute('href')).toBe('https://aetox.dev')
    expect(host.querySelector('.fn-ref')).toBeNull()
  })

  it('leaves the syntax inside code exactly as written', () => {
    const inline = render('เขียน `[^1]` แบบนี้')
    expect(inline.querySelector('.fn-ref')).toBeNull()
    expect(inline.querySelector('code')?.textContent).toBe('[^1]')

    const fenced = render('```\n[^1]: ไม่ใช่เชิงอรรถ\n```')
    expect(fenced.querySelector('.fn-notes')).toBeNull()
    expect(fenced.textContent).toContain('[^1]: ไม่ใช่เชิงอรรถ')
  })

  it('leaves an answer with no citations completely alone', () => {
    const host = render('ธรรมดา')

    expect(host.querySelector('.fn-notes')).toBeNull()
    expect(host.querySelector('.fn-ref')).toBeNull()
  })

  // Every answer numbers its own notes from 1, so nothing may survive between
  // two renders.
  it('does not carry notes from one answer into the next', () => {
    render(CITED)
    const host = render('คำตอบถัดไป')

    expect(host.querySelector('.fn-notes')).toBeNull()
    expect(renderMarkdown(CITED)).toBe(renderMarkdown(CITED))
  })
})
