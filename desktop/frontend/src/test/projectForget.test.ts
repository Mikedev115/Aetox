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
import { ForgetProject, DeleteSpace, CurrentSessionID, SessionMode } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setShell } from '../lib/shell.svelte'
import { setLocale } from '../lib/i18n.svelte'

const project = { key: 'k-downloads', name: 'Downloads', path: 'C:\\Users\\phrms\\Downloads', ago: '' }
const space = { name: 'เปิดร้านกาแฟ', chats: 2, updatedAt: new Date().toISOString() }

const moreButton = () => screen.getByLabelText('More')
const openMenu = async () => {
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
  it('keeps its actions behind one button until asked', () => {
    render(Sidebar, { onOpenSettings: () => {} })
    expect(moreButton()).toBeTruthy()
    expect(screen.queryByText('Remove from the list')).toBeNull()
    expect(screen.queryByText('Pin to the top')).toBeNull()
  })

  it('offers pin and the way off the list, in that order', async () => {
    render(Sidebar, { onOpenSettings: () => {} })
    await openMenu()
    expect(screen.getByText('Pin to the top')).toBeTruthy()
    expect(screen.getByText('Remove from the list')).toBeTruthy()
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
    await openMenu()
    expect(screen.getByText('Pin to the top')).toBeTruthy()
    expect(screen.getByText('Delete project')).toBeTruthy()
  })

  // A folder of copies of the user's files is not a list entry, and one click
  // must not take it.
  it('asks through the dialog before deleting the folder', async () => {
    render(Sidebar, { onOpenSettings: () => {} })
    await openMenu()
    await fireEvent.click(screen.getByText('Delete project'))
    expect(DeleteSpace).not.toHaveBeenCalled()
    expect(screen.getByText('Delete this project?')).toBeTruthy()

    await fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(DeleteSpace).toHaveBeenCalledWith(space.name))
  })
})
