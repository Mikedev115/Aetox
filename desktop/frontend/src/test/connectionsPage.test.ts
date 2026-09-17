// การเชื่อมต่อ — a register of external accounts and self-run services: one
// line per service until you open it, and inside, who may use it. Since
// 14 ก.ย. 2026 it is a heading of ห้องความสามารถ (Capability.svelte), moved
// whole out of ตั้งค่า; these tests moved with it out of Settings.test.ts and
// open it off the room's rail.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import {
  Connections, ConnectAccount, SetConnectionTargets, DisconnectAccount,
  ListMCPServers, ListSubagentProfiles, PlacementTargets, ListTools,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'capability'
  cockpit.capabilityIntent = null
  vi.mocked(ListMCPServers).mockResolvedValue([] as any)
  vi.mocked(ListSubagentProfiles).mockResolvedValue([] as any)
  vi.mocked(ListTools).mockResolvedValue([] as any)
  // Where an account can be pointed. The placement tests assert this exact
  // list, by name.
  vi.mocked(PlacementTargets).mockResolvedValue([
    { id: 'assistant', name: 'ผู้ช่วย', kind: 'desk' },
    { id: 'coding', name: 'โค้ด', kind: 'desk' },
    { id: 'agent:researcher', name: 'researcher', kind: 'agent' },
  ] as any)
})

// Render the room and walk to the register's row on the rail.
const openConnections = async () => {
  const rendered = render(Capability, { onClose: () => {} })
  await waitFor(() => expect(PlacementTargets).toHaveBeenCalled())
  const row = Array.from(rendered.container.querySelectorAll('.settings-nav-item'))
    .find((el) => el.textContent?.includes('บริการที่เชื่อมไว้'))
  if (!row) throw new Error('rail row "บริการที่เชื่อมไว้" not found')
  await fireEvent.click(row)
  await waitFor(() => expect(Connections).toHaveBeenCalled())
  return rendered
}

// Opens the row itself, which is where everything but the name and the state
// lives.
const expandRow = async (container: HTMLElement) => {
  const head = container.querySelector('.reg-head')
  if (!head) throw new Error('no connection row to open')
  await fireEvent.click(head)
}

const githubRow = (over: Record<string, unknown> = {}) => [{
  id: 'github', label: 'GitHub', kind: 'token',
  token_url: 'https://github.com/settings/tokens/new',
  connected: false, env_override: false, for: [], configured: false,
  system: true,
  tools: ['github', 'plugin_install'],
  ...over,
}]

const connectedRow = (over: Record<string, unknown> = {}) => githubRow({
  connected: true, login: 'mike', source: 'connection', for: ['coding'], configured: true, ...over,
})

// The register remains generic enough for a future connection that genuinely
// has audiences. GitHub itself is not that example anymore: it is a system
// credential and must not draw or persist this second gate.
const placeableGithubRow = (over: Record<string, unknown> = {}) => githubRow({ system: false, ...over })
const connectedPlaceableRow = (over: Record<string, unknown> = {}) => placeableGithubRow({
  connected: true, login: 'mike', source: 'connection', for: ['coding'], configured: true, ...over,
})

// A service the user hosts. n8n and Windmill live wherever the user put them,
// so the row has to ask where before a token means anything.
const selfHostedRow = (over: Record<string, unknown> = {}) => [{
  id: 'n8n', label: 'n8n', kind: 'token',
  connected: false, env_override: false, for: [], configured: false,
  tools: ['n8n_workflow_list', 'n8n_workflow_create'],
  needs_base_url: true, base_url_hint: 'http://localhost:5678',
  ...over,
}]

const telegramRow = (over: Record<string, unknown> = {}) => [{
  id: 'telegram', label: 'Telegram', kind: 'token',
  connected: true, login: '@Aetoxbot', source: 'connection', env_override: false,
  for: [], configured: false, tools: [], channel: true, paired: true,
  channel_desk: 'assistant', channel_assistant: 'มะลิ',
  channel_provider: 'openai', channel_model: 'gpt-5.6',
  ...over,
}]

