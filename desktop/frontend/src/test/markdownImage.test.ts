import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../lib/markdown'

// A picture in an answer is a file on the desk.
//
// `![a cat](art/cat.jpg)` is what a model writes the moment it has made one,
// and it is the right place for the picture to be — in the sentence announcing
// it. What it used to render to was <img src="art/cat.jpg">, resolved by the
// webview against its own origin, where that file has never existed: the first
// real image_make call produced a picture and drew a broken icon (owner,
// 7 ก.ย., with the screenshot).
//
// These pin the translation. The half that matters most is the second block —
// an address that is already absolute must be left alone, or a picture from the
// web becomes a 404 on the desk.
describe('a relative picture in prose goes through the file host', () => {
  const srcOf = (md: string) => /<img[^>]*\ssrc="([^"]*)"/.exec(renderMarkdown(md))?.[1] ?? ''

  it('rewrites a project-relative path', () => {
    expect(srcOf('![a cat](art/cat.jpg)')).toBe('/aetox-file/art/cat.jpg')
  })

  it('treats a leading slash and ./ as the same project-relative path', () => {
    expect(srcOf('![a cat](/art/cat.jpg)')).toBe('/aetox-file/art/cat.jpg')
    expect(srcOf('![a cat](./art/cat.jpg)')).toBe('/aetox-file/art/cat.jpg')
  })

  it('encodes a name that needs it, without eating the separators', () => {
    // Thai names are ordinary here — the assistant writes them. And a space
    // only reaches this code through markdown's angle-bracket form, which is
    // the only spelling marked accepts for one.
    expect(srcOf('![x](art/แมว.png)')).toBe('/aetox-file/art/' + encodeURIComponent('แมว.png'))
    expect(srcOf('![x](<my art/a cat.png>)')).toBe('/aetox-file/my%20art/a%20cat.png')
  })

  it('leaves an address that already resolves exactly as it is', () => {
    for (const src of [
      'https://example.com/cat.jpg',
      'http://example.com/cat.jpg',
      '/aetox-file/art/cat.jpg',
    ]) {
      expect(srcOf(`![x](${src})`)).toBe(src)
    }
  })

  it('does not translate a second time', () => {
    // The DOM pass can run over markup that already carries a translated src —
    // markdown passes raw HTML through, and a re-render is ordinary. Two passes
    // must produce the same address as one.
    const once = renderMarkdown('![a cat](art/cat.jpg)')
    expect(srcOf(once)).toBe('/aetox-file/art/cat.jpg')
  })
})
