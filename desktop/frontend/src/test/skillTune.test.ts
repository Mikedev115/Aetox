// ปรับสกิลอัตโนมัติ: since 14 ก.ย. 2026 the fourth row under สกิล in ห้องความสามารถ,
// moved whole out of ตั้งค่า. A drafted skill fix is shown with its diff, applies
// only on approval, and the switch that decides whether it drafts on its own
// persists. The rail row, the switch and the queue keep their settings.* keys:
// the words did not change, only the address.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import {
  ListSkillProposals, SkillTuneAuto, SetSkillTuneAuto, RunSkillTuneup,
  ApprovePendingChange, ListMCPServers, ListExternalSkills, ListSubagentProfiles, PlacementTargets, ListTools,
} from './mocks/wailsApp'

const skillProposal = (over: Record<string, unknown> = {}) => ({
  id: 7, kind: 'skill', scope: 'aetox-slides', target: '',
  op: 'add', before: '', body: 'เพิ่มกฎ: ถ้า OCR ได้น้อยกว่า 3 บรรทัด ให้ถามผู้ใช้ก่อน',
  reason: 'โดน 👎 3 ครั้ง', evidence: 'jobs:1,2,3',
  source: 'optimizer', state: 'pending', createdAt: '2026-08-26T10:00:00Z', decidedAt: '',
  ...over,
})

const openSection = async (container: HTMLElement, label: string) => {
  const item = Array.from(container.querySelectorAll('.settings-nav-item'))
    .find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}

const SKILLTUNE = 'ปรับสกิลอัตโนมัติ'
// The room's first load lands on a page of its own choosing; the click has to
// come after it, the way a person's does.
const openRoom = async () => {
  const r = render(Capability, { onClose: () => {} })
  await waitFor(() => expect(document.querySelector('.office-grid')).toBeTruthy())
  return r
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(SkillTuneAuto).mockResolvedValue(false)
  vi.mocked(ListMCPServers).mockResolvedValue([] as any)
  vi.mocked(ListExternalSkills).mockResolvedValue([] as any)
  vi.mocked(ListSubagentProfiles).mockResolvedValue([] as any)
  vi.mocked(PlacementTargets).mockResolvedValue([] as any)
  vi.mocked(ListTools).mockResolvedValue([] as any)
  vi.mocked(ListSkillProposals).mockResolvedValue([] as any)
})

describe('the skill-tuning room', () => {
  it('shows a drafted skill fix with its diff, and applies it only on approval', async () => {
    vi.mocked(ListSkillProposals).mockResolvedValue([skillProposal()] as any)
    const { container } = await openRoom()
    await openSection(container, SKILLTUNE)

    // The skill it edits and what it would add — the diff a person reads before
    // saying yes.
    await waitFor(() => expect(screen.getByText('aetox-slides')).toBeTruthy())
    expect(screen.getByText(/เพิ่มกฎ/)).toBeTruthy()

    // Nothing applied yet; the approve button is what applies it.
    expect(ApprovePendingChange).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByText('อนุมัติ'))
    expect(ApprovePendingChange).toHaveBeenCalledWith(7)
  })

  it('empties to a message rather than a blank card', async () => {
    const { container } = await openRoom()
    await openSection(container, SKILLTUNE)
    await waitFor(() => expect(screen.getByText(/ยังไม่มีอะไรให้ตรวจ/)).toBeTruthy())
  })

  it('persists the auto-draft switch', async () => {
    const { container } = await openRoom()
    await openSection(container, SKILLTUNE)
    const sw = await waitFor(() => container.querySelector('.mswitch input') as HTMLInputElement)
    await fireEvent.click(sw)
    expect(SetSkillTuneAuto).toHaveBeenCalledWith(true)
  })

  it('runs a tuneup on demand', async () => {
    vi.mocked(RunSkillTuneup).mockResolvedValue(1 as any)
    const { container } = await openRoom()
    await openSection(container, SKILLTUNE)
    await fireEvent.click(screen.getByText('ตรวจตอนนี้'))
    expect(RunSkillTuneup).toHaveBeenCalled()
  })
})
