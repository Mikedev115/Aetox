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
import { ReadFile, SessionEdits, ListArtifactsForSession } from './mocks/wailsApp'
import type { PlanReport } from '../lib/types'

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

const names = (container: HTMLElement) =>
  Array.from(container.querySelectorAll('.art-item-name')).map((el) => el.textContent?.trim())

const report = (over: Partial<PlanReport> = {}): PlanReport => ({
  run: 1,
  planVersion: 1,
  title: 'ย้ายพาเนลชิ้นงานให้เหลือแผนกับรายงาน',
  sections: [
    { heading: 'What was done', body: 'ตัดการกวาดโฟลเดอร์ output ออก' },
    { heading: 'How it was checked', body: 'go test ./desktop ผ่าน 214' },
  ],
  done: 3, failed: 0, total: 3,
  elapsedSecs: 840, sentBack: 1,
  at: '2026-09-12T10:42:00+07:00',
  ...over,
})

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.plan = null
  cockpit.planReports = []
  cockpit.chat = []
  workbench.tabs = []
  workbench.activeId = ''
  // The chat on screen. The pane reads it from the store rather than asking the
  // engine for it (see the note at the call site), so a test states it here.
  cockpit.openSession = 'session-1'
  vi.mocked(ReadFile).mockResolvedValue('')
})

