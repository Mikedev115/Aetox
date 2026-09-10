import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import FileChangeReview from '../lib/FileChangeReview.svelte'
import Chat from '../lib/Chat.svelte'
import { extractChangedFiles } from '../lib/fileChange'
import { setLocale } from '../lib/i18n.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import * as workbenchStore from '../lib/stores/workbench.svelte'

describe('extractChangedFiles', () => {
  it('extracts unique files and aggregates additions, deletions, and diffs', () => {
    const steps: any[] = [
      {
        name: 'edit',
        subject: '/d:/Aetox/Aetox/desktop/frontend/src/style.css',
        added: 7,
        removed: 1,
        diff: '@@ -1,1 +1,1 @@\n-old\n+new',
      },
      {
        name: 'read',
        subject: '/d:/Aetox/Aetox/desktop/frontend/src/App.svelte',
        range: '1-50',
      },
      {
        name: 'write',
        subject: '/d:/Aetox/Aetox/desktop/frontend/src/style.css',
        added: 3,
        removed: 0,
        diff: '@@ -10,1 +10,1 @@\n+added',
      },
      {
        name: 'replace_file_content',
        subject: 'C:\\Users\\phrms\\project\\main.go',
        added: 12,
        removed: 4,
        diff: '@@ -5,1 +5,1 @@\n-go\n+go2',
      },
    ]

    const files = extractChangedFiles(steps)
    expect(files.length).toBe(2)

    const styleFile = files.find((f) => f.name === 'style.css')
    expect(styleFile).toBeDefined()
    expect(styleFile?.dir).toBe('/d:/Aetox/Aetox/desktop/frontend/src')
    expect(styleFile?.added).toBe(10)
    expect(styleFile?.removed).toBe(1)
    expect(styleFile?.diff).toContain('old')
    expect(styleFile?.diff).toContain('added')

    const mainFile = files.find((f) => f.name === 'main.go')
    expect(mainFile).toBeDefined()
    expect(mainFile?.dir).toBe('C:/Users/phrms/project')
    expect(mainFile?.added).toBe(12)
    expect(mainFile?.removed).toBe(4)
  })
})

