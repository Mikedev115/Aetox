// The four starting files Settings offers when the identity folder is empty.
//
// What is pinned here is not their wording — it is that they follow the
// language the user picked. They were Thai string literals in identity.svelte.ts
// while every other user-facing string was in the locale files, which made this
// the one screen that ignored the choice: someone who picked English opened the
// file where they write down who they are and found Thai in it.
//
// The filenames are asserted to be the SAME in every language on purpose. They
// are addresses the engine reads (internal/prompt folds every *.md in the
// identity dir into every prompt), not labels, and translating one would quietly
// create a second file instead of opening the first.
import { describe, it, expect, afterEach } from 'vitest'
import { identityTemplates } from '../lib/identity.svelte'
import { setLocale } from '../lib/i18n.svelte'

const names = () => identityTemplates().map((t) => t.name)

afterEach(() => setLocale('th'))

describe('identity templates', () => {
  it('ship identity and thinking in English everywhere, and context.md blank', () => {
    setLocale('th')
    const thai = identityTemplates()
    setLocale('en')
    const english = identityTemplates()

    expect(thai).toHaveLength(3)
    expect(english).toHaveLength(3)

    for (const [i, tpl] of english.entries()) {
      expect(tpl.content).not.toMatch(/[ก-๙]/)
      // identity.md and thinking.md are prompt text and ship in English in
      // every locale (14 ก.ย. 2026). context.md is the person's own words about
      // themselves and starts blank in every locale (owner, the same day:
      // "ควรจะโล่งเป็นค่าเริ่มต้น") — the engine folds nothing from an empty file,
      // so the blank costs no prompt.
      expect(tpl.content).toBe(thai[i].content)
      if (tpl.name === 'context.md') {
        expect(tpl.content).toBe('')
      } else {
        expect(tpl.content.startsWith('# ')).toBe(true)
        expect(tpl.content.trim().length).toBeGreaterThan(20)
      }
    }
  })

  it('keep the same filenames in every language', () => {
    setLocale('th')
    const thai = names()
    setLocale('en')
    expect(names()).toEqual(thai)
    // skills.md left 14 ก.ย. 2026 (owner: no clear use).
    expect(thai).toEqual(['identity.md', 'thinking.md', 'context.md'])
  })
})
