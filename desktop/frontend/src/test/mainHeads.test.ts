// ตัวหลัก (14 ก.ย. 2026): the two heads a person actually talks to, each with a
// page in the shape every specialist agent already has. The feedback that
// started this was "ไม่รู้ว่าตัวหลักปรับแต่งได้" — a thing with no page reads as
// unconfigurable. What is pinned: two cards and never a third, the agent
// editor's tab bar, the doors on each tab, and that what is about the person
// is NOT here (เกี่ยวกับคุณ). The memory tab is pinned in learningReview.test.ts.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import {
  ListModes, LearnedScopeInfos, LearnedEntries, ListPendingChanges,
  ListIdentityFiles, ReadIdentityFile, SaveIdentityFile, ReadDeskFile, SaveDeskFile, ResetDeskFile,
  ListMCPServers, SetMCPServerTargets, ListExternalSkills, DeskStarters, SaveDeskStarters, HeadName, SetHeadName,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

const openSection = async (container: HTMLElement, label: string) => {
  const items = Array.from(container.querySelectorAll('.settings-nav-item'))
  const item = items.find((el) => el.textContent?.trim() === label) ?? items.find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}
const cards = (c: HTMLElement) => Array.from(c.querySelectorAll<HTMLElement>('.main-card'))
const tabs = (c: HTMLElement) => Array.from(c.querySelectorAll<HTMLElement>('.ag-tabs-bar [role="tab"]'))

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'settings'
  cockpit.capabilityIntent = null
  vi.mocked(ListModes).mockResolvedValue([
    { name: 'assistant', description: 'โต๊ะผู้ช่วย, ทำได้ทุกอย่างบนเครื่อง' },
    { name: 'coding', description: 'โต๊ะเขียนโค้ด' },
    { name: 'specialized', description: 'โต๊ะเอเจน' },
  ] as any)
  vi.mocked(LearnedScopeInfos).mockResolvedValue([{ scope: '', orphan: false }, { scope: 'mode:coding', orphan: false, projectsUnder: true }] as any)
  vi.mocked(LearnedEntries).mockImplementation(async (scope: string) => (scope === '' ? ['a', 'b'] : []) as any)
  vi.mocked(ListPendingChanges).mockResolvedValue([] as any)
})

describe('the list', () => {
  it('draws the two heads as agent cards with the desk file\'s own line, and no third', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    const names = cards(container).map((c) => c.querySelector('.chair-name')?.textContent?.trim())
    expect(names).toEqual(['ผู้ช่วย', 'โค้ด'])
    // The face is the head's own rig, not an icon, with its rank on the
    // corner (RankedFace) like every other face in the app.
    expect(cards(container)[0].querySelector('.mascot')).toBeTruthy()
    expect(cards(container)[0].querySelector('.rank-corner.rank-head')).toBeTruthy()
    await waitFor(() => expect(cards(container)[0].textContent).toContain('ทำได้ทุกอย่างบนเครื่อง'))
    // No memory count on the card (owner, 14 ก.ย.: "จำไว้ 0 บรรทัด เอาออก");
    // the number is on the memory tab, one click in.
    expect(cards(container)[0].textContent).not.toContain('บรรทัด')
    // The person's layer is named as elsewhere, never drawn here.
    expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตัวหลัก')
    expect(container.textContent).not.toContain('USER.md')
    expect(container.querySelector('.office-note .linklike')?.textContent).toBe('เกี่ยวกับคุณ')
  })

  it('counts what waits, on the card and on the rail', async () => {
    vi.mocked(ListPendingChanges).mockResolvedValue([
      { id: 1, kind: 'memory', scope: 'mode:coding', op: 'add', body: 'x', state: 'pending' },
      { id: 2, kind: 'memory', scope: 'user:profile', op: 'add', body: 'y', state: 'pending' },
    ] as any)
    cockpit.pendingLearned = 2
    const { container } = render(Settings, { onClose: () => {} })
    const row = Array.from(container.querySelectorAll('.settings-nav-item')).find((el) => el.textContent?.includes('ตัวหลัก'))!
    expect(row.querySelector('.nav-count')?.textContent?.trim()).toBe('2')
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await waitFor(() => expect(cards(container)[1].textContent).toContain('รออนุมัติ 1'))
    expect(cards(container)[0].textContent).not.toContain('รออนุมัติ')
    cockpit.pendingLearned = 0
  })
})

