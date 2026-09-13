// ห้องความสามารถ. Since 12 ก.ย. 2026 the room holds one kind of thing, the MCP
// server, and is the ONE place it is handled: connected, tested, configured,
// handed to a desk or an agent, thrown away. ตั้งค่า › MCP is gone from the
// menu and the agent's own settings page only counts. Since 13 ก.ย. it is a
// rail of four pages in ตั้งค่า's frame: MCP server ของคุณ (readiness only:
// test, sign in, keys, tools), ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด and ตั้งค่า MCP
// สำหรับเอเจนเฉพาะทาง (placement, a card per target and one picker), ห้องสมุด MCP.
// The same day สกิล became the rail's second heading — three pages moved whole
// out of ตั้งค่า (and a fourth, ปรับสกิลอัตโนมัติ, on 14 ก.ย. — skillTune.test.ts): สกิลของคุณ (the shelf: what is on it, what did not read,
// the three install roads), ตั้งค่าสกิลสำหรับเอเจนเฉพาะทาง (a card per agent and
// one sheet that COPIES a shelf skill into the agent's own folder — there is no
// `for:` on a skill, every desk carries the whole shelf), ห้องสมุดสกิล.
//
// So what is pinned here is in four parts:
//   - what the room must NOT have any more (DESIGN.md §6: lock the forbidden
//     thing, it is what rots silently): no tab bar of kinds, no tiles that can
//     contradict each other, no second copy of the placement editor in
//     Settings, no bracketed placeholder printed to the user;
//   - the four status words, which the previous room could not say because it
//     compared against a value the engine never sends;
//   - the one picker, opened from a target card on either placement page,
//     which is the only writer of `for:`; nothing on ของคุณ places a server;
//   - the sheet itself, which is the only form.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent, within } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import { NAV, navFor } from '../lib/desks'
import { MCP_PRESETS, needsPaste, presetFor, presetConfig } from '../lib/mcpShelf'
import {
  ListMCPServers, ListExternalSkills, ListTools, SaveMCPServer,
  PlacementTargets, SetMCPServerTargets, RemoveMCPServer, ToggleMCPServer, TestMCPServer,
  StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus, ListSubagentProfiles,
  MCPConfigPath, OpenMCPFolder,
  SkillsDir, SkillScanIssues, OpenSkillsFolder, InstallSkillFromZip, InstallSkillFromGitHub, RemoveExternalSkill,
  AgentSkills, CopySkillToAgent, RemoveAgentSkill, OpenAgentSkillsFolder,
} from './mocks/wailsApp'
import { BrowserOpenURL } from './mocks/wailsRuntime'
import { cockpit } from '../lib/stores/cockpit.svelte'

const tool = (over: Record<string, unknown> = {}) => ({
  name: 'read', description: 'อ่านไฟล์', source: 'builtin', category: 'files', ...over,
})
const TARGETS = [
  { id: 'assistant', name: 'assistant', detail: 'โต๊ะผู้ช่วย', kind: 'desk' },
  { id: 'coding', name: 'coding', detail: 'โต๊ะโค้ด', kind: 'desk' },
  { id: 'agent:deepresearch', name: 'deepresearch', detail: 'เอเจนหาข้อมูล', kind: 'agent' },
  { id: 'agent:editor', name: 'editor', detail: 'เอเจนตัดวิดีโอ', kind: 'agent' },
]
const server = (over: Record<string, unknown> = {}) =>
  ({ name: 'firecrawl', disabled: false, status: 'idle', tools: 0, for: [], url: 'https://mcp.firecrawl.dev/v2/mcp', ...over })

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'capability'
  cockpit.capabilityIntent = null
  vi.mocked(ListMCPServers).mockResolvedValue([] as any)
  vi.mocked(ListExternalSkills).mockResolvedValue([] as any)
  vi.mocked(ListSubagentProfiles).mockResolvedValue([] as any)
  vi.mocked(PlacementTargets).mockResolvedValue(TARGETS as any)
  vi.mocked(ListTools).mockResolvedValue([tool(), tool({ name: 'web_fetch', category: 'web' })] as any)
})

const open = async (rows: Record<string, unknown>[] = []) => {
  vi.mocked(ListMCPServers).mockResolvedValue(rows as any)
  const r = render(Capability, { onClose: () => {} })
  await waitFor(() => expect(PlacementTargets).toHaveBeenCalled())
  await waitFor(() => expect(document.querySelector('.office-grid')).toBeTruthy())
  return r
}
const card = (name: string) => screen.getByText(name, { selector: '.chair-name .nm' }).closest('.chair-card')! as HTMLElement
const rail = () => Array.from(document.querySelectorAll<HTMLElement>('.settings-nav-item'))
const railTo = async (label: string) => {
  await fireEvent.click(rail().find((x) => x.textContent?.includes(label))!)
  await waitFor(() => expect(document.querySelector('.office-grid')).toBeTruthy())
}
const activePage = () => document.querySelector('.settings-nav-item.active')?.textContent?.trim()

describe('the room in the column', () => {
  it('sits directly under ผู้ช่วย behind the assistant door', () => {
    const rooms = navFor('assistant')
    expect(rooms[0].id).toBe('assistant')
    expect(rooms[1].id).toBe('capability')
  })
  it('is one row rather than a branch', () => {
    expect(NAV.find((n) => n.id === 'capability')?.kind).toBe('page')
  })
})

