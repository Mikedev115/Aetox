// A panel's own newlines came back as <br> between its divs.
//
// internal/prompt's `panel` layer asks the model for a small block of styled
// divs when a comparison reads better as a grid than as a table. The model
// writes it the way anyone writes markup — one element per line — and glues it
// to the end of the sentence that introduces it, which makes the whole thing
// inline HTML as far as markdown is concerned. `breaks: true` then turns every
// newline inside it into a <br>.
//
// In ordinary flow that is a stack of blank lines nobody asked for. Inside
// `display:grid` a <br> is a grid ITEM: a header row of four labels became nine
// items and wrapped into three rows, so "หน้า" sat above "แอดมินสนาม" in the
// second column while the body rows underneath — written one line each, so free
// of <br> — lined up perfectly (owner, 6 ก.ย., with the screenshot).
import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../lib/markdown'

const header = [
  'ตารางสิทธิ์ครับ<div style="border:1px solid var(--border-subtle);">',
  '  <div style="display:grid; grid-template-columns:minmax(0,1fr) 90px 110px 110px;">',
  '    <div>หน้า</div>',
  '    <div style="text-align:center;">ซุปเปอร์</div>',
  '    <div style="text-align:center;">แอดมินสนาม</div>',
  '    <div style="text-align:center;">เมเนเจอร์ (ใหม่)</div>',
  '  </div>',
  '</div>',
].join('\n')

function render(text: string): HTMLElement {
  const host = document.createElement('div')
  host.innerHTML = renderMarkdown(text)
  return host
}

describe('a panel written one element per line', () => {
  it('gives its grid exactly the items the model wrote', () => {
    const host = render(header)
    const grid = host.querySelector('div[style*="display:grid"]')

    expect(grid).not.toBeNull()
    // Four labels, four items. Nine was the bug.
    expect(grid!.children).toHaveLength(4)
    expect(Array.from(grid!.children, (c) => c.textContent)).toEqual([
      'หน้า', 'ซุปเปอร์', 'แอดมินสนาม', 'เมเนเจอร์ (ใหม่)',
    ])
  })

  it('leaves no line breaks anywhere in it', () => {
    expect(render(header).querySelectorAll('br')).toHaveLength(0)
  })

  // The <br> at the very start of the outer div is the same stray newline, and
  // it drew a band of empty space above the header before anything else did.
  it('drops one standing at the edge of a container', () => {
    const host = render('ดูนี่<div class="p">\n  <div>a</div>\n</div>')

    expect(host.querySelectorAll('br')).toHaveLength(0)
    expect(host.querySelector('.p')!.children).toHaveLength(1)
  })

  // A blank line inside the panel leaves two. Judged one at a time, each keeps
  // the other alive — the run has to be read as the one newline it came from.
  it('drops a run of them between two blocks', () => {
    const host = render('ดูนี่<div class="p">\n<div>a</div>\n\n<div>b</div>\n</div>')

    expect(host.querySelectorAll('br')).toHaveLength(0)
    expect(host.querySelector('.p')!.children).toHaveLength(2)
  })

  // markdown's paragraph machinery leaves its own phantom at the tail of a
  // panel: a <p> that closes with nothing in it, which a grid counts as an item
  // the same way it counts a <br>.
  it('leaves no empty paragraph inside it', () => {
    const host = render('ดูนี่<div class="p">\n  <div>a</div>\n</div>')

    expect(Array.from(host.querySelectorAll('p'), (p) => p.textContent)).not.toContain('')
  })
})

// The other half, and the reason this is a removal rather than turning `breaks`
// off: a line break next to real text is the one the user meant.
describe('an ordinary line break in prose', () => {
  it('survives inside a paragraph', () => {
    expect(render('บรรทัดแรก\nบรรทัดสอง').querySelectorAll('br')).toHaveLength(1)
  })

  it('survives between a sentence and a block that follows it', () => {
    const host = render('บอกไว้ก่อน<br>\n<div>แล้วค่อยดูตาราง</div>')

    expect(host.querySelectorAll('br').length).toBeGreaterThanOrEqual(1)
  })
})