describe('a head\'s page', () => {
  it('opens from the card in the agent editor\'s shape, and crosses to the other head', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await fireEvent.click(cards(container)[0].querySelector('.icobtn')!)
    await waitFor(() => expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตั้งค่า ผู้ช่วย'))
    expect(tabs(container).map((t) => t.textContent?.trim())).toEqual(['ตัวตน', 'ตั้งค่า MCP', 'สกิล', 'เปิดบทสนทนา', 'ความจำ'])
    expect(container.querySelector('.main-head .rank-corner.rank-head')).toBeTruthy()
    expect(container.querySelector('.main-head .mascot')).toBeTruthy()
    // The first tab reads as the three questions (DECISIONS §270.3), naming the
    // head: the desk row is the job, not the head's description again (the
    // hero has that).
    const firstTab = container.querySelector('.ag-tab-panel.on')?.textContent ?? ''
    expect(firstTab).toContain('หน้าที่หลัก: ผู้ช่วยทำอะไร')
    expect(firstTab).toContain('ตัวตนของผู้ช่วย')
    expect(firstTab).toContain('ท่าทีของผู้ช่วย')
    expect(firstTab).not.toContain('มัน')
    expect(firstTab).not.toContain('ไฟล์โต๊ะ')
    expect(container.querySelector('.ag-tab-panel.on')?.textContent).toContain('modes/assistant.md')

    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl')).find((b) => b.textContent?.includes('ไปที่ โค้ด'))!)
    await waitFor(() => expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตั้งค่า โค้ด'))
    expect(container.querySelector('.ag-tab-panel.on')?.textContent).toContain('modes/coding.md')

    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl'))[0])
    await waitFor(() => expect(cards(container).length).toBe(2))
  })

  // The agent editor's MCP box, for a desk: every live server with this
  // desk's switch on it, written through the room's one writer at once.
  it('MCP is switches on the page, writing the desk into the server\'s for: list', async () => {
    vi.mocked(ListMCPServers).mockResolvedValue([
      { name: 'firecrawl', disabled: false, status: 'ok', tools: 25, for: ['coding'] },
      { name: 'notion', disabled: false, status: 'idle', tools: 0, for: [] },
      { name: 'old', disabled: true, status: 'off', tools: 0, for: ['coding'] },
    ] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await fireEvent.click(cards(container)[1].querySelector('.icobtn')!)
    await fireEvent.click(tabs(container).find((t) => t.textContent?.includes('ตั้งค่า MCP'))!)
    const rows = await waitFor(() => {
      const r = Array.from(container.querySelectorAll('.ag-tab-panel.on .ag-reachrow'))
      expect(r.length).toBe(2)
      return r
    })
    expect(rows.map((r) => r.querySelector('.t')?.textContent?.trim())).toEqual(['firecrawl', 'notion'])
    expect((rows[0].querySelector('input') as HTMLInputElement).checked).toBe(true)
    expect((rows[1].querySelector('input') as HTMLInputElement).checked).toBe(false)
    await fireEvent.click(rows[1].querySelector('input')!)
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledWith('notion', ['coding']))
    await fireEvent.click(rows[0].querySelector('input')!)
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledWith('firecrawl', []))
    // The tool register stays a door: it is read-only everywhere.
    await fireEvent.click(Array.from(container.querySelectorAll('.ag-tab-panel.on .ctrl')).find((b) => b.textContent?.includes('เปิด'))!)
    expect(cockpit.capabilityIntent).toEqual({ page: 'tools' })
  })

  // The shelf, listed: every desk sees all of it, so the tab shows rather
  // than ticks, and the door leads to where the shelf changes.
  it('skills lists the whole shelf under this head, yours before bundled', async () => {
    vi.mocked(ListExternalSkills).mockResolvedValue([
      { name: 'aetox-slides', description: 'สไลด์', dir: 'x', bundled: true },
      { name: 'my-deck', description: 'ของผม', dir: 'y' },
    ] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await fireEvent.click(cards(container)[0].querySelector('.icobtn')!)
    await fireEvent.click(tabs(container).find((t) => t.textContent?.trim() === 'สกิล')!)
    await waitFor(() => expect(container.querySelector('.ag-tab-panel.on .ag-count')?.textContent).toBe('2'))
    // The agent skills box's own rows: yours first, a bundled one badged.
    const rows = Array.from(container.querySelectorAll('.ag-tab-panel.on .set-row')).filter((r) => r.querySelector('.cap-mark'))
    expect(rows.map((r) => r.querySelector('.t')?.textContent?.trim())).toEqual(['my-deck', 'slides มากับแอป'])
    expect(rows[1].querySelector('.badge')).toBeTruthy()
    expect(container.querySelector('.ag-tab-panel.on input[type="checkbox"]')).toBeNull()
  })

  // The opening: the worker's form, aimed at the desk's own file. Read for
  // this head, written for this head, and the other head reads its own.
  it('opening edits this desk\'s STARTERS.md through the same form a worker has', async () => {
    vi.mocked(DeskStarters).mockImplementation(async (desk: string) =>
      (desk === 'coding'
        ? { headline: '{ชื่อ} วันนี้จะแก้ตรงไหน?', cards: [{ title: 'รันเทสต์', prompt: 'รันเทสต์ทั้งหมดแล้วบอกผล', icon: 'play' }] }
        : { headline: '', cards: [] }) as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await fireEvent.click(cards(container)[1].querySelector('.icobtn')!)
    await waitFor(() => expect(DeskStarters).toHaveBeenCalledWith('coding', expect.any(String)))
    await fireEvent.click(tabs(container).find((t) => t.textContent?.includes('เปิดบทสนทนา'))!)
    const head = await waitFor(() => {
      const i = container.querySelector('.ag-tab-panel.on input.ctrl') as HTMLInputElement
      expect(i?.value).toBe('{ชื่อ} วันนี้จะแก้ตรงไหน?')
      return i
    })
    expect(container.querySelector('.ag-tab-panel.on .mono-dim')?.textContent).toContain('modes/coding/STARTERS.md')
    await fireEvent.input(head, { target: { value: '{ชื่อ} เริ่มจากไฟล์ไหนดี?' } })
    await fireEvent.click(Array.from(container.querySelectorAll('.ag-tab-panel.on .ctrl-primary')).at(-1)!)
    await waitFor(() => expect(SaveDeskStarters).toHaveBeenCalledWith('coding', expect.any(String), expect.objectContaining({ headline: '{ชื่อ} เริ่มจากไฟล์ไหนดี?' })))
    // Crossing heads reads the other file, never this one's rows.
    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl')).find((b) => b.textContent?.includes('ไปที่ ผู้ช่วย'))!)
    await waitFor(() => expect(DeskStarters).toHaveBeenCalledWith('assistant', expect.any(String)))
    await fireEvent.click(tabs(container).find((t) => t.textContent?.includes('เปิดบทสนทนา'))!)
    await waitFor(() => expect((container.querySelector('.ag-tab-panel.on input.ctrl') as HTMLInputElement).value).toBe(''))
  })

})

// ---- ตัวตน ------------------------------------------------------------------
// The desk file and the head's own identity files, edited here since 14 ก.ย.
// 2026: "คำสั่งประจำตัวพวกนี้ผูกกับเอเจนหลัก … แยกกันทั้งสองตัว เอาไว้ที่ส่วนตัวตน",
// "เอา โต๊ะ ออก แล้วเอา modes/coding.md มาแสดงให้คนปรับแต่งได้ … คืนค่าเริ่มต้นได้เสมอ
// ก่อนคืนค่าให้ถามยืนยัน", "เอา เพิ่มไฟล์คำสั่งใหม่ ออก".
const openHead = async (which: 0 | 1) => {
  const r = render(Settings, { onClose: () => {} })
  await openSection(r.container, 'ตัวหลัก')
  await waitFor(() => expect(cards(r.container).length).toBe(2))
  await fireEvent.click(cards(r.container)[which].querySelector('.icobtn')!)
  await waitFor(() => expect(r.container.querySelector('.ag-tab-panel.on')?.textContent).toContain('modes/'))
  return r
}
const panel = (c: HTMLElement) => c.querySelector('.ag-tab-panel.on') as HTMLElement
const rowNames = (c: HTMLElement) => Array.from(panel(c).querySelectorAll('.set-row .you-file')).map((x) => x.textContent?.trim())

describe('ตัวตน', () => {
  beforeEach(() => {
    vi.mocked(ListIdentityFiles).mockImplementation(async (head: string) =>
      (head === 'coding' ? [{ name: 'identity.md' }] : [{ name: 'context.md' }, { name: 'notes.md' }]) as any)
    vi.mocked(ReadIdentityFile).mockResolvedValue('# บริบทผู้ใช้\n\n- ทำ Aetox อยู่')
  })

  // One folder per head: the list is the head's own, and crossing to the
  // other head reads the other folder. The three files are always offered;
  // a hand-made file on disk is still listed; nothing here makes a new one.
  it("lists the desk file and this head's own four files, and reads the other head's when crossing", async () => {
    const { container } = await openHead(0)
    await waitFor(() => expect(ListIdentityFiles).toHaveBeenCalledWith('assistant'))
    await waitFor(() => expect(rowNames(container)).toEqual(['modes/assistant.md', 'identity.md', 'thinking.md', 'context.md', 'notes.md']))
    // context.md exists on this head, identity.md does not: one is opened, the other created.
    const rows = Array.from(panel(container).querySelectorAll('.set-row'))
    expect(rows.find((r) => r.textContent?.includes('context.md'))?.textContent).toContain('เปิดแก้ไข')
    expect(rows.find((r) => r.textContent?.includes('identity.md'))?.textContent).toContain('สร้างไฟล์')
    expect(container.querySelector('.identity-newfile-input')).toBeNull()
    expect(panel(container).textContent).not.toContain('เพิ่มไฟล์คำสั่งใหม่')

    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl')).find((b) => b.textContent?.includes('ไปที่ โค้ด'))!)
    await waitFor(() => expect(ListIdentityFiles).toHaveBeenCalledWith('coding'))
    await waitFor(() => expect(rowNames(container)).toEqual(['modes/coding.md', 'identity.md', 'thinking.md', 'context.md']))
  })

  // The editor is a row until asked for; a save goes to THIS head's folder.
  it('opens a file behind its row and saves it to this head', async () => {
    const { container } = await openHead(0)
    await waitFor(() => expect(rowNames(container)).toContain('context.md'))
    expect(container.querySelector('textarea[aria-label="context.md"]')).toBeNull()
    const row = Array.from(panel(container).querySelectorAll('.set-row')).find((r) => r.textContent?.includes('context.md'))!
    await fireEvent.click(row.querySelector('.ctrl')!)
    const box = await waitFor(() => container.querySelector('textarea[aria-label="context.md"]') as HTMLTextAreaElement)
    expect(ReadIdentityFile).toHaveBeenCalledWith('assistant', 'context.md')
    await waitFor(() => expect(box.value).toContain('ทำ Aetox อยู่'))
    await fireEvent.input(box, { target: { value: '# บริบทผู้ใช้\n\n- เครื่อง Windows' } })
    await fireEvent.click(container.querySelector('.identity-save')!)
    await waitFor(() => expect(SaveIdentityFile).toHaveBeenCalledWith('assistant', 'context.md', expect.stringContaining('เครื่อง Windows')))
  })

  // "+" writes the template into this head's folder and opens it.
  it('creates a missing file from its template, for this head', async () => {
    const { container } = await openHead(1)
    await waitFor(() => expect(rowNames(container)).toContain('thinking.md'))
    const row = Array.from(panel(container).querySelectorAll('.set-row')).find((r) => r.textContent?.includes('thinking.md'))!
    await fireEvent.click(row.querySelector('.ctrl')!)
    await waitFor(() => expect(SaveIdentityFile).toHaveBeenCalledWith('coding', 'thinking.md', expect.stringMatching(/.+/)))
    await waitFor(() => expect(container.querySelector('textarea[aria-label="thinking.md"]')).toBeTruthy())
  })

  // The desk file: the whole text behind its row, a save through the
  // binding, and the way back to the bundled file — asked first.
  it('edits the desk file whole, and restores the default only after asking', async () => {
    const { container } = await openHead(1)
    await waitFor(() => expect(ReadDeskFile).toHaveBeenCalledWith('coding'))
    // Not overridden yet: nothing to restore.
    expect(Array.from(panel(container).querySelectorAll('.ctrl')).some((b) => b.textContent?.includes('คืนค่าเริ่มต้น'))).toBe(false)
    const deskRow = Array.from(panel(container).querySelectorAll('.set-row')).find((r) => r.textContent?.includes('modes/coding.md'))!
    await fireEvent.click(deskRow.querySelector('.ctrl')!)
    const box = await waitFor(() => container.querySelector('textarea[aria-label="modes/coding.md"]') as HTMLTextAreaElement)
    expect(box.value).toContain('This session is coding work.')
    const edited = box.value + '\n\nAlways run the tests.'
    await fireEvent.input(box, { target: { value: edited } })
    vi.mocked(ReadDeskFile).mockResolvedValue({ name: 'coding', text: edited, overrides: true } as any)
    await fireEvent.click(Array.from(container.querySelectorAll('.you-save .ctrl-primary'))[0])
    await waitFor(() => expect(SaveDeskFile).toHaveBeenCalledWith('coding', expect.stringContaining('Always run the tests.')))
    await waitFor(() => expect(panel(container).textContent).toContain('แก้ไว้เอง'))

    // Now overridden: the way back is offered, and it asks before acting.
    const reset = await waitFor(() => Array.from(panel(container).querySelectorAll('.ctrl')).find((b) => b.textContent?.includes('คืนค่าเริ่มต้น'))!)
    await fireEvent.click(reset)
    expect(ResetDeskFile).not.toHaveBeenCalled()
    await waitFor(() => expect(screen.getByText('คืนหน้าที่เป็นค่าเริ่มต้นของแอป?')).toBeTruthy())
    const confirm = Array.from(document.querySelectorAll('.confirm-actions button, .modal button')).find((b) => b.textContent?.trim() === 'คืนค่าเริ่มต้น') as HTMLButtonElement
    await fireEvent.click(confirm)
    await waitFor(() => expect(ResetDeskFile).toHaveBeenCalledWith('coding'))
  })

  // context.md is the head's, not the person's: on this tab and nowhere on
  // เกี่ยวกับคุณ (it sat there for a morning).
  it('keeps context.md here and off เกี่ยวกับคุณ', async () => {
    const { container } = await openHead(0)
    await waitFor(() => expect(rowNames(container)).toContain('context.md'))
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(screen.getByText('ชื่อของคุณ')).toBeTruthy())
    expect(container.textContent).not.toContain('context.md')
    // And there is no คำสั่งประจำตัว page any more.
    expect(Array.from(container.querySelectorAll('.settings-nav-item')).some((n) => n.textContent?.trim() === 'คำสั่งประจำตัว')).toBe(false)
  })
})

// The head's name is a field, not a line in a file (owner, 14 ก.ย. 2026:
// "ชื่อควรจะเป็นชื่อที่เปลี่ยนได้"): written on change through SetHeadName,
// worn by the card and the page title, with the desk's word as the badge.
describe('the name', () => {
  it('is edited on ตัวตน, saved on change, and worn by the card', async () => {
    vi.mocked(HeadName).mockImplementation(async (h: string) => (h === 'coding' ? 'Dev' : ''))
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await waitFor(() => expect(cards(container)[1].querySelector('.chair-name')?.textContent).toContain('Dev'))
    expect(cards(container)[1].querySelector('.chair-name .badge')?.textContent).toBe('โค้ด')
    expect(cards(container)[0].querySelector('.chair-name .badge')).toBeNull()

    await fireEvent.click(cards(container)[0].querySelector('.icobtn')!)
    const box = await waitFor(() => container.querySelector('.ag-tab-panel.on input[aria-label="ชื่อ"]') as HTMLInputElement)
    expect(box.value).toBe('')
    await fireEvent.input(box, { target: { value: ' Nova ' } })
    await fireEvent.change(box)
    await waitFor(() => expect(SetHeadName).toHaveBeenCalledWith('assistant', 'Nova'))
    await waitFor(() => expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตั้งค่า Nova'))
  })
})
