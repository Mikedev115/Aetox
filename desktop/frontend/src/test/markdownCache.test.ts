// An answer is rendered once and remembered.
//
// Turning a reply into markup is the most expensive thing markdown.ts does —
// marked, then highlight.js per fenced block, then DOMPurify, then a dozen
// passes over the fragment — and the window asks for it far more often than a
// reply changes. Measured in Chromium on 9 ก.ย.: a 7.2KB answer took 3.8ms to
// render and 0.015ms to hand back. A chat switch re-renders every bubble in the
// transcript (`cockpit.chat = restoreTranscript(...)`), so on a long
// conversation that difference is the whole freeze.
//
// What is checked here is not the speed — it is the one thing a cache can get
// wrong, which is answering with markup that is no longer true.
import { describe, it, expect } from 'vitest'
import { renderMarkdown, setRunnableLanguages } from '../lib/markdown'
import { setLocale } from '../lib/i18n.svelte'

const FENCE = '```bash\necho hello\n```'

describe('a rendered answer, asked for twice', () => {
  it('comes back the same', () => {
    const text = '# หัวข้อ\n\nข้อความ **หนา** และ `code`\n\n' + FENCE
    expect(renderMarkdown(text)).toBe(renderMarkdown(text))
  })

  // The Run button is part of a rendered block and the list behind it arrives
  // from the engine after boot (App.RunnableLanguages), so answers drawn before
  // it landed were drawn without one. Remembering those would keep the button
  // off them for the life of the app — the one stale answer this cache could
  // give, and the reason setRunnableLanguages throws its contents away.
  it('is drawn again when the engine says which fences can run', () => {
    const text = 'ลองสิ\n\n' + FENCE
    setRunnableLanguages({})
    expect(renderMarkdown(text)).not.toContain('code-run')

    setRunnableLanguages({ bash: 'shell' })

    expect(renderMarkdown(text)).toContain('code-run')
    setRunnableLanguages({})
  })

  // The markup carries WORDS — a code block's คัดลอก button is
  // t('chat.copyCode') — so the same answer is a different answer in another
  // language. Keyed rather than cleared, because a language is switched back.
  it('is drawn again in the language the reader switched to', () => {
    const text = 'สวัสดี\n\n' + FENCE
    setLocale('th')
    const thai = renderMarkdown(text)
    setLocale('en')
    const english = renderMarkdown(text)

    expect(english).not.toBe(thai)
    setLocale('th')
    expect(renderMarkdown(text)).toBe(thai)
  })
})
