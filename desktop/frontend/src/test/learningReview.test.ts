// The approval surface. The whole learning design rests on nothing taking
// effect until a person allows it, which only holds up if the person can see
// what is waiting, judge it, and be told it is there at all.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import Sidebar from '../lib/Sidebar.svelte'
import {
  ListPendingChanges, ListDecidedChanges, LearnedMemory, LearnedEntries, SaveLearnedEntry, MoveLearnedEntry,
  LearningEnabled, ApprovePendingChange, ApprovePendingChangeTo, RejectPendingChange, SetLearningEnabled,
  PendingLearnedCount, LearnedScopeInfos, ForgetMemoryScope, AdoptMemoryScope, RecentProjects,
  ConsolidateMemory, ApplyMemoryLines, ListSubagentProfiles, ReadSubagentProfile,
} from './mocks/wailsApp'
import { cockpit, applyPendingLearned, refreshPendingLearned } from '../lib/stores/cockpit.svelte'

const proposal = (over: Record<string, unknown> = {}) => ({
  id: 1, kind: 'memory', scope: '', target: 'C:/aetox/memory/MEMORY.md',
  op: 'add', before: '', body: 'เครื่องนี้ไม่มี Excel ติดตั้ง',
  reason: 'เปิดไฟล์ .xlsx แล้วไม่มีโปรแกรมรับ', evidence: 'session:1',
  source: 'agent', state: 'pending', createdAt: '2026-08-04T10:00:00Z', decidedAt: '',
  ...over,
})

const openSection = async (container: HTMLElement, label: string) => {
  const item = Array.from(container.querySelectorAll('.settings-nav-item'))
    .find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}

// ตัวหลัก › <head> › ความจำ — where a desk's file, its projects and its queue
// live since 14 ก.ย. 2026. The card's gear opens the editor; the tab is the
// agent editor's own bar.
const openHeadMemory = async (container: HTMLElement, head: 'ผู้ช่วย' | 'โค้ด') => {
  await openSection(container, 'ตัวหลัก')
  const card = await waitFor(() => {
    const c = Array.from(container.querySelectorAll('.main-card')).find((x) => x.querySelector('.chair-name')?.textContent?.trim() === head)
    expect(c).toBeTruthy()
    return c!
  })
  await fireEvent.click(card.querySelector('.icobtn')!)
  const tab = await waitFor(() => {
    const b = Array.from(container.querySelectorAll('.ag-tabs-bar [role="tab"]')).find((x) => x.textContent?.includes('ความจำ'))
    expect(b).toBeTruthy()
    return b!
  })
  await fireEvent.click(tab)
}

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.pendingLearned = 0
  vi.mocked(LearningEnabled).mockResolvedValue(true)
  vi.mocked(LearnedMemory).mockResolvedValue('')
  vi.mocked(LearnedEntries).mockResolvedValue([] as any)
  vi.mocked(LearnedScopeInfos).mockResolvedValue([] as any)
  vi.mocked(ListDecidedChanges).mockResolvedValue([] as any)
  vi.mocked(ListPendingChanges).mockResolvedValue([proposal()] as any)
})

