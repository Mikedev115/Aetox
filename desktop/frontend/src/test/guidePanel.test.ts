import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import GuidePanel from '../lib/guide/GuidePanel.svelte'
import { guidePrefs, resetGuidePrefs, setGuidePref } from '../lib/guide/guidePrefs.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import {
  EnabledProviders,
  ListModelsForProvider,
  SupportedThinkLevelsFor,
  ListTTSEngines,
  ListTTSVoices,
  RefreshTTSVoices,
  TTSStatus,
} from './mocks/wailsApp'
import { BrowserOpenURL } from './mocks/wailsRuntime'

describe('GuidePanel think level per model', () => {
  beforeEach(() => {
    localStorage.clear()
    resetGuidePrefs()
    vi.clearAllMocks()
    cockpit.model.provider = 'codex'
    cockpit.model.modelName = 'gpt-5.6-luna'
    vi.mocked(EnabledProviders).mockResolvedValue(['codex', 'deepseek', 'ollama'] as any)
    vi.mocked(ListModelsForProvider).mockResolvedValue(['gpt-5.6-luna', 'gpt-5-pro'] as any)
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue([])
    vi.mocked(TTSStatus).mockResolvedValue('')
    vi.mocked(ListTTSEngines).mockResolvedValue([
      { id: 'windows', label: 'Windows', active: true, install: '', installCommand: [], models: [], activeModel: '' },
    ] as any)
    vi.mocked(ListTTSVoices).mockResolvedValue([
      { id: 'pattara', name: 'Pattara', lang: 'th-TH', active: true },
    ] as any)
    vi.mocked(RefreshTTSVoices).mockResolvedValue([
      { id: 'pattara', name: 'Pattara', lang: 'th-TH', active: true },
    ] as any)
  })

  it('hides think level dropdown when model does not support thinking levels', async () => {
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue([])
    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    await fireEvent.click(view.getByRole('menuitem', { name: /สมองของไกด์/ }))

    await waitFor(() => {
      expect(SupportedThinkLevelsFor).toHaveBeenCalledWith('codex', 'gpt-5.6-luna')
    })
    expect(view.container.querySelector('#guide-think')).toBeNull()
  })

  it('displays supported think levels for a reasoning model', async () => {
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue(['off', 'low', 'medium', 'high'])
    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    await fireEvent.click(view.getByRole('menuitem', { name: /สมองของไกด์/ }))

    await waitFor(() => {
      expect(view.container.querySelector('#guide-think')).not.toBeNull()
    })
    const select = view.container.querySelector('#guide-think') as HTMLSelectElement
    const options = Array.from(select.options).map((o) => o.value)
    expect(options).toEqual(['', 'off', 'low', 'medium', 'high'])
  })

  it('updates guidePrefs.think when selecting an option', async () => {
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue(['low', 'medium', 'high'])
    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    await fireEvent.click(view.getByRole('menuitem', { name: /สมองของไกด์/ }))

    await waitFor(() => {
      expect(view.container.querySelector('#guide-think')).not.toBeNull()
    })
    const select = view.container.querySelector('#guide-think') as HTMLSelectElement
    await fireEvent.change(select, { target: { value: 'high' } })
    expect(guidePrefs.think).toBe('high')
  })

  it('queries think levels for pinned guide provider and model', async () => {
    setGuidePref('provider', 'deepseek')
    setGuidePref('model', 'deepseek-reasoner')
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue(['low', 'high'])

    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    await fireEvent.click(view.getByRole('menuitem', { name: /สมองของไกด์/ }))

    await waitFor(() => {
      expect(SupportedThinkLevelsFor).toHaveBeenCalledWith('deepseek', 'deepseek-reasoner')
    })
    await waitFor(() => {
      expect(view.container.querySelector('#guide-think')).not.toBeNull()
    })
    const select = view.container.querySelector('#guide-think') as HTMLSelectElement
    const options = Array.from(select.options).map((o) => o.value)
    expect(options).toEqual(['', 'low', 'high'])
  })

  it('opens settings as a nested menu and returns to the root', async () => {
    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    expect(view.getByRole('menu', { name: 'ตั้งค่าไกด์' })).not.toBeNull()

    await fireEvent.click(view.getByRole('menuitem', { name: /ขนาดอวตาร/ }))
    const slider = view.container.querySelector<HTMLInputElement>('#guide-size')
    expect(slider).not.toBeNull()
    await fireEvent.input(slider!, { target: { value: '120' } })
    expect(guidePrefs.size).toBe(120)

    await fireEvent.click(view.getByRole('button', { name: 'กลับ' }))
    expect(view.getByRole('menu', { name: 'ตั้งค่าไกด์' })).not.toBeNull()
  })

  it('offers voice installation only when Windows has no voice for the guide language', async () => {
    vi.mocked(ListTTSVoices).mockResolvedValue([
      { id: 'zira', name: 'Zira', lang: 'en-US', active: true },
    ] as any)
    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    await fireEvent.click(view.getByRole('menuitem', { name: /การพูด/ }))

    const install = await view.findByRole('button', { name: 'ติดตั้งเสียง' })
    expect(view.getByText('ยังไม่มีเสียงภาษาไทยพร้อมใช้')).not.toBeNull()
    await fireEvent.click(install)
    expect(vi.mocked(BrowserOpenURL)).toHaveBeenCalledWith('ms-settings:speech')
  })

  it('checks newly installed voices again without restarting Aetox', async () => {
    vi.mocked(ListTTSVoices).mockResolvedValue([
      { id: 'zira', name: 'Zira', lang: 'en-US', active: true },
    ] as any)
    vi.mocked(RefreshTTSVoices).mockResolvedValue([
      { id: 'pattara', name: 'Pattara', lang: 'th-TH', active: true },
    ] as any)
    const view = render(GuidePanel, { onPick: () => {}, onClose: () => {} })
    await fireEvent.click(view.getByRole('menuitem', { name: /การพูด/ }))
    const recheck = await view.findByRole('button', { name: 'ตรวจอีกครั้ง' })

    await fireEvent.click(recheck)
    await waitFor(() => expect(view.queryByRole('button', { name: 'ติดตั้งเสียง' })).toBeNull())
    expect(vi.mocked(RefreshTTSVoices)).toHaveBeenCalledTimes(1)
  })
})
