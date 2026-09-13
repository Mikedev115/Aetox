import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import RemoteEngine from '../lib/RemoteEngine.svelte'
import RemoteDirPicker from '../lib/RemoteDirPicker.svelte'
import EngineStatus from '../lib/EngineStatus.svelte'
import TopBar from '../lib/TopBar.svelte'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import {
  RemoteHosts, SaveRemoteHost, ConnectRemote, DisconnectRemote, ListDir, HomeDir,
  OpenProjectFolder, EngineStatus as engineStatus,
} from './mocks/wailsApp'
import { engine, resetEngineStore, applyEngineStatus } from '../lib/stores/engine.svelte'
import { openFolder, cockpit } from '../lib/stores/cockpit.svelte'

// The engine on another machine (§248 phase 3): the Settings section that
// lists hosts and connects, the picker that browses the host's folders
// through the engine, and the status card's words for the road there.
// i18n defaults to Thai.

const box = { name: 'box', target: 'user@box', root: '/home/u/proj', version: '1.5.28', arch: 'amd64', lastUsed: '', spec: 'Ubuntu 26.04 LTS · 4 CPU · 8 GB · amd64' }
const thisPC = { hostname: 'PC', os: 'windows', arch: 'amd64', version: 'Windows 11 (26200)', cpu: 'AMD Ryzen 7', cpus: 16, memBytes: 32 * 2 ** 30 }

function status(st: Record<string, unknown>) {
  return { state: 'connected', detail: '', restarts: 0, pid: 0, address: '', mode: 'local', host: '', ...st } as any
}

beforeEach(() => {
  resetEngineStore()
  vi.mocked(EventsOn).mockClear()
  vi.mocked(RemoteHosts).mockClear()
  vi.mocked(SaveRemoteHost).mockClear()
  vi.mocked(ConnectRemote).mockClear()
  vi.mocked(DisconnectRemote).mockClear()
  vi.mocked(OpenProjectFolder).mockClear()
  vi.mocked(ListDir).mockClear()
  vi.mocked(RemoteHosts).mockResolvedValue({ this: thisPC, active: '', hosts: [box], ssh: 'C:\\ssh.exe', sshError: '', engine: 'release' } as any)
  vi.mocked(engineStatus).mockResolvedValue(status({}))
})

describe('Settings › เครื่องระยะไกล', () => {
  it('lists the hosts and connects with one press', async () => {
    render(RemoteEngine)
    await waitFor(() => expect(screen.getByText('box')).toBeTruthy())
    expect(screen.getByText('user@box')).toBeTruthy()
    expect(screen.getByText(/เครื่องยนต์ 1\.5\.28 \(amd64\)/)).toBeTruthy()
    expect(screen.getByText('เครื่องยนต์อยู่ที่เครื่องนี้')).toBeTruthy()
    // This machine is the first row, with its spec; the host row carries
    // what its engine said last time.
    expect(screen.getByText('เครื่องนี้')).toBeTruthy()
    expect(screen.getByText('PC')).toBeTruthy()
    expect(screen.getByText(/Windows 11 \(26200\) · 16 CPU · 32 GB · amd64 · AMD Ryzen 7/)).toBeTruthy()
    expect(screen.getByText('Ubuntu 26.04 LTS · 4 CPU · 8 GB · amd64')).toBeTruthy()
    await fireEvent.click(screen.getByText('เชื่อมต่อ'))
    expect(ConnectRemote).toHaveBeenCalledWith('box')
  })

  it('offers the way back when the window is on a host', async () => {
    vi.mocked(RemoteHosts).mockResolvedValue({ this: thisPC, active: 'box', hosts: [box], ssh: 'C:\\ssh.exe', sshError: '', engine: 'release' } as any)
    applyEngineStatus(status({ mode: 'remote', host: 'box' }))
    render(RemoteEngine)
    await waitFor(() => expect(screen.getByText('เครื่องยนต์อยู่ที่ box')).toBeTruthy())
    expect(screen.queryByText('เชื่อมต่อ')).toBeNull()
    await fireEvent.click(screen.getAllByText('กลับมาเครื่องนี้')[0])
    expect(DisconnectRemote).toHaveBeenCalledTimes(1)
  })

  it('saves a new host from the form', async () => {
    render(RemoteEngine)
    await waitFor(() => expect(RemoteHosts).toHaveBeenCalled())
    // The form is a dialog behind the + button since 13 ก.ย. (5f343f9a), and
    // its fields carry labels rather than long placeholders.
    await fireEvent.click(screen.getByText('เพิ่มเครื่อง'))
    const target = (await screen.findByPlaceholderText('user@host')) as HTMLInputElement
    await fireEvent.input(target, { target: { value: 'dev@10.0.0.5' } })
    await fireEvent.click(screen.getByText('บันทึก'))
    await waitFor(() => expect(SaveRemoteHost).toHaveBeenCalledWith('', 'dev@10.0.0.5', ''))
  })

  it('cannot connect without ssh, and says why', async () => {
    vi.mocked(RemoteHosts).mockResolvedValue({ this: thisPC, active: '', hosts: [box], ssh: '', sshError: 'ไม่พบ ssh ในเครื่องนี้', engine: 'release' } as any)
    render(RemoteEngine)
    await waitFor(() => expect(screen.getByText('ไม่พบ ssh ในเครื่องนี้')).toBeTruthy())
    expect((screen.getByText('เชื่อมต่อ') as HTMLButtonElement).disabled).toBe(true)
  })
})

