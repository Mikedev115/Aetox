import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import EngineStatus from '../lib/EngineStatus.svelte'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { EngineStatus as engineStatus, RestartEngine } from './mocks/wailsApp'

// The engine is a process beside the window (§248 phase 2). This card is
// the only thing that says so when it is not there. i18n defaults to Thai.

type Status = { state: string; detail: string; restarts: number; pid: number; address: string }

function announce(st: Partial<Status>) {
  const call = vi.mocked(EventsOn).mock.calls.find((c) => c[0] === 'engine:status')
  if (!call) throw new Error('nothing subscribed to engine:status')
  ;(call[1] as (st: Status) => void)({ state: 'connected', detail: '', restarts: 0, pid: 1, address: '', ...st })
}

beforeEach(() => {
  vi.mocked(EventsOn).mockClear()
  vi.mocked(RestartEngine).mockClear()
  vi.mocked(engineStatus).mockResolvedValue({ state: 'connected', detail: '', restarts: 0, pid: 1, address: '' })
})

describe('EngineStatus', () => {
  it('says nothing while the engine is connected', async () => {
    const { container } = render(EngineStatus)
    await waitFor(() => expect(engineStatus).toHaveBeenCalled())
    expect(container.querySelector('.upd-card')).toBeNull()
  })

  // A launch starts the engine every time; a card for that every time would
  // be noise. A start that is still going after the settle wait is news.
  it('does not flash a card for the ordinary start', async () => {
    vi.useFakeTimers()
    try {
      const { container } = render(EngineStatus)
      announce({ state: 'starting' })
      await vi.advanceTimersByTimeAsync(500)
      expect(container.querySelector('.upd-card')).toBeNull()
      await vi.advanceTimersByTimeAsync(1500)
      expect(container.querySelector('.upd-card')).not.toBeNull()
      expect(screen.getByText('กำลังเริ่มเครื่องยนต์')).toBeTruthy()
    } finally {
      vi.useRealTimers()
    }
  })

  it('shows a failure at once, with the reason and a way to start again', async () => {
    render(EngineStatus)
    announce({ state: 'failed', detail: 'aetox-engine.exe not found', restarts: 3 })
    await waitFor(() => expect(screen.getByText('เครื่องยนต์หยุดทำงาน')).toBeTruthy())
    expect(screen.getByText('aetox-engine.exe not found')).toBeTruthy()
    expect(screen.getByText('เริ่มใหม่แล้ว 3 ครั้งในรอบนี้')).toBeTruthy()

    await fireEvent.click(screen.getByText('เริ่มเครื่องยนต์ใหม่'))
    expect(RestartEngine).toHaveBeenCalledTimes(1)
  })

  it('leaves when the engine is back, and returns for the next change', async () => {
    const { container } = render(EngineStatus)
    announce({ state: 'failed', detail: 'x' })
    await waitFor(() => expect(container.querySelector('.upd-card')).not.toBeNull())
    announce({ state: 'connected' })
    await waitFor(() => expect(container.querySelector('.upd-card')).toBeNull())
    // Hidden by the ×, then a new state: the card comes back.
    announce({ state: 'failed', detail: 'y' })
    await waitFor(() => expect(container.querySelector('.upd-card')).not.toBeNull())
    await fireEvent.click(screen.getByLabelText('ซ่อน'))
    await waitFor(() => expect(container.querySelector('.upd-card')).toBeNull())
    announce({ state: 'restarting', restarts: 1 })
    await waitFor(() => expect(screen.getByText('เครื่องยนต์หยุดไป กำลังเริ่มใหม่')).toBeTruthy())
    expect(document.querySelector('.upd-bar.indeterminate')).not.toBeNull()
  })
})