describe('ArtifactsPane', () => {
  it('shows the empty state when nothing has been handed over', async () => {
    render(ArtifactsPane)
    await waitFor(() => {
      expect(screen.getByText('แชตนี้ยังไม่มีชิ้นงาน')).toBeTruthy()
    })
  })

  it('renders the plan as the first row when cockpit.plan exists', async () => {
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
      expect(screen.getByText(/1\/2 ขั้น/)).toBeTruthy()
    })
  })

  // ONE REPORT PER ROUND, NEWEST FIRST. The row says what the round came to
  // without opening it — a finished round by its steps, a held round by the
  // hold's own words — and the stage draws the report as a card of the plan's
  // own family, numbers first and the model's words under them.
  it('lists every round of the plan as a report, newest first, and draws the chosen one', async () => {
    cockpit.plan = { title: 'ย้ายพาเนล', sections: [{ heading: 'What to change', body: 'x' }], steps: [], version: 3, updated: '' } as any
    cockpit.planReports = [
      report({ run: 1, done: 1, failed: 0, total: 3, stopped: 'รอก่อนทำข้อ 2', elapsedSecs: 120 }),
      report({ run: 2, done: 2, failed: 1, total: 3 }),
    ]

    const { container } = render(ArtifactsPane)
    await waitFor(() => expect(names(container)).toEqual(['ย้ายพาเนล', 'รอบที่ 2', 'รอบที่ 1']))

    const metas = Array.from(container.querySelectorAll('.art-item-meta')).map((el) => el.textContent?.trim() ?? '')
    expect(metas[1]).toContain('3/3 ขั้น')
    expect(metas[1]).toContain('ล้ม 1')
    expect(metas[1]).toContain('14 นาที')
    expect(metas[2]).toContain('รอก่อนทำข้อ 2')

    const rows = container.querySelectorAll('.art-item')
    await fireEvent.click(rows[1])
    await waitFor(() => {
      expect(container.querySelector('.report-hero')).toBeTruthy()
      expect(screen.getByText('ตรวจอย่างไร ผลเป็นอย่างไร')).toBeTruthy()
      expect(screen.getByText(/go test \.\/desktop/)).toBeTruthy()
      // The engine's numbers, not the model's words: the round failed a step,
      // so the badge is not "finished as planned".
      expect(screen.queryByText('เสร็จตามแผน')).toBeNull()
      expect(screen.getByText('ล้ม 1')).toBeTruthy()
    })
  })

  // WHAT A TURN HANDED OVER, SORTED — AND NOTHING ELSE.
  //
  // The signal is the flag a tool set (skill.Output.Artifacts), and the pane
  // sorts what it flagged into documents, decks and pages. A picture, a sheet,
  // and source code are not listed however they arrived: each has a home
  // already, and this pane was the third copy that buried the plan (owner, 12
  // ก.ย., with a screenshot of twelve rows: "มันไม่ควรแสดงทุกอย่างดิตรงนี้").
  it('sorts the handed-over files into documents, slides and pages, and lists nothing else', async () => {
    cockpit.chat = [
      {
        role: 'agent',
        text: 'ทำเสร็จแล้วครับ',
        time: '12:00',
        producedFiles: [
          'output/session-1/findings.md',
          'output/session-1/สรุปไตรมาส.html',
          'output/session-1/dashboard.html',
          'output/session-1/work/page-1.png',
          'output/session-1/hero.png',
          'output/session-1/book.xlsx',
          'desktop/app.go',
          'output/session-1/brief.docx',
        ],
      },
    ] as any
    vi.mocked(ReadFile).mockImplementation(async (path: string) => {
      if (path.endsWith('สรุปไตรมาส.html')) return '<!doctype html><html><body><section class="slide">1</section><section class="slide">2</section></body></html>'
      if (path.endsWith('dashboard.html')) return '<!doctype html><html><body><h1>Dashboard</h1></body></html>'
      if (path.endsWith('.md')) return '# สรุป\n\nใช้งานได้ดี'
      return ''
    })

    const { container } = render(ArtifactsPane)
    await waitFor(() => expect(names(container).length).toBe(4))

    // Grouped in the pane's order, each group only when it has something.
    const heads = Array.from(container.querySelectorAll('.art-group-head')).map((el) => el.textContent?.trim().replace(/\s*\d+$/, ''))
    expect(heads).toEqual(['เอกสาร', 'สไลด์', 'หน้าเว็บ'])
    expect(names(container)).toEqual(['findings.md', 'brief.docx', 'สรุปไตรมาส.html', 'dashboard.html'])
    for (const gone of ['page-1.png', 'hero.png', 'book.xlsx', 'app.go']) {
      expect(names(container)).not.toContain(gone)
    }
    // The deck knows it is one, and says how long it is.
    expect(screen.getByText('2 สไลด์')).toBeTruthy()

    // A document opens on the stage.
    const rows = Array.from(container.querySelectorAll('.art-item')) as HTMLElement[]
    await fireEvent.click(rows[0])
    await waitFor(() => expect(screen.getByText('สรุป')).toBeTruthy())
  })

  // A DECK AND A PAGE ARE SENT OFF, NOT PREVIEWED. Each already has a room
  // that can show it properly; a second preview inside a 320px pane was the
  // thing the owner could not read. The row says where it goes and goes there,
  // and the stage keeps whatever it was showing.
  it('sends a deck to the slides room and a page to the browser instead of drawing them', async () => {
    cockpit.chat = [
      { role: 'agent', text: 'x', time: '12:00', producedFiles: ['output/session-1/notes.md', 'output/session-1/deck.html', 'output/session-1/page.html'] },
    ] as any
    vi.mocked(ReadFile).mockImplementation(async (path: string) => {
      if (path.endsWith('deck.html')) return '<!doctype html><html><body><section class="slide">1</section></body></html>'
      if (path.endsWith('notes.md')) return 'notes'
      return '<!doctype html><html><body>page</body></html>'
    })

    const { container } = render(ArtifactsPane)
    await waitFor(() => expect(names(container)).toEqual(['notes.md', 'deck.html', 'page.html']))
    const tags = Array.from(container.querySelectorAll('.art-handoff')).map((el) => el.textContent?.trim())
    expect(tags).toEqual(['ห้องสไลด์', 'เบราว์เซอร์'])

    const rows = Array.from(container.querySelectorAll('.art-item')) as HTMLElement[]
    await fireEvent.click(rows[1])
    await waitFor(() => expect(workbench.tabs.some((t) => t.kind === 'file' && t.path === 'output/session-1/deck.html')).toBe(true))
    await fireEvent.click(rows[2])
    await waitFor(() => expect(workbench.tabs.some((t) => t.kind === 'browser' && (t.url ?? '').includes('page.html'))).toBe(true))
    // The stage still shows the document; a send-off row is never "chosen".
    expect(container.querySelector('.art-item.active')?.textContent).toContain('notes.md')
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

  // NO SWEEP, NO SESSION EDITS. The two sources that used to fill this list
  // with every file in output/<session> and every file the session touched
  // are not asked at all — not filtered afterwards, not asked. What is here
  // came through the flag or the database.
  it('never sweeps the output folder or the session edits', async () => {
    cockpit.chat = [{ role: 'agent', text: 'x', time: '12:00', producedFiles: ['output/session-1/notes.md'] }] as any
    const { container } = render(ArtifactsPane)
    await waitFor(() => expect(names(container)).toEqual(['notes.md']))
    expect(vi.mocked(ListArtifactsForSession)).not.toHaveBeenCalled()
    expect(vi.mocked(SessionEdits)).not.toHaveBeenCalled()
  })

  it('reads nothing at all while it is behind another tab', async () => {
    // The pane stays mounted behind the tab in front, so its effect used to run
    // for the whole turn — every message and every plan step, each one a read,
    // for a list with no reader (owner, 11 ก.ย.: "ทำไมเปิดอยู่ถึงรอนานและกระตุก").
    cockpit.chat = [{ role: 'agent', text: 'x', time: '12:00', producedFiles: ['output/session-1/deck.html'] }] as any
    render(ArtifactsPane, { props: { active: false } })
    await new Promise((r) => setTimeout(r, 20))

    expect(vi.mocked(ReadFile)).not.toHaveBeenCalled()
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
// and therefore the width at which the layout turns into a column) and the
// band's cap, which is what lets a short list show every row rather than a
// fixed 240px with the send-off rows below the fold.
describe('the artifacts pane at the width it is allowed to be', () => {
  it('turns into a column rather than squeezing the stage', () => {
    // The basis is the turning point: 220 (list) + 340 (stage) = 560.
    expect(rule('.art-stage')).toMatch(/flex: ?1 1 340px/)
    const narrow = paneSrc.slice(paneSrc.indexOf('@container artifacts'))
    expect(narrow).toContain('flex-direction: column')
  })

  it('measures itself rather than the window, since a drag handle sets its width', () => {
    const pane = rule('.art-pane')
    expect(pane).toContain('container-type: inline-size')
    expect(pane).toContain('container-name: artifacts')
    // A named container, so the query cannot be answered by some other
    // container that happens to be added around this pane later.
    expect(paneSrc).toContain('@container artifacts (max-width: 559px)')
  })

  it('lets the band size to its rows, up to half the pane', () => {
    // Sized to content with a cap, not a fixed height: a list of six rows is
    // six rows tall, and the stage takes every pixel under it. A fixed band
    // cut the last rows off — the deck and the page, the rows this list is
    // now for.
    const narrow = paneSrc.slice(paneSrc.indexOf('@container artifacts'))
    expect(narrow).toContain('max-height: 52%')
    expect(narrow).toMatch(/\.art-stage \{[^}]*flex: 1 1 0/)
  })

  it('keeps the percentage the layout is built on honest', () => {
    // The list's width and the inspector's floor are what make the column
    // happen at a width this pane really reaches; a change to either is a
    // change to this whole rule, and this is the line that says so.
    expect(rule('.art-list')).toContain('width: 220px')
    const app = readFileSync('src/App.svelte', 'utf8')
    expect(app).toMatch(/inspector:[\s\S]{0,120}?min: 320/)
  })
})
