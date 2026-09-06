// ห้องความสามารถ. The room exists to announce, not to organise — a person using
// Aetox had no way to learn it speaks MCP at all, because the register sat three
// levels inside Settings and the word appeared nowhere a reader would pass.
//
// So what is pinned here is the announcing, which is the part that quietly stops
// working: the room opening on the ห้องสมุด rather than on your own empty list,
// that library being the real list rather than a second copy of it, and the two
// rooms this one deliberately does not swallow.
//
// Since 4 ก.ย. 2026 the ของคุณ half also EDITS one field — `for:`, who carries a
// server — so the last block pins both halves of that: that the panel is the
// register's own and writes through the register's own call, and that nothing
// ELSE followed it in.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import { NAV, navFor } from '../lib/desks'
import { MCP_PRESETS, needsPaste, presetFor, presetConfig } from '../lib/mcpShelf'
import { SKILL_PRESETS } from '../lib/skillShelf'
import {
  ListMCPServers, ListExternalSkills, ListTools, SaveMCPServer,
  PlacementTargets, SetMCPServerTargets, RemoveMCPServer, ToggleMCPServer,
  StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus,
  InstallSkillFromGitHub,
} from './mocks/wailsApp'
import { BrowserOpenURL } from './mocks/wailsRuntime'
import { cockpit } from '../lib/stores/cockpit.svelte'

const tool = (over: Record<string, unknown> = {}) => ({
  name: 'read', description: 'อ่านไฟล์', source: 'builtin', category: 'files', ...over,
})

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'capability'
  vi.mocked(ListMCPServers).mockResolvedValue([] as any)
  vi.mocked(ListExternalSkills).mockResolvedValue([] as any)
  vi.mocked(ListTools).mockResolvedValue([tool(), tool({ name: 'web_fetch', category: 'web' })] as any)
})

describe('the room in the column', () => {
  // The position is the feature. Last in the list is the right order for a
  // column being organised and the wrong one for a column that has to announce:
  // slot two is the most-seen row after the room the app opens on (owner,
  // 2026-09-03, reversing my own first placement).
  it('sits directly under ผู้ช่วย behind the assistant door', () => {
    const rooms = navFor('assistant')
    expect(rooms[0].id).toBe('assistant')
    expect(rooms[1].id).toBe('capability')
  })

  // A `page`, not a group with children. A submenu that has to be opened
  // announces nothing to the person who does not already know it is there —
  // which is the only person this room was built for.
  it('is one row rather than a branch', () => {
    const entry = NAV.find((n) => n.id === 'capability')
    expect(entry?.kind).toBe('page')
  })
})

describe('one shelf, not two', () => {
  // The list moved out of Settings.svelte so that the room and the register read
  // the same one. Two copies of a preset table go stale on one of them, and the
  // stale one is always the copy nobody opened.
  it('every preset can be turned into a saved server', async () => {
    for (const p of MCP_PRESETS) {
      const cfg = await presetConfig(p)
      expect(cfg.name).toBe(p.name)
      // An http preset carries its URL; a stdio one carries a command. Never
      // neither — that is an entry that could not connect if it were saved.
      expect(Boolean(cfg.url) || Array.isArray(cfg.command)).toBe(true)
    }
  })

  // The header-splitting line existed twice before this file did (addPreset and
  // installNeeded), which is exactly the kind of thing that gets fixed in one
  // copy. github's header carries the value's prefix after the colon and must
  // survive the split whole.
  it('splits a header on its first colon, keeping the value prefix', async () => {
    const github = MCP_PRESETS.find((p) => p.name === 'github')!
    const cfg = await presetConfig(github)
    expect(cfg.headers).toEqual({ Authorization: 'Bearer ${connect:github}' })
  })

  // `presetFor` is the one-click path an agent's card uses, and `needsPaste` is
  // what decides who gets it: a header still waiting for a raw token cannot be
  // finished in one press, so it keeps the door to the form instead.
  //
  // Every non-oauth preset on the shelf clears that bar, and github is the one
  // worth pinning — it carries a header and is STILL one click, because
  // `${connect:github}` resolves at connect time from a credential the app
  // already holds. Reading "has a header" as "needs a paste" would send the user
  // to a form to type something Aetox already has.
  it('sends only a preset waiting on a raw token to the form', () => {
    expect(needsPaste(['Authorization: Bearer ${connect:github}'])).toBe(false)
    expect(needsPaste(['Authorization: Bearer ${env:FIRECRAWL_API_KEY}'])).toBe(false)
    expect(needsPaste(['X-Api-Key'])).toBe(true)

    for (const p of MCP_PRESETS.filter((p) => !p.oauth)) {
      expect(presetFor(p.name)?.name, `${p.name} should be a one-click install`).toBe(p.name)
    }
  })

  // An oauth preset's header (`${connect:name}`) matches the exact same
  // pattern github's does, so needsPaste alone reads it as one-click too —
  // the `oauth` flag is the only thing telling presetFor these still need a
  // browser round trip before SaveMCPServer would connect to anything.
  it('keeps an oauth preset out of the one-click install path despite its header', () => {
    const oauthPresets = MCP_PRESETS.filter((p) => p.oauth)
    expect(oauthPresets.length).toBeGreaterThan(0)
    for (const p of oauthPresets) {
      expect(needsPaste(p.headers), `${p.name}'s header syntax`).toBe(false)
      expect(presetFor(p.name), `${p.name} must not be offered as one-click`).toBeUndefined()
    }
  })
})