describe('where the register is', () => {
  // A heading of one row on the room's rail, after การใช้คอมพิวเตอร์: the last
  // two are the reaches that moved out of ตั้งค่า together.
  it('is a heading of the capability rail, not a page of ตั้งค่า', async () => {
    const { container } = render(Capability, { onClose: () => {} })
    await waitFor(() => expect(PlacementTargets).toHaveBeenCalled())
    const groups = Array.from(container.querySelectorAll('.settings-nav .settings-group-label')).map((x) => x.textContent?.trim())
    expect(groups.indexOf('การเชื่อมต่อ')).toBeGreaterThan(groups.indexOf('การใช้คอมพิวเตอร์'))
  })

  // The engine chip in the composer and the agent editor's "needs" row both
  // door here (openCapabilityAt('connections')); the room must land on it.
  it('opens straight from openCapabilityAt', async () => {
    vi.mocked(Connections).mockResolvedValue(githubRow() as any)
    cockpit.capabilityIntent = { page: 'connections' }
    const { container } = render(Capability, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText('GitHub')).toBeTruthy())
    expect(container.querySelector('.settings-nav-item.active')?.textContent?.trim()).toBe('บริการที่เชื่อมไว้')
  })

  // A door can land here before the room's own load has brought the targets;
  // a draft built over none would tick nothing, so the page fetches them.
  it('fetches the targets itself when the door arrives first', async () => {
    vi.mocked(Connections).mockResolvedValue(placeableGithubRow() as any)
    cockpit.capabilityIntent = { page: 'connections' }
    const { container } = render(Capability, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText('GitHub')).toBeTruthy())
    await expandRow(container)
    const chips = Array.from(container.querySelectorAll('.conn-chip'))
    expect(chips.map((c) => c.getAttribute('aria-pressed'))).toEqual(['true', 'true', 'false'])
  })
})

// Two pages over one register, split by the catalog's `family`, for nine days;
// the owner's verdict on 19 ส.ค. undid the split: *"มันคืออันเดียวกันแท้ๆ
// เชื่อมต่อแอปภายนอก เอาคีย์ไปใส่"*. What these pin is that NOTHING is filtered
// out — the way that fails is silent: an account the user cannot find.
describe('one page for everything the agent connects to', () => {
  it('shows accounts and self-run engines together', async () => {
    vi.mocked(Connections).mockResolvedValue([
      ...githubRow(), ...selfHostedRow({ family: 'automation' }),
    ] as any)
    await openConnections()

    await waitFor(() => expect(screen.getByText('GitHub')).toBeTruthy())
    expect(screen.getByText('n8n')).toBeTruthy()
  })

  // The family is still a real fact — it is what the composer's engine picker
  // asks (connect.InFamily) — it is just no longer allowed to decide what a
  // page draws.
  it('shows a service whatever family it declares', async () => {
    vi.mocked(Connections).mockResolvedValue(
      githubRow({ id: 'gmail', label: 'Gmail', family: 'mail' }) as any)
    await openConnections()

    await waitFor(() => expect(screen.getByText('Gmail')).toBeTruthy())
  })
})