describe('one shelf, not two', () => {
  it('every preset can be turned into a saved server', async () => {
    for (const p of MCP_PRESETS) {
      const cfg = await presetConfig(p)
      expect(cfg.name).toBe(p.name)
      expect(Boolean(cfg.url) || Array.isArray(cfg.command)).toBe(true)
    }
  })
  it('splits a header on its first colon, keeping the value prefix', async () => {
    const github = MCP_PRESETS.find((p) => p.name === 'github')!
    const cfg = await presetConfig(github)
    expect(cfg.headers).toEqual({ Authorization: 'Bearer ${connect:github}' })
  })
  it('sends only a preset waiting on a raw token to the form', () => {
    expect(needsPaste(['Authorization: Bearer ${connect:github}'])).toBe(false)
    expect(needsPaste(['Authorization: Bearer ${env:X}'])).toBe(false)
    expect(needsPaste(['Authorization: Bearer'])).toBe(true)
  })
  // The stdio spelling of the same blank: an env line with nothing after its
  // `=`. A line that already carries a value is not a blank, and a program
  // with only those is one click.
  it('sends a program with an env blank to the form, and saves only the filled lines', async () => {
    expect(needsPaste(undefined, ['OAUTHLIB_INSECURE_TRANSPORT=1'])).toBe(false)
    expect(needsPaste(undefined, ['GOOGLE_OAUTH_CLIENT_ID='])).toBe(true)
    const gw = MCP_PRESETS.find((p) => p.name === 'google-workspace')!
    expect(needsPaste(gw.headers, gw.env)).toBe(true)
    expect(presetFor('google-workspace')).toBeUndefined()
    const cfg = await presetConfig(gw)
    expect(cfg.command?.[0]).toBe('uvx')
    expect(cfg.environment).toEqual({ OAUTHLIB_INSECURE_TRANSPORT: '1' })
  })
  it('keeps an oauth preset out of the one-click install path despite its header', () => {
    for (const p of MCP_PRESETS.filter((x) => x.oauth)) expect(presetFor(p.name)).toBeUndefined()
    expect(presetFor('firecrawl')?.name).toBe('firecrawl')
  })
  // The bracketed placeholder used to live INSIDE `why` and was printed to the
  // user. Now it is a field, and an unproven entry has no `why` at all.
  it('says in a field, not in prose, which entries nobody has tried', () => {
    for (const p of MCP_PRESETS) {
      expect(p.why).not.toMatch(/\[รอเจ้าของ/)
      if (!p.proven) expect(p.why).toBe('')
      else expect(p.why.length).toBeGreaterThan(0)
    }
    expect(MCP_PRESETS.some((p) => !p.proven)).toBe(true)
  })
})

describe('what the room must not have any more', () => {
  // A rail of three pages, each one kind of thing, and no tab bar of KINDS:
  // nothing here switches between MCP, skills and tools (the registers still
  // in ตั้งค่า are linked from the foot, not drawn as rows that point away).
  it('is a rail of two headings — four MCP pages, four skill pages — and has no kind tabs', async () => {
    await open()
    expect(rail().map((x) => x.textContent?.trim())).toEqual([
      'MCP server ของคุณ', 'ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด', 'ตั้งค่า MCP สำหรับเอเจนเฉพาะทาง', 'ห้องสมุด MCP',
      'สกิลของคุณ', 'ตั้งค่าสกิลสำหรับเอเจนเฉพาะทาง', 'ห้องสมุดสกิล', 'ปรับสกิลอัตโนมัติ',
    ])
    expect(Array.from(document.querySelectorAll('.settings-nav .settings-group-label')).map((x) => x.textContent?.trim())).toEqual(['MCP', 'สกิล'])
    expect(screen.queryAllByRole('tablist').length).toBe(0) // the sheet's tabs exist only while it is open
    // The registers still in ตั้งค่า are not drawn as rows that point away.
    expect(rail().some((x) => /เครื่องมือในตัว|บัญชี/.test(x.textContent ?? ''))).toBe(false)
    expect(document.querySelector('.cap-tile')).toBeNull()
    expect(document.querySelector('.cap-person')).toBeNull()
  })

  it('opens on the library with nothing connected, and on yours once there is something', async () => {
    await open()
    expect(activePage()).toBe('ห้องสมุด MCP')
    document.body.innerHTML = ''
    await open([server()])
    expect(activePage()).toBe('MCP server ของคุณ')
    expect(card('firecrawl')).toBeTruthy()
  })
  it('never prints a placeholder to the user', async () => {
    await open()
    expect(document.body.textContent).not.toMatch(/\[รอเจ้าของ/)
  })
  // The one register still in ตั้งค่า is announced in the foot of ของคุณ as a
  // door to where it lives today; สกิล is on the rail and gets no door.
  it('names the built-in tools with their count and links that register only', async () => {
    await open([server()])
    expect(screen.getByRole('heading', { level: 2 }).textContent).toMatch(/MCP server/)
    expect(screen.getByText(/เครื่องมือในตัว 2 ชิ้น/)).toBeTruthy()
    expect(screen.getByText('ตั้งค่า › เครื่องมือ')).toBeTruthy()
    expect(screen.queryByText('ตั้งค่า › สกิล')).toBeNull()
  })
})

// ---- สกิล ------------------------------------------------------------------
// Moved whole from ตั้งค่า › สกิล on 13 ก.ย. 2026. The shelf tests below were
// that page's (path from the engine, folder button, unreadable files, zip),
// carried over so nothing the page proved is lost in the move.
const SHELF = [
  { name: 'aetox-slides', description: 'สไลด์', dir: '', bundled: true, before: 'making a deck' },
  { name: 'gridgeist', description: 'grid design', dir: 'C:/Users/x/.aetox/skills/gridgeist' },
]
describe('สกิลของคุณ', () => {
  beforeEach(() => {
    vi.mocked(ListExternalSkills).mockResolvedValue(SHELF as any)
    vi.mocked(SkillScanIssues).mockResolvedValue([] as any)
    vi.mocked(SkillsDir).mockResolvedValue('C:/Users/x/.aetox/skills')
  })
  const openSkills = async () => {
    await open()
    await railTo('สกิลของคุณ')
    await waitFor(() => expect(screen.getByText('gridgeist')).toBeTruthy())
  }

  it('draws the shelf in two bands — yours, then bundled — with the engine\'s own folder path', async () => {
    await openSkills()
    expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('สกิลของคุณ')
    const bands = Array.from(document.querySelectorAll('.ag-band .lab')).map((x) => x.textContent)
    expect(bands).toEqual(['ที่คุณเพิ่ม', 'ติดมากับแอป'])
    expect(screen.getByText(/C:\/Users\/x\/\.aetox\/skills/)).toBeTruthy()
    expect(document.body.textContent).not.toContain('.agents/skills')
    // A bundled card has no folder and no ลบ; a user's has both.
    expect(within(card('aetox-slides')).queryByLabelText('ลบ')).toBeNull()
    expect(within(card('gridgeist')).getByLabelText('ลบ')).toBeTruthy()
    // The moment a skill names for itself (§221) is on its card.
    expect(within(card('aetox-slides')).getByText(/อ่านก่อน: making a deck/)).toBeTruthy()
  })

  it('offers to open the folder, and refresh', async () => {
    await openSkills()
    await fireEvent.click(screen.getAllByText('เปิดโฟลเดอร์')[0])
    expect(vi.mocked(OpenSkillsFolder)).toHaveBeenCalled()
  })

  it('says when a SKILL.md was found but could not be read, and counts it on the rail', async () => {
    vi.mocked(SkillScanIssues).mockResolvedValue(['C:/Users/x/.aetox/skills/broken/SKILL.md: missing description'] as any)
    await openSkills()
    expect(document.querySelector('.cap-issue')?.textContent).toContain('broken/SKILL.md')
    expect(rail().find((x) => x.textContent?.includes('สกิลของคุณ'))?.querySelector('.nav-count')?.textContent).toBe('1')
  })

  it('stays quiet when every file read cleanly', async () => {
    await openSkills()
    expect(document.querySelector('.cap-issue')).toBeNull()
    expect(rail().find((x) => x.textContent?.includes('สกิลของคุณ'))?.querySelector('.nav-count')).toBeNull()
  })

  it('removing a skill asks first and names the folder that will be deleted', async () => {
    await openSkills()
    await fireEvent.click(within(card('gridgeist')).getByLabelText('ลบ'))
    expect(vi.mocked(RemoveExternalSkill)).not.toHaveBeenCalled()
    expect(document.querySelector('.confirm-detail')?.textContent?.trim()).toBe('C:/Users/x/.aetox/skills/gridgeist')
    await fireEvent.click(document.querySelector('.confirm-go')!)
    await waitFor(() => expect(vi.mocked(RemoveExternalSkill)).toHaveBeenCalledWith('gridgeist'))
  })

  it('installs from a picked zip on the install sheet and reports what landed', async () => {
    vi.mocked(InstallSkillFromZip).mockResolvedValue('ติดตั้งแล้ว 1 สกิล (5 ไฟล์): pdf\nลงที่: C:/Users/x/.aetox/skills' as any)
    await openSkills()
    await fireEvent.click(screen.getByText('ติดตั้งสกิล'))
    await fireEvent.click(screen.getByText('เลือกไฟล์ zip…'))
    await waitFor(() => expect(document.querySelector('.skill-result')?.textContent).toContain('5 ไฟล์'))
    // The shelf has to be re-read, or the skill just installed is not on screen.
    expect(vi.mocked(ListExternalSkills).mock.calls.length).toBeGreaterThan(1)
  })

  it('treats a dismissed picker as nothing happening, and a refused archive as an error', async () => {
    vi.mocked(InstallSkillFromZip).mockResolvedValue('' as any)
    await openSkills()
    await fireEvent.click(screen.getByText('ติดตั้งสกิล'))
    await fireEvent.click(screen.getByText('เลือกไฟล์ zip…'))
    await waitFor(() => expect(vi.mocked(InstallSkillFromZip)).toHaveBeenCalled())
    // run() reloads the room after the call; wait for the button to come back.
    await waitFor(() => expect(screen.getByText('เลือกไฟล์ zip…')).toBeTruthy())
    expect(document.querySelector('.skill-result')).toBeNull()
    expect(document.querySelector('.mset-error')).toBeNull()

    vi.mocked(InstallSkillFromZip).mockRejectedValue(new Error('ไฟล์ zip มีเส้นทางที่ออกนอกโฟลเดอร์ติดตั้ง: ../../evil.txt'))
    await fireEvent.click(screen.getByText('เลือกไฟล์ zip…'))
    await waitFor(() => expect(document.querySelector('.mset-error')?.textContent).toContain('evil.txt'))
  })

  it('installs from a GitHub URL typed on the sheet', async () => {
    await openSkills()
    await fireEvent.click(screen.getByText('ติดตั้งสกิล'))
    const input = document.querySelector<HTMLInputElement>('.cap-sheet input.key-input')!
    await fireEvent.input(input, { target: { value: 'https://github.com/x/y' } })
    await fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(vi.mocked(InstallSkillFromGitHub)).toHaveBeenCalledWith('https://github.com/x/y'))
  })
})

describe('ตั้งค่าสกิลสำหรับเอเจนเฉพาะทาง', () => {
  beforeEach(() => {
    vi.mocked(ListExternalSkills).mockResolvedValue(SHELF as any)
    vi.mocked(AgentSkills).mockImplementation(async (name: string) =>
      name === 'deepresearch'
        ? [{ name: 'source-trust', description: 'ใครเชื่อได้', bundled: true }, { name: 'gridgeist', description: 'grid design', bundled: false }]
        : [])
  })
  const openAgents = async () => {
    await open()
    await railTo('ตั้งค่าสกิลสำหรับเอเจนเฉพาะทาง')
    await waitFor(() => expect(vi.mocked(AgentSkills)).toHaveBeenCalledWith('editor'))
  }

  it('is a card per agent showing what its own folder holds, never dimmed for holding nothing', async () => {
    await openAgents()
    await waitFor(() => expect(within(card('deepresearch')).getByText('source-trust')).toBeTruthy())
    expect(within(card('deepresearch')).getByText('gridgeist')).toBeTruthy()
    expect(within(card('editor')).getByText(/ยังไม่มีสกิลของตัวเอง/)).toBeTruthy()
    expect(card('editor').classList.contains('off')).toBe(false)
    // The empty card gets the primary button; the held one does not.
    expect(within(card('editor')).getByRole('button', { name: /เพิ่มสกิลให้ editor/ }).classList.contains('ctrl-primary')).toBe(true)
    expect(within(card('deepresearch')).getByRole('button', { name: /แก้สกิลของ deepresearch/ }).classList.contains('ctrl-primary')).toBe(false)
  })

  it('the sheet ticks a shelf skill INTO the agent\'s folder, and unticks a copy out of it', async () => {
    await openAgents()
    await fireEvent.click(within(card('editor')).getByRole('button', { name: /เพิ่มสกิลให้ editor/ }))
    const sheet = document.querySelector('.cap-sheet')!
    expect(within(sheet as HTMLElement).getByRole('heading', { level: 3 }).textContent).toBe('สกิลของ editor')
    const rows = Array.from(sheet.querySelectorAll<HTMLElement>('.cap-pickrow'))
    expect(rows.map((r) => r.getAttribute('aria-checked'))).toEqual(['false', 'false'])
    await fireEvent.click(rows.find((r) => r.textContent?.includes('gridgeist'))!)
    await waitFor(() => expect(vi.mocked(CopySkillToAgent)).toHaveBeenCalledWith('editor', 'gridgeist'))
    expect(vi.mocked(SetMCPServerTargets)).not.toHaveBeenCalled()
  })

  it('a copy already in the folder is ticked, and the tick removes it; what shipped is listed, not tickable', async () => {
    await openAgents()
    await fireEvent.click(within(card('deepresearch')).getByRole('button', { name: /แก้สกิลของ deepresearch/ }))
    const sheet = document.querySelector('.cap-sheet')! as HTMLElement
    const rows = Array.from(sheet.querySelectorAll<HTMLElement>('.cap-pickrow'))
    const grid = rows.find((r) => r.textContent?.includes('gridgeist'))!
    expect(grid.getAttribute('aria-checked')).toBe('true')
    // source-trust shipped with the profile: shown under its own group, no row.
    expect(within(sheet).getByText('source-trust')).toBeTruthy()
    expect(rows.some((r) => r.textContent?.includes('source-trust'))).toBe(false)
    await fireEvent.click(grid)
    await waitFor(() => expect(vi.mocked(RemoveAgentSkill)).toHaveBeenCalledWith('deepresearch', 'gridgeist'))
    await fireEvent.click(within(sheet).getByText('เปิดโฟลเดอร์สกิล'))
    expect(vi.mocked(OpenAgentSkillsFolder)).toHaveBeenCalledWith('deepresearch')
  })
})

// The agent editor's doors (ตั้งค่า › เอเจนเฉพาะทาง › MCP / สกิล) ask for a
// page AND an agent (cockpit.capabilityIntent): the room opens on that page
// with that agent's sheet already up, instead of on its front page with the
// reader left to find the card (owner, 13 ก.ย. 2026: "กดปุ่มความสามารถแล้ว
// ควรพามาหน้าตั้งค่า MCP สำหรับเอเจนเฉพาะทางสิครับ").
describe('arriving from the agent editor', () => {
  it("opens the MCP placement page on the agent's picker, and consumes the intent", async () => {
    cockpit.capabilityIntent = { page: 'agents', agent: 'editor' }
    await open([{ name: 'context7', command: ['npx'], disabled: false, status: 'connected', tools: 2, for: [] }])
    await waitFor(() => expect(activePage()).toBe('ตั้งค่า MCP สำหรับเอเจนเฉพาะทาง'))
    await waitFor(() => expect(document.getElementById('cap-pick-title')?.textContent).toContain('editor'))
    expect(cockpit.capabilityIntent).toBeNull()
  })

  it("opens the skill page on the agent's sheet", async () => {
    vi.mocked(ListExternalSkills).mockResolvedValue(SHELF as any)
    vi.mocked(AgentSkills).mockResolvedValue([] as any)
    cockpit.capabilityIntent = { page: 'skagents', agent: 'editor' }
    await open()
    await waitFor(() => expect(activePage()).toBe('ตั้งค่าสกิลสำหรับเอเจนเฉพาะทาง'))
    await waitFor(() => expect(document.querySelector('.cap-sheet h3')?.textContent).toBe('สกิลของ editor'))
  })
})

describe('ห้องสมุดสกิล', () => {
  it('offers every curated pack whole, marks a partial one, and keeps the two other doors', async () => {
    vi.mocked(ListExternalSkills).mockResolvedValue([{ name: 'wayfinder', description: '', dir: 'C:/x/wayfinder' }] as any)
    await open()
    await railTo('ห้องสมุดสกิล')
    expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ห้องสมุดสกิล')
    const mp = card('mattpocock/skills')
    expect(within(mp).getByText(/มีอยู่แล้ว 1 จาก/)).toBeTruthy()
    await fireEvent.click(within(mp).getByRole('button', { name: /เพิ่มทั้งชุด/ }))
    await waitFor(() => expect(vi.mocked(InstallSkillFromGitHub)).toHaveBeenCalledWith('https://github.com/mattpocock/skills'))
    expect(screen.getByText('ติดตั้งจาก URL หรือ zip')).toBeTruthy()
    expect(screen.getByText('ให้ผู้ช่วยหาสกิลมาให้')).toBeTruthy()
  })
})

describe('the library', () => {
  // Every preset not yet added is a card, under one of two bands split by
  // the fact the reader acts on: whether it signs in through their account.
  // The sign-in band used to be named "ยังไม่ได้ลองจริง" and was read as "may
  // not work" (owner, 13 ก.ย.); that word is gone.
  it('offers every preset you do not have, instant ones first, sign-in ones under their own band', async () => {
    await open()
    const names = Array.from(document.querySelectorAll('.chair-name .nm')).map((n) => n.textContent)
    for (const p of MCP_PRESETS) expect(names).toContain(p.name)
    const bands = Array.from(document.querySelectorAll('.ag-band .lab')).map((x) => x.textContent)
    expect(bands).toEqual(['ต่อได้ทันที', 'เข้าสู่ระบบด้วยบัญชีของคุณ'])
    expect(document.body.textContent).not.toMatch(/ยังไม่ได้ลองจริง|ลองต่อ|พิสูจน์แล้ว/)
    const first = names.indexOf(MCP_PRESETS.find((p) => !p.oauth)!.name)
    const firstSignIn = names.indexOf(MCP_PRESETS.find((p) => p.oauth)!.name)
    expect(first).toBeLessThan(firstSignIn)
    // A sign-in card is not dimmed: it works, after the login.
    expect(card('figma').classList.contains('off')).toBe(false)
  })
  it('says what a preset costs per message, from a measurement, or says nothing', async () => {
    await open()
    expect(card('exa').querySelector('.cap-cost')?.textContent).toBe('2 เครื่องมือ · 553 โทเคนต่อข้อความ')
    expect(card('semgrep').querySelector('.cap-cost')).toBeNull()
    for (const p of MCP_PRESETS) {
      if (p.toolCount !== undefined) expect(p.measured).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    }
  })

  it('does not offer a server you already have', async () => {
    await open([server({ name: 'exa', status: 'connected', tools: 6, for: ['assistant'] })])
    expect(screen.getAllByText('exa', { selector: '.chair-name .nm' }).length).toBe(1)
    expect(card('exa').querySelector('.cap-st')).toBeTruthy() // the server card, not a shelf card
  })

  it('saves the preset the card button belongs to', async () => {
    await open()
    await fireEvent.click(card('firecrawl').querySelector('.cap-act')!)
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect(vi.mocked(SaveMCPServer).mock.calls[0][1].name).toBe('firecrawl')
  })

  // A program whose env has blanks opens the form with those lines already
  // written, the same door github's pasted token goes through, and says where
  // the values come from. Nothing is saved by the press.
  it('opens the form for a program with env blanks, with the lines written and the hint under them', async () => {
    await open()
    expect(within(card('google-workspace')).getByText('ต้องใช้คีย์')).toBeTruthy()
    await fireEvent.click(card('google-workspace').querySelector('.cap-act')!)
    const dlg = await screen.findByRole('dialog')
    expect(SaveMCPServer).not.toHaveBeenCalled()
    const env = dlg.querySelector<HTMLTextAreaElement>('.mcp-lines')!
    expect(env.value.split('\n')).toEqual(['GOOGLE_OAUTH_CLIENT_ID=', 'GOOGLE_OAUTH_CLIENT_SECRET=', 'USER_GOOGLE_EMAIL=', 'OAUTHLIB_INSECURE_TRANSPORT=1'])
    expect((within(dlg).getByPlaceholderText(/uvx|npx/) as HTMLInputElement).value).toMatch(/^uvx workspace-mcp/)
    // The note: a title naming the server, the hint as numbered steps with
    // the values a person will type set as code, and the foot.
    const note = dlg.querySelector('.cap-keynote')!
    expect(note.querySelector('.cap-keynote-t')?.textContent).toBe('google-workspace ต้องใช้ค่าที่เว้นว่างไว้')
    expect(note.querySelectorAll('.cap-keynote-steps li').length).toBe(5)
    expect(Array.from(note.querySelectorAll('code')).map((c) => c.textContent)).toContain('http://localhost:8000/oauth2callback')
    expect(note.textContent).not.toContain('`')
    expect(note.querySelector('.cap-keynote-f')?.textContent).toContain('ยังไม่มีอะไรถูกบันทึก')
  })

  it('filters by what kind of thing a server reaches', async () => {
    await open()
    await fireEvent.click(screen.getByText('โค้ดและรีโป'))
    expect(screen.queryByText('firecrawl', { selector: '.chair-name .nm' })).toBeNull()
    expect(card('github')).toBeTruthy()
  })

  it('waits for a browser sign-in before saving an oauth preset', async () => {
    await open()
    vi.mocked(MCPSignInStatus).mockResolvedValue({ provider: 'semgrep', signed_in: false } as any)
    vi.mocked(StartMCPSignIn).mockResolvedValue(
      { provider: 'semgrep', kind: 'browser', url: 'https://login.semgrep.dev/oauth2/authorize?x=1' } as any)
    let finishSignIn!: () => void
    vi.mocked(CompleteMCPSignIn).mockReturnValue(
      new Promise<undefined>((resolve) => { finishSignIn = () => resolve(undefined) }))

    await fireEvent.click(card('semgrep').querySelector('.cap-act')!)
    await waitFor(() => expect(BrowserOpenURL).toHaveBeenCalledWith('https://login.semgrep.dev/oauth2/authorize?x=1'))
    expect(screen.getByText('กำลังรอการอนุมัติในเบราว์เซอร์…')).toBeTruthy()
    expect(SaveMCPServer).not.toHaveBeenCalled()

    finishSignIn()
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect(vi.mocked(SaveMCPServer).mock.calls[0][1].name).toBe('semgrep')
  })

  it('skips the browser round trip for an oauth preset already signed in', async () => {
    vi.mocked(MCPSignInStatus).mockResolvedValue({ provider: 'grafana', signed_in: true } as any)
    await open()
    await fireEvent.click(card('grafana').querySelector('.cap-act')!)
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect(StartMCPSignIn).not.toHaveBeenCalled()
    expect(BrowserOpenURL).not.toHaveBeenCalled()
  })

  it('cancels an in-flight oauth sign-in rather than leaving it running', async () => {
    await open()
    vi.mocked(MCPSignInStatus).mockResolvedValue({ provider: 'netlify', signed_in: false } as any)
    vi.mocked(StartMCPSignIn).mockResolvedValue(
      { provider: 'netlify', kind: 'browser', url: 'https://netlify-mcp.netlify.app/oauth-server/auth?x=1' } as any)
    vi.mocked(CompleteMCPSignIn).mockReturnValue(new Promise<undefined>(() => {}))

    const c = card('netlify')
    await fireEvent.click(c.querySelector('.cap-act')!)
    await waitFor(() => expect(within(c).getByText('ยกเลิก')).toBeTruthy())
    await fireEvent.click(within(c).getByText('ยกเลิก'))
    expect(CancelMCPSignIn).toHaveBeenCalledWith('netlify')
    await waitFor(() => expect(c.querySelector('.cap-act')?.textContent?.trim()).toBe('เพิ่ม'))
    expect(SaveMCPServer).not.toHaveBeenCalled()
  })
})

describe('what a server card says', () => {
  // Four words. The previous room compared `status` against 'ok', which the
  // engine never sends (idle / connected / failed), so its green never lit and
  // a server that could not connect looked like one nobody had tried.
  it('says connected, with the tool count, from the value the engine actually sends', async () => {
    await open([server({ status: 'connected', tools: 25, tokens: 3200, for: ['assistant'] })])
    expect(card('firecrawl').querySelector('.cap-st')?.textContent).toContain('ต่อได้')
    expect(card('firecrawl').querySelector('.cap-cost')?.textContent).toBe('25 เครื่องมือ · 3.2k โทเคนต่อข้อความ')
  })
  it('says never connected for idle, and offers ทดสอบ on the card', async () => {
    await open([server({ status: 'idle' })])
    const c = card('firecrawl')
    expect(c.querySelector('.cap-st')?.textContent).toContain('ยังไม่เคยต่อ')
    await fireEvent.click(c.querySelector('.cap-act')!)
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledWith('firecrawl'))
  })
  it('says cannot connect and prints the engine error on the card', async () => {
    await open([server({ status: 'failed', err: '401 Unauthorized: token expired' })])
    const c = card('firecrawl')
    expect(c.querySelector('.cap-st')?.textContent).toContain('ต่อไม่ได้')
    expect(c.textContent).toContain('401 Unauthorized: token expired')
  })
  it('says switched off, and the card button switches it back on', async () => {
    await open([server({ disabled: true, status: 'disabled' })])
    const c = card('firecrawl')
    expect(c.querySelector('.cap-st')?.textContent).toContain('ปิดอยู่')
    await fireEvent.click(c.querySelector('.cap-act')!)
    await waitFor(() => expect(ToggleMCPServer).toHaveBeenCalledWith('firecrawl', false))
  })
  // Readiness only (owner, 13 ก.ย.): who holds a server is the placement
  // pages' question, and nothing on this card says or changes it.
  it('says nothing about who holds it', async () => {
    await open([server({ for: ['agent:deepresearch', 'coding'] })])
    const c = card('firecrawl')
    expect(within(c).queryByTitle('โค้ด')).toBeNull()
    expect(c.textContent).not.toMatch(/deepresearch|เปิดให้ใคร/)
  })
  // An oauth server with no token on this machine cannot be tested before a
  // login, so the login is the card's one action, and the sentence counts it.
  it('offers a sign-in on an oauth server that has none, and tests otherwise', async () => {
    vi.mocked(MCPSignInStatus).mockImplementation(async (name: string) => ({ signed_in: name !== 'canva' }) as any)
    vi.mocked(StartMCPSignIn).mockResolvedValue({ url: 'https://canva/auth' } as any)
    await open([
      server({ name: 'canva', url: 'https://mcp.canva.com/mcp', headers: { Authorization: 'Bearer ${connect:canva}' } }),
      server({ name: 'notion', url: 'https://mcp.notion.com/mcp', headers: { Authorization: 'Bearer ${connect:notion}' } }),
    ])
    await waitFor(() => expect(within(card('canva')).getByText('ยังไม่ได้เข้าสู่ระบบ')).toBeTruthy())
    expect(within(card('notion')).getByText('เข้าสู่ระบบแล้ว')).toBeTruthy()
    expect(document.querySelector('.cap-line')!.textContent).toMatch(/ยังไม่ได้เข้าสู่ระบบ 1: canva/)
    expect(rail()[0].querySelector('.nav-count')?.textContent).toBe('1')
    await fireEvent.click(within(card('canva')).getByText('เข้าสู่ระบบ'))
    await waitFor(() => expect(StartMCPSignIn).toHaveBeenCalledWith('canva', 'https://mcp.canva.com/mcp'))
    expect(BrowserOpenURL).toHaveBeenCalledWith('https://canva/auth')
    await waitFor(() => expect(CompleteMCPSignIn).toHaveBeenCalledWith('canva'))
    expect(vi.mocked(TestMCPServer).mock.calls.map((c: any[]) => c[0])).not.toContain('canva')
  })
})

