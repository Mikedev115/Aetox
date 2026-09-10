import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import ArtifactsPane from '../lib/workbench/ArtifactsPane.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { workbench } from '../lib/stores/workbench.svelte'
import {
  CurrentSessionID, SessionEdits, ListArtifactsIn, ReadFile,
} from './mocks/wailsApp'

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.plan = null
  cockpit.chat = []
  workbench.tabs = []
  workbench.activeId = ''
  vi.mocked(CurrentSessionID).mockResolvedValue('session-1')
  vi.mocked(SessionEdits).mockResolvedValue({ files: [], total: 0 })
  vi.mocked(ListArtifactsIn).mockResolvedValue({ files: [], range: 'all', total: 0 } as any)
  vi.mocked(ReadFile).mockResolvedValue('')
})

describe('ArtifactsPane', () => {
  it('shows empty state when session has no plan and no produced files', async () => {
    render(ArtifactsPane)
    await waitFor(() => {
      expect(screen.getByText('ยังไม่มีชิ้นงานในเซสชันนี้')).toBeTruthy()
    })
  })

  it('renders plan as an artifact item when cockpit.plan exists', async () => {
    cockpit.plan = {
      title: 'สร้างระบบ Artifacts',
      steps: [
        { n: 1, text: 'ออกแบบ UI', state: 'done' },
        { n: 2, text: 'เขียนโค้ด', state: 'doing' },
      ],
      sections: [{ heading: 'เป้าหมาย', body: 'เพิ่มพาเนลชิ้นงาน' }],
    } as any

    render(ArtifactsPane)

    await waitFor(() => {
      expect(screen.getAllByText('สร้างระบบ Artifacts').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText(/1\/2 ขั้นตอน/)).toBeTruthy()
    })
  })

  it('collects produced files from chat messages and displays them with icons and badges', async () => {
    cockpit.chat = [
      {
        role: 'agent',
        text: 'ทำเสร็จแล้วครับ',
        time: '12:00',
        producedFiles: [
          'output/session-1/walkthrough.md',
          'output/session-1/mockup.html',
          'output/session-1/diagram.png',
        ],
      },
    ] as any

    vi.mocked(ReadFile).mockImplementation(async (path: string) => {
      if (path.endsWith('.md')) return '# สรุปการทำงาน\n\nระบบใช้งานได้ดี'
      if (path.endsWith('.html')) return '<h1>Mockup</h1>'
      return ''
    })

    const { container } = render(ArtifactsPane)

    await waitFor(() => {
      expect(screen.getAllByText('walkthrough.md').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('mockup.html')).toBeTruthy()
      expect(screen.getByText('diagram.png')).toBeTruthy()
    })

    // Click on walkthrough.md item in the list to view
    const itemBtn = container.querySelector('.art-item') as HTMLElement
    await fireEvent.click(itemBtn)
    await waitFor(() => {
      expect(screen.getByText('สรุปการทำงาน')).toBeTruthy()
    })
  })

  it('allows filtering by kind', async () => {
    cockpit.chat = [
      {
        role: 'agent',
        text: 'ผลลัพธ์',
        time: '12:00',
        producedFiles: [
          'output/session-1/notes.md',
          'output/session-1/page.html',
        ],
      },
    ] as any

    const { container } = render(ArtifactsPane)

    await waitFor(() => {
      expect(container.querySelector('.art-item')).toBeTruthy()
    })

    // Click filter UI / เว็บ
    const pageFilter = screen.getByText('UI / เว็บ')
    await fireEvent.click(pageFilter)

    await waitFor(() => {
      const items = Array.from(container.querySelectorAll('.art-item-name')).map((el) => el.textContent?.trim())
      expect(items).toContain('page.html')
      expect(items).not.toContain('notes.md')
    })
  })

  it('toggles between UI Preview and Source Code for HTML files', async () => {
    cockpit.chat = [
      {
        role: 'agent',
        text: 'หน้าเว็บ',
        time: '12:00',
        producedFiles: ['output/session-1/index.html'],
      },
    ] as any

    vi.mocked(ReadFile).mockResolvedValue('<!DOCTYPE html><html><body>Hello World</body></html>')

    const { container } = render(ArtifactsPane)

    await waitFor(() => {
      expect(screen.getAllByText('index.html').length).toBeGreaterThanOrEqual(1)
    })

    await waitFor(() => {
      expect(container.querySelector('iframe.art-iframe')).toBeTruthy()
    })

    // Toggle to source code
    const codeBtn = screen.getByText('ซอร์สโค้ด')
    await fireEvent.click(codeBtn)

    await waitFor(() => {
      expect(screen.getByText(/Hello World/)).toBeTruthy()
    })
  })

  it('deduplicates files referenced by both chat and disk output scan', async () => {
    cockpit.chat = [
      {
        role: 'agent',
        text: 'สร้างข้อมูล',
        time: '12:00',
        producedFiles: ['users_data.csv'],
      },
    ] as any

    vi.mocked(ListArtifactsIn).mockResolvedValue({
      range: 'all',
      total: 1,
      files: [
        {
          name: 'users_data.csv',
          path: 'C:/Users/phrms/aetox/output/20260910/users_data.csv',
          size: 316,
          sessionId: 'test-session-id',
        } as any,
      ],
    })

    const { container } = render(ArtifactsPane)

    await waitFor(() => {
      const items = Array.from(container.querySelectorAll('.art-item-name')).map((el) => el.textContent?.trim())
      const occurrences = items.filter((name) => name === 'users_data.csv')
      expect(occurrences.length).toBe(1)
    })
  })

  it('renders Plan & Stepper 2.0 with hero progress bar and vertical timeline', async () => {
    cockpit.plan = {
      title: 'ทดสอบส่วนเพิ่มเติม (ตารางข้อมูลและเอกสารคู่มือ)',
      sections: [
        { heading: 'ขั้นตอนการดำเนินการ', body: 'สร้างข้อมูลทดสอบ' },
        { heading: 'จุดที่อาจมีปัญหาหรือความเสี่ยง', body: 'ไฟล์ไม่แสดงในหมวดหมู่' },
        { heading: 'เกณฑ์วัดความสำเร็จ', body: 'ชิ้นงานปรากฏในพาเนล' },
      ],
      steps: [
        { n: 1, text: 'สร้างไฟล์ตารางข้อมูลผู้ใช้งาน `users_data.csv`', state: 'done' },
        { n: 2, text: 'สร้างไฟล์สรุปการใช้งานเพิ่มเติม system_guide.md', state: 'doing' },
      ],
      version: 2,
      updated: '2026-09-10T03:00:00Z',
    }

    const { container } = render(ArtifactsPane)

    await waitFor(() => {
      // Hero Header
      expect(container.querySelector('.hero-title')?.textContent).toContain('ทดสอบส่วนเพิ่มเติม')
      // Progress bar (1 of 2 done = 50%)
      expect(container.querySelector('.progress-percentage')?.textContent).toBe('50%')
      // Section callouts
      expect(container.querySelector('.plan-sec-callout.kind-scope')).toBeTruthy()
      expect(container.querySelector('.plan-sec-callout.kind-risk')).toBeTruthy()
      expect(container.querySelector('.plan-sec-callout.kind-success')).toBeTruthy()
      // Stepper items
      const stepperRows = container.querySelectorAll('.stepper-item-row')
      expect(stepperRows.length).toBe(2)
      // Step code highlight
      expect(container.querySelector('.step-code')?.textContent).toBe('users_data.csv')
    })
  })
})

