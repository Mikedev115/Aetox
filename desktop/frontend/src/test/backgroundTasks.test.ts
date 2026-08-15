import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import {
  cockpit, applyBackgroundTasks, refreshBackgroundTasks,
  clearFinishedBackgroundTasks, visibleBackgroundTasks, runningBackgroundCount,
} from '../lib/stores/cockpit.svelte'
import BackgroundPane from '../lib/workbench/BackgroundPane.svelte'
import { BackgroundTasks, StopBackgroundTask } from './mocks/wailsApp'
import type { BackgroundTask } from '../lib/types'

const running = (id: string, command = 'npm run dev'): BackgroundTask =>
  ({ id, command, running: true, killed: false, exitError: '', elapsedMs: 5_000 })
const finished = (id: string, over: Partial<BackgroundTask> = {}): BackgroundTask =>
  ({ id, command: 'go test ./...', running: false, killed: false, exitError: '', elapsedMs: 12_000, ...over })

beforeEach(() => {
  cockpit.backgroundTasks = []
  cockpit.backgroundTasksAt = 0
  cockpit.backgroundCleared = []
  vi.mocked(BackgroundTasks).mockClear()
  vi.mocked(BackgroundTasks).mockResolvedValue([])
  vi.mocked(StopBackgroundTask).mockClear()
  vi.mocked(StopBackgroundTask).mockResolvedValue(undefined)
})

describe('background task store', () => {
  it('applyBackgroundTasks replaces the list wholesale and tolerates junk', () => {
    applyBackgroundTasks([running('bg_1')])
    expect(cockpit.backgroundTasks).toHaveLength(1)
    expect(cockpit.backgroundTasksAt).toBeGreaterThan(0)
    applyBackgroundTasks(undefined as never)
    expect(cockpit.backgroundTasks).toEqual([])
  })

  it('refresh fetches what the backend holds', async () => {
    vi.mocked(BackgroundTasks).mockResolvedValue([running('bg_2')] as any)
    await refreshBackgroundTasks()
    expect(cockpit.backgroundTasks).toHaveLength(1)
    expect(cockpit.backgroundTasks[0].id).toBe('bg_2')
  })

  it('clear finished hides finished rows only, and never the badge count', () => {
    applyBackgroundTasks([running('bg_1'), finished('bg_2'), finished('bg_3', { killed: true })])
    clearFinishedBackgroundTasks()
    expect(visibleBackgroundTasks().map((t) => t.id)).toEqual(['bg_1'])
    expect(runningBackgroundCount()).toBe(1)
  })

  it('a cleared id expires when the registry forgets it or a fresh engine reuses it', () => {
    applyBackgroundTasks([finished('bg_1')])
    clearFinishedBackgroundTasks()
    expect(visibleBackgroundTasks()).toEqual([])

    // A re-bootstrap starts ids over: the "same" id arriving as a running job
    // is a new job and must be visible — finished included, later.
    applyBackgroundTasks([running('bg_1')])
    expect(cockpit.backgroundCleared).toEqual([])
    expect(visibleBackgroundTasks().map((t) => t.id)).toEqual(['bg_1'])
  })
})

describe('BackgroundPane', () => {
  // The pane refreshes from the binding on mount, so the mock must serve the
  // same list the test staged — otherwise the mount wipes it back to [].
  function stage(list: BackgroundTask[]) {
    vi.mocked(BackgroundTasks).mockResolvedValue(list as any)
    applyBackgroundTasks(list)
  }

  it('shows running and finished sections with command lines and states', async () => {
    stage([running('bg_1', 'npm run dev'), finished('bg_2', { killed: true })])
    render(BackgroundPane)
    expect(screen.getByText('npm run dev')).toBeTruthy()
    expect(screen.getByText('go test ./...')).toBeTruthy()
    // Default locale is Thai.
    expect(screen.getByText(/กำลังทำงาน \(1\)/)).toBeTruthy()
    expect(screen.getByText(/เสร็จแล้ว \(1\)/)).toBeTruthy()
    expect(screen.getByText(/ถูกหยุดหลัง/)).toBeTruthy()
  })

  it('empty state renders when nothing has run', () => {
    render(BackgroundPane)
    expect(document.querySelector('.empty')).toBeTruthy()
  })

  it('stop is two-step: first click arms, second click kills', async () => {
    stage([running('bg_1')])
    render(BackgroundPane)
    const stop = screen.getByRole('button', { name: 'หยุด' })
    await fireEvent.click(stop)
    expect(vi.mocked(StopBackgroundTask)).not.toHaveBeenCalled()
    await fireEvent.click(stop)
    await waitFor(() => expect(vi.mocked(StopBackgroundTask).mock.calls[0][0]).toBe('bg_1'))
  })

  it('a failed stop surfaces its reason instead of vanishing', async () => {
    vi.mocked(StopBackgroundTask).mockRejectedValue(new Error('no background command'))
    stage([running('bg_1')])
    render(BackgroundPane)
    const stop = screen.getByRole('button', { name: 'หยุด' })
    await fireEvent.click(stop)
    await fireEvent.click(stop)
    await waitFor(() => expect(document.querySelector('.bg-error')?.textContent).toContain('no background command'))
  })

  it('clear finished removes the finished section', async () => {
    stage([finished('bg_2')])
    render(BackgroundPane)
    expect(screen.getByText('go test ./...')).toBeTruthy()
    await fireEvent.click(screen.getByText(/ล้างที่เสร็จแล้ว/))
    await waitFor(() => expect(screen.queryByText('go test ./...')).toBeNull())
  })
})