describe('the probe', () => {
  // Never-connected servers are connected by the room when ของคุณ opens, two
  // at a time, each card saying so, and never twice per visit. A server that
  // wants a login is left alone (it would only 401), so is one switched off.
  it('connects never-tested servers on its own, two at a time, and leaves the sign-in and off ones alone', async () => {
    const pending: Array<() => void> = []
    vi.mocked(TestMCPServer).mockImplementation(() => new Promise<any>((r) => { pending.push(() => r({} as any)) }))
    vi.mocked(MCPSignInStatus).mockImplementation(async (name: string) => ({ signed_in: name !== 'canva' }) as any)
    await open([server(), server({ name: 'exa' }), server({ name: 'deepwiki' }), server({ name: 'canva', url: 'https://mcp.canva.com/mcp', headers: { Authorization: 'Bearer ${connect:canva}' } }), server({ name: 'off', disabled: true, status: 'disabled' })])
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledTimes(2))
    expect(vi.mocked(TestMCPServer).mock.calls.map((c: any[]) => c[0])).toEqual(['firecrawl', 'exa'])
    expect(card('firecrawl').querySelector('.cap-act')?.textContent).toContain('กำลังทดสอบ')
    expect(card('deepwiki').querySelector('.cap-act')?.textContent).not.toContain('กำลังทดสอบ')
    expect(document.querySelector('.cap-line')!.textContent).toMatch(/กำลังทดสอบ 2 ตัว/)
    pending.shift()!()
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledTimes(3))
    expect((vi.mocked(TestMCPServer).mock.calls as any[][])[2][0]).toBe('deepwiki')
    pending.shift()!(); pending.shift()!()
    await waitFor(() => expect(document.querySelectorAll('.cap-act.testing').length).toBe(0))
    expect(TestMCPServer).toHaveBeenCalledTimes(3) // canva (sign-in) and off (switched off) untouched
    vi.mocked(TestMCPServer).mockImplementation(async () => ({} as any))
  })
  it('does not probe again on a second visit to ของคุณ', async () => {
    await open([server()])
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledTimes(1))
    await railTo('ห้องสมุด MCP')
    await railTo('MCP server ของคุณ')
    expect(TestMCPServer).toHaveBeenCalledTimes(1)
  })
})

