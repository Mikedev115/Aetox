// The working tree as a room (DECISIONS §161.4).
//
// Three claims worth pinning: the room says which desk it belongs to, it does
// no work until a row is opened, and the diff it draws comes from the engine
// rather than from anything computed here.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent, waitFor } from '@testing-library/svelte'
import GitPane from '../lib/workbench/GitPane.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import {
  GitWorkingTree,
  GitFileDiff,
  GitCommitFiles,
  GitSuggestCommitMessage,
  GitSuggestSplitCommits,
  GitSplitCancel,
} from './mocks/wailsApp'
import { EventsOn } from './mocks/wailsRuntime'

const CHANGED = [
  { path: 'internal/skill/hunk.go', status: 'U', added: 284, removed: 0 },
  { path: 'ARCHITECTURE.md', status: 'M', added: 5, removed: 1 },
]

const DIFF = '+++ ARCHITECTURE.md\n@@ -565,1 +565,2 @@\n old row\n+| §161 | a new row |'

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  cockpit.project.branch = 'main'
  vi.mocked(GitWorkingTree).mockResolvedValue(CHANGED as any)
  vi.mocked(GitFileDiff).mockResolvedValue(DIFF as any)
  vi.mocked(GitCommitFiles).mockResolvedValue(undefined as any)
  vi.mocked(GitSuggestCommitMessage).mockResolvedValue('feat: generated commit message' as any)
  vi.mocked(GitSuggestSplitCommits).mockResolvedValue([] as any)
})

