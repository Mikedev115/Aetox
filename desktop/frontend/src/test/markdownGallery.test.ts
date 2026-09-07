// Four posters, four columns.
//
// A model asked for four versions of a poster answers with four images, and
// until now the paragraph drew them as four tall columns down the answer. The
// CSS aimed at it — `p > img:not(:only-child)`, capped at a third of the bubble
// — but a cap cannot make portrait art sit side by side, and not one of the
// four could be clicked, enlarged or saved. The owner screenshotted it.
//
// jsdom lays nothing out, so what these pin is the structure the fix is made
// of: a run of adjacent pictures becomes ONE object with a stage and a
// filmstrip, a lone picture is left exactly as it was, and the collecting is
// idempotent — an answer re-renders on every frame while it streams.
import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../lib/markdown'

const render = (src: string) => {
  const host = document.createElement('div')
  host.innerHTML = renderMarkdown(src)
  return host
}

const FOUR = ['![หนึ่ง](a.png)', '![สอง](b.png)', '![สาม](c.png)', '![สี่](d.png)'].join('\n')
// The same four with a blank line between each, which is four paragraphs — and
// which is what the model that produced the owner's screenshot actually wrote.
const APART = ['![หนึ่ง](a.png)', '![สอง](b.png)', '![สาม](c.png)', '![สี่](d.png)'].join('\n\n')

describe('a run of images becomes one gallery', () => {
  it('collects every picture onto one stage with one filmstrip', () => {
    const host = render(FOUR)

    expect(host.querySelectorAll('.img-gallery').length).toBe(1)
    expect(host.querySelectorAll('.gallery-stage .gallery-shot').length).toBe(4)
    expect(host.querySelectorAll('.gallery-strip .gallery-thumb').length).toBe(4)
  })

  it('shows the first one and says which of how many', () => {
    const host = render(FOUR)

    const shown = host.querySelectorAll('.gallery-shot.shown')
    expect(shown.length).toBe(1)
    expect(shown[0].getAttribute('src')).toBe('/aetox-file/a.png')
    expect(host.querySelector('.gallery-count')?.textContent).toBe('1 / 4')
    expect(host.querySelector('.img-gallery')?.getAttribute('data-shown')).toBe('0')
  })

  // The handler in Chat.svelte has this markup and nothing else — no component
  // holds the index, because {@html} markup cannot carry one.
  it('gives the handler somewhere to steer from', () => {
    const host = render(FOUR)

    expect(host.querySelector('.gallery-prev')?.getAttribute('data-step')).toBe('-1')
    expect(host.querySelector('.gallery-next')?.getAttribute('data-step')).toBe('1')
    expect(
      Array.from(host.querySelectorAll('.gallery-thumb'), (b) => b.getAttribute('data-at')),
    ).toEqual(['0', '1', '2', '3'])
    expect(host.querySelector('.gallery-open')).toBeTruthy()
  })

  // Spans, not divs. The run lives inside the <p> markdown wrapped it in, and a
  // <div> there is closed out of its paragraph by the HTML parser the moment
  // this string is set as innerHTML — which is how every one of these tests
  // reads it back, and how the app does too.
  it('survives being parsed back inside its paragraph', () => {
    // A run that shares its paragraph with a sentence, so the paragraph stays
    // and the gallery has to be able to live inside it. (A paragraph of
    // nothing but pictures is dissolved instead — see below.)
    const host = render('![a](a.png)\n![b](b.png)\nสี่แบบครับ')

    const gallery = host.querySelector('.img-gallery')
    expect(gallery?.tagName).toBe('SPAN')
    expect(gallery?.closest('p')).not.toBeNull()
  })

  // markdown lifts a picture per line into <img><br><img>, and a break between
  // two pictures is punctuation, not something standing between them.
  it('reads one picture per line as adjacent', () => {
    expect(render(FOUR).querySelectorAll('.gallery-shot').length).toBe(4)
    expect(render('![a](a.png) ![b](b.png)').querySelectorAll('.gallery-shot').length).toBe(2)
  })

  // The shape that actually arrived. A model writes a blank line between its
  // images as readily as a single one, and four paragraphs of one picture each
  // are four pictures — the first rule here missed every one of them and left
  // the columns standing in a build that had the whole gallery in it.
  it('collects pictures markdown put in a paragraph each', () => {
    const host = render(APART)

    expect(host.querySelectorAll('.img-gallery').length).toBe(1)
    expect(host.querySelectorAll('.gallery-shot').length).toBe(4)
    expect(host.querySelector('.gallery-count')?.textContent).toBe('1 / 4')
  })

  // The paragraphs are gone, not left standing empty around it — an empty <p>
  // still holds its own margins open.
  it('dissolves the paragraphs it swallowed', () => {
    const host = render(APART)

    expect(host.querySelectorAll('p').length).toBe(0)
    expect(host.querySelector('.img-gallery')?.parentElement).toBe(host)
  })

  it('collects a mixture of paragraphs and lines into one gallery', () => {
    const host = render('![a](a.png)\n![b](b.png)\n\n![c](c.png)')

    expect(host.querySelectorAll('.img-gallery').length).toBe(1)
    expect(host.querySelectorAll('.gallery-shot').length).toBe(3)
  })

  it('keeps the sentence that introduced them', () => {
    const host = render(`ได้แล้วครับ ออกมา 4 แบบให้เลือก:\n\n${APART}`)

    expect(host.textContent).toContain('ได้แล้วครับ')
    expect(host.querySelectorAll('.gallery-shot').length).toBe(4)
    expect(host.querySelectorAll('.img-gallery').length).toBe(1)
  })

  // And takes the punctuation with it. `breaks:true` puts a <br> between each
  // pair, and four pictures lifted onto a stage left three of them standing in
  // the paragraph — three blank lines under the gallery, growing with the
  // number of pictures the way the columns used to.
  it('takes the breaks that were between the pictures', () => {
    expect(render(FOUR).querySelectorAll('br').length).toBe(0)
  })

  // The break after the last picture is not one of those: it is what separates
  // the gallery from the sentence written under it.
  it('leaves the break that ends the run', () => {
    const host = render('![a](a.png)\n![b](b.png)\nสี่แบบครับ')

    expect(host.querySelector('.img-gallery')).toBeTruthy()
    expect(host.querySelectorAll('br').length).toBe(1)
    expect(host.textContent).toContain('สี่แบบครับ')
  })

  it('keeps the alt text on the picture and gives the thumbnail a name', () => {
    const host = render(FOUR)

    expect(host.querySelector('.gallery-shot')?.getAttribute('alt')).toBe('หนึ่ง')
    expect(host.querySelector('.gallery-thumb')?.getAttribute('aria-label')).toBe('หนึ่ง')
  })
})