describe('what the room says when you walk in', () => {
  // Point 2 of the room's whole design: it opens on the servers you do NOT
  // have. A register opening on your own empty list can only ever confirm what
  // you already knew, which is how the old page failed.
  it('opens on the library, showing every preset, with nothing connected', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(ListMCPServers).toHaveBeenCalled())
    for (const p of MCP_PRESETS) {
      expect(screen.getByText(p.name)).toBeTruthy()
    }
  })

  // Point 1: the announcement is a sentence, not a tab label. Someone who has
  // never heard of either word has to be able to read both without clicking.
  it('names MCP and สกิล in prose above the tabs', async () => {
    render(Capability, { onClose: () => {} })
    const lede = await screen.findByText(/MCP server/)
    expect(lede.textContent).toMatch(/สกิล/)
    // The count is every tool it can actually run, from all three sources —
    // which is why the page prints one number and not three.
    expect(lede.textContent).toMatch(/2/)
  })

  // A preset already in the config is not offered again. Getting this wrong
  // means a shelf that invites you to install what you have, which reads as the
  // page not knowing your own machine.
  it('marks a server you already have instead of offering it', async () => {
    vi.mocked(ListMCPServers).mockResolvedValue(
      [{ name: 'exa', disabled: false, status: 'ok', tools: 6, for: ['assistant'] }] as any)
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(screen.getAllByText('เพิ่มแล้ว').length).toBe(1))
  })

  it('saves the preset the shelf button belongs to', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(ListMCPServers).toHaveBeenCalled())
    // firecrawl needs no key, so its button completes here rather than sending
    // the user to the register's form.
    const card = screen.getByText('firecrawl').closest('article')!
    await fireEvent.click(card.querySelector('button')!)
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect(vi.mocked(SaveMCPServer).mock.calls[0][1].name).toBe('firecrawl')
  })

  // The third button state: an oauth preset's header reads exactly like a
  // one-click preset's (`${connect:semgrep}`), so the room has to actually
  // walk through a browser sign-in rather than trusting the header syntax —
  // this is the test that would have caught SaveMCPServer firing with no
  // credential behind that header yet.
  it('waits for a browser sign-in before saving an oauth preset', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(ListMCPServers).toHaveBeenCalled())

    vi.mocked(MCPSignInStatus).mockResolvedValue({ provider: 'semgrep', signed_in: false } as any)
    vi.mocked(StartMCPSignIn).mockResolvedValue(
      { provider: 'semgrep', kind: 'browser', url: 'https://login.semgrep.dev/oauth2/authorize?x=1' } as any)
    // Held open deliberately: the real CompleteMCPSignIn blocks until the
    // redirect lands, and the assertion below needs a moment where the sign-in
    // is genuinely still in flight to catch a save that jumped the gun.
    let finishSignIn!: () => void
    vi.mocked(CompleteMCPSignIn).mockReturnValue(
      new Promise<undefined>((resolve) => { finishSignIn = () => resolve(undefined) }))

    const card = screen.getByText('semgrep').closest('article')!
    await fireEvent.click(card.querySelector('button')!)

    await waitFor(() => expect(BrowserOpenURL).toHaveBeenCalledWith('https://login.semgrep.dev/oauth2/authorize?x=1'))
    expect(screen.getByText('กำลังรอการอนุมัติในเบราว์เซอร์…')).toBeTruthy()
    expect(SaveMCPServer).not.toHaveBeenCalled()

    finishSignIn()
    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect(vi.mocked(SaveMCPServer).mock.calls[0][1].name).toBe('semgrep')
  })

  // Already signed in from a previous visit: the credential is already in the
  // store, so the button saves immediately rather than opening a browser for
  // a sign-in that already happened.
  it('skips the browser round trip for an oauth preset already signed in', async () => {
    vi.mocked(MCPSignInStatus).mockResolvedValue({ provider: 'grafana', signed_in: true } as any)
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(ListMCPServers).toHaveBeenCalled())

    const card = screen.getByText('grafana').closest('article')!
    await fireEvent.click(card.querySelector('button')!)

    await waitFor(() => expect(SaveMCPServer).toHaveBeenCalled())
    expect(StartMCPSignIn).not.toHaveBeenCalled()
    expect(BrowserOpenURL).not.toHaveBeenCalled()
  })

  // Cancel has to actually free the backend's blocked CompleteMCPSignIn call
  // (CancelMCPSignIn), not just clear the room's own local waiting state —
  // otherwise the button here looks fine while a sign-in listener is still
  // running underneath it.
  it('cancels an in-flight oauth sign-in rather than leaving it running', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(ListMCPServers).toHaveBeenCalled())

    // vi.clearAllMocks() in beforeEach resets call history, not a
    // .mockResolvedValue override from an earlier test — the previous test
    // in this file leaves MCPSignInStatus answering signed_in:true, which
    // this one needs false to actually reach the sign-in flow it is testing.
    vi.mocked(MCPSignInStatus).mockResolvedValue({ provider: 'netlify', signed_in: false } as any)
    vi.mocked(StartMCPSignIn).mockResolvedValue(
      { provider: 'netlify', kind: 'browser', url: 'https://netlify-mcp.netlify.app/oauth-server/auth?x=1' } as any)
    vi.mocked(CompleteMCPSignIn).mockReturnValue(new Promise<undefined>(() => {})) // never resolves on its own

    const card = screen.getByText('netlify').closest('article')!
    await fireEvent.click(card.querySelector('button')!)
    await waitFor(() => expect(screen.getByText('ยกเลิก')).toBeTruthy())

    await fireEvent.click(screen.getByText('ยกเลิก'))
    expect(CancelMCPSignIn).toHaveBeenCalledWith('netlify')
    // Scoped to netlify's own card: every other preset's button also reads
    // "เพิ่ม", so a page-wide query here would be ambiguous rather than wrong.
    await waitFor(() => expect(card.querySelector('button')?.textContent).toBe('เพิ่ม'))
    expect(SaveMCPServer).not.toHaveBeenCalled()
  })
})

