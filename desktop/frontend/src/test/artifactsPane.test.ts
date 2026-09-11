// The layout half of this pane is pinned here rather than in a rendering test,
// for the reason composerNarrow.test.ts reads its sheet off disk: jsdom has no
// layout at all, so a rule checked against a rendered component passes whatever
// the rule says. What that protects is the failure the owner hit by hand — the
// pane drawn at the width it is allowed to be, list 220px and stage 100px, with
// a file clicked and nothing to show for it.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import ArtifactsPane from '../lib/workbench/ArtifactsPane.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { workbench } from '../lib/stores/workbench.svelte'
import {
  SessionEdits, ListArtifactsForSession, ReadFile,
} from './mocks/wailsApp'

const paneSrc = readFileSync('src/lib/workbench/ArtifactsPane.svelte', 'utf8')
/** The declarations of the rule whose selector matches exactly.
 *
 *  Anchored on the two-space indent a component's own rules carry, because
 *  `.art-list` and `.art-stage` each appear twice: once here and once inside
 *  the `@container` block, four spaces in. Asking for "the first match" answers
 *  with the narrow-layout copy of the rule and the test then passes or fails
 *  about the wrong line. */
function rule(selector: string): string {
  const re = new RegExp(`^  ${selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*\\{([^}]*)\\}`, 'm')
  const m = paneSrc.match(re)
  if (!m) throw new Error(`no rule for ${selector}`)
  return m[1]
}

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.plan = null
  cockpit.chat = []
  workbench.tabs = []
  workbench.activeId = ''
  // The chat on screen. The pane reads it from the store rather than asking the
  // engine for it (see the note at the call site), so a test states it here.
  cockpit.openSession = 'session-1'
  vi.mocked(SessionEdits).mockResolvedValue({ files: [], total: 0 })
  vi.mocked(ListArtifactsForSession).mockResolvedValue({ files: [], range: 'all', total: 0 } as any)
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

  // SOURCE CODE IS NOT AN ARTIFACT.
  //
  // A session that works on this repository edits .go and .svelte files all day,
  // and every one of them used to land in this list — the turn's produced files
  // and the session's edits both carry them — where they buried the plan, the
  // walkthrough and the pictures under a directory listing. The owner, looking
  // at exactly that (11 ก.ย.): *"ตรงนี้ไม่ควรแสดงโค้ดสิ มันควรแสดงแค่ ไฟล์
  // ตัวอย่าง หรือ แผน หรืออะไรพวกนี้"*.
  //
  // What it edits has two homes already — the turn draws those files with their
  // diff, and the session strip lists them next to the room's sources — so the
  // rule is about what this pane is FOR: a thing Aetox made.
  it('leaves source code out, whatever brought it in', async () => {
    cockpit.chat = [
      {
        role: 'agent',
        text: 'แก้ให้แล้วครับ',
        time: '12:00',
        producedFiles: [
          'desktop/app.go',
          'src/lib/Chat.svelte',
          'docs/walkthrough.md',
          'scratch/mockup.html',
          'deck/สรุปงาน.pptx',
        ],
      },
    ] as any
    vi.mocked(SessionEdits).mockResolvedValue({
      files: [
        { path: 'desktop/plan.go', label: 'plan.go', dir: 'desktop', gone: false },
        { path: 'package.json', label: 'package.json', dir: '', gone: false },
        { path: 'docs/notes.md', label: 'notes.md', dir: 'docs', gone: false },
      ],
      total: 3,
    } as any)

    const { container } = render(ArtifactsPane)
    await waitFor(() => expect(container.querySelector('.art-item')).toBeTruthy())

    const names = Array.from(container.querySelectorAll('.art-item-name')).map((el) => el.textContent?.trim())
    // The deliverables, from both paths.
    for (const kept of ['walkthrough.md', 'mockup.html', 'สรุปงาน.pptx', 'notes.md']) {
      expect(names).toContain(kept)
    }
    // The work itself, from both paths. A .pptx is kept above on purpose: a file
    // this pane cannot preview is still a thing Aetox made, which is not the
    // same answer as the project's own source.
    for (const gone of ['app.go', 'Chat.svelte', 'plan.go', 'package.json']) {
      expect(names).not.toContain(gone)
    }
    // And a chat that produced nothing but code reads as a chat with nothing to
    // show, rather than as a list of the files it was asked to change.
    expect(names.length).toBe(4)
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

    vi.mocked(ListArtifactsForSession).mockResolvedValue({
      range: 'all',
      total: 1,
      files: [
        {
          name: 'users_data.csv',
          path: 'C:/Users/phrms/aetox/output/20260910/users_data.csv',
          size: 316,
          sessionId: 'session-1',
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

// TWO PANES, ONE COLUMN WHEN THERE IS NO ROOM FOR TWO.
//
// The failure, in the owner's words on 11 ก.ย.: *"ในนี้กดไม่ได้ ผมดูไม่ได้"*,
// with a screenshot of this list — a file clicked, and no preview to show for
// it. The arithmetic is the whole story: the list is a fixed 220px and the
// stage was `flex:1`, while App.svelte lets this pane be 320px wide, so the
// preview was a ~100px column. Everything "worked" and there was nothing to
// see.
//
// What is pinned here is the pair of numbers that stops it happening again:
// the stage's basis (the width below which a preview is not worth looking at,
// and therefore the width at which the layout wraps) and the two heights that
// keep the wrapped lines from splitting the pane between them.
describe('the artifacts pane at the width it is allowed to be', () => {
  it('wraps the stage under the list rather than squeezing it', () => {
    expect(rule('.art-pane')).toContain('flex-wrap: wrap')
    // The basis is the wrap point: 220 (list) + 340 (stage) = 560.
    expect(rule('.art-stage')).toMatch(/flex: ?1 1 340px/)
  })

  it('measures itself rather than the window, since a drag handle sets its width', () => {
    const pane = rule('.art-pane')
    expect(pane).toContain('container-type: inline-size')
    expect(pane).toContain('container-name: artifacts')
    // A named container, so the query cannot be answered by some other
    // container that happens to be added around this pane later.
    expect(paneSrc).toContain('@container artifacts (max-width: 559px)')
  })

  it('gives the stage every pixel the band does not take', () => {
    // No max-height here: two flex lines left to size themselves split the
    // pane's free space equally, which would hand the list half a pane for rows
    // it is not showing. Both heights are definite, so there is nothing to split.
    const band = paneSrc.slice(paneSrc.indexOf('@container artifacts'))
    expect(band).toContain('height: min(240px, 42%)')
    expect(band).toContain('height: calc(100% - min(240px, 42%))')
  })

  it('asks for the files of one chat, not every file on the machine', async () => {
    // The whole point of the session-scoped binding: the rows this pane keeps
    // were always the rows in `output/<id>`, and the walk that found them was
    // every root this install knows. The pane still drops a row whose sessionId
    // is not the chat on screen, because a load can land after a switch.
    render(ArtifactsPane)
    await waitFor(() => expect(ListArtifactsForSession).toHaveBeenCalled())

    expect(vi.mocked(ListArtifactsForSession).mock.calls.at(-1)?.[0]).toBe('session-1')
  })

  it('reads nothing at all while it is behind another tab', async () => {
    // The pane stays mounted behind the tab in front, so its effect used to run
    // for the whole turn — every message and every plan step, each one a sweep,
    // for a list with no reader (owner, 11 ก.ย.: "ทำไมเปิดอยู่ถึงรอนานและกระตุก").
    render(ArtifactsPane, { props: { active: false } })
    await new Promise((r) => setTimeout(r, 20))

    expect(vi.mocked(ListArtifactsForSession)).not.toHaveBeenCalled()
    expect(vi.mocked(SessionEdits)).not.toHaveBeenCalled()
  })

  it('keeps the percentage the layout is built on honest', () => {
    // The list's width and the inspector's floor are what make the wrap happen
    // at a width this pane really reaches; a change to either is a change to
    // this whole rule, and this is the line that says so.
    expect(rule('.art-list')).toContain('width: 220px')
    const app = readFileSync('src/App.svelte', 'utf8')
    expect(app).toMatch(/inspector:[\s\S]{0,120}?min: 320/)
  })
})