describe('the sentence above the cards', () => {
  const line = () => document.querySelector('.cap-line')!.textContent!
  // The probe runs on open; these sentences are read once it has come back
  // (the mock engine leaves every server as it was).
  const settled = () => waitFor(() => expect(document.querySelectorAll('.cap-act.testing').length).toBe(0))
  // One sentence, and only the clauses that are true. The four tiles this
  // replaced said "ต่ออยู่ 5 · 0 เครื่องมือ" and "ต่อได้ 0 · ต่อไม่ได้ 0 · ทุกตัวปกติ"
  // on a machine where nothing had ever been tested.
  it('says none tested yet, not all fine, when nothing has ever connected', async () => {
    await open([server({ for: ['coding'] }), server({ name: 'exa', for: ['agent:deepresearch'] })])
    await settled()
    expect(line()).toMatch(/^2 ตัว/)
    expect(line()).toMatch(/ยังไม่เคยทดสอบสักตัว/)
    expect(line()).not.toMatch(/ต่อได้ทุกตัว/)
    expect(line()).not.toMatch(/ผู้ช่วย|ถือ/) // placement is the next pages' sentence
  })
  it('names what failed, and counts the rest', async () => {
    await open([
      server({ status: 'connected', tools: 25, tokens: 3200, for: ['assistant'] }),
      server({ name: 'exa', status: 'failed', err: '401', for: [] }),
      server({ name: 'kinocut', url: '', command: ['kino'], for: ['agent:editor'] }),
    ])
    await settled()
    expect(line()).toMatch(/3 ตัว/)
    expect(line()).toMatch(/ยังไม่เคยทดสอบ 1/)
    expect(line()).toMatch(/ต่อไม่ได้ 1: exa/)
    expect(rail()[0].querySelector('.nav-count')?.textContent).toBe('1')
  })
  it('says all connected only when every live server is', async () => {
    await open([server({ status: 'connected', tools: 2, for: ['assistant'] }), server({ name: 'exa', disabled: true, status: 'disabled', for: [] })])
    expect(line()).toMatch(/ต่อได้ทุกตัว/)
    expect(line()).toMatch(/ปิดอยู่ 1/)
  })
  it('tests every live server from one button, naming each as it goes', async () => {
    await open([server(), server({ name: 'exa' }), server({ name: 'off', disabled: true, status: 'disabled' })])
    await waitFor(() => expect(document.querySelectorAll('.cap-act.testing').length).toBe(0))
    vi.mocked(TestMCPServer).mockClear()
    await fireEvent.click(screen.getByText('ทดสอบทั้งหมด'))
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledTimes(2))
    expect(vi.mocked(TestMCPServer).mock.calls.map((c: any[]) => c[0])).toEqual(['firecrawl', 'exa'])
  })
  it('says there is nothing on ของคุณ, with a door to the library', async () => {
    await open([])
    await railTo('MCP server ของคุณ')
    expect(screen.getByText(/ยังไม่ได้ต่อ MCP server ไหนเลย/)).toBeTruthy()
    await fireEvent.click(within(document.querySelector('.chair-card.empty') as HTMLElement).getByText('ห้องสมุด MCP'))
    expect(card('exa')).toBeTruthy()
  })
})