describe('what the room deliberately does not hold', () => {
  // เอเจนเฉพาะทาง has its own row four lines below this one, and a second door
  // into the same page is exactly what ระบบออโตเมชั่น was removed for. ซับเอเจน
  // is not a tab for a different reason: the user cannot add, edit or talk to
  // one, so a tab would be an announcement with nothing behind it.
  it('offers three tabs, and neither เอเจน nor ซับเอเจน is one', async () => {
    render(Capability, { onClose: () => {} })
    const tabs = await screen.findAllByRole('tab')
    const labels = tabs.map((el) => el.textContent?.trim())
    expect(labels).toContain('MCP')
    expect(labels).toContain('สกิล')
    expect(labels).toContain('เครื่องมือในตัว')
    expect(labels).not.toContain('เอเจน')
    expect(labels).not.toContain('ซับเอเจน')
  })

  // เครื่องมือในตัว is an inventory, not a register: there is no shelf of
  // built-in tools to browse, so offering the pair would put half a choice on
  // screen that leads nowhere.
  it('drops the library/yours switch on the built-in tab', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText('ห้องสมุด')).toBeTruthy())
    await fireEvent.click(screen.getByText('เครื่องมือในตัว'))
    await waitFor(() => expect(screen.queryByText('ห้องสมุด')).toBeNull())
  })

  // The skills shelf was empty for as long as it existed and said so, on the
  // argument that drawing a list it did not have would break its own promise.
  // skillShelf.ts is that list, so what is pinned now is that the tab draws it
  // — and the older test's point survives as the assertion that every card
  // comes from the file rather than from the markup.
  it('draws the skill shelf from skillShelf.ts', async () => {
    render(Capability, { onClose: () => {} })
    await fireEvent.click(screen.getByText('สกิล'))
    for (const p of SKILL_PRESETS) {
      expect(await screen.findByText(p.name)).toBeTruthy()
    }
  })

  // The one thing a person cannot find out after pressing: the button installs
  // the WHOLE repository, because InstallSkillFromGitHub cannot take less than
  // one. A card that showed only the pack's headline would be hiding fifty
  // skills behind a one-word button.
  it('says how many skills a card writes before it is pressed', async () => {
    render(Capability, { onClose: () => {} })
    await fireEvent.click(screen.getByText('สกิล'))
    const p = SKILL_PRESETS[0]
    const card = (await screen.findByText(p.name)).closest('article')!
    expect(card.textContent).toContain(String(p.installs.length))
  })

  // Pressing เพิ่ม hands the repo URL to the engine's own installer and nothing
  // else — no second copy of the URL, no client-side unpacking.
  it('installs a pack through the engine, by its own repo url', async () => {
    render(Capability, { onClose: () => {} })
    await fireEvent.click(screen.getByText('สกิล'))
    const p = SKILL_PRESETS[0]
    const card = (await screen.findByText(p.name)).closest('article')!
    await fireEvent.click(card.querySelector('button')!)
    await waitFor(() => expect(InstallSkillFromGitHub).toHaveBeenCalledWith(p.repo))
  })

  // A pack whose skills are all on disk is เพิ่มแล้ว and has no button, the
  // same way an added MCP server does. Matched on the pack's own `installs`
  // list, because the installer's unit is the repository: one skill in common
  // is not the pack, and a missing one is not a reason to redraw the button as
  // if nothing were there.
  it('marks a pack added only when all of its skills are present', async () => {
    const p = SKILL_PRESETS[0]
    vi.mocked(ListExternalSkills).mockResolvedValue(
      p.installs.map((name) => ({ name, description: '', dir: 'd' })) as any)
    render(Capability, { onClose: () => {} })
    await fireEvent.click(screen.getByText('สกิล'))
    const card = (await screen.findByText(p.name)).closest('article')!
    await waitFor(() => expect(card.querySelector('button')).toBeNull())
  })

  it('offers to install the rest when a pack is only half there', async () => {
    const p = SKILL_PRESETS[0]
    vi.mocked(ListExternalSkills).mockResolvedValue(
      [{ name: p.installs[0], description: '', dir: 'd' }] as any)
    render(Capability, { onClose: () => {} })
    await fireEvent.click(screen.getByText('สกิล'))
    const card = (await screen.findByText(p.name)).closest('article')!
    await waitFor(() => expect(card.querySelector('button')).not.toBeNull())
    expect(card.textContent).toContain('1')
  })
})

