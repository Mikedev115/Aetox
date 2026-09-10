import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import PRPane from '../lib/workbench/PRPane.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import {
  PullRequests,
  PullRequestsState,
  PullRequestFiles,
  PullRequestChecks,
  SuggestPRDetails,
  ReviewPullRequest,
} from './mocks/wailsApp'

const pr = {
  number: 7, title: 'Collapse the diffs', url: 'https://example.test/pr/7',
  state: 'open', draft: false, author: 'mike', headRef: 'work', baseRef: 'main',
  headSHA: 'abc123', mergeable: true, additions: 40, deletions: 3, changedFiles: 2,
}

const closedPR = {
  number: 4, title: 'Merged feature branch', url: 'https://example.test/pr/4',
  state: 'closed', draft: false, author: 'mike', headRef: 'feat-1', baseRef: 'main',
  headSHA: 'xyz987', mergeable: true, additions: 15, deletions: 2, changedFiles: 1,
}

const files = [
  { path: 'a.go', status: 'modified', additions: 20, deletions: 1, patch: '@@ -1 +1 @@\n-old line a\n+new line a' },
  { path: 'b.go', status: 'modified', additions: 20, deletions: 2, patch: '@@ -1 +1 @@\n-old line b\n+new line b' },
]

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  cockpit.project.branch = 'feature/my-branch'
  vi.mocked(PullRequests).mockResolvedValue({ repo: 'o/r', reason: '', connected: true, items: [pr] } as any)
  vi.mocked(PullRequestsState).mockImplementation(async (st) => {
    if (st === 'closed') {
      return { repo: 'o/r', reason: '', connected: true, items: [closedPR] } as any
    }
    return { repo: 'o/r', reason: '', connected: true, items: [pr] } as any
  })
  vi.mocked(PullRequestFiles).mockResolvedValue(files as any)
  vi.mocked(PullRequestChecks).mockResolvedValue([] as any)
  vi.mocked(SuggestPRDetails).mockResolvedValue({
    title: 'feat: AI generated title',
    body: '## Summary\nAI generated body',
  })
  vi.mocked(ReviewPullRequest).mockResolvedValue('### 📋 Overview & Highlights\nAll looks clean.')
})

describe('the pull request room', () => {
  it('lists the files without drawing a single patch', async () => {
    render(PRPane)
    await waitFor(() => expect(screen.getByText(/Collapse the diffs/)).toBeTruthy())
    await fireEvent.click(screen.getByText(/Collapse the diffs/))

    await waitFor(() => expect(screen.getByText('a.go')).toBeTruthy())
    expect(screen.getByText('b.go')).toBeTruthy()
    expect(screen.queryByText(/new line a/)).toBeNull()
    expect(screen.queryByText(/new line b/)).toBeNull()
  })

  it('draws only the file you opened', async () => {
    render(PRPane)
    await waitFor(() => expect(screen.getByText(/Collapse the diffs/)).toBeTruthy())
    await fireEvent.click(screen.getByText(/Collapse the diffs/))
    await waitFor(() => expect(screen.getByText('a.go')).toBeTruthy())

    await fireEvent.click(screen.getByText('a.go'))
    await waitFor(() => expect(screen.getByText(/new line a/)).toBeTruthy())
    expect(screen.queryByText(/new line b/)).toBeNull()

    await fireEvent.click(screen.getByText('a.go'))
    await waitFor(() => expect(screen.queryByText(/new line a/)).toBeNull())
  })

  it('switches between Open and Closed PR tabs', async () => {
    const { container } = render(PRPane)
    await waitFor(() => expect(container.querySelectorAll('.pr-tab').length).toBe(2))

    // Default tab is open
    expect(screen.getByText(/Collapse the diffs/)).toBeTruthy()

    // Click Closed tab
    const tabs = container.querySelectorAll('.pr-tab')
    await fireEvent.click(tabs[1] as HTMLElement)

    await waitFor(() => expect(screen.getByText(/Merged feature branch/)).toBeTruthy())
    expect(vi.mocked(PullRequestsState)).toHaveBeenCalledWith('closed')
  })

  it('renders smart empty state with current branch and create action when no PRs open', async () => {
    vi.mocked(PullRequestsState).mockResolvedValue({ repo: 'o/r', reason: '', connected: true, items: [] } as any)
    const { container } = render(PRPane)

    await waitFor(() => expect(container.querySelector('.pr-smart-empty')).not.toBeNull())
    expect(screen.getByText('feature/my-branch')).toBeTruthy()
    expect(screen.getByText(/No open pull requests/)).toBeTruthy()

    // Click CTA button to open form
    const ctaBtn = container.querySelector('.pr-empty-cta') as HTMLButtonElement
    expect(ctaBtn).not.toBeNull()
    await fireEvent.click(ctaBtn)

    await waitFor(() => expect(container.querySelector('.pr-form')).not.toBeNull())
    const headInput = container.querySelector('.pr-in.mono') as HTMLInputElement
    expect(headInput.value).toBe('feature/my-branch')
  })

  it('supports AI PR description and title drafting in the form', async () => {
    const { container } = render(PRPane)
    await waitFor(() => expect(container.querySelector('.icobtn[title*="Open"]')).not.toBeNull())

    // Click + button to open form
    const plusBtn = container.querySelector('.icobtn[title*="Open"]') as HTMLButtonElement
    await fireEvent.click(plusBtn)

    await waitFor(() => expect(container.querySelector('.pr-ai-draft-btn')).not.toBeNull())

    // Click Draft with AI
    const draftBtn = container.querySelector('.pr-ai-draft-btn') as HTMLButtonElement
    await fireEvent.click(draftBtn)

    expect(vi.mocked(SuggestPRDetails)).toHaveBeenCalledWith('feature/my-branch', '')
    await waitFor(() => {
      const titleInput = container.querySelector('.pr-title-row .pr-in') as HTMLInputElement
      expect(titleInput.value).toBe('feat: AI generated title')
      const bodyTextarea = container.querySelector('.pr-in.pr-body') as HTMLTextAreaElement
      expect(bodyTextarea.value).toContain('AI generated body')
    })
  })

  it('supports AI Code Review on an expanded pull request', async () => {
    const { container } = render(PRPane)
    await waitFor(() => expect(screen.getByText(/Collapse the diffs/)).toBeTruthy())

    // Expand the PR row
    await fireEvent.click(screen.getByText(/Collapse the diffs/))
    await waitFor(() => expect(container.querySelector('.pr-ai-review-btn')).not.toBeNull())

    // Click AI Review
    const reviewBtn = container.querySelector('.pr-ai-review-btn') as HTMLButtonElement
    await fireEvent.click(reviewBtn)

    expect(vi.mocked(ReviewPullRequest)).toHaveBeenCalledWith(7)
    await waitFor(() => expect(container.querySelector('.pr-ai-review-box')).not.toBeNull())
    expect(container.querySelector('.pr-ai-review-content')?.textContent).toContain('All looks clean')
  })
})

