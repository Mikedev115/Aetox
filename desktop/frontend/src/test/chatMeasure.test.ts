// How wide each side of the conversation is allowed to get.
//
// Read off disk like themeContrast and composerNarrow do: vitest stubs CSS
// imports to "", and a rule checked against an empty stylesheet passes.
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'

const css = readFileSync('src/style.css', 'utf8')

/** The declarations of the rule whose selector list ends with `selector`. */
function rule(selector: string): string {
  const at = css.indexOf(selector + ' {')
  if (at < 0) throw new Error('no rule for ' + selector)
  const open = css.indexOf('{', at)
  const close = css.indexOf('}', open)
  return css.slice(open + 1, close)
}

describe('the answer is held to a reading measure', () => {
  it('keeps the assistant row to four fifths of the conversation column', () => {
    expect(rule('.msg.bot')).toContain('max-width:80%')
  })

  // The row owns the requested 80% speaker measure. Neither the bubble nor its
  // prose gets a second cap, so text, code, tables and drawings share one edge.
  it('does not cap the bubble itself', () => {
    expect(rule('.bubble')).not.toContain('max-width')
  })

  it('does not narrow assistant prose a second time', () => {
    expect(css).not.toContain('.msg:not(.user) .bubble .markdown-body > p')
  })
})
