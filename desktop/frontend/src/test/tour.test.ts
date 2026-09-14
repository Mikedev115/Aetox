// รู้จักกับ Aetox — the first-run tour (DECISIONS §279). What these pin is the
// shape a person can rely on: nine scenes, every one reachable, a skip that
// lands on the last, names that are saved for real, and the words the
// scenes make claims with — the ones a release must re-measure.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Tour from '../lib/Tour.svelte'
import { SetHeadName, SetUserName, HeadName } from './mocks/wailsApp'
import { tourState, openTour } from '../lib/tourState.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(HeadName).mockResolvedValue('')
  vi.useRealTimers()
})

const dots = () => [...document.querySelectorAll('.tour-dot')]
const onDot = () => dots().findIndex((d) => d.classList.contains('on'))

describe('รู้จักกับ Aetox', () => {
  it('is nine scenes, opening on the app as a window with four rooms', async () => {
    render(Tour, { onDone: () => {} })
    expect(dots().length).toBe(9)
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
    expect(onDot()).toBe(8)
    expect(screen.getByText('ปิด')).toBeTruthy()
    expect(done).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByText('ปิด'))
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('saves the names typed in scene 3, and only the ones typed', async () => {
    const done = vi.fn()
    render(Tour, { onDone: done })
    await fireEvent.click(dots()[2])
    const [bot, you] = [...document.querySelectorAll('.names input')] as HTMLInputElement[]
    await fireEvent.input(bot, { target: { value: '  ลูน่า ' } })
    // The name is used at once, before it is saved: the greeting and every
    // later scene call the assistant what was typed.
    expect(document.querySelector('.tbubble')?.textContent).toContain('ลูน่า')
    await fireEvent.click(screen.getByText('ข้าม'))
    expect(screen.getByText('พร้อมแล้ว ลูน่า รออยู่ที่โต๊ะ')).toBeTruthy()
    await fireEvent.click(screen.getByText('ปิด'))
    expect(vi.mocked(SetHeadName)).toHaveBeenCalledWith('assistant', 'ลูน่า')
    expect(vi.mocked(SetUserName)).not.toHaveBeenCalled()
    expect(you.value).toBe('')
  })

  it('opens scene 3 with the name the head already has', async () => {
    vi.mocked(HeadName).mockResolvedValue('มายด์')
    render(Tour, { onDone: () => {} })
    await fireEvent.click(dots()[2])
    await waitFor(() => expect((document.querySelector('.names input') as HTMLInputElement).value).toBe('มายด์'))
  })

  it('shows every scene the doc promises, by its heading', async () => {
    render(Tour, { onDone: () => {} })
    const headings: string[] = []
    for (let i = 0; i < 9; i++) {
      await fireEvent.click(dots()[i])
      headings.push(document.querySelector('.tour-screen h2')?.textContent?.trim() ?? '')
    }
    expect(headings).toEqual([
      'Aetox AI ที่ออกแบบมาเพื่อเป็นผู้ช่วยในคอมพิวเตอร์ของคุณ',
      'คุณมีผู้ช่วยสองคน',
      'ตั้งชื่อกันได้',
      'จำได้ข้ามแชท แต่ไม่มีอะไรถูกจำโดยคุณไม่อนุมัติ',
      'ทุกอย่างอยู่ในเครื่องคุณ',
      'หนึ่งบริษัทเล็ก ๆ บนเครื่องคุณ',
      'ตั้งทีมให้พนักงาน แล้วAetoxแบ่งงานให้ทีมทำ',
      'เบากว่า harness ที่มีอยู่ และทำงานกับโมเดลเล็กได้ดี ทั้งที่มีเบราว์เซอร์ เทอร์มินัล และไฟล์ทำงานอยู่ในตัว',
      'พร้อมแล้ว Aetox รออยู่ที่โต๊ะ',
    ])
  })

  it('marks every RAM figure that was not measured here', async () => {
    render(Tour, { onDone: () => {} })
    await fireEvent.click(dots()[7])
    const rows = [...document.querySelectorAll('.bar:not(.me)')]
    expect(rows.length).toBe(10)
    // Aetox's own row carries the measured split and both figures.
    expect(document.querySelector('.bar.me .bar-val')?.textContent).toBe('455 MB - 706 MB')
    // A reported figure wears the asterisk; a measured one does not.
    const starred = rows.filter((r) => r.querySelector('sup')).map((r) => r.querySelector('.bar-name')?.firstChild?.textContent?.trim())
    expect(starred).toEqual(['OpenClaw', 'Hermes Agent', 'Windsurf', 'Claude Desktop', 'OpenCode', 'Codex'])
  })

  it('in the wizard, ends by handing over to the three setup steps', async () => {
    const done = vi.fn()
    render(Tour, { onDone: done, flow: 'setup' })
    await fireEvent.click(screen.getByText('ข้าม'))
    expect(screen.getByText('รู้จักกันแล้ว ต่อไปตั้งค่าระบบให้พร้อมก่อน')).toBeTruthy()
    expect([...document.querySelectorAll('.step-ahead')].length).toBe(3)
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