describe('the folder door on a host', () => {
  it('raises the picker instead of the native dialog', async () => {
    applyEngineStatus(status({ mode: 'remote', host: 'box' }))
    await openFolder()
    expect(OpenProjectFolder).not.toHaveBeenCalled()
    expect(engine.pickerOpen).toBe(true)
  })

  it('uses the native dialog on this machine', async () => {
    await openFolder()
    expect(OpenProjectFolder).toHaveBeenCalledTimes(1)
    expect(engine.pickerOpen).toBe(false)
  })
})

describe('RemoteDirPicker', () => {
  it('lists the host home, walks into a folder, and hands the path back', async () => {
    vi.mocked(HomeDir).mockResolvedValue('/home/u')
    vi.mocked(ListDir).mockImplementation(async (p: string) => {
      if (p === '/home/u/proj') return { path: '/home/u/proj', parent: '/home/u', entries: [], truncated: false } as any
      return {
        path: '/home/u', parent: '/home',
        entries: [{ name: 'proj', path: '/home/u/proj', hidden: false }, { name: '.ssh', path: '/home/u/.ssh', hidden: true }],
        truncated: false,
      } as any
    })
    const onPick = vi.fn()
    render(RemoteDirPicker, { props: { host: 'box', onPick, onCancel: vi.fn() } })
    await waitFor(() => expect(screen.getByText('proj')).toBeTruthy())
    expect(screen.getByText('เลือกโฟลเดอร์บน box')).toBeTruthy()
    // Hidden folders wait behind the checkbox.
    expect(screen.queryByText('.ssh')).toBeNull()
    await fireEvent.click(screen.getByLabelText('แสดงโฟลเดอร์ที่ซ่อน'))
    expect(screen.getByText('.ssh')).toBeTruthy()

    await fireEvent.click(screen.getByText('proj'))
    await waitFor(() => expect(ListDir).toHaveBeenCalledWith('/home/u/proj'))
    await waitFor(() => expect(screen.getByText('ไม่มีโฟลเดอร์ย่อย')).toBeTruthy())
    await fireEvent.click(screen.getByText('เลือกโฟลเดอร์นี้'))
    expect(onPick).toHaveBeenCalledWith('/home/u/proj')
  })

  it('goes where the path box says on Enter', async () => {
    vi.mocked(HomeDir).mockResolvedValue('/home/u')
    render(RemoteDirPicker, { props: { host: 'box', onPick: vi.fn(), onCancel: vi.fn() } })
    await waitFor(() => expect(ListDir).toHaveBeenCalledWith('/home/u'))
    const input = screen.getByLabelText('ที่อยู่โฟลเดอร์') as HTMLInputElement
    await fireEvent.input(input, { target: { value: '/srv' } })
    await fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(ListDir).toHaveBeenCalledWith('/srv'))
  })
})

describe('EngineStatus on the road to a host', () => {
  function announce(st: Record<string, unknown>) {
    const call = vi.mocked(EventsOn).mock.calls.find((c) => c[0] === 'engine:status')
    if (!call) throw new Error('nothing subscribed to engine:status')
    ;(call[1] as (st: unknown) => void)(status(st))
  }

  it('shows each step at once, with a way back to this machine', async () => {
    const { container } = render(EngineStatus)
    announce({ state: 'starting', mode: 'remote', host: 'box', detail: 'กำลังส่งเครื่องยนต์ไปที่ box (40%)' })
    await waitFor(() => expect(container.querySelector('.upd-card')).not.toBeNull())
    expect(screen.getByText('กำลังไปที่ box')).toBeTruthy()
    expect(screen.getByText('กำลังส่งเครื่องยนต์ไปที่ box (40%)')).toBeTruthy()
    await fireEvent.click(screen.getByText('ใช้เครื่องนี้แทน'))
    expect(DisconnectRemote).toHaveBeenCalledTimes(1)
  })

  it('names the host in a failure and offers both ways out', async () => {
    render(EngineStatus)
    announce({ state: 'failed', mode: 'remote', host: 'box', detail: 'ssh user@box: Permission denied (publickey)' })
    await waitFor(() => expect(screen.getByText('ไปที่ box ไม่สำเร็จ')).toBeTruthy())
    expect(screen.getByText('ssh user@box: Permission denied (publickey)')).toBeTruthy()
    expect(screen.getByText('ลองอีกครั้ง')).toBeTruthy()
    expect(screen.getByText('ใช้เครื่องนี้แทน')).toBeTruthy()
  })
})

describe('the host badge on the top bar', () => {
  const props = {
    inspectorCollapsed: false, onToggleInspector: () => {},
    sidebarCollapsed: false, onToggleSidebar: () => {},
  }

  it('names the host while the engine is there, and is absent at home', async () => {
    const { unmount } = render(TopBar, props)
    expect(document.querySelector('.host-badge')).toBeNull()
    unmount()

    applyEngineStatus(status({ mode: 'remote', host: 'box' }))
    render(TopBar, props)
    const badge = document.querySelector('.host-badge') as HTMLButtonElement
    expect(badge).not.toBeNull()
    expect(badge.textContent).toContain('box')
    expect(badge.title).toBe('เครื่องยนต์อยู่ที่ box')
    await fireEvent.click(badge)
    expect(cockpit.activeView).toBe('settings')
  })
})