describe('GitPane', () => {
  it('names the branch, the totals and every changed file', async () => {
    const { container, getByText } = render(GitPane)
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))

    expect(container.querySelector('.gp-where b')?.textContent).toBe('main')
    expect(getByText('hunk.go')).toBeTruthy()
    expect(getByText('internal/skill')).toBeTruthy()
    // The header totals the rows rather than asking for a second number that
    // could disagree with them.
    expect(container.querySelector('.gp-totals .add')?.textContent).toBe('+289')
    expect(container.querySelector('.gp-totals .del')?.textContent).toBe('-1')
  })

  // Said out loud, because a room that exists on one desk and nowhere else owes
  // the reader that sentence instead of leaving them to hunt.
  it('says it is on the โค้ด desk only', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-note')).not.toBeNull())
    expect(container.querySelector('.gp-note')?.textContent).toContain('Code desk only')
  })

  // A working tree of forty files is ordinary. Forty diffs fetched for a list
  // nobody has opened is work done on the chance it is wanted.
  it('fetches nothing until a row is opened, then fetches once', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))
    expect(vi.mocked(GitFileDiff)).not.toHaveBeenCalled()

    await fireEvent.click(container.querySelectorAll('.gp-row')[1] as HTMLElement)
    await waitFor(() => expect(container.querySelector('.dl.add')).not.toBeNull())
    expect(vi.mocked(GitFileDiff)).toHaveBeenCalledWith('ARCHITECTURE.md')
    expect(container.querySelector('.dl.add .tx')?.textContent).toContain('§161')

    // Shut and reopened, the answer is the one already in hand.
    await fireEvent.click(container.querySelectorAll('.gp-row')[1] as HTMLElement)
    await fireEvent.click(container.querySelectorAll('.gp-row')[1] as HTMLElement)
    expect(vi.mocked(GitFileDiff)).toHaveBeenCalledTimes(1)
  })

  // The list most often goes stale because the agent above it just finished
  // editing. A panel still saying "clean" beside a chat reporting three edited
  // files is worse than no panel: it is confidently wrong.
  //
  // Rendered as a tab that is NOT the one in front: this is the turn-end path
  // on its own, and the live timer has a test of its own below.
  it('reads the tree again when a turn finishes', async () => {
    cockpit.awaitingReply = false
    const { container } = render(GitPane, { active: false })
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))
    expect(vi.mocked(GitWorkingTree)).toHaveBeenCalledTimes(1)

    cockpit.awaitingReply = true
    await waitFor(() => expect(vi.mocked(GitWorkingTree)).toHaveBeenCalledTimes(1))
    cockpit.awaitingReply = false
    await waitFor(() => expect(vi.mocked(GitWorkingTree)).toHaveBeenCalledTimes(2))
  })

  // The other half of staying current, and the one a turn cannot cover: the tree
  // changes for reasons this pane never hears about — an editor, a formatter, a
  // build, git in a terminal beside it — so while it is the tab in front it
  // re-reads on its own rather than waiting to be asked (owner, 2026-09-11:
  // "แสดงแบบเรียลไทม์ไม่ใช่ต้องมาคอยกดรีเองบ่อยๆ").
  it('re-reads on a timer while it is the tab in front', async () => {
    vi.useFakeTimers()
    try {
      render(GitPane, { active: true })
      // The first read is on mount, not on the first tick.
      await vi.advanceTimersByTimeAsync(0)
      const mounted = vi.mocked(GitWorkingTree).mock.calls.length
      expect(mounted).toBeGreaterThan(0)

      // No press, no turn, no event: whatever arrives here is the timer's.
      await vi.advanceTimersByTimeAsync(2100)
      expect(vi.mocked(GitWorkingTree).mock.calls.length).toBeGreaterThan(mounted)
    } finally {
      vi.useRealTimers()
    }
  })

  // Every other slot in the desk is `display: none`, and a working tree nobody
  // can see is git processes spent on nothing. Switching away has to stop it.
  it('stops reading while it is not the tab in front', async () => {
    vi.useFakeTimers()
    try {
      const { rerender } = render(GitPane, { active: true })
      await vi.advanceTimersByTimeAsync(0)

      await rerender({ active: false })
      await vi.advanceTimersByTimeAsync(0)
      const hidden = vi.mocked(GitWorkingTree).mock.calls.length
      await vi.advanceTimersByTimeAsync(6000)
      expect(vi.mocked(GitWorkingTree).mock.calls.length).toBe(hidden)
    } finally {
      vi.useRealTimers()
    }
  })

  // A read that failed is not a tree that is clean. git ran out of its budget
  // (owner, 12 ก.ย., on a machine where a git process was taking seconds), and
  // the honest answer is the rows from the read before it plus a line saying
  // so — never "nothing changed" over fifty-eight files.
  it('keeps the last tree and says so when a read fails', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))
    expect(container.querySelector('.gp-stale')).toBeNull()

    vi.mocked(GitWorkingTree).mockRejectedValueOnce(new Error('git took too long to answer'))
    await fireEvent.click(container.querySelector('.gp-head .icobtn') as HTMLElement)
    await waitFor(() => expect(container.querySelector('.gp-stale')).not.toBeNull())
    expect(container.querySelectorAll('.gp-row').length).toBe(2)
    expect(container.querySelector('.gp-commit-area')).not.toBeNull()

    // And the next read that answers clears the line.
    await fireEvent.click(container.querySelector('.gp-head .icobtn') as HTMLElement)
    await waitFor(() => expect(container.querySelector('.gp-stale')).toBeNull())
  })

  // The tick is paced by the read: a git that takes long is asked less often,
  // not queued behind itself. Four times the read, never under two seconds.
  it('waits longer between reads when the last one was slow', async () => {
    vi.useFakeTimers()
    try {
      // The mount read takes three seconds of (fake) time to answer.
      vi.mocked(GitWorkingTree).mockImplementationOnce(
        () => new Promise((resolve) => setTimeout(() => resolve(CHANGED as any), 3000)),
      )
      render(GitPane, { active: true })
      await vi.advanceTimersByTimeAsync(3000)
      const afterMount = vi.mocked(GitWorkingTree).mock.calls.length
      expect(afterMount).toBeGreaterThan(0)

      // The old two-second tick would have read again by now; the paced one
      // waits four times three seconds.
      await vi.advanceTimersByTimeAsync(2100)
      expect(vi.mocked(GitWorkingTree).mock.calls.length).toBe(afterMount)
      await vi.advanceTimersByTimeAsync(10000)
      expect(vi.mocked(GitWorkingTree).mock.calls.length).toBeGreaterThan(afterMount)
    } finally {
      vi.useRealTimers()
    }
  })

  it('says the tree is clean rather than drawing an empty list', async () => {
    vi.mocked(GitWorkingTree).mockResolvedValue([] as any)
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-empty')).not.toBeNull())
    expect(container.querySelector('.gp-empty')?.textContent).toContain('Nothing changed')
    expect(container.querySelector('.gp-list')).toBeNull()
    expect(container.querySelector('.gp-commit-area')).toBeNull()
  })

  it('renders commit area with mode tabs when changes exist', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-commit-area')).not.toBeNull())

    const tabs = container.querySelectorAll('.gp-mode-tab')
    expect(tabs.length).toBe(2)
    expect(tabs[0].classList.contains('active')).toBe(true) // default: split
  })

  it('supports manual commit flow: message entry and committing selected files', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-commit-area')).not.toBeNull())

    // Switch to manual mode
    const tabs = container.querySelectorAll('.gp-mode-tab')
    await fireEvent.click(tabs[1] as HTMLElement)

    const textarea = container.querySelector('.gp-msg-input') as HTMLTextAreaElement
    expect(textarea).not.toBeNull()

    // Type commit message
    await fireEvent.input(textarea, { target: { value: 'feat: manual commit test' } })

    const commitBtn = container.querySelector('.gp-manual-box .gp-commit-btn') as HTMLButtonElement
    expect(commitBtn.disabled).toBe(false)
    await fireEvent.click(commitBtn)

    expect(vi.mocked(GitCommitFiles)).toHaveBeenCalledWith(
      'feat: manual commit test',
      ['internal/skill/hunk.go', 'ARCHITECTURE.md'],
    )
  })

  it('supports generating commit message via AI in manual mode', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-commit-area')).not.toBeNull())

    // Switch to manual mode
    const tabs = container.querySelectorAll('.gp-mode-tab')
    await fireEvent.click(tabs[1] as HTMLElement)

    const genBtn = container.querySelector('.gp-gen-btn') as HTMLButtonElement
    expect(genBtn).not.toBeNull()
    await fireEvent.click(genBtn)

    expect(vi.mocked(GitSuggestCommitMessage)).toHaveBeenCalledWith([
      'internal/skill/hunk.go',
      'ARCHITECTURE.md',
    ])

    await waitFor(() => {
      const textarea = container.querySelector('.gp-msg-input') as HTMLTextAreaElement
      expect(textarea.value).toBe('feat: generated commit message')
    })
  })

  it('supports committing a subset of selected files in manual mode', async () => {
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))

    // Deselect first file checkbox
    const checkboxes = container.querySelectorAll('.gp-row-checkbox') as NodeListOf<HTMLInputElement>
    expect(checkboxes.length).toBe(2)
    await fireEvent.click(checkboxes[0])

    // Switch to manual mode
    const tabs = container.querySelectorAll('.gp-mode-tab')
    await fireEvent.click(tabs[1] as HTMLElement)

    const textarea = container.querySelector('.gp-msg-input') as HTMLTextAreaElement
    await fireEvent.input(textarea, { target: { value: 'docs: update architecture only' } })

    const commitBtn = container.querySelector('.gp-manual-box .gp-commit-btn') as HTMLButtonElement
    await fireEvent.click(commitBtn)

    // Should only commit the second file (ARCHITECTURE.md)
    expect(vi.mocked(GitCommitFiles)).toHaveBeenCalledWith(
      'docs: update architecture only',
      ['ARCHITECTURE.md'],
    )
  })

  it('supports smart split commit: groups changes and commits single group', async () => {
    vi.mocked(GitSuggestSplitCommits).mockResolvedValue([
      {
        title: 'skill: hunk updates',
        message: 'feat(skill): update hunk parser',
        files: ['internal/skill/hunk.go'],
      },
      {
        title: 'docs: architecture record',
        message: 'docs: update architecture record',
        files: ['ARCHITECTURE.md'],
      },
    ] as any)

    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-split-section')).not.toBeNull())

    // Click smart split button to analyze
    const splitBtn = container.querySelector('.gp-split-section .gp-commit-btn') as HTMLButtonElement
    await fireEvent.click(splitBtn)

    // Wait for cards to render
    await waitFor(() => expect(container.querySelectorAll('.gp-split-card').length).toBe(2))
    expect(vi.mocked(GitSuggestSplitCommits)).toHaveBeenCalled()

    const cards = container.querySelectorAll('.gp-split-card')
    expect(cards[0].querySelector('.gp-split-title')?.textContent).toContain('skill: hunk updates')
    expect(cards[1].querySelector('.gp-split-title')?.textContent).toContain('docs: architecture record')

    // Commit only the first group
    const commitGroupBtn = cards[0].querySelector('.gp-split-commit-btn') as HTMLButtonElement
    await fireEvent.click(commitGroupBtn)

    expect(vi.mocked(GitCommitFiles)).toHaveBeenCalledWith(
      'feat(skill): update hunk parser',
      ['internal/skill/hunk.go'],
    )
  })

  it('supports committing all split groups in sequence', async () => {
    vi.mocked(GitSuggestSplitCommits).mockResolvedValue([
      {
        title: 'skill: hunk updates',
        message: 'feat(skill): update hunk parser',
        files: ['internal/skill/hunk.go'],
      },
      {
        title: 'docs: architecture record',
        message: 'docs: update architecture record',
        files: ['ARCHITECTURE.md'],
      },
    ] as any)

    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-split-section')).not.toBeNull())

    // Click analyze
    const splitBtn = container.querySelector('.gp-split-section .gp-commit-btn') as HTMLButtonElement
    await fireEvent.click(splitBtn)

    await waitFor(() => expect(container.querySelectorAll('.gp-split-card').length).toBe(2))

    // Click Commit All Groups
    const commitAllBtn = container.querySelector('.gp-split-all-btn') as HTMLButtonElement
    expect(commitAllBtn).not.toBeNull()
    await fireEvent.click(commitAllBtn)

    expect(vi.mocked(GitCommitFiles)).toHaveBeenCalledTimes(2)
    expect(vi.mocked(GitCommitFiles)).toHaveBeenNthCalledWith(
      1,
      'feat(skill): update hunk parser',
      ['internal/skill/hunk.go'],
    )
    expect(vi.mocked(GitCommitFiles)).toHaveBeenNthCalledWith(
      2,
      'docs: update architecture record',
      ['ARCHITECTURE.md'],
    )
  })

  // The messages stream in after the groups (DECISIONS: smart split, two
  // beats). A card with no message yet says the model is writing and cannot
  // be committed; chunks land in it; the message event closes it.
  it('streams each group message into its card after the groups arrive', async () => {
    vi.mocked(GitSuggestSplitCommits).mockResolvedValue([
      { title: 'hunk parser', message: '', files: ['internal/skill/hunk.go'] },
      { title: 'architecture record', message: '', files: ['ARCHITECTURE.md'] },
    ] as any)
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-split-section')).not.toBeNull())
    const handler = (name: string) =>
      vi.mocked(EventsOn).mock.calls.find((c) => c[0] === name)?.[1] as (p: any) => void
    expect(handler('git:split:chunk')).toBeTypeOf('function')

    await fireEvent.click(container.querySelector('.gp-split-section .gp-commit-btn') as HTMLButtonElement)
    await waitFor(() => expect(container.querySelectorAll('.gp-split-card').length).toBe(2))

    const cards = () => container.querySelectorAll('.gp-split-card')
    expect(cards()[0].querySelector('.gp-split-note.writing')).not.toBeNull()
    expect((cards()[0].querySelector('.gp-split-commit-btn') as HTMLButtonElement).disabled).toBe(true)
    // The header offers to stop the run, not to commit half-written cards.
    expect(container.querySelector('.gp-split-all-btn')?.hasAttribute('disabled')).toBe(true)
    expect(container.querySelector('.gp-split-redo-btn')?.textContent).toContain('Stop writing')

    handler('git:split:chunk')({ index: 0, text: 'feat(skill): parse ' })
    handler('git:split:chunk')({ index: 0, text: 'hunks\n\n- one' })
    await waitFor(() =>
      expect((cards()[0].querySelector('.gp-split-msg-input') as HTMLTextAreaElement).value).toBe('feat(skill): parse hunks\n\n- one'),
    )
    handler('git:split:message')({ index: 0, message: 'feat(skill): parse hunks\n\n- one\n- two', source: 'model' })
    await waitFor(() => expect(cards()[0].querySelector('.gp-split-note')).toBeNull())
    expect((cards()[0].querySelector('.gp-split-msg-input') as HTMLTextAreaElement).value).toBe('feat(skill): parse hunks\n\n- one\n- two')
    expect((cards()[0].querySelector('.gp-split-commit-btn') as HTMLButtonElement).disabled).toBe(false)
    // The second card is still the model's to write.
    expect(cards()[1].querySelector('.gp-split-note.writing')).not.toBeNull()

    // Nobody wrote the second one: the card says so, and why.
    handler('git:split:message')({ index: 1, message: 'chore(root): แก้ 1 ไฟล์', source: 'fallback', reason: 'model wrote nothing' })
    await waitFor(() => expect(cards()[1].querySelector('.gp-split-note.fallback')?.textContent).toContain('model wrote nothing'))
    expect(container.querySelector('.gp-split-all-btn')?.hasAttribute('disabled')).toBe(false)
  })

  // Committing the first card must not hand its message to the second — the
  // messages were keyed by array index once, and the array shifts when a
  // committed card leaves it.
  it('keeps each card its own message after the card above it is committed', async () => {
    vi.mocked(GitSuggestSplitCommits).mockResolvedValue([
      { title: 'hunk parser', message: 'feat(skill): parse hunks', files: ['internal/skill/hunk.go'] },
      { title: 'architecture record', message: 'docs: architecture record', files: ['ARCHITECTURE.md'] },
    ] as any)
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-split-section')).not.toBeNull())
    await fireEvent.click(container.querySelector('.gp-split-section .gp-commit-btn') as HTMLButtonElement)
    await waitFor(() => expect(container.querySelectorAll('.gp-split-card').length).toBe(2))

    // After the commit the tree no longer has hunk.go.
    vi.mocked(GitWorkingTree).mockResolvedValue([CHANGED[1]] as any)
    await fireEvent.click(container.querySelectorAll('.gp-split-commit-btn')[0] as HTMLButtonElement)
    await waitFor(() => expect(container.querySelectorAll('.gp-split-card').length).toBe(1))

    const left = container.querySelector('.gp-split-card') as HTMLElement
    expect(left.querySelector('.gp-split-title')?.textContent).toContain('architecture record')
    expect((left.querySelector('.gp-split-msg-input') as HTMLTextAreaElement).value).toBe('docs: architecture record')
    await fireEvent.click(left.querySelector('.gp-split-commit-btn') as HTMLButtonElement)
    expect(vi.mocked(GitCommitFiles)).toHaveBeenLastCalledWith('docs: architecture record', ['ARCHITECTURE.md'])
  })

  it('stops the run on request and leaves the unwritten cards editable', async () => {
    vi.mocked(GitSuggestSplitCommits).mockResolvedValue([
      { title: 'hunk parser', message: '', files: ['internal/skill/hunk.go'] },
    ] as any)
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelector('.gp-split-section')).not.toBeNull())
    await fireEvent.click(container.querySelector('.gp-split-section .gp-commit-btn') as HTMLButtonElement)
    await waitFor(() => expect(container.querySelectorAll('.gp-split-card').length).toBe(1))

    await fireEvent.click(container.querySelector('.gp-split-redo-btn') as HTMLButtonElement)
    expect(vi.mocked(GitSplitCancel)).toHaveBeenCalled()
    await waitFor(() => expect(container.querySelector('.gp-split-note.fallback')?.textContent).toContain('Stopped'))
    expect((container.querySelector('.gp-split-msg-input') as HTMLTextAreaElement).readOnly).toBe(false)
  })

  it('detects dangerous files, displays warning banner, unchecks them by default, and supports asking assistant', async () => {
    vi.mocked(GitWorkingTree).mockResolvedValue([
      { path: '.env', status: '?', added: 5, removed: 0 },
      { path: 'src/main.ts', status: 'M', added: 10, removed: 2 },
    ] as any)

    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))

    // Danger banner should be present
    const banner = container.querySelector('.gp-danger-banner')
    expect(banner).not.toBeNull()
    expect(banner?.textContent).toContain('.env')

    // Dangerous row badge should be rendered for .env
    const badges = container.querySelectorAll('.gp-row-danger-badge')
    expect(badges.length).toBe(1)
    expect(badges[0].textContent).toContain('Sensitive')

    // Checkboxes: .env should be unchecked (false), while src/main.ts should be checked (true)
    const checkboxes = container.querySelectorAll('.gp-row-checkbox') as NodeListOf<HTMLInputElement>
    expect(checkboxes.length).toBe(2)
    expect(checkboxes[0].checked).toBe(false)
    expect(checkboxes[1].checked).toBe(true)

    // Ask assistant button should be present and switch to chat when clicked
    const askBtn = container.querySelector('.gp-ask-assistant-btn') as HTMLButtonElement
    expect(askBtn).not.toBeNull()
    await fireEvent.click(askBtn)
    expect(cockpit.activeView).toBe('chat')
  })
})