describe('a row speaks for its own service', () => {
  // These strings were GitHub's copy hardcoded, so the n8n row asked for a
  // "PERSONAL ACCESS TOKEN" starting `ghp_…` and promised to check it with
  // GitHub. Wrong on every row but one.
  it('does not put GitHub words on a self-hosted engine', async () => {
    vi.mocked(Connections).mockResolvedValue(selfHostedRow({ family: 'automation' }) as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    // Scoped to the register: the page's own description names GitHub as the
    // example of an account that takes only a key. What must not carry
    // GitHub's words is the ROW.
    const register = container.querySelector('.settings-card')
    expect(register?.textContent).not.toContain('GitHub')
    expect(container.querySelector('input[type="password"]')?.getAttribute('placeholder')).toBe('')
    expect(container.textContent).toContain('n8n')
  })

  // Two questions, and the page has to look like it knows they are two:
  // whether the PROGRAM is up, and whether the KEY works.
  it('splits the server from the account, in order', async () => {
    vi.mocked(Connections).mockResolvedValue(selfHostedRow({ family: 'automation' }) as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    const heads = Array.from(container.querySelectorAll('.conn-part-head'))
      .map((el) => el.textContent?.trim())
    expect(heads).toEqual(['1 · ตัวเซิร์ฟเวอร์', '2 · บัญชีและคีย์'])
  })
})

describe('disconnecting', () => {
  // The one destructive button on this page that just did it. An n8n key is
  // shown once at creation and never again, so a mis-click costs a trip to
  // another program to mint a new one.
  it('asks before throwing a credential away', async () => {
    vi.mocked(Connections).mockResolvedValue(connectedPlaceableRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    await expandRow(container)

    await fireEvent.click(screen.getByText('เลิกเชื่อม'))
    expect(vi.mocked(DisconnectAccount)).not.toHaveBeenCalled()

    // And what survives is said, because "disconnect" reads like "start over"
    // and it is not: the address and the placement are still there.
    await waitFor(() => expect(screen.getByText(/เลิกเชื่อม GitHub\?/)).toBeTruthy())
  })
})

describe('the register', () => {
  // Collapsed is the state a returning user arrives in, so the line has to
  // carry the whole answer: which service, whether it is attached, a way in.
  it('lists each service as one line with its state and a way to connect', async () => {
    vi.mocked(Connections).mockResolvedValue(githubRow() as any)
    const { container } = await openConnections()

    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    expect(screen.getByText('GitHub')).toBeTruthy()
    expect(screen.getByText('เชื่อม')).toBeTruthy()
    // Nothing is open, so no token box is on screen waiting to be pasted into.
    expect(container.querySelector('input[type="password"]')).toBeNull()
    expect(container.querySelector('.conn-body')).toBeNull()
  })

  it('opens one row at a time, and the connect button opens it too', async () => {
    vi.mocked(Connections).mockResolvedValue(githubRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())

    await fireEvent.click(screen.getByText('เชื่อม'))
    await waitFor(() => expect(container.querySelector('.conn-body')).toBeTruthy())
    expect(container.querySelector('input[type="password"]')).toBeTruthy()

    // And the header closes it again.
    await expandRow(container)
    await waitFor(() => expect(container.querySelector('.conn-body')).toBeNull())
  })

  // A connected row says where it reaches without being opened, by name —
  // "2 desks" would make you open it to find out which two.
  it('names the desks on the collapsed row of a connected service', async () => {
    vi.mocked(Connections).mockResolvedValue(connectedPlaceableRow() as any)
    const { container } = await openConnections()

    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    expect(container.querySelector('.mcp-badge')?.textContent?.trim()).toBe('โค้ด')
  })

  // Never placed is not "off" — it is carried everywhere, and the row must say
  // which of the two it is.
  it('says every desk on a connected service nobody has placed yet', async () => {
    vi.mocked(Connections).mockResolvedValue(connectedPlaceableRow({ for: [], configured: false }) as any)
    const { container } = await openConnections()

    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    expect(container.querySelector('.mcp-badge')?.textContent?.trim()).toBe('ทุกโต๊ะ')
    expect(container.querySelector('.mcp-badge-warn')).toBeNull()
  })

  // Connected and placed nowhere looks healthy and reaches no one — the one
  // state worth interrupting for, same as the MCP cards call out.
  it('calls out a connection that serves nobody', async () => {
    vi.mocked(Connections).mockResolvedValue(connectedPlaceableRow({ for: [] }) as any)
    const { container } = await openConnections()

    await waitFor(() => expect(screen.getByText('ไม่มีใคร')).toBeTruthy())
    expect(container.querySelector('.mcp-badge-warn')).toBeTruthy()
  })

  it('offers every desk and agent as a placement, desks picked and agents not', async () => {
    vi.mocked(Connections).mockResolvedValue(placeableGithubRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    const chips = Array.from(container.querySelectorAll('.conn-chip'))
    expect(chips.map((c) => c.textContent?.trim())).toEqual(['ผู้ช่วย', 'โค้ด', 'researcher'])
    // An agent is handed things on purpose; a desk is where work already happens.
    expect(chips.map((c) => c.getAttribute('aria-pressed'))).toEqual(['true', 'true', 'false'])
  })

  it('does not draw or send a second audience gate for a system connection', async () => {
    vi.mocked(Connections).mockResolvedValue(githubRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('GitHub')).toBeTruthy())
    await expandRow(container)

    expect(container.querySelector('.conn-targets')).toBeNull()
    expect(container.querySelector('.mcp-badge')).toBeNull()

    const field = container.querySelector('input[type="password"]') as HTMLInputElement
    await fireEvent.input(field, { target: { value: 'ghp_system' } })
    await fireEvent.click(screen.getByText('เชื่อม'))
    await waitFor(() =>
      expect(vi.mocked(ConnectAccount)).toHaveBeenCalledWith('github', 'ghp_system', '', []))
  })

  // A service Aetox hosts nowhere cannot be reached until the user says where
  // it is, and a token checked against the wrong host fails in a way that
  // reads as a bad token — so the address is asked for first and the button
  // stays down until it is there.
  it('asks a self-hosted service where it lives before it will connect', async () => {
    vi.mocked(Connections).mockResolvedValue(selfHostedRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    const address = container.querySelector('input[type="text"].key-input') as HTMLInputElement
    expect(address).toBeTruthy()
    expect(address.placeholder).toBe('http://localhost:5678')

    const token = container.querySelector('input[type="password"]') as HTMLInputElement
    await fireEvent.input(token, { target: { value: 'n8n_api_key' } })
    const connect = screen.getByText('เชื่อม') as HTMLButtonElement
    expect(connect.disabled).toBe(true)

    await fireEvent.input(address, { target: { value: '  http://box.local:5678  ' } })
    expect(connect.disabled).toBe(false)
    await fireEvent.click(connect)

    // The address goes trimmed, and beside the token in one call.
    await waitFor(() => expect(vi.mocked(ConnectAccount)).toHaveBeenCalled())
    expect(vi.mocked(ConnectAccount).mock.calls[0].slice(0, 3))
      .toEqual(['n8n', 'n8n_api_key', 'http://box.local:5678'])
  })

  // GitHub is one host for everybody and states it as a constant. Drawing an
  // address field on its row would ask a question with one possible answer.
  it('does not ask a hosted service for an address', async () => {
    vi.mocked(Connections).mockResolvedValue(githubRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    expect(container.querySelector('input[type="text"].key-input')).toBeNull()
  })

  it('connects with the pasted token and the chosen desks in one call', async () => {
    vi.mocked(Connections)
      .mockResolvedValueOnce(placeableGithubRow() as any)
      .mockResolvedValue(connectedPlaceableRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    const chips = Array.from(container.querySelectorAll('.conn-chip'))
    await fireEvent.click(chips[0])

    const field = container.querySelector('input[type="password"]') as HTMLInputElement
    await fireEvent.input(field, { target: { value: '  ghp_example  ' } })
    await fireEvent.click(screen.getByText('เชื่อม'))

    // The empty string is the base URL: GitHub is one host for everybody and
    // states it as a constant, so the field is not even drawn on its row.
    await waitFor(() =>
      expect(vi.mocked(ConnectAccount)).toHaveBeenCalledWith('github', 'ghp_example', '', ['coding']))
    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    expect(container.textContent).toContain('สิทธิ์: repo')
    expect(container.querySelector('input[type="password"]')).toBeNull()
  })

  // Flipping a switch on a connected row writes straight through. It must not
  // go via the connect path, which would send the token field along with it.
  it('moves a connected account between desks without touching its token', async () => {
    vi.mocked(Connections).mockResolvedValue(connectedPlaceableRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    await expandRow(container)

    const chips = Array.from(container.querySelectorAll('.conn-chip'))
    await fireEvent.click(chips[0])

    await waitFor(() =>
      expect(vi.mocked(SetConnectionTargets)).toHaveBeenCalledWith('github', ['coding', 'assistant']))
    expect(vi.mocked(ConnectAccount)).not.toHaveBeenCalled()
  })

  it('keeps the typed token when the service rejects it', async () => {
    vi.mocked(Connections).mockResolvedValue(githubRow() as any)
    vi.mocked(ConnectAccount).mockRejectedValueOnce(new Error('GitHub rejected this token'))
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
    await expandRow(container)

    const field = container.querySelector('input[type="password"]') as HTMLInputElement
    await fireEvent.input(field, { target: { value: 'ghp_wrong' } })
    await fireEvent.click(screen.getByText('เชื่อม'))

    await waitFor(() => expect(container.textContent).toContain('GitHub rejected this token'))
    expect((container.querySelector('input[type="password"]') as HTMLInputElement).value).toBe('ghp_wrong')
  })

  // Disconnect lives inside the row on purpose: it is not something to do by
  // accident while scanning a list.
  it('disconnects from inside the opened row', async () => {
    vi.mocked(Connections)
      .mockResolvedValueOnce(connectedRow() as any)
      .mockResolvedValue(githubRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    expect(screen.queryByText('เลิกเชื่อม')).toBeNull()

    await expandRow(container)
    await fireEvent.click(screen.getByText('เลิกเชื่อม'))
    // Through the confirm gate: the dialog's own button carries the same
    // word, so the second click is the one that acts.
    const confirm = await waitFor(() =>
      Array.from(document.querySelectorAll('.confirm-actions button, .modal button'))
        .find((b) => b.textContent?.trim() === 'เลิกเชื่อม') as HTMLButtonElement)
    await fireEvent.click(confirm)

    await waitFor(() => expect(vi.mocked(DisconnectAccount)).toHaveBeenCalledWith('github'))
    await waitFor(() => expect(screen.getByText('ยังไม่ได้เชื่อม')).toBeTruthy())
  })

  // A token exported in a shell profile is why GitHub keeps working for someone
  // who never connected anything. The page has to say so, and must still offer
  // the form, since connecting an account is how you take that override back.
  it('says when the token comes from the environment, and still offers to connect', async () => {
    vi.mocked(Connections).mockResolvedValue(
      githubRow({ connected: true, source: 'environment', env_override: true }) as any)
    const { container } = await openConnections()

    await waitFor(() => expect(screen.getByText('กำลังใช้ token จาก environment')).toBeTruthy())
    await expandRow(container)
    expect(container.querySelector('input[type="password"]')).toBeTruthy()
    expect(screen.queryByText('เลิกเชื่อม')).toBeNull()
  })

  it('discloses an environment token that the connected account is overriding', async () => {
    vi.mocked(Connections).mockResolvedValue(connectedRow({ env_override: true }) as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('เชื่อมแล้วในชื่อ mike')).toBeTruthy())
    await expandRow(container)

    expect(container.textContent).toContain('แต่ Aetox ใช้บัญชีที่เชื่อมไว้')
  })
})

describe('bot channels name their real destination', () => {
  it('shows the runtime assistant identity, model, desk, and isolated history', async () => {
    vi.mocked(Connections).mockResolvedValue(telegramRow() as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('Telegram')).toBeTruthy())
    await expandRow(container)

    expect(screen.getByText('ปลายทางจริง · ผู้ช่วยหลัก “มะลิ”')).toBeTruthy()
    expect(container.textContent).toContain('openai · gpt-5.6')
    expect(container.textContent).toContain('หน้าผู้ช่วยเท่านั้น')
    expect(container.textContent).toContain('ห้องนี้มีประวัติของตัวเอง')
    expect(container.querySelector('.conn-route')?.getAttribute('data-channel-desk')).toBe('assistant')
    expect(container.querySelector('.conn-route-icon .scope-face .mascot')).toBeTruthy()
    expect(container.textContent).toContain('พร้อมใช้งาน · ห้องนี้คุยกับผู้ช่วยหลัก “มะลิ”')
  })

  it('separates a connected token from authorizing the room', async () => {
    vi.mocked(Connections).mockResolvedValue(telegramRow({ paired: false, pairing_code: '941012' }) as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('Telegram')).toBeTruthy())
    await expandRow(container)

    expect(container.textContent).toContain('Bot token เชื่อมแล้ว · เหลืออนุญาตห้องนี้ครั้งเดียว')
    expect(container.textContent).toContain('/pair 941012')
    expect(container.textContent).toContain('กันคนอื่นที่ค้นเจอชื่อบอท')
  })

  it('draws the real service marks and a neutral fallback for an unknown future service', async () => {
    vi.mocked(Connections).mockResolvedValue([
      ...githubRow(),
      ...telegramRow(),
      ...telegramRow({ id: 'discord', label: 'Discord' }),
      ...selfHostedRow(),
      ...selfHostedRow({ id: 'windmill', label: 'Windmill' }),
      ...githubRow({ id: 'future-service', label: 'Future service' }),
    ] as any)
    const { container } = await openConnections()
    await waitFor(() => expect(screen.getByText('Future service')).toBeTruthy())

    expect(container.querySelectorAll('.reg-head').length).toBe(6)
    expect(Array.from(container.querySelectorAll('[data-connection-mark]')).map((el) =>
      el.getAttribute('data-connection-mark'))).toEqual([
      'github', 'telegram', 'discord', 'n8n', 'windmill', 'fallback',
    ])
    expect(container.querySelectorAll('.conn-service-icon svg').length).toBe(6)
  })
})
