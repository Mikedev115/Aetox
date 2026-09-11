// Reactive code status store: tracks uncommitted working tree changes and open pull requests.
// Used by Workbench, tab badges, and panel toggles to notify the user of code changes.

import { GitWorkingTree, PullRequestsState } from '../../../wailsjs/go/main/App'
import { cockpit } from './cockpit.svelte'

export const codeStatus = $state<{
  gitChangedCount: number
  gitAdded: number
  gitRemoved: number
  openPRCount: number
  hasFailingPR: boolean
  lastChecked: number
  loading: boolean
  // Bumped by the git room when a commit lands. The timeline (GitLogPane)
  // re-reads its first page on the bump rather than polling — history only
  // moves when this moves, or when something outside the window commits, and
  // the second case is what coming back to the tab is for.
  commitsLanded: number
}>({
  gitChangedCount: 0,
  gitAdded: 0,
  gitRemoved: 0,
  openPRCount: 0,
  hasFailingPR: false,
  lastChecked: 0,
  loading: false,
  commitsLanded: 0,
})

export function noteCommitLanded(): void {
  codeStatus.commitsLanded++
}

let refreshTimer: ReturnType<typeof setTimeout> | undefined

export function updateCodeStatusFromGitTree(tree: Array<{ added?: number; removed?: number }>): void {
  codeStatus.gitChangedCount = tree.length
  codeStatus.gitAdded = tree.reduce((acc, f) => acc + (f.added ?? 0), 0)
  codeStatus.gitRemoved = tree.reduce((acc, f) => acc + (f.removed ?? 0), 0)
  codeStatus.lastChecked = Date.now()
}

export async function refreshCodeStatus(): Promise<void> {
  if (codeStatus.loading) return
  codeStatus.loading = true
  try {
    // 1. Fetch git working tree changes. A read that failed (git ran out of
    //    its budget) keeps the last count: a badge that drops to nothing over
    //    fifty changed files is a lie, and the next read will say more.
    try {
      const tree = await GitWorkingTree()
      if (Array.isArray(tree)) {
        codeStatus.gitChangedCount = tree.length
        codeStatus.gitAdded = tree.reduce((acc, f) => acc + (f.added ?? 0), 0)
        codeStatus.gitRemoved = tree.reduce((acc, f) => acc + (f.removed ?? 0), 0)
      }
    } catch {
      /* keep what was known */
    }

    // 2. Fetch open pull requests count
    try {
      const room = await PullRequestsState('open')
      if (room && Array.isArray(room.items)) {
        codeStatus.openPRCount = room.items.length
      } else {
        codeStatus.openPRCount = 0
      }
    } catch {
      codeStatus.openPRCount = 0
    }

    codeStatus.lastChecked = Date.now()
  } finally {
    codeStatus.loading = false
  }
}

// Debounced trigger helper
export function scheduleCodeStatusRefresh(delayMs = 400): void {
  if (refreshTimer) clearTimeout(refreshTimer)
  refreshTimer = setTimeout(() => {
    void refreshCodeStatus()
  }, delayMs)
}

// Auto-refresh when model stops working (e.g. after editing files)
let prevAwaiting = false
$effect.root(() => {
  $effect(() => {
    const awaiting = cockpit.awaitingReply
    if (prevAwaiting && !awaiting) {
      scheduleCodeStatusRefresh(500)
    }
    prevAwaiting = awaiting
  })

  // Refresh on branch or project changes
  $effect(() => {
    const _br = cockpit.project.branch
    const _path = cockpit.project.path
    if (_path || _br) {
      scheduleCodeStatusRefresh(300)
    }
  })
})
