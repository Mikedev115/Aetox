// ห้องความสามารถ. The room exists to announce, not to organise — a person using
// Aetox had no way to learn it speaks MCP at all, because the register sat three
// levels inside Settings and the word appeared nowhere a reader would pass.
//
// So what is pinned here is the announcing, which is the part that quietly stops
// working: the room opening on the shelf rather than on your own empty list, the
// shelf being the real list rather than a second copy of it, and the two rooms
// this one deliberately does not swallow.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import { NAV, navFor } from '../lib/desks'
import { MCP_PRESETS, needsPaste, presetFor, presetConfig } from '../lib/mcpShelf'
import {
  ListMCPServers, ListExternalSkills, ListTools, SaveMCPServer,
  StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus,
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
  it('opens on the shelf, showing every preset, with nothing connected', async () => {
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
  it('drops the shelf/yours switch on the built-in tab', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText('ชั้นวาง')).toBeTruthy())
    await fireEvent.click(screen.getByText('เครื่องมือในตัว'))
    await waitFor(() => expect(screen.queryByText('ชั้นวาง')).toBeNull())
  })

  // The skills shelf is empty and says so. MCP has eight curated entries
  // because mcpShelf.ts was written; skills have no such list, and drawing one
  // it does not have would be the shelf breaking its own promise.
  it('admits there is no skill shelf rather than inventing one', async () => {
    render(Capability, { onClose: () => {} })
    await fireEvent.click(screen.getByText('สกิล'))
    expect(await screen.findByText('ยังไม่มีชั้นวางสกิล')).toBeTruthy()
  })
})
