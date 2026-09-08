// The table that turns three engine fields into a row a person can read.
//
// Tested here rather than through the chat because it is a table: what is worth
// pinning is that `search` with action `grep` is a READ and says "ค้นในโค้ด",
// and that a `browser` row whose action never arrived still says a true
// sentence. Rendering a whole transcript to assert that would test the renderer.
import { describe, it, expect } from 'vitest'
import {
  toolFamily, toolIcon, toolVerbKey, toolFallbackVerb, toolSubject, toolServer,
  mcpParts, serverSlot, splitSubject, linkDomain, linkInitials, markOpenedLinks,
} from '../lib/toolFace'
import type { ToolStep } from '../lib/types'

const step = (s: Partial<ToolStep>): ToolStep =>
  ({ label: '', state: 'done', startedAt: 0, ...s })

describe('which family a tool belongs to', () => {
  // The two vocabularies §99 left behind: the packed name a live event carries,
  // and the per-action name a turn stored before packing carries. Both have to
  // land in the same place or one half of the transcript loses its colours.
  it('reads the packed name and the per-action name alike', () => {
    expect(toolFamily(step({ name: 'search' }))).toBe('read')
    expect(toolFamily(step({ name: 'grep' }))).toBe('read')
    expect(toolFamily(step({ name: 'change' }))).toBe('write')
    expect(toolFamily(step({ name: 'write' }))).toBe('write')
    expect(toolFamily(step({ name: 'browser' }))).toBe('web')
    expect(toolFamily(step({ name: 'browser_click' }))).toBe('web')
    expect(toolFamily(step({ name: 'shell' }))).toBe('shell')
    expect(toolFamily(step({ name: 'pr' }))).toBe('shell')
    expect(toolFamily(step({ name: 'media_read' }))).toBe('media')
    expect(toolFamily(step({ name: 'task' }))).toBe('task')
  })

  // A row stored before `name` existed has only the joined label, and its first
  // word is the tool name by construction.
  it('falls back to the first word of an old label', () => {
    expect(toolFamily(step({ label: 'read internal/skill/edit.go' }))).toBe('read')
  })

  // The row the whole job started from. A bridged tool's WORK is unknowable
  // from here — nobody can enumerate every server anyone will connect — but its
  // provenance is exact, because internal/mcp names every one `server_tool`.
  it('recognises a bridged tool by the name its adapter built', () => {
    expect(toolFamily(step({ name: 'canva_generate-design' }))).toBe('mcp')
    expect(toolIcon('mcp')).toBe('plug')
    expect(toolServer(step({ name: 'canva_generate-design' }))).toBe('canva')
    // The identifier taken off what is already a verb phrase.
    expect(toolFallbackVerb(step({ name: 'canva_generate-design' }))).toBe('generate design')
    expect(toolFallbackVerb(step({ name: 'slack_send_message' }))).toBe('send message')
  })

  // The guard that keeps the split off our own tools. Every one of these
  // carries an underscore, and `skills_list` read as server `skills` doing
  // `list` would be a bridged tool that is not one.
  it('never reads a first-party underscore as a server', () => {
    for (const name of ['web_search', 'shell_output', 'skills_list', 'todo_write', 'image_ocr', 'desk_close', 'computer_apps', 'computer_read', 'computer_capture']) {
      expect({ name, parts: mcpParts(name) }).toEqual({ name, parts: null })
    }
  })

  // A colour is only worth having if it is stable: two Canva rows must look
  // like two Canva rows, in this turn and in one opened next month.
  it('gives a server the same chart slot every time', () => {
    expect(serverSlot('canva')).toBe(serverSlot('canva'))
    expect(serverSlot('canva')).toBeGreaterThanOrEqual(1)
    expect(serverSlot('canva')).toBeLessThanOrEqual(5)
  })

  // What is left over is genuinely unclassifiable, and gets the neutral tile.
  // Inventing a colour for it would make the palette say less, not more.
  it('puts anything else in the neutral pile', () => {
    expect(toolFamily(step({ name: 'time' }))).toBe('other')
    expect(toolIcon('other')).toBe('wrench')
  })
})