// The memory list is the one thing on this page the *user* writes. Approving is
// how the agent's proposals get in; this is how a line that got in and turned
// out to be noise gets fixed, without leaving the app for a text editor.
describe('editing what is already remembered', () => {
  // The case the row index exists for. A real file collects lines that differ
  // only at the end, and the fourth must be the one that moves.
  const shell = (status: string) =>
    `เครื่องมือ shell เคยล้มซ้ำ ๆ ด้วยเหตุเดียวกัน: "exit status ${status}"`

  const openMemory = async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([{ scope: '', orphan: false }] as any)
    vi.mocked(LearnedEntries).mockResolvedValue(
      ['เครื่องผู้ใช้เป็น Windows', shell('1'), shell('2'), shell('124')] as any,
    )
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(4))
    return container
  }

  it('gives every remembered line its own row', async () => {
    const container = await openMemory()
    const rows = container.querySelectorAll('.mem-row .mem-text')
    expect(rows[0].textContent).toContain('Windows')
    expect(rows[3].textContent).toContain('exit status 124')
  })

  it('saves the row that was edited, by its position and not by its text', async () => {
    const container = await openMemory()
    const rows = Array.from(container.querySelectorAll('.mem-row'))
    await fireEvent.click(rows[3].querySelector('.icobtn')!)

    const box = container.querySelector('.mem-input') as HTMLTextAreaElement
    expect(box.value).toContain('exit status 124')
    await fireEvent.input(box, { target: { value: 'shell timeout บ่อยในโปรเจกต์นี้' } })
    // Scoped to the open row: the approval card above has a .ctrl-primary too,
    // and it comes first in the document.
    await fireEvent.click(container.querySelector('.mem-row.editing .ctrl-primary')!)

    // Index 3 — the row on screen. The three lines above it all contain
    // "exit status", so anything matching on text would take the wrong one.
    await waitFor(() =>
      expect(SaveLearnedEntry).toHaveBeenCalledWith('', 3, 'shell timeout บ่อยในโปรเจกต์นี้'))
  })

  it('forgets a line by saving it empty, and reloads so the rows renumber', async () => {
    const container = await openMemory()
    const rows = Array.from(container.querySelectorAll('.mem-row'))
    await fireEvent.click(rows[1].querySelector('.mem-forget')!)

    await waitFor(() => expect(SaveLearnedEntry).toHaveBeenCalledWith('', 1, ''))
    // A delete moves every row below it, so the positions the next edit sends
    // have to come from the file again rather than from the stale array.
    // Awaited: the reload asks which scopes exist before it asks for any lines,
    // so the re-read lands a tick later than it used to.
    await waitFor(() => expect(LearnedEntries).toHaveBeenCalledTimes(2))
  })

  it('says why a save failed instead of quietly leaving the line as it was', async () => {
    const container = await openMemory()
    vi.mocked(SaveLearnedEntry).mockRejectedValueOnce(new Error('memory is full'))
    await fireEvent.click(container.querySelectorAll('.mem-row')[0].querySelector('.icobtn')!)
    await fireEvent.input(container.querySelector('.mem-input')!, { target: { value: 'x' } })
    // Scoped to the open row: the approval card above has a .ctrl-primary too,
    // and it comes first in the document.
    await fireEvent.click(container.querySelector('.mem-row.editing .ctrl-primary')!)

    await waitFor(() => expect(container.textContent).toContain('memory is full'))
  })

  // A line can land in three different files now — the shared one, a desk's, a
  // project's — and this page was built when there was only one. A file it
  // cannot show is a file only the folder knows about, which is the whole thing
  // the page exists to avoid.
  it('shows every memory that holds something, each under whose it is', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: '', orphan: false }, { scope: 'mode:coding', orphan: false, projectsUnder: true }, { scope: 'project:Aetox-1a2b3c4d', orphan: false },
    ] as any)
    vi.mocked(LearnedEntries).mockImplementation(async (scope: string) => {
      if (scope === '') return ['เครื่องผู้ใช้เป็น Windows'] as any
      if (scope === 'mode:coding') return ['เจ้าของอ่านดิฟก่อนเสมอ'] as any
      return ['เราตกลงกันว่า statereport เป็นคนบอกว่า error มาจากโลกภายนอก'] as any
    })

    const { container } = render(Settings, { onClose: () => {} })
    // The profile's block is เกี่ยวกับคุณ's (14 ก.ย. 2026), drawn even when
    // empty; การเรียนรู้ keeps one block per desk, with the project nested
    // under the desk whose sessions write it.
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(container.querySelectorAll('.mem-scope-name').length).toBe(1))
    expect(container.querySelector('.mem-scope-name')?.textContent?.trim()).toBe('เกี่ยวกับคุณ')
    // And ตัวหลัก: the assistant's page holds its file alone; the coder's
    // holds its own with the project nested under it (projectsUnder).
    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(1))
    expect(Array.from(container.querySelectorAll('.mem-scope-name')).map((el) => el.textContent?.trim())).toEqual(['ผู้ช่วย'])
    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl')).find((b) => b.textContent?.includes('ไปที่ โค้ด'))!)
    await fireEvent.click(Array.from(container.querySelectorAll('.ag-tabs-bar [role="tab"]')).find((x) => x.textContent?.includes('ความจำ'))!)
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(2))
    const heads = Array.from(container.querySelectorAll('.mem-scope-name')).map((el) => el.textContent?.trim())
    expect(heads).toEqual(['โค้ด', 'โปรเจกต์ Aetox'])
    expect(container.querySelector('.mem-sub .mem-scope-name')?.textContent).toContain('Aetox')
    // Each heading says who reads the file — the label alone never did.
    const auds = Array.from(container.querySelectorAll('.mem-scope .learn-aud')).map((el) => el.textContent?.trim())
    expect(auds).toEqual(['เฉพาะโค้ด ทุกโปรเจกต์', 'เฉพาะตอนเปิดโฟลเดอร์ Aetox'])
    // The hash half of a project key is identity, not information — a person
    // recognises the folder, not the digest. It stays in the file badge only,
    // because that badge is the name on disk.
    expect(heads.join(' ')).not.toContain('1a2b3c4d')
    expect(container.querySelector('.mem-sub .mem-badge-file')?.textContent).toBe('projects/Aetox-1a2b3c4d.md')
  })

  // The ceiling, on the page. A full profile used to be a fact only the tool
  // knew — proposals refused, the session review skipping silently.
  it('draws each file\'s meter and says when one is full', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: 'user:profile', orphan: false, bytes: 3900, maxBytes: 4096, full: true },
      { scope: '', orphan: false, bytes: 1214, maxBytes: 8192, full: false },
    ] as any)
    vi.mocked(LearnedEntries).mockResolvedValue(['x'] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(container.querySelectorAll('.mem-cap').length).toBe(1))
    const cap = container.querySelector('.mem-cap')!
    expect(cap.textContent).toContain('3,900 / 4,096')
    expect(cap.classList.contains('mem-cap-full')).toBe(true)
    expect(container.querySelectorAll('.mem-cap-note').length).toBe(1)
    expect(container.querySelector('.mem-cap-note')?.textContent).toContain('เต็มแล้ว')

    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(container.querySelectorAll('.mem-cap').length).toBe(1))
    expect(container.querySelector('.mem-cap')!.classList.contains('mem-cap-ok')).toBe(true)
    expect(container.querySelectorAll('.mem-cap-note').length).toBe(0)
  })

  // A project's memory file is keyed by the folder's path, so a moved or
  // deleted folder strands its file — readable in the folder, reachable by no
  // session, and indistinguishable on disk from a live one. The page is the
  // only place that can say so, and a label without its exits is a nagging
  // sign: move the lines to the project the folder became, or let them go.
  it('names an orphaned project memory and offers its two exits', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: '', orphan: false }, { scope: 'mode:coding', orphan: false, projectsUnder: true },
      { scope: 'project:old-app-99aa88bb', orphan: true },
    ] as any)
    vi.mocked(LearnedEntries).mockImplementation(async (scope: string) =>
      (scope === '' ? ['เครื่องผู้ใช้เป็น Windows'] : scope === 'mode:coding' ? [] : ['ตกลงกันว่าใช้ PowerShell']) as any)
    vi.mocked(RecentProjects).mockResolvedValue([
      { key: 'old-app-11223344', name: 'old-app', rootPath: 'D:/work/old-app', openedAt: '', snippet: '' },
    ] as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'โค้ด')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(1))

    // The live file carries no mark; the orphan carries exactly one.
    expect(container.querySelectorAll('.mem-orphan').length).toBe(1)
    const live = container.querySelector('.mem-scope[data-mem-scope="mode:coding"]')!
    const orphan = container.querySelector('.mem-scope[data-mem-scope="project:old-app-99aa88bb"]')!
    expect(live.querySelector('.mem-orphan')).toBeNull()
    expect(orphan.textContent).toContain('โฟลเดอร์นี้ไม่อยู่แล้ว')

    // Exit one: move into a project the store still knows.
    await fireEvent.click(orphan.querySelector('.mem-orphan-actions .ctrl')!)
    const target = Array.from(container.querySelectorAll('.mem-adopt .ctrl'))
      .find((el) => el.textContent?.includes('old-app'))
    await fireEvent.click(target!)
    await waitFor(() =>
      expect(AdoptMemoryScope).toHaveBeenCalledWith('project:old-app-99aa88bb', 'D:/work/old-app'))

    // Exit two: delete — and it reaches the whole-file door, never a row's.
    await fireEvent.click(orphan.querySelector('.mem-orphan-actions .mem-forget')!)
    await waitFor(() =>
      expect(ForgetMemoryScope).toHaveBeenCalledWith('project:old-app-99aa88bb'))
  })

  // Editing has to reach the file the row came from. Every group counts its own
  // rows from zero, so a save that forgot the scope would rewrite line 0 of the
  // main memory while the user was looking at line 0 of a project's.
  it('edits the line in the file it belongs to', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: '', orphan: false }, { scope: 'mode:coding', orphan: false, projectsUnder: true }, { scope: 'project:Aetox-1a2b3c4d', orphan: false },
    ] as any)
    vi.mocked(LearnedEntries).mockImplementation(async (scope: string) =>
      (scope === '' ? ['เครื่องผู้ใช้เป็น Windows'] : scope === 'mode:coding' ? [] : ['ตกลงกันว่าใช้ PowerShell']) as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'โค้ด')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(1))

    const projectRow = container.querySelectorAll('.mem-row')[0]
    await fireEvent.click(projectRow.querySelector('.mem-forget')!)

    await waitFor(() =>
      expect(SaveLearnedEntry).toHaveBeenCalledWith('project:Aetox-1a2b3c4d', 0, ''))
  })

  // A full destination (the owner's USER.md at 4,086 of 4,096 bytes) made
  // "ย้ายทั้ง 4" look like a dead button: Go refused, and the error landed at
  // the top of the page, off-screen from the block it was pressed in. The
  // refusal now shows where the press was, the button is disabled ahead of
  // it, and the move menu marks the full file rather than offering it.
  it('says in place when the destination is full, and does not offer it', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: 'user:profile', orphan: false, bytes: 4086, maxBytes: 4096, full: true },
      { scope: '', orphan: false, bytes: 936, maxBytes: 8192, full: false },
      { scope: 'mode:coding', orphan: false, bytes: 0, maxBytes: 8192, full: false, projectsUnder: true },
    ] as any)
    vi.mocked(LearnedEntries).mockImplementation(async (scope: string) =>
      (scope === '' ? ['User likes cloning Framer templates'] : scope === 'user:profile' ? ['ผู้ใช้พูดไทย'] : []) as any)
    vi.mocked(MoveLearnedEntry).mockRejectedValue(new Error("this scope's memory is full (4100 bytes, limit 4096)"))

    const { container } = render(Settings, { onClose: () => {} })
    // The banner offering the move sits where the lines would land.
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(container.querySelector('.mem-quick-banner')).toBeTruthy())
    const migrate = container.querySelector('.mem-quick-banner .ctrl') as HTMLButtonElement
    expect(migrate.disabled).toBe(true)

    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(1))
    const mainRow = container.querySelectorAll('.mem-row')[0]
    await fireEvent.click(mainRow.querySelector('.mem-action-move')!)
    const full = mainRow.querySelector('.mem-menu-i.full') as HTMLButtonElement
    expect(full.textContent).toContain('เกี่ยวกับคุณ')
    expect(full.textContent).toContain('เต็มแล้ว')
    expect(full.disabled).toBe(true)

    // The other target still works, and a refusal from Go shows in the block.
    const coding = Array.from(mainRow.querySelectorAll('.mem-menu-i')).find((el) => el.textContent?.includes('โค้ด'))!
    await fireEvent.click(coding)
    await waitFor(() => expect(MoveLearnedEntry).toHaveBeenCalledWith('', 'mode:coding', 0))
    const block = container.querySelector('.mem-desk .mem-scope[data-mem-scope=""]')!.closest('.mem-desk')!
    await waitFor(() => expect(block.querySelector('.mem-move-error')?.textContent).toContain('เต็มแล้ว'))
  })

  // "ให้ผู้ช่วยช่วยสรุป": a full file offers the model's shorter list, read
  // beside the current one; nothing is written until the user applies it.
  it('offers a merged draft for a full file and writes only on apply', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: 'user:profile', orphan: false, bytes: 4086, maxBytes: 4096, full: true },
      { scope: '', orphan: false, bytes: 100, maxBytes: 8192, full: false },
    ] as any)
    vi.mocked(LearnedEntries).mockImplementation(async (scope: string) =>
      (scope === 'user:profile' ? ['ผู้ใช้เก็บเอกสารที่ D:/docs', 'ผู้ใช้มีโฟลเดอร์ทำงานที่ D:/docs'] : ['x']) as any)
    vi.mocked(ConsolidateMemory).mockResolvedValue({
      scope: 'user:profile', before: ['ผู้ใช้เก็บเอกสารที่ D:/docs', 'ผู้ใช้มีโฟลเดอร์ทำงานที่ D:/docs'],
      after: ['ผู้ใช้เก็บเอกสารและทำงานที่ D:/docs'], note: 'รวมสองบรรทัดเรื่องโฟลเดอร์', bytes: 400, maxBytes: 4096,
    } as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(container.querySelector('.mem-cap-note.mem-cap-full')).toBeTruthy())

    // Only the full file offers it.
    expect(container.querySelectorAll('.mem-consolidate').length).toBe(1)
    await fireEvent.click(container.querySelector('.mem-consolidate')!)
    await waitFor(() => expect(container.querySelector('.mem-draft')).toBeTruthy())
    expect(ConsolidateMemory).toHaveBeenCalledWith('user:profile')
    expect(ApplyMemoryLines).not.toHaveBeenCalled()

    const draft = container.querySelector('.mem-draft')!
    expect(draft.querySelectorAll('.mem-draft-line.was').length).toBe(2)
    expect(draft.querySelectorAll('.mem-draft-line:not(.was)').length).toBe(1)
    expect(draft.textContent).toContain('รวมสองบรรทัดเรื่องโฟลเดอร์')
    expect(draft.textContent).toContain('400 B')

    await fireEvent.click(draft.querySelector('.ctrl-primary')!)
    await waitFor(() => expect(ApplyMemoryLines).toHaveBeenCalledWith('user:profile', ['ผู้ใช้เก็บเอกสารและทำงานที่ D:/docs']))
    await waitFor(() => expect(container.querySelector('.mem-draft')).toBeNull())
  })

  // The folder is the promise that this is plain markdown you can take away.
  it('keeps the way out to the folder', async () => {
    const container = await openMemory()
    expect(container.querySelector('.learn-foot .ctrl')?.textContent)
      .toContain('เปิดโฟลเดอร์ความจำ')
  })
})