// ── the shelf file itself ─────────────────────────────────────────────────────
//
// mcpShelf.ts has a Go test guarding it because its keys have to line up with a
// second file. This one has no twin, so what is guarded is the promise the file
// makes in its own header: every entry is a real GitHub repository, and the
// `installs` list is the measured truth the card counts from — a list that is
// empty, duplicated, or shared between two packs would make the count and the
// เพิ่มแล้ว tick both wrong, silently.
describe('the skill shelf', () => {
  it('points every card at a github repository', () => {
    for (const p of SKILL_PRESETS) {
      expect({ name: p.name, ok: /^https:\/\/github\.com\/[^/]+\/[^/]+$/.test(p.repo) })
        .toEqual({ name: p.name, ok: true })
    }
  })

  it('lists what each pack writes, with no name twice', () => {
    for (const p of SKILL_PRESETS) {
      expect(p.installs.length).toBeGreaterThan(0)
      expect(new Set(p.installs).size).toBe(p.installs.length)
    }
  })

  // Two packs claiming the same skill folder would fight over one directory on
  // disk, and the room would draw both as added the moment either was.
  it('never has two packs claiming one skill folder', () => {
    const seen = new Map<string, string>()
    for (const p of SKILL_PRESETS) {
      for (const name of p.installs) {
        expect({ name, owner: seen.get(name) ?? p.name }).toEqual({ name, owner: p.name })
        seen.set(name, p.name)
      }
    }
  })

  // The charter's rule 1, as far as a test can read it: a pack that only
  // repeats a bundled aetox-* skill is a second copy of an answer the machine
  // already gives. The overlaps that do exist are named in the file's own
  // comments; what must never happen is a pack whose whole content is one.
  it('does not ship a pack that only repeats a bundled skill', () => {
    for (const p of SKILL_PRESETS) {
      expect({ name: p.name, count: p.installs.length }).toEqual({ name: p.name, count: p.installs.length })
      expect(p.installs.every((s) => s.startsWith('aetox-'))).toBe(false)
    }
  })
})