describe('the verb the row says', () => {
  // The whole reason ToolEvent carries Act. `browser` alone is twelve sentences
  // and `change` is five; without the action the row can only name the pack.
  it('picks the act inside a packed tool', () => {
    expect(toolVerbKey(step({ name: 'browser', act: 'open' }))).toBe('tool.browserOpen')
    expect(toolVerbKey(step({ name: 'browser', act: 'click' }))).toBe('tool.browserClick')
    expect(toolVerbKey(step({ name: 'change', act: 'delete' }))).toBe('tool.delete')
    expect(toolVerbKey(step({ name: 'search', act: 'grep' }))).toBe('tool.grep')
    // The pair that were one row until Act reached the UI: hiring somebody, and
    // sitting waiting on somebody already hired.
    expect(toolVerbKey(step({ name: 'task', act: 'start' }))).toBe('tool.delegate')
    expect(toolVerbKey(step({ name: 'task', act: 'collect' }))).toBe('tool.awaitDelegate')
  })

  // A turn stored before ToolPart carried the act, and a call whose arguments
  // never parsed. The bare-name entry has to be a true sentence, not a stub.
  it('still says something true with no act at all', () => {
    expect(toolVerbKey(step({ name: 'browser' }))).toBe('tool.browserOpen')
    expect(toolVerbKey(step({ name: 'read' }))).toBe('tool.read')
  })

  // Nothing in the table, and no underscore to read a server out of: a
  // first-party tool added after this file was last looked at. It still says
  // which of its actions ran, which is more than the old label managed.
  it('names the tool and its action when nothing is mapped', () => {
    expect(toolVerbKey(step({ name: 'canva_generate-design' }))).toBe('')
    expect(toolFallbackVerb(step({ name: 'frobnicate', act: 'render' }))).toBe('frobnicate render')
    expect(toolFallbackVerb(step({ name: 'frobnicate' }))).toBe('frobnicate')
  })
})

describe('the subject', () => {
  it('prefers the engine field and falls back to the label', () => {
    expect(toolSubject(step({ subject: 'a/b.go', label: 'read a/b.go' }))).toBe('a/b.go')
    expect(toolSubject(step({ label: 'read a/b.go' }))).toBe('a/b.go')
    // A tool that takes nothing nameable — todo_write — has no subject in
    // either field, and must not be handed its own name as one.
    expect(toolSubject(step({ label: 'todo_write' }))).toBe('')
  })

  // Dim where, bright what. Split at the last separator only: nobody reads a
  // path segment by segment, they read the end of it.
  it('cuts a path at its last separator', () => {
    expect(splitSubject('internal/skill/edit.go')).toEqual({ head: 'internal/skill/', tail: 'edit.go' })
    expect(splitSubject('C:\\Users\\x\\note.md')).toEqual({ head: 'C:\\Users\\x\\', tail: 'note.md' })
    // Nothing to locate: a query, a bare file name.
    expect(splitSubject('react server components')).toEqual({ head: '', tail: 'react server components' })
    // A shell command has slashes and is not a path. Cut at the last one it
    // read `go test ./internal/` dim and `...` bright — the emphasis inverted.
    expect(splitSubject('go test ./internal/...')).toEqual({ head: '', tail: 'go test ./internal/...' })
    // A trailing separator names nothing after it, so the whole thing is name.
    expect(splitSubject('src/lib/')).toEqual({ head: '', tail: 'src/lib/' })
  })
})

describe('a search result on the card', () => {
  it('reads the host and its initials off the URL', () => {
    expect(linkDomain('https://www.go.dev/blog/go1.24')).toBe('go.dev')
    expect(linkInitials('https://react.dev/rfc')).toBe('RE')
    // A URL that will not parse still draws as something.
    expect(linkDomain('go.dev/blog')).toBe('go.dev')
  })

  // The badge cannot ride on the search's own event: the fetch that earns it
  // happens two or three calls later. It is a fact about the turn.
  it('marks the results the agent went back and read', () => {
    const steps: ToolStep[] = [
      step({
        name: 'web_search', subject: 'rsc',
        links: [
          { title: 'RFC', url: 'https://react.dev/rfc' },
          { title: 'Notes', url: 'https://go.dev/blog/go1.24' },
        ],
      }),
      step({ name: 'web_fetch', subject: 'https://react.dev/rfc/' }),
    ]
    markOpenedLinks(steps)
    expect(steps[0].links?.[0].opened).toBe(true)
    expect(steps[0].links?.[1].opened).toBeUndefined()
  })

  // A trailing slash and a #fragment are what a model adds on the way past when
  // it copies a URL out of a result list into a fetch call.
  it('ignores a trailing slash and a fragment', () => {
    const steps: ToolStep[] = [
      step({ name: 'web_search', links: [{ title: 'X', url: 'https://go.dev/blog' }] }),
      step({ name: 'browser', act: 'open', subject: 'https://go.dev/blog/#intro' }),
    ]
    markOpenedLinks(steps)
    expect(steps[0].links?.[0].opened).toBe(true)
  })

  // A click carries a selector, not a URL. Listing which browser actions count
  // says so, rather than leaving it to the fact that a selector never matches.
  it('does not count a click as reading the page', () => {
    const steps: ToolStep[] = [
      step({ name: 'web_search', links: [{ title: 'X', url: 'https://go.dev/blog' }] }),
      step({ name: 'browser', act: 'click', subject: 'https://go.dev/blog' }),
    ]
    markOpenedLinks(steps)
    expect(steps[0].links?.[0].opened).toBeUndefined()
  })
})