describe('the learning review page', () => {
  // Approving a sentence without its reasoning is signing for an assertion
  // with no provenance, which is the thing this page exists to prevent.
  it('shows what would be remembered, whose memory it is, and why', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')

    await waitFor(() => expect(screen.getByText('เครื่องนี้ไม่มี Excel ติดตั้ง')).toBeTruthy())
    expect(screen.getByText('เปิดไฟล์ .xlsx แล้วไม่มีโปรแกรมรับ')).toBeTruthy()
    // The verb in the user's language, never the database's own enum; whose
    // file, and — since 11 ก.ย. — who reads it, which is the decision.
    const head = container.querySelector('.learn-row .learn-head')!
    expect(head.textContent).toContain('ขอจำเรื่องนี้ไว้')
    expect(head.textContent).toContain('ผู้ช่วย')
    expect(head.textContent).toContain('เฉพาะแชทกับผู้ช่วย')
    expect(head.textContent).not.toContain('add')
  })

  // "เก็บที่อื่น": a proposal can be approved into a file other than the one
  // it was aimed at, and the menu says who would read it there. Only a new
  // line offers it — a replace names a line that lives in one file.
  it('lets a new line be kept in a different file', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: 'user:profile', orphan: false }, { scope: '', orphan: false }, { scope: 'mode:coding', orphan: false, projectsUnder: true },
    ] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(screen.getByText('เครื่องนี้ไม่มี Excel ติดตั้ง')).toBeTruthy())

    await fireEvent.click(screen.getByText('เก็บที่อื่น'))
    const items = Array.from(container.querySelectorAll('.learn-row .mem-menu-i'))
    // Every other file, never the one it is already aimed at.
    expect(items.map((el) => el.querySelector('.learn-scope')?.textContent?.trim())).toEqual(['เกี่ยวกับคุณ', 'โค้ด'])
    expect(items[0].textContent).toContain('ทั้งผู้ช่วย โค้ด และทุกลูกมือจะเห็น')
    await fireEvent.click(items[1])
    await waitFor(() => expect(ApprovePendingChangeTo).toHaveBeenCalledWith(1, 'mode:coding'))

    vi.mocked(ListPendingChanges).mockResolvedValue([proposal({ op: 'replace', before: 'x', body: 'y' })] as any)
    const second = render(Settings, { onClose: () => {} })
    await openHeadMemory(second.container, 'ผู้ช่วย')
    await waitFor(() => expect(second.container.textContent).toContain('ขอแก้สิ่งที่จำไว้'))
    expect(second.container.querySelector('.learn-row .mem-move')).toBeNull()
  })

  // A delegate's memory is not the assistant's, and the row has to say so —
  // scope is the difference between "everything you ask it" and "one job".
  it('puts a delegate\'s proposal on the delegate\'s own page, not on a head\'s', async () => {
    vi.mocked(ListPendingChanges).mockResolvedValue([proposal({ scope: 'explore' })] as any)
    vi.mocked(ListSubagentProfiles).mockResolvedValue([{ name: 'explore', description: 'ค้นไฟล์', prompt: 'role', builtin: true }] as any)
    vi.mocked(ReadSubagentProfile).mockResolvedValue('---\ndescription: ค้นไฟล์\n---\nYou search files.' as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(container.querySelectorAll('.mem-scope-name').length).toBe(1))
    expect(container.querySelector('.learn-row')).toBeNull()

    // The intent road opens an editor only through 'team'; a helper's page
    // is the same editor pane, so it serves to show where the queue landed.
    cockpit.settingsIntent = { section: 'team', agent: 'explore' }
    const own = render(Settings, { onClose: () => {} })
    await waitFor(() => expect(own.container.querySelector('.ag-body')).toBeTruthy())
    await waitFor(() => expect(own.container.querySelector('.learn-row .learn-scope')?.textContent).toContain('explore'))
  })

  // What a change overwrites is part of the decision.
  it('shows the line a replacement would overwrite', async () => {
    vi.mocked(ListPendingChanges).mockResolvedValue([
      proposal({ op: 'replace', before: 'สแกนเนอร์เขียนลง D:\\Scans', body: 'สแกนเนอร์เขียนลง E:\\Scans' }),
    ] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')

    await waitFor(() => expect(screen.getByText('สแกนเนอร์เขียนลง D:\\Scans')).toBeTruthy())
    expect(screen.getByText('สแกนเนอร์เขียนลง E:\\Scans')).toBeTruthy()
  })

  it('approves and discards through to the engine', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(screen.getByText('อนุมัติ')).toBeTruthy())

    await fireEvent.click(screen.getByText('อนุมัติ'))
    await waitFor(() => expect(ApprovePendingChange).toHaveBeenCalledWith(1))

    await fireEvent.click(screen.getByText('ไม่เอา'))
    await waitFor(() => expect(RejectPendingChange).toHaveBeenCalledWith(1))
  })

  // An approval that could not be applied leaves the proposal in the list, and
  // a button that appears to do nothing reads as a broken feature.
  //
  // The mocked error is the one Apply can still produce — the scope filled up
  // between the proposal and the click. It used to be "no remembered line
  // contains", which §139 removed: a stale revision now lands instead of
  // erroring, so a test still waving that message would be rehearsing a
  // failure the engine can no longer send.
  it('says so when an approval could not be applied', async () => {
    vi.mocked(ApprovePendingChange).mockRejectedValueOnce(
      new Error("this scope's memory is full (8215 bytes, limit 8192) — merge or drop an existing line first"))
    const { container } = render(Settings, { onClose: () => {} })
    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(screen.getByText('อนุมัติ')).toBeTruthy())

    await fireEvent.click(screen.getByText('อนุมัติ'))
    await waitFor(() => expect(screen.getByText(/memory is full/)).toBeTruthy())
  })

  it('carries the kill switch, and turning it off reaches the engine', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การเรียนรู้')
    await waitFor(() => expect(screen.getByText('ให้ Aetox เรียนรู้จากงานที่ทำ')).toBeTruthy())

    const box = container.querySelector('.mswitch input') as HTMLInputElement
    expect(box.checked).toBe(true)
    await fireEvent.change(box)
    await waitFor(() => expect(SetLearningEnabled).toHaveBeenCalledWith(false))
  })

  it('separates user memory from assistant memory and provides move action', async () => {
    vi.mocked(LearnedScopeInfos).mockResolvedValue([
      { scope: 'user:profile', orphan: false },
      { scope: '', orphan: false },
    ] as any)
    vi.mocked(LearnedEntries).mockImplementation(async (scope: string) => {
      if (scope === 'user:profile') return ['ผู้ใช้ชอบภาษาไทย'] as any
      return ['User is developing Aetox'] as any
    })

    const { container } = render(Settings, { onClose: () => {} })
    // Two pages since 14 ก.ย. 2026: the person's file on เกี่ยวกับคุณ, with
    // the banner for lines that belong there; the assistant's on การเรียนรู้.
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(1))
    expect(container.textContent).toContain('ความจำเกี่ยวกับคุณ')
    expect(container.textContent).toContain('USER.md')
    expect(container.textContent).not.toContain('ความจำของผู้ช่วยและโค้ด')
    // Quick migrate banner is rendered because Main has "User is developing Aetox"
    expect(container.querySelector('.mem-quick-banner')).toBeTruthy()

    await openHeadMemory(container, 'ผู้ช่วย')
    await waitFor(() => expect(container.querySelectorAll('.mem-row').length).toBe(1))
    expect(container.textContent).toContain('MEMORY.md')
    expect(container.textContent).not.toContain('ความจำเกี่ยวกับคุณ')
    expect(container.querySelector('.mem-quick-banner')).toBeNull()

    // The move button opens a menu of every other file; a line about the user
    // sitting in the assistant's file has the profile marked as the suggestion.
    const mainRow = container.querySelectorAll('.mem-row')[0]
    await fireEvent.click(mainRow.querySelector('.mem-action-move')!)
    const rec = mainRow.querySelector('.mem-menu-i.rec')!
    expect(rec.textContent).toContain('เกี่ยวกับคุณ')
    expect(rec.textContent).toContain('แนะนำ')
    await fireEvent.click(rec)
    await waitFor(() => expect(MoveLearnedEntry).toHaveBeenCalledWith('', 'user:profile', 0))
  })
})

describe('being told there is something waiting', () => {
  it('marks the way into settings, and only when something is waiting', async () => {
    const { container } = render(Sidebar, {
      onOpenSettings: () => {}, onOpenFile: () => {}, collapsed: false, onToggle: () => {},
    } as any)
    expect(container.querySelector('.gear.has-pending')).toBeNull()

    applyPendingLearned(2)
    await waitFor(() => expect(container.querySelector('.gear.has-pending')).toBeTruthy())
  })

  it('counts the row that opens them', async () => {
    applyPendingLearned(3)
    const { container } = render(Settings, { onClose: () => {} })
    const badge = container.querySelector('.nav-count')
    expect(badge?.textContent?.trim()).toBe('3')
  })

  // Anything left undecided in an earlier session is still undecided, and
  // nothing would emit an event for it.
  it('picks up a queue left over from a previous session', async () => {
    vi.mocked(PendingLearnedCount).mockResolvedValue(4)
    await refreshPendingLearned()
    expect(cockpit.pendingLearned).toBe(4)
  })
})
