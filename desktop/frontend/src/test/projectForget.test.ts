// The two things you do to a project row once, behind one button.
//
// Owner, 7 ก.ย.: "ทำไมลบโปรเจกต์ไม่ได้หน้าโค้ด" — about a Downloads row the file
// panel had made a project an hour earlier, on a list that only ever grew. Then,
// once the way off existed: "ควรจะห่อไว้ใน สามจุดนะ ทั้งปักหมุดและลบ" and
// "หน้าผู้ช่วยอีก".
//
// So both doors' rows carry the same three dots. What ลบ *means* is where they
// differ, and the tests say so: the workshop's row forgets a folder (two clicks,
// nothing on disk touched), the storefront's deletes a โปรเจกต์ whose folder
// holds copies of the user's files — through the app's one dialog, which is the
// door the โปรเจกต์ page already uses.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte'
import Sidebar from '../lib/Sidebar.svelte'
import { ForgetProject, DeleteSpace, CurrentSessionID, SessionMode, UpdateProjectMeta } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setShell } from '../lib/shell.svelte'
import { setLocale } from '../lib/i18n.svelte'

const project = { key: 'k-downloads', name: 'Downloads', path: 'C:\\Users\\phrms\\Downloads', ago: '' }
const space = { name: 'เปิดร้านกาแฟ', chats: 2, updatedAt: new Date().toISOString() }

const moreButton = () => screen.getByLabelText('More')
const openMenu = async (projectFirst = true) => {
  if (projectFirst) await fireEvent.click(screen.getByText('All projects'))
  await fireEvent.click(moreButton())
}

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  setShell('code')
  cockpit.desk = ''
  cockpit.activeView = 'chat'
  cockpit.history.length = 0
  cockpit.projects = [project]
  cockpit.spaces = []
  cockpit.project.focused = false
  cockpit.project.path = ''
  vi.mocked(CurrentSessionID).mockResolvedValue('20260907-120000.000')
  vi.mocked(SessionMode).mockResolvedValue('')
})

describe('a project row in the workshop', () => {
  it('keeps management actions off the recent-project picker', () => {
    render(Sidebar, { onOpenSettings: () => {} })
    expect(screen.queryByLabelText('More')).toBeNull()
    expect(screen.queryByText('Remove from the list')).toBeNull()
  })

  it('offers editing and the way off from the all-projects view', async () => {
    render(Sidebar, { onOpenSettings: () => {} })
    await fireEvent.click(screen.getByText('All projects'))
    expect(screen.getByLabelText('Edit name and description')).toBeTruthy()
    await fireEvent.click(moreButton())
    expect(screen.getByText('Remove from the list')).toBeTruthy()
  })

  it('saves a display name and description without changing the folder path', async () => {
    vi.mocked(UpdateProjectMeta).mockResolvedValueOnce({
      key: project.key, name: 'Client portal', folder: 'Downloads',
      description: 'Checkout redesign', rootPath: project.path,
      openedAt: '', sessions: 0, snippet: '',
    })
    render(Sidebar, { onOpenSettings: () => {} })
    await fireEvent.click(screen.getByText('All projects'))
    await fireEvent.click(screen.getByLabelText('Edit name and description'))
    await fireEvent.input(screen.getByLabelText('Display name'), { target: { value: 'Client portal' } })
    await fireEvent.input(screen.getByLabelText('Project description'), { target: { value: 'Checkout redesign' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(UpdateProjectMeta).toHaveBeenCalledWith(
      project.path, 'Client portal', 'Checkout redesign',
    ))
    expect(cockpit.projects[0]?.path).toBe(project.path)
    expect(cockpit.projects[0]?.name).toBe('Client portal')
  })

  it('asks once before it removes anything', async () => {
    render(Sidebar, { onOpenSettings: () => {} })
    await openMenu()
    await fireEvent.click(screen.getByText('Remove from the list'))
    expect(ForgetProject).not.toHaveBeenCalled()

    await fireEvent.click(screen.getByText('Sure?'))
    await waitFor(() => expect(ForgetProject).toHaveBeenCalledWith(project.path))
  })
})

describe('a โปรเจกต์ row at the storefront', () => {
  beforeEach(() => {
    setShell('assistant')
    cockpit.projects = []
    cockpit.spaces = [space]
  })

  it('carries the same three dots', async () => {
    render(Sidebar, { onOpenSettings: () => {} })
    await openMenu(false)
    expect(screen.getByText('Pin to the top')).toBeTruthy()
    expect(screen.getByText('Delete project')).toBeTruthy()
  })

  // A folder of copies of the user's files is not a list entry, and one click
  // must not take it.
  it('asks through the dialog before deleting the folder', async () => {
    render(Sidebar, { onOpenSettings: () => {} })
    await openMenu(false)
    await fireEvent.click(screen.getByText('Delete project'))
    expect(DeleteSpace).not.toHaveBeenCalled()
    expect(screen.getByText('Delete this project?')).toBeTruthy()

    await fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(DeleteSpace).toHaveBeenCalledWith(space.name))
  })
})
