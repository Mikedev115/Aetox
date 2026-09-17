// รู้จักกับ Aetox — the first-run tour (DECISIONS §279). What these pin is the
// shape a person can rely on: the short first-run story plus the measured
// harness comparison, every scene reachable, and a skip that lands on it.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { readFileSync } from 'node:fs'
import Tour from '../lib/Tour.svelte'
import { HeadName } from './mocks/wailsApp'
import { tourState, openTour } from '../lib/tourState.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(HeadName).mockResolvedValue('')
  vi.useRealTimers()
})

const dots = () => [...document.querySelectorAll('.tour-dot')]
const onDot = () => dots().findIndex((d) => d.classList.contains('on'))

describe('รู้จักกับ Aetox', () => {
  it('is four scenes, opening on the app as a window with four rooms', async () => {
    render(Tour, { onDone: () => {} })
    expect(dots().length).toBe(4)
    expect(onDot()).toBe(0)
    expect(screen.getByText('Aetox AI ที่ออกแบบมาเพื่อเป็นผู้ช่วยในคอมพิวเตอร์ของคุณ')).toBeTruthy()
    expect(document.querySelectorAll('.pane').length).toBe(4)
    // No dash in anything a person reads (owner, 14 ก.ย.: "เอา — ออก").
    expect(document.querySelector('.tour-screen')?.textContent).not.toContain('—')
  })

  it('walks forward and back, and ข้าม lands on the last scene', async () => {
    const done = vi.fn()
    render(Tour, { onDone: done })
    await fireEvent.click(screen.getByText('ถัดไป →'))
    expect(onDot()).toBe(1)
    expect(screen.getByText('คุณมีผู้ช่วยสองคน')).toBeTruthy()
    await fireEvent.click(screen.getByText('← ก่อนหน้า'))
    expect(onDot()).toBe(0)
    await fireEvent.click(screen.getByText('ข้าม'))
    expect(onDot()).toBe(3)
    expect(screen.getByText(/RAM ตอนทำงาน/)).toBeTruthy()
    expect(screen.getByText('ปิด')).toBeTruthy()
    expect(done).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByText('ปิด'))
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('shows the four first-run ideas in order', async () => {
    render(Tour, { onDone: () => {} })
    const headings: string[] = []
    for (let i = 0; i < 4; i++) {
      await fireEvent.click(dots()[i])
      headings.push(document.querySelector('.tour-screen h2')?.textContent?.trim() ?? '')
    }
    expect(headings).toEqual([
      'Aetox AI ที่ออกแบบมาเพื่อเป็นผู้ช่วยในคอมพิวเตอร์ของคุณ',
      'คุณมีผู้ช่วยสองคน',
      'ทุกอย่างอยู่ในเครื่องคุณ',
      'เบากว่า harness ที่มีอยู่ และทำงานกับโมเดลเล็กได้ดี ทั้งที่มีเบราว์เซอร์ เทอร์มินัล และไฟล์ทำงานอยู่ในตัว',
    ])
  })

  it('keeps every RAM comparison row and marks figures reported elsewhere', async () => {
    render(Tour, { onDone: () => {} })
    await fireEvent.click(dots()[3])
    const rows = [...document.querySelectorAll('.bar:not(.me)')]
    expect(rows.length).toBe(10)
    expect(document.querySelector('.bar.me .bar-val')?.textContent).toBe('455 MB - 706 MB')
    const starred = rows.filter((r) => r.querySelector('sup')).map((r) => r.querySelector('.bar-name')?.firstChild?.textContent?.trim())
    expect(starred).toEqual(['OpenClaw', 'Hermes Agent', 'Windsurf', 'Claude Desktop', 'OpenCode', 'Codex'])
  })

  it('has narrow and short viewport layouts for the dense comparison', () => {
    const css = readFileSync('src/style.css', 'utf8')
    expect(css).toContain('@media (max-width: 640px)')
    expect(css).toContain('@media (max-height: 760px)')
    expect(css).toMatch(/\.tour-scene-perf[\s\S]*?min-height:0/)
    expect(css).toMatch(/\.tour \.perf[\s\S]*?overflow:auto/)
    expect(css).toMatch(/\.tour \.perf[^}]*align-items:center/)
    expect(css).toMatch(/\.tour \.perf[^}]*scrollbar-gutter:stable both-edges/)
    expect(css).toMatch(/\.tour \.bars[^}]*margin-inline:auto/)
  })

  it('in the wizard, continues from the comparison to setup', async () => {
    const done = vi.fn()
    render(Tour, { onDone: done, flow: 'setup' })
    await fireEvent.click(screen.getByText('ข้าม'))
    expect(screen.getByText(/RAM ตอนทำงาน/)).toBeTruthy()
    expect([...document.querySelectorAll('.step-ahead')].length).toBe(0)
    expect(screen.queryByText('ปิด')).toBeNull()
    await fireEvent.click(screen.getByText('ไปตั้งค่า'))
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('the replay flag opens and closes', () => {
    expect(tourState.open).toBe(false)
    openTour()
    expect(tourState.open).toBe(true)
    tourState.open = false
  })
})