describe('what is not a run', () => {
  it('leaves a lone picture exactly as it was', () => {
    const host = render('![คนเดียว](a.png)')

    expect(host.querySelector('.img-gallery')).toBeNull()
    // "as it was" is about the STAGE: a lone picture is not collected onto one.
    // Its address is translated like every other picture's — that pass runs
    // before the galleries are built and is not what this test is guarding
    // (markdown.ts, hostRelativeImage).
    expect(host.querySelector('img')?.getAttribute('src')).toBe('/aetox-file/a.png')
  })

  // A picture with a sentence beside it is illustrating that sentence.
  it('does not reach across words', () => {
    const host = render('![a](a.png) แล้วก็ ![b](b.png)')

    expect(host.querySelector('.img-gallery')).toBeNull()
    expect(host.querySelectorAll('img').length).toBe(2)
  })

  // A caption is prose, and prose ends a run.
  it('does not reach past a paragraph that says something', () => {
    const host = render('![a](a.png)\n\nแบบแรก\n\n![b](b.png)')

    expect(host.querySelector('.img-gallery')).toBeNull()
    expect(host.querySelectorAll('img').length).toBe(2)
  })

  // An answer re-renders on every frame while it streams, and a gallery built
  // inside a gallery would nest one per frame.
  it('does not build a gallery inside a gallery on a second pass', () => {
    const once = renderMarkdown(FOUR)
    expect(renderMarkdown(FOUR)).toBe(once)

    const host = document.createElement('div')
    host.innerHTML = renderMarkdown(once)
    expect(host.querySelectorAll('.img-gallery').length).toBe(1)
    expect(host.querySelectorAll('.gallery-shot').length).toBe(4)
  })
})

// A shop's results are not bare images: each product is a picture wrapped in a
// link to the page you would buy it on. Reading only for <img> saw none of them
// and left the columns exactly as they were (owner, ตอนเอาลิงก์ร้านค้ามา).
describe('pictures that are also links', () => {
  const SHOP = [
    '[![กระเป๋า](a.png)](https://shop.example.com/a)',
    '[![รองเท้า](b.png)](https://shop.example.com/b)',
    '[![หมวก](c.png)](https://shop.example.com/c)',
  ].join('\n')

  it('collects them, anchors and all', () => {
    const host = render(SHOP)

    expect(host.querySelectorAll('.gallery-shot').length).toBe(3)
    expect(host.querySelector('.gallery-shot')?.tagName).toBe('A')
  })

  // The link is what the model sent. Moving the picture must not lose it —
  // clicking the poster still has to reach the shop, through the same handler
  // any link in an answer already uses.
  it('keeps the destination on the picture', () => {
    const host = render(SHOP)

    expect(host.querySelector('.gallery-shot')?.getAttribute('href')).toBe('https://shop.example.com/a')
    expect(host.querySelector('.gallery-shot img')?.getAttribute('src')).toBe('/aetox-file/a.png')
  })

  // Inside the anchor, so it appears and disappears with the shot it names and
  // cannot go out of step with what is on screen.
  it('says where it goes, on the shot itself', () => {
    const host = render(SHOP)

    const badge = host.querySelector('.gallery-shot .gallery-link')
    expect(badge?.textContent).toContain('shop.example.com')
  })

  // The filmstrip chooses which picture is up. A thumbnail that navigated would
  // leave no way to look at the thing before buying it.
  it('builds the filmstrip out of buttons, not links', () => {
    const host = render(SHOP)

    const picks = host.querySelectorAll('.gallery-thumb')
    expect(picks.length).toBe(3)
    for (const pick of picks) expect(pick.tagName).toBe('BUTTON')
    expect(host.querySelector('.gallery-thumb img')?.getAttribute('src')).toBe('/aetox-file/a.png')
  })

  it('mixes a linked product and a plain picture into one gallery', () => {
    const host = render('[![ก](a.png)](https://shop.example.com/a)\n![ข](b.png)')

    expect(host.querySelectorAll('.gallery-shot').length).toBe(2)
    expect(host.querySelectorAll('.gallery-link').length).toBe(1)
  })

  // A link with a word in it is a sentence with a picture in it.
  it('leaves a link that carries text alone', () => {
    const host = render(
      '[![ก](a.png) ดูร้าน](https://shop.example.com/a)\n[![ข](b.png) ดูร้าน](https://shop.example.com/b)',
    )

    expect(host.querySelector('.img-gallery')).toBeNull()
  })
})