describe('placement: two pages, one picker', () => {
  const TARGETS_WITH_OFFICE = [...TARGETS, { id: 'specialized', name: 'specialized', detail: 'โต๊ะออฟฟิศ', kind: 'desk' }]
  // Open the picker for a target from its card on the page it lives on.
  const openPicker = async (rows: Record<string, unknown>[], who = 'ผู้ช่วย', profiles: Record<string, unknown>[] = []) => {
    vi.mocked(PlacementTargets).mockResolvedValue(TARGETS_WITH_OFFICE as any)
    vi.mocked(ListSubagentProfiles).mockResolvedValue(profiles as any)
    await open(rows)
    await railTo(['ผู้ช่วย', 'โค้ด'].includes(who) ? 'ตั้งค่า MCP ฝั่งผู้ช่วย' : 'เอเจนเฉพาะ')
    await fireEvent.click(card(who).querySelector('.cap-act')!)
    const dlg = await screen.findByRole('dialog')
    expect(within(dlg).getByText(`ให้ ${who} ใช้ตัวไหน`)).toBeTruthy()
    return dlg
  }

  // The grid that stood here for an afternoon is gone (header note). What the
  // room must NOT have again is a switchboard on the page itself: a placement
  // page is cards and one button each, every tick is on the picker.
  it('ตั้งค่า MCP ฝั่งผู้ช่วย is a card per side, in the sidebar words, with no third desk and no switch on the page', async () => {
    vi.mocked(PlacementTargets).mockResolvedValue(TARGETS_WITH_OFFICE as any)
    await open([server({ status: 'connected', tools: 25, tokens: 3200, for: ['assistant', 'agent:editor'] }), server({ name: 'exa', for: [] }), server({ name: 'kinocut', url: '', command: ['kino'], for: ['coding'] })])
    await railTo('ตั้งค่า MCP ฝั่งผู้ช่วย')
    expect(document.querySelector('table')).toBeNull()
    expect(screen.queryAllByRole('switch').length).toBe(0)
    expect(Array.from(document.querySelectorAll('.cap-target .chair-name .nm')).map((n) => n.textContent)).toEqual(['ผู้ช่วย', 'โค้ด'])
    // Each side wears the memory page's identity for the desk, and its chips.
    const grid = document.querySelector('.cap-tiergrid')!
    expect(grid.querySelector('.mem-scope-ic.mem-tone-assistant')).toBeTruthy()
    expect(grid.querySelector('.mem-scope-ic.mem-tone-desk')).toBeTruthy()
    expect(within(card('ผู้ช่วย')).getByText('เฉพาะแชทกับผู้ช่วย')).toBeTruthy()
    expect(within(card('ผู้ช่วย')).getByText('firecrawl')).toBeTruthy()
    expect(within(card('ผู้ช่วย')).getByText('รวม 25 เครื่องมือ · 3.2k โทเคนต่อข้อความ')).toBeTruthy()
    expect(within(card('โค้ด')).getByText('kinocut')).toBeTruthy()
    // What nobody holds is said on this page, since this is where it is fixed.
    expect(screen.getByText(/ยังไม่มีใครถือ 1: exa/)).toBeTruthy()
    expect(SetMCPServerTargets).not.toHaveBeenCalled()
  })

  // A chat with no desk set carries every server whatever `for:` says. Reading
  // `for:` alone would tell the owner "cannot reach anything" while the
  // session held 44 notion tools.
  it('says the session still holds MCP tools when no side is ticked but the registry has them', async () => {
    vi.mocked(ListTools).mockResolvedValue([tool(), tool({ name: 'notion_search', source: 'mcp', category: '' })] as any)
    await open([server({ for: ['agent:deepresearch'] })])
    await railTo('ตั้งค่า MCP ฝั่งผู้ช่วย')
    expect(screen.getByText(/ยังถือเครื่องมือ MCP อยู่/)).toBeTruthy()
  })

  // The agents' page: a card per teammate saying what it is for, what it
  // holds, and what its own file says it needs and has not got. The rail
  // counts the teammates that cannot work yet.
  it('ตั้งค่า MCP สำหรับเอเจนเฉพาะทาง is a card per agent with its needs, and its button opens the picker', async () => {
    vi.mocked(ListSubagentProfiles).mockResolvedValue(
      [{ name: 'deepresearch', description: 'เอเจนหาข้อมูลเชิงลึก — ไล่หลายแหล่ง', needs: ['mcp:firecrawl'] }, { name: 'editor', description: 'เอเจนตัดต่อวิดีโอ — ดูฟุตเทจ', needs: ['mcp:kinocut'] }] as any)
    await open([server({ status: 'connected', tools: 25, tokens: 3200, for: ['assistant', 'agent:editor'] }), server({ name: 'kinocut', url: '', command: ['kino'], for: [] })])
    expect(rail().find((x) => x.textContent?.includes('เอเจนเฉพาะ'))?.querySelector('.nav-count')?.textContent).toBe('2')
    await railTo('เอเจนเฉพาะ')
    expect(screen.queryAllByRole('switch').length).toBe(0)
    const dr = card('deepresearch')
    expect(within(dr).getByText('หาข้อมูลเชิงลึก')).toBeTruthy() // the profile's role, before its dash, without "เอเจน"
    expect(within(dr).getByText('! firecrawl')).toBeTruthy()
    const ed = card('editor')
    expect(within(ed).getByText('firecrawl')).toBeTruthy()
    expect(within(ed).getByText('! kinocut')).toBeTruthy()
    expect(within(ed).getByText('รวม 25 เครื่องมือ · 3.2k โทเคนต่อข้อความ')).toBeTruthy()
    // The button on the card that needs something is the primary one.
    expect(dr.querySelector('.cap-act')?.classList.contains('ctrl-primary')).toBe(true)
    await fireEvent.click(dr.querySelector('.cap-act')!)
    const dlg = await screen.findByRole('dialog')
    expect(within(dlg).getByText('ให้ deepresearch ใช้ตัวไหน')).toBeTruthy()
    const chip = within(dlg).getByRole('switch', { name: /^firecrawl\b/ })
    expect(chip.classList.contains('need')).toBe(true)
    expect(chip.getAttribute('aria-checked')).toBe('false')
    await fireEvent.click(chip)
    // The same writer as the sides' picker: the whole list, through the register's call.
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledWith('firecrawl', ['assistant', 'agent:editor', 'agent:deepresearch']))
    await fireEvent.keyDown(dlg, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  // Every row says what that server adds per message, and the card totals it.
  // Before a server has connected, the library's measurement stands in,
  // marked ~, and the note says so; a measured number carries no mark.
  it('shows what each server adds per message on the picker rows, ~ for the library estimate', async () => {
    const dlg = await openPicker([server({ status: 'connected', tools: 3, tokens: 3200, for: ['assistant'] }), server({ name: 'exa', for: [] })])
    const fc = within(dlg).getByRole('switch', { name: /^firecrawl\b/ })
    expect(fc.querySelector('.cost')?.textContent).toMatch(/^3\.2k/)
    const exa = within(dlg).getByRole('switch', { name: /^exa\b/ })
    expect(exa.querySelector('.cost')?.textContent).toMatch(/^~553/) // mcpShelf: exa measured 553 on 2026-09-12
    expect(within(dlg).getByText(/ผู้ช่วย แบก รวม 3 เครื่องมือ · 3\.2k โทเคนต่อข้อความ/)).toBeTruthy()
    expect(within(dlg).getByText(/ตัวเลขที่มี ~ ประมาณจากห้องสมุด/)).toBeTruthy()
  })

  it('writes the whole new list through the register own call, not a per-target one', async () => {
    const dlg = await openPicker([server({ for: ['agent:deepresearch'] })])
    const chip = within(dlg).getByRole('switch', { name: /^firecrawl\b/ })
    expect(chip.getAttribute('aria-checked')).toBe('false')
    await fireEvent.click(chip)
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledWith('firecrawl', ['agent:deepresearch', 'assistant']))
    expect(vi.mocked(ListMCPServers).mock.calls.length).toBeGreaterThan(1)
  })

  it('keeps the office desk in the stored list without ever drawing it', async () => {
    const dlg = await openPicker([server({ for: ['specialized'] })], 'โค้ด')
    await fireEvent.click(within(dlg).getByRole('switch', { name: /^firecrawl\b/ }))
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledWith('firecrawl', ['specialized', 'coding']))
    expect(screen.queryByText('specialized', { selector: '.chair-name .nm' })).toBeNull()
  })

  it('ticks and clears every live server for one target from its own link', async () => {
    const dlg = await openPicker([server({ for: [] }), server({ name: 'exa', for: ['assistant'] }), server({ name: 'off', disabled: true, status: 'disabled', for: [] })])
    await fireEvent.click(within(dlg).getByText('เอาทั้งหมด'))
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledTimes(1))
    expect(SetMCPServerTargets).toHaveBeenCalledWith('firecrawl', ['assistant'])
  })

  it('refuses to pretend a switched-off server can be handed to anyone', async () => {
    const dlg = await openPicker([server({ disabled: true, status: 'disabled' })])
    const chip = within(dlg).getByRole('switch', { name: /^firecrawl\b/ })
    expect((chip as HTMLButtonElement).disabled).toBe(true)
    await fireEvent.click(chip)
    expect(SetMCPServerTargets).not.toHaveBeenCalled()
  })

  // "หลังจากเพิ่ม MCP ควรจะมีแจ้งเตือนว่าไปตั้งค่า MCP ก่อน" (owner, 12 ก.ย.):
  // a server that was just added is one nobody has decided about. ของคุณ
  // says so and names the two placement pages; it does not open a picker,
  // because this page is readiness only (13 ก.ย.).
  it('lands on ของคุณ with a notice naming the two placement pages when a server is added', async () => {
    await open([])
    vi.mocked(ListMCPServers).mockResolvedValue([server({ name: 'firecrawl', for: ['assistant'] })] as any)
    await fireEvent.click(card('firecrawl').querySelector('.cap-act')!)
    await waitFor(() => expect(activePage()).toBe('MCP server ของคุณ'))
    const notice = screen.getByText(/เพิ่ม firecrawl แล้ว/).closest('.cap-notice')! as HTMLElement
    expect(screen.queryByRole('dialog')).toBeNull()
    await fireEvent.click(within(notice).getByText('ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด'))
    expect(activePage()).toBe('ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด')
    expect(card('ผู้ช่วย')).toBeTruthy()
  })
})

