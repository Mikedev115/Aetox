// The file tree wearing git's answer.
//
// The badges were already there and the owner's screen showed none of them: the
// tree opens collapsed, every changed file sat inside a folder, and a folder
// carried no mark. So the marks have to survive the state the panel is actually
// in — a folder saying something inside it changed, a file saying what happened
// and how much (owner, 7 ก.ย.: "แบบไฟล์ไหนแก้อะไรยังไง").
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte'
import FilesPane from '../lib/workbench/FilesPane.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { BrowseFolder, OpenProjectFolder, StopBrowsing } from './mocks/wailsApp'
import type { TreeNode } from '../lib/types'

const tree = (nodes: TreeNode[]) => {
  cockpit.tree = nodes
}

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  cockpit.tree = []
  cockpit.browseRoot = ''
  cockpit.project.focused = true
})

describe('the file tree', () => {
  it('says how much a changed file changed', () => {
    tree([{ label: 'app.go', path: 'app.go', kind: 'file', depth: 0, status: 'M', added: 12, removed: 3 }])
    render(FilesPane)
    expect(screen.getByText('M')).toBeTruthy()
    expect(screen.getByText('+12')).toBeTruthy()
    expect(screen.getByText('−3')).toBeTruthy()
  })

  it('gives a new file its letter and no counts', () => {
    tree([{ label: 'new.md', path: 'new.md', kind: 'file', depth: 0, status: 'U' }])
    const { container } = render(FilesPane)
    expect(screen.getByText('U')).toBeTruthy()
    expect(container.querySelector('.add')).toBeNull()
    expect(container.querySelector('.rem')).toBeNull()
  })

  it('marks a collapsed folder, which is the state the tree lives in', () => {
    tree([
      { label: 'backend', path: 'backend', kind: 'dir', depth: 0, status: 'M' },
      { label: 'docs', path: 'docs', kind: 'dir', depth: 0, status: 'U' },
      { label: 'scripts', path: 'scripts', kind: 'dir', depth: 0 },
    ])
    const { container } = render(FilesPane)
    expect(container.querySelector('.dot.m')).toBeTruthy()
    expect(container.querySelector('.dot.u')).toBeTruthy()
    // A folder with nothing to say says nothing: three folders, two dots.
    expect(container.querySelectorAll('.dot').length).toBe(2)
  })

  it('tints the name as well as the mark at the end of the row', () => {
    tree([
      { label: 'app.go', path: 'app.go', kind: 'file', depth: 0, status: 'M' },
      { label: 'docs', path: 'docs', kind: 'dir', depth: 0, status: 'U' },
    ])
    const { container } = render(FilesPane)
    expect(container.querySelector('.lbl.m')?.textContent).toBe('app.go')
    expect(container.querySelector('.lbl.u')?.textContent).toBe('docs')
  })

  it('leaves an unchanged file unmarked', () => {
    tree([{ label: 'README.md', path: 'README.md', kind: 'file', depth: 0 }])
    const { container } = render(FilesPane)
    expect(container.querySelector('.gitmark')).toBeNull()
    expect(container.querySelector('.lbl.m, .lbl.u, .lbl.d')).toBeNull()
  })
})

// Looking at a folder, without moving into it.
//
// The panel is empty on the ผู้ช่วย door by design — the assistant is tied to no
// project — and the one button it offered was เปิดโฟลเดอร์, which focuses the
// engine on the folder, retargets what a new chat is born into, and files it in
// recent projects for good. The owner pressed it on 7 ก.ย. and the app kept the
// folder; worse, his chat's reach shrank from the whole machine to Downloads
// with nothing on screen saying so.
describe('the panel with no project', () => {
  beforeEach(() => {
    cockpit.project.focused = false
  })

  it('does not offer to open a project', () => {
    render(FilesPane)
    expect(screen.queryByText('Open Folder')).toBeNull()
    expect(screen.getByText(/not tied to a project/i)).toBeTruthy()
  })

  it('browses instead, which opens nothing', async () => {
    render(FilesPane)
    await fireEvent.click(screen.getByText(/Look at a folder/i))
    await waitFor(() => expect(BrowseFolder).toHaveBeenCalled())
    expect(OpenProjectFolder).not.toHaveBeenCalled()
  })

  it('says the folder is only being looked at, and offers the way back', async () => {
    cockpit.browseRoot = 'C:\\Users\\phrms\\Downloads'
    render(FilesPane)
    expect(screen.getByText('Downloads')).toBeTruthy()
    expect(screen.getByText(/not opened as a project/i)).toBeTruthy()

    await fireEvent.click(screen.getByLabelText(/Stop looking/i))
    await waitFor(() => expect(StopBrowsing).toHaveBeenCalled())
  })

  it('keeps the project door on a focused project that happens to be empty', () => {
    cockpit.project.focused = true
    render(FilesPane)
    expect(screen.getByText('Open Folder')).toBeTruthy()
  })
})