// ── ของคุณ: the one field this room writes ────────────────────────────────────
//
// The half that used to be a three-column table. The owner's reading of it was
// that it printed owner names in a column and then sent him to another page to
// change one, which is a page that knows the answer and refuses to act on it.
//
// What is pinned is the boundary, not the layout: `for:` is written HERE, in the
// register's own panel, through the register's own call — and nothing else came
// with it. A room that grows an edit a release is how the second editor for one
// config file gets built by accident.
describe('who carries a server, changed in the room', () => {
  const TARGETS = [
    { id: 'assistant', name: 'assistant', detail: 'โต๊ะผู้ช่วย', kind: 'desk' },
    { id: 'coding', name: 'coding', detail: 'โต๊ะโค้ด', kind: 'desk' },
    { id: 'agent:deepresearch', name: 'deepresearch', detail: 'เอเจนหาข้อมูล', kind: 'agent' },
  ]
  const server = (over: Record<string, unknown> = {}) =>
    ({ name: 'firecrawl', disabled: false, status: 'ok', tools: 4, for: [], ...over })

  const openMine = async (rows: Record<string, unknown>[]) => {
    vi.mocked(PlacementTargets).mockResolvedValue(TARGETS as any)
    vi.mocked(ListMCPServers).mockResolvedValue(rows as any)
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(PlacementTargets).toHaveBeenCalled())
    await fireEvent.click(screen.getByText('ของคุณ'))
    return await screen.findByRole('button', { name: /firecrawl/ })
  }

  it('writes the whole new list through the register own call, not a per-target one', async () => {
    const head = await openMine([server({ for: ['agent:deepresearch'] })])
    await fireEvent.click(head) // กางแถว

    const sw = await screen.findByRole('switch', { name: /assistant/ })
    expect(sw.getAttribute('aria-checked')).toBe('false')
    await fireEvent.click(sw)

    // The whole list, because that is what the engine stores — a per-target
    // call would need the engine to merge, and two places deciding what the
    // list is now is how one of them ends up wrong.
    await waitFor(() => expect(SetMCPServerTargets).toHaveBeenCalledWith('firecrawl', ['agent:deepresearch', 'assistant']))
    // And it re-reads, so the panel agrees with disk rather than with itself.
    expect(vi.mocked(ListMCPServers).mock.calls.length).toBeGreaterThan(1)
  })

  // The fact the old column could not state. Every switch under every row can be
  // off for the desks and five healthy-looking rows still read as a working
  // list — which is exactly what the owner's own machine looked like.
  it('says out loud when the assistant you talk to carries nothing', async () => {
    await openMine([server({ for: ['agent:deepresearch'] })])
    expect(screen.getByText(/ยังไม่มีเซิร์ฟเวอร์ตัวไหนเปิดให้โต๊ะเลย/)).toBeTruthy()
  })

  it('drops that warning as soon as one desk carries something', async () => {
    await openMine([server({ for: ['coding'] })])
    expect(screen.queryByText(/ยังไม่มีเซิร์ฟเวอร์ตัวไหนเปิดให้โต๊ะเลย/)).toBeNull()
  })

  // The measurement that caught this sentence lying, kept as the test that stops
  // it lying again. A chat with no desk set is the pre-modes full desk and
  // carries every server whatever `for:` says — the owner's own machine, where
  // nothing was on for a desk and 44 notion tools were in the session anyway.
  // Reading `for:` alone would have shown him "the assistant cannot reach
  // anything through MCP" while it was holding 44 of them.
  it('stays quiet when the session is actually holding MCP tools, whatever for: says', async () => {
    vi.mocked(ListTools).mockResolvedValue(
      [tool(), tool({ name: 'notion_search', source: 'mcp', category: '' })] as any)
    await openMine([server({ for: ['agent:deepresearch'] })])
    expect(screen.queryByText(/ยังไม่มีเซิร์ฟเวอร์ตัวไหนเปิดให้โต๊ะเลย/)).toBeNull()
  })

  // โต๊ะ is the settings page's word, and settings may say it bare: whoever got
  // three levels in went looking for it. This room is the announcement, so it is
  // read by someone meeting the word for the first time.
  it('translates โต๊ะ and เอเจน where the switches are, which settings does not', async () => {
    const head = await openMine([server()])
    await fireEvent.click(head)
    expect(await screen.findByText(/ผู้ช่วยหลักที่คุณคุยด้วยในหน้าแชท ทำงานอยู่บนโต๊ะพวกนี้/)).toBeTruthy()
    expect(screen.getByText(/คนในทีมที่คุยตรงได้จากหน้าทีมงาน/)).toBeTruthy()
  })

  // Reversed on 4 ก.ย. 2026, and the reversal is the point. This used to assert
  // that ลบ was NOT here — a boundary that read as principled and was not: the
  // room was already a second editor, just one with its arms cut off, so a
  // person on a row had to know which half of the job lived on which page. The
  // owner found it by trying to delete a server he was looking straight at.
  it('deletes a server from the row, behind the register own confirm', async () => {
    await openMine([server()])
    await fireEvent.click(screen.getByRole('button', { name: 'จัดการเซิร์ฟเวอร์นี้' }))
    await fireEvent.click(await screen.findByRole('button', { name: /ลบ/ }))

    // Never on the first click: this throws away a configuration that has to be
    // typed again from nothing.
    expect(RemoveMCPServer).not.toHaveBeenCalled()
    expect(screen.getByText('ลบ MCP server?')).toBeTruthy()

    // The menu shut when the dialog opened, so the only ลบ left on screen is
    // the one that actually removes it.
    await fireEvent.click(screen.getByRole('button', { name: 'ลบ' }))
    // With the name, not with '' — the first cut read it back out of reactive
    // state after clearing it, so the call went through and removed nothing.
    await waitFor(() => expect(RemoveMCPServer).toHaveBeenCalledWith('firecrawl'))
  })

  // The other two the register keeps on its row. Switching one off matters most:
  // the room grew a "ปิดอยู่" state before it grew any way back out of it.
  it('switches a server off and on from the same row', async () => {
    await openMine([server({ disabled: true, status: 'disabled' })])
    await fireEvent.click(screen.getByRole('button', { name: 'จัดการเซิร์ฟเวอร์นี้' }))
    await fireEvent.click(await screen.findByRole('button', { name: /เปิดใช้/ }))
    await waitFor(() => expect(ToggleMCPServer).toHaveBeenCalledWith('firecrawl', false))
  })

  // The form does NOT come with them — eleven fields copied is the duplication
  // the room's original no-edit rule was actually about. The label says where it
  // goes rather than pretending the field is here.
  it('sends the address and key to the register instead of copying the form', async () => {
    await openMine([server()])
    await fireEvent.click(screen.getByRole('button', { name: 'จัดการเซิร์ฟเวอร์นี้' }))
    expect(await screen.findByRole('button', { name: /ที่ทะเบียน/ })).toBeTruthy()
    for (const field of ['cwd', 'timeout', 'env']) {
      expect(screen.queryByLabelText(field)).toBeNull()
    }
  })

  // A switched-off server carries nothing whoever it is pointed at —
  // config.mcpServersFor drops it before it ever reads `for:`. The register has
  // always greyed its switches for that reason; the room copied the panel and
  // not the guard, so it took the click and wrote a value the engine ignores.
  it('refuses to pretend a switched-off server can be handed to anyone', async () => {
    const head = await openMine([server({ disabled: true, status: 'disabled' })])
    await fireEvent.click(head)

    const sw = await screen.findByRole('switch', { name: /assistant/ })
    expect((sw as HTMLButtonElement).disabled).toBe(true)
    await fireEvent.click(sw)
    expect(SetMCPServerTargets).not.toHaveBeenCalled()

    // And it says why, rather than leaving a dead control with no reason —
    // "off" has to be a word, not only a grey dot (DESIGN.md §4).
    expect(screen.getByText('ปิดอยู่')).toBeTruthy()
    expect(screen.getByText(/เปิดมันจากเมนูจัดการที่ท้ายแถวก่อน/)).toBeTruthy()
  })

  // A count is the register's answer, because that page is about servers. This
  // page has one question and it is WHO, so the shut row answers it by name.
  it('names the carriers on the shut row rather than counting them', async () => {
    await openMine([server({ for: ['agent:deepresearch'] })])
    expect(screen.getByText('deepresearch')).toBeTruthy()
    expect(screen.queryByText('เปิดให้ 1 ที่')).toBeNull()
  })
})