describe('the sheet, the one form', () => {
  const openSheet = async (rows: Record<string, unknown>[], tab: 'การเชื่อมต่อ' | 'เครื่องมือ' = 'การเชื่อมต่อ') => {
    await open(rows)
    await fireEvent.click(within(card(rows[0].name as string)).getByLabelText(`ตั้งค่า ${rows[0].name}`))
    const dlg = await screen.findByRole('dialog')
    await fireEvent.click(within(dlg).getByRole('tab', { name: new RegExp('^' + tab) }))
    return dlg
  }

  it('opens from the gear with the server own fields, and saves under its original name', async () => {
    const dlg = await openSheet([server({ headers: { Authorization: 'Bearer ${connect:x}' }, timeoutMs: 30000 })])
    const url = within(dlg).getByPlaceholderText(/https/) as HTMLInputElement
    expect(url.value).toBe('https://mcp.firecrawl.dev/v2/mcp')
    await fireEvent.input(url, { target: { value: 'https://mcp.firecrawl.dev/v3/mcp' } })
    await fireEvent.click(within(dlg).getByText('บันทึก'))
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    const [original, cfg] = vi.mocked(SaveMCPServer).mock.calls[0] as any
    expect(original).toBe('firecrawl')
    expect(cfg.url).toBe('https://mcp.firecrawl.dev/v3/mcp')
    expect(cfg.timeoutMs).toBe(30000)
    expect(cfg.headers).toEqual({ Authorization: 'Bearer ${connect:x}' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('refuses to save a header left as a scheme with no credential', async () => {
    const dlg = await openSheet([server()])
    await fireEvent.input(dlg.querySelector('.mcp-lines')!, { target: { value: 'Authorization: Bearer' } })
    await fireEvent.click(within(dlg).getByText('บันทึก'))
    await waitFor(() => expect(dlg.querySelector('.mset-error')?.textContent).toContain('Authorization'))
    expect(SaveMCPServer).not.toHaveBeenCalled()
  })

  it('adds a new server from the section button, with nothing written until saved', async () => {
    await open([server({ name: 'exa' })])
    await fireEvent.click(screen.getByText('เพิ่มเซิร์ฟเวอร์', { selector: '.sec-head button' }))
    const dlg = await screen.findByRole('dialog')
    expect(SaveMCPServer).not.toHaveBeenCalled()
    expect(within(dlg).getByRole('tab', { name: 'การเชื่อมต่อ' }).getAttribute('aria-selected')).toBe('true')
    expect(within(dlg).getAllByRole('tab').map((x) => x.textContent?.trim())).toEqual(['การเชื่อมต่อ', 'เครื่องมือ'])
    expect((within(dlg).getByText('เพิ่ม', { selector: '.ctrl-primary' }) as HTMLButtonElement).disabled).toBe(true)
    await fireEvent.input(within(dlg).getByPlaceholderText('ชื่อ เช่น context7'), { target: { value: 'mine' } })
    await fireEvent.input(within(dlg).getByPlaceholderText(/https/), { target: { value: 'https://x.test/mcp' } })
    vi.mocked(ListMCPServers).mockResolvedValue([server({ name: 'mine', url: 'https://x.test/mcp', for: [] })] as any)
    await fireEvent.click(within(dlg).getByText('เพิ่ม', { selector: '.ctrl-primary' }))
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalledWith('', expect.objectContaining({ name: 'mine', url: 'https://x.test/mcp' })))
    // And the sheet stays, now on the saved server, asking who gets it.
    await waitFor(() => expect(screen.getByText(/เพิ่ม mine แล้ว/)).toBeTruthy())
  })

  it('deletes a server from the sheet, behind the confirm, and not before', async () => {
    const dlg = await openSheet([server()])
    await fireEvent.click(within(dlg).getByText('ลบ'))
    expect(RemoveMCPServer).not.toHaveBeenCalled()
    expect(document.querySelector('.confirm-detail')?.textContent?.trim()).toBe('firecrawl')
    await fireEvent.click(document.querySelector('.confirm-go')!)
    await waitFor(() => expect(RemoveMCPServer).toHaveBeenCalledWith('firecrawl'))
  })

  it('pauses and tests from the sheet too', async () => {
    const dlg = await openSheet([server({ status: 'connected', tools: 3 })])
    await fireEvent.click(within(dlg).getByText('ปิดชั่วคราว'))
    await waitFor(() => expect(ToggleMCPServer).toHaveBeenCalledWith('firecrawl', true))
    await fireEvent.click(within(dlg).getByText('ทดสอบ'))
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledWith('firecrawl'))
  })

  // The allowlist is the one field that is destructive when it round-trips
  // wrong: shown blank and saved blank, it silently widens the server back out
  // to everything it offers. Both directions are pinned.
  // The allowlist is a list of checkboxes now, not a box of names. Before the
  // server has connected this session the stored names are the rows, all kept;
  // unticking one drops it from what is written back.
  it('lists the stored allowlist as ticked rows and writes back what stays ticked', async () => {
    const dlg = await openSheet([server({ allowed: ['resolve-library-id', 'get-library-docs'] })], 'เครื่องมือ')
    const boxes = within(dlg).getAllByRole('checkbox') as HTMLInputElement[]
    expect(boxes.map((b) => b.checked)).toEqual([true, true])
    expect(within(dlg).getByText(/เอา 2 จาก 2 ตัว/)).toBeTruthy()
    await fireEvent.click(boxes[1])
    await fireEvent.click(within(dlg).getByText('บันทึก'))
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect((vi.mocked(SaveMCPServer).mock.calls.at(-1) as any)[1].tools).toEqual(['resolve-library-id'])
  })

  // After a connect the engine reports the whole list with a cost per tool,
  // which is what a person picks from; "keep all" writes an empty list, the
  // engine's own word for everything.
  it('lists what the server offered, with tokens per tool, and keeps all as an empty list', async () => {
    const dlg = await openSheet([server({ status: 'connected', tools: 3, tokens: 900, toolList: [
      { name: 'firecrawl_scrape', tokens: 400 }, { name: 'firecrawl_search', tokens: 300 }, { name: 'firecrawl_map', tokens: 200 },
    ] })], 'เครื่องมือ')
    expect(within(dlg).getByText('~400')).toBeTruthy()
    expect(within(dlg).getByText(/เอา 3 จาก 3 ตัว · 900 โทเคนต่อข้อความ/)).toBeTruthy()
    await fireEvent.click(within(dlg).getAllByRole('checkbox')[2])
    expect(within(dlg).getByText(/เอา 2 จาก 3 ตัว · 700 โทเคนต่อข้อความ/)).toBeTruthy()
    await fireEvent.click(within(dlg).getByText('เอาทั้งหมด'))
    await fireEvent.click(within(dlg).getByText('บันทึก'))
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect((vi.mocked(SaveMCPServer).mock.calls.at(-1) as any)[1].tools).toEqual([])
  })

  // The list lives on the server. Opening the tab on a server that has never
  // connected starts the connect itself, names what it is doing, and shows
  // the list when it lands (owner, 13 ก.ย., on an empty tab: "ทำไมว่างเปล่า").
  it('connects for the tool list when the tab is opened on a never-connected server', async () => {
    let done!: () => void
    // The probe's own connect is done and came back empty (the engine kept
    // idle); the tab starts one of its own.
    vi.mocked(TestMCPServer).mockImplementation(() => new Promise<any>((r) => { done = () => r({} as any) }))
    const dlg = await openSheet([server()], 'เครื่องมือ')
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledWith('firecrawl'))
    expect(within(dlg).getByText(/กำลังต่อ firecrawl เพื่อขอรายชื่อเครื่องมือ/)).toBeTruthy()
    expect(within(dlg).queryAllByRole('checkbox').length).toBe(0)
    vi.mocked(ListMCPServers).mockResolvedValue([server({ status: 'connected', tools: 1, toolList: [{ name: 'scrape', tokens: 900 }] })] as any)
    done()
    await waitFor(() => expect(within(dlg).getAllByRole('checkbox').length).toBe(1))
    vi.mocked(TestMCPServer).mockImplementation(async () => ({} as any))
  })
  it('does not connect on its own for a server that wants a sign-in first, and offers the sign-in instead', async () => {
    vi.mocked(MCPSignInStatus).mockResolvedValue({ signed_in: false } as any)
    const dlg = await openSheet([server({ name: 'canva', url: 'https://mcp.canva.com/mcp', headers: { Authorization: 'Bearer ${connect:canva}' } })], 'เครื่องมือ')
    expect(TestMCPServer).not.toHaveBeenCalled()
    expect(within(dlg).getByText(/canva ต้องเข้าสู่ระบบก่อน/)).toBeTruthy()
    // One sign-in button, in the status row; the tab does not repeat it.
    expect(within(dlg).getAllByRole('button', { name: 'เข้าสู่ระบบ' }).length).toBe(1)
  })
  it('says why the tab is empty when the connect failed, and leaves the error and retry to the status row', async () => {
    const dlg = await openSheet([server({ status: 'failed', err: '401 Unauthorized' })], 'เครื่องมือ')
    expect(TestMCPServer).not.toHaveBeenCalled()
    expect(within(dlg).getByText(/ต่อ firecrawl ไม่ได้ เลยยังไม่มีรายชื่อ/)).toBeTruthy()
    expect(within(dlg).getAllByText('401 Unauthorized').length).toBe(1)
  })

  // ขั้นสูง is a fold: closed for a server with nothing in it, open by itself
  // for one that carries a value, so a stored setting is never hidden.
  it('folds ขั้นสูง away unless the server already carries a value there', async () => {
    const dlg = await openSheet([server()])
    expect(within(dlg).queryByPlaceholderText(/รันจากโฟลเดอร์/)).toBeNull()
    await fireEvent.click(within(dlg).getByRole('button', { name: /ขั้นสูง/ }))
    expect(within(dlg).getByPlaceholderText(/รันจากโฟลเดอร์/)).toBeTruthy()
    document.body.innerHTML = ''
    const dlg2 = await openSheet([server({ url: '', command: ['kino'], timeoutMs: 30000 })])
    expect(within(dlg2).getByPlaceholderText(/รันจากโฟลเดอร์/)).toBeTruthy()
  })

  it('round-trips the working directory and timeout of a local server', async () => {
    const dlg = await openSheet([{ name: 'local', command: ['node', 'server.js'], cwd: 'D:/work', timeoutMs: 45000, disabled: false, status: 'connected', tools: 2, for: [] }])
    const inputs = Array.from(dlg.querySelectorAll('.pp-field input')) as HTMLInputElement[]
    expect(inputs.map((i) => i.value)).toEqual(['D:/work', '45000'])
    await fireEvent.click(within(dlg).getByText('บันทึก'))
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    const saved = (vi.mocked(SaveMCPServer).mock.calls.at(-1) as any)[1]
    expect(saved.cwd).toBe('D:/work')
    expect(saved.timeoutMs).toBe(45000)
    expect(saved.command).toEqual(['node', 'server.js'])
  })

  it('cancelling the confirm keeps the server, and Escape cancels with focus on Cancel', async () => {
    const dlg = await openSheet([server()])
    await fireEvent.click(within(dlg).getByText('ลบ'))
    expect(document.activeElement).toBe(document.querySelector('.confirm-cancel'))
    await fireEvent.click(document.querySelector('.confirm-cancel')!)
    expect(document.querySelector('.confirm-overlay')).toBeNull()
    await fireEvent.click(within(dlg).getByText('ลบ'))
    await fireEvent.keyDown(document.querySelector('.confirm-overlay')!, { key: 'Escape' })
    await waitFor(() => expect(document.querySelector('.confirm-overlay')).toBeNull())
    expect(RemoveMCPServer).not.toHaveBeenCalled()
  })

  // Where the servers are persisted is the engine's answer, printed so the
  // file is findable for backup and inspection, with the folder one click away.
  it('shows the file the servers are persisted to, and opens its folder', async () => {
    vi.mocked(MCPConfigPath).mockResolvedValue('C:/Users/x/AppData/Roaming/aetox/mcp-servers.json' as any)
    await open([server()])
    expect(screen.getByText('C:/Users/x/AppData/Roaming/aetox/mcp-servers.json')).toBeTruthy()
    await fireEvent.click(screen.getByText('เปิดโฟลเดอร์'))
    expect(OpenMCPFolder).toHaveBeenCalled()
  })

  it('closes on Escape without writing', async () => {
    const dlg = await openSheet([server()])
    await fireEvent.keyDown(dlg, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(SaveMCPServer).not.toHaveBeenCalled()
  })
})