describe('FileChangeReview component', () => {
  beforeEach(() => {
    setLocale('en')
  })

  const sampleFiles = [
    {
      path: '/d:/Aetox/Aetox/desktop/frontend/src/style.css',
      name: 'style.css',
      dir: '/d:/Aetox/Aetox/desktop/frontend/src',
      added: 7,
      removed: 1,
      diff: '@@ -10,2 +10,2 @@\n-color: red;\n+color: blue;\n',
    },
  ]

  it('renders file change summary header with counts and review button', () => {
    const { container } = render(FileChangeReview, { files: sampleFiles })

    const count = container.querySelector('.fcr-count')
    expect(count?.textContent).toContain('1 file changed')

    const add = container.querySelector('.fcr-head .fcr-add')
    const del = container.querySelector('.fcr-head .fcr-del')
    expect(add?.textContent).toBe('+7')
    expect(del?.textContent).toBe('-1')

    const reviewBtn = container.querySelector('.fcr-review-btn')
    expect(reviewBtn).not.toBeNull()
    expect(reviewBtn?.textContent).toContain('Review')
  })

  it('renders file row with braces, filename, directory, and stats', async () => {
    const { container } = render(FileChangeReview, { files: sampleFiles })

    // Expand header
    await fireEvent.click(container.querySelector('.fcr-head') as HTMLElement)

    const braces = container.querySelector('.fcr-braces')
    expect(braces?.textContent).toContain('{ }')

    const name = container.querySelector('.fcr-name')
    expect(name?.textContent).toBe('style.css')

    const dir = container.querySelector('.fcr-dir')
    expect(dir?.textContent).toBe('/d:/Aetox/Aetox/desktop/frontend/src')

    const statAdd = container.querySelector('.fcr-file-row .fcr-add')
    const statDel = container.querySelector('.fcr-file-row .fcr-del')
    expect(statAdd?.textContent).toBe('+7')
    expect(statDel?.textContent).toBe('-1')
  })

  it('toggles file diff when clicking file row', async () => {
    const { container } = render(FileChangeReview, { files: sampleFiles })

    // Expand header
    await fireEvent.click(container.querySelector('.fcr-head') as HTMLElement)

    expect(container.querySelector('.fcr-diff')).toBeNull()

    const fileRow = container.querySelector('.fcr-file-row') as HTMLElement
    await fireEvent.click(fileRow)

    expect(container.querySelector('.fcr-diff')).not.toBeNull()
    expect(container.querySelector('.dl.del .tx')?.textContent).toBe('color: red;')
    expect(container.querySelector('.dl.add .tx')?.textContent).toBe('color: blue;')

    await fireEvent.click(fileRow)
    expect(container.querySelector('.fcr-diff')).toBeNull()
  })

  it('calls openGitTab when clicking Review button', async () => {
    const spy = vi.spyOn(workbenchStore, 'openGitTab').mockImplementation(() => {})
    const { container } = render(FileChangeReview, { files: sampleFiles })

    const reviewBtn = container.querySelector('.fcr-review-btn') as HTMLElement
    await fireEvent.click(reviewBtn)

    expect(spy).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('calls openFileTab when clicking open in editor button', async () => {
    const spy = vi.spyOn(workbenchStore, 'openFileTab').mockImplementation(async () => {})
    const { container } = render(FileChangeReview, { files: sampleFiles })

    // Expand header
    await fireEvent.click(container.querySelector('.fcr-head') as HTMLElement)

    const openBtn = container.querySelector('.fcr-open-btn') as HTMLElement
    await fireEvent.click(openBtn)

    expect(spy).toHaveBeenCalledWith(sampleFiles[0].path, sampleFiles[0].name)
    spy.mockRestore()
  })

  it('renders plural files changed for multiple files', () => {
    const multiFiles = [
      ...sampleFiles,
      {
        path: '/d:/Aetox/Aetox/desktop/frontend/src/App.svelte',
        name: 'App.svelte',
        dir: '/d:/Aetox/Aetox/desktop/frontend/src',
        added: 15,
        removed: 3,
        diff: '@@ -1,1 +1,1 @@\n-old\n+new\n',
      },
    ]

    const { container } = render(FileChangeReview, { files: multiFiles })
    const count = container.querySelector('.fcr-count')
    expect(count?.textContent).toContain('2 files changed')

    const add = container.querySelector('.fcr-head .fcr-add')
    const del = container.querySelector('.fcr-head .fcr-del')
    expect(add?.textContent).toBe('+22')
    expect(del?.textContent).toBe('-4')
  })
})

describe('FileChangeReview in Chat message', () => {
  it('renders at the bottom of the message, after markdown prose and before time actions', () => {
    cockpit.desk = 'coding'
    cockpit.chat = []
    const props = {
      messages: [{
        role: 'agent',
        text: 'Finished editing app.css',
        time: '3:09 AM',
        steps: [{
          name: 'edit',
          label: 'edit landing/src/app.css',
          subject: 'landing/src/app.css',
          diff: '@@ -1,1 +1,2 @@\n-old\n+new\n',
          added: 1,
          removed: 1,
        }],
      }] as any,
      task: { title: '', steps: [] } as any,
      awaitingReply: false,
      agentStatus: '',
      toolSteps: [],
      streamingText: '',
      reasoningText: '',
      onSend: () => {},
      onSwitchProvider: async () => {},
      onSwitchThinkLevel: async () => {},
      onSwitchModel: async () => {},
      onCancelPendingModel: async () => {},
      onSubmitAPIKey: async () => {},
      model: { provider: 'deepseek', modelName: 'v4', thinkLevel: 'high', approval: 'ask', wireFormat: '' } as any,
    }

    const { container } = render(Chat, props)
    const card = container.querySelector('.file-change-review')
    expect(card).not.toBeNull()

    const bubble = container.querySelector('.msg.bot .bubble')
    const children = Array.from(bubble?.children ?? [])
    const markdownIdx = children.findIndex((el) => el.classList.contains('markdown-body'))
    const cardIdx = children.findIndex((el) => el.classList.contains('file-change-review'))
    const timeIdx = children.findIndex((el) => el.classList.contains('time'))

    expect(markdownIdx).toBeGreaterThanOrEqual(0)
    expect(cardIdx).toBeGreaterThan(markdownIdx)
    expect(timeIdx).toBeGreaterThan(cardIdx)
  })

  it('renders strictly below revertedFiles note when answer was regenerated', () => {
    cockpit.desk = 'coding'
    cockpit.chat = []
    const props = {
      messages: [{
        role: 'agent',
        text: 'เพิ่มคอมเมนต์เรียบร้อยแล้วครับ',
        time: '3:13 AM',
        revertedFiles: ['landing/src/app.css'],
        steps: Array.from({ length: 10 }, (_, idx) => ({
          name: idx === 9 ? 'edit' : 'read',
          label: idx === 9 ? 'edit landing/src/app.css' : `read step ${idx}`,
          subject: 'landing/src/app.css',
          diff: idx === 9 ? '@@ -1,1 +1,2 @@\n-old\n+new\n' : undefined,
          added: idx === 9 ? 1 : 0,
          removed: 0,
        })),
      }] as any,
      task: { title: '', steps: [] } as any,
      awaitingReply: false,
      agentStatus: '',
      toolSteps: [],
      streamingText: '',
      reasoningText: '',
      onSend: () => {},
      onSwitchProvider: async () => {},
      onSwitchThinkLevel: async () => {},
      onSwitchModel: async () => {},
      onCancelPendingModel: async () => {},
      onSubmitAPIKey: async () => {},
      model: { provider: 'deepseek', modelName: 'v4', thinkLevel: 'high', approval: 'ask', wireFormat: '' } as any,
    }

    const { container } = render(Chat, props)
    const bubble = container.querySelector('.msg.bot .bubble')
    const children = Array.from(bubble?.children ?? [])
    const metaIdx = children.findIndex((el) => el.classList.contains('meta-row'))
    const markdownIdx = children.findIndex((el) => el.classList.contains('markdown-body'))
    const revertedIdx = children.findIndex((el) => el.classList.contains('msg-reverted'))
    const cardIdx = children.findIndex((el) => el.classList.contains('file-change-review'))
    const timeIdx = children.findIndex((el) => el.classList.contains('time'))

    expect(metaIdx).toBeGreaterThanOrEqual(0)
    expect(markdownIdx).toBeGreaterThan(metaIdx)
    expect(revertedIdx).toBeGreaterThan(markdownIdx)
    expect(cardIdx).toBeGreaterThan(revertedIdx)
    expect(timeIdx).toBeGreaterThan(cardIdx)
  })
})
