import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import Projects from '../lib/Projects.svelte'
import {
  CreateSpace, PickSpaceImage, SessionsInSpace, Spaces, UpdateSpaceDescription,
} from './mocks/wailsApp'

const project = {
  name: 'Aetox',
  description: '',
  image: '',
  path: 'C:/data/project/Aetox',
  contextPath: 'C:/data/project/Aetox/context',
  contextFiles: [],
  contextModified: {},
  chats: 2,
  updatedAt: new Date().toISOString(),
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(Spaces).mockResolvedValue([project])
  vi.mocked(SessionsInSpace).mockResolvedValue([])
})

describe('project presentation metadata', () => {
  it('uses an uploaded picture as the project card cover', async () => {
    vi.mocked(Spaces).mockResolvedValue([{ ...project, image: 'data:image/png;base64,cGljdHVyZQ==' }])
    render(Projects, { onClose: () => {} })

    await screen.findByText(project.name, { selector: '.pp-title' })
    expect(document.querySelector('.proj-card-cover > img')).toBeTruthy()
    expect(document.querySelector('.proj-card.pp-card')).toBeNull()
  })

  it('saves the optional description while creating a project', async () => {
    render(Projects, { onClose: () => {} })
    await fireEvent.click(await screen.findByText('สร้างโปรเจกต์', { selector: 'button' }))

    await fireEvent.input(screen.getByPlaceholderText('ชื่อโปรเจกต์ เช่น เปิดร้านกาแฟ'), { target: { value: 'Launch' } })
    await fireEvent.input(screen.getByPlaceholderText('โปรเจกต์นี้มีไว้ทำอะไร'), { target: { value: 'เตรียมเปิดตัวสินค้า' } })
    await fireEvent.click(screen.getByText('สร้าง', { selector: 'button' }))

    await waitFor(() => expect(CreateSpace).toHaveBeenCalledWith('Launch'))
    await waitFor(() => expect(UpdateSpaceDescription).toHaveBeenCalledWith('Launch', 'เตรียมเปิดตัวสินค้า'))
  })

  it('edits the description from the project header', async () => {
    render(Projects, { onClose: () => {} })
    fireEvent.click(await screen.findByText(project.name, { selector: '.pp-title' }))

    fireEvent.click(await screen.findByText(/เพิ่มคำอธิบายสั้น/))
    const field = await screen.findByPlaceholderText('โปรเจกต์นี้มีไว้ทำอะไร')
    await fireEvent.input(field, { target: { value: 'เดสก์ท็อปแอปสำหรับทำงานกับผู้ช่วย' } })
    await fireEvent.click(screen.getByText('บันทึก'))

    await waitFor(() => expect(UpdateSpaceDescription).toHaveBeenCalledWith(
      project.name, 'เดสก์ท็อปแอปสำหรับทำงานกับผู้ช่วย',
    ))
  })

  it('opens the picture picker from the project avatar', async () => {
    vi.mocked(PickSpaceImage).mockResolvedValue('data:image/png;base64,cGljdHVyZQ==')
    render(Projects, { onClose: () => {} })
    fireEvent.click(await screen.findByText(project.name, { selector: '.pp-title' }))

    await fireEvent.click(await screen.findByLabelText('อัปโหลดรูป'))
    await fireEvent.click(await screen.findByText('อัปโหลดรูป'))

    await waitFor(() => expect(PickSpaceImage).toHaveBeenCalledWith(project.name))
    await waitFor(() => expect(document.querySelector('.proj-avatar-button img')).toBeTruthy())
  })
})
