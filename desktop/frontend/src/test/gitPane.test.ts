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
} from './mocks/wailsApp'

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
  it('reads the tree again when a turn finishes', async () => {
    cockpit.awaitingReply = false
    const { container } = render(GitPane)
    await waitFor(() => expect(container.querySelectorAll('.gp-row').length).toBe(2))
    expect(vi.mocked(GitWorkingTree)).toHaveBeenCalledTimes(1)

    cockpit.awaitingReply = true
    await waitFor(() => expect(vi.mocked(GitWorkingTree)).toHaveBeenCalledTimes(1))
    cockpit.awaitingReply = false
    await waitFor(() => expect(vi.mocked(GitWorkingTree)).toHaveBeenCalledTimes(2))
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


