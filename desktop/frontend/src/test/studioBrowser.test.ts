import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import StudioBrowser from '../lib/StudioBrowser.svelte'
import { StudioAssets, StudioThumbs, StudioSetKind, StudioSetHidden } from './mocks/wailsApp'

vi.mock('../../wailsjs/go/main/App', () => import('./mocks/wailsApp'))
vi.mock('../../wailsjs/runtime/runtime', () => import('./mocks/wailsRuntime'))

const row = (id: string, ext: string, kind: string, extra: Record<string, unknown> = {}) => ({
  id, name: id, path: `X/${id}.${ext}`, kind, category: 'X', duration: 1.2, width: 0, height: 0,
  alpha: false, bytes: 10, ext, url: `/aetox-shelf/lib/${id}.${ext}`, library: 'L', libraryId: 'lib',
  thumb: '', playable: ['mp4', 'webm', 'gif', 'png', 'jpg', 'wav', 'mp3'].includes(ext), ...extra,
})

// The browser asks the engine the agent's own question and draws what comes
// back by medium: a sound is a row with a play button, a clip a <video>, a
// still an <img>; transparency gets its badge. The page count comes from the
// engine, and paging asks for the next one.
describe('StudioBrowser', () => {
  beforeEach(() => {
    vi.mocked(StudioAssets).mockReset()
    vi.mocked(StudioAssets).mockResolvedValue({
      rows: [
        row('whoosh', 'wav', 'sfx'),
        row('grid', 'mp4', 'background', { width: 1080, height: 1920, duration: 12.3, thumb: '/aetox-shelf/thumb/grid.jpg' }),
        row('burn', 'mov', 'overlay', { alpha: true }),
        row('rocket', 'png', 'icon', { duration: 0, width: 512, height: 512 }),
      ],
      total: 698, page: 1, pages: 12,
      categories: [{ name: 'FILM BURN', count: 300 }, { name: 'X', count: 4 }],
    } as any)
  })

  it('opens on the kind it was given and draws each medium its own way', async () => {
    const { container } = render(StudioBrowser, { kind: 'overlay', library: '', onClose: () => {} })
    await waitFor(() => expect(vi.mocked(StudioAssets)).toHaveBeenCalled())
    const q = vi.mocked(StudioAssets).mock.calls.at(-1)![0] as any
    expect(q.kind).toBe('overlay')
    expect(q.page).toBe(1)

    const tiles = await waitFor(() => {
      const found = container.querySelectorAll('.sb-tile')
      expect(found.length).toBe(4)
      return Array.from(found)
    })
    expect(tiles[0].querySelector('.sb-play')).toBeTruthy()
    expect(tiles[0].classList.contains('sound')).toBe(true)
    // display: the clip with a rendered poster shows it — and NOT the file.
    expect(tiles[1].querySelector('img.sb-poster')?.getAttribute('src')).toBe('/aetox-shelf/thumb/grid.jpg')
    expect(tiles[1].querySelector('video')).toBeNull()
    expect(tiles[1].textContent).toContain('1080×1920')
    expect(tiles[2].classList.contains('alpha')).toBe(true)
    expect(tiles[2].textContent).toContain('alpha')
    // render: posters not made yet are asked for once, for exactly those rows,
    // and the tile shows its kind's icon (pulsing) meanwhile.
    expect(tiles[2].querySelector('.sb-kind-mark.pending')).toBeTruthy()
    expect(tiles[3].querySelector('.sb-kind-mark.pending')).toBeTruthy()
    await waitFor(() => expect(vi.mocked(StudioThumbs)).toHaveBeenCalledWith(['burn', 'rocket']))
    // load: hovering a playable clip mounts the real file; a ProRes .mov
    // (not playable) mounts nothing — its poster is all it shows.
    await fireEvent.mouseEnter(tiles[1])
    await waitFor(() => expect(tiles[1].querySelector('video.sb-live')?.getAttribute('src')).toBe('/aetox-shelf/lib/grid.mp4'))
    await fireEvent.mouseLeave(tiles[1])
    await waitFor(() => expect(tiles[1].querySelector('video.sb-live')).toBeNull())
    await fireEvent.mouseEnter(tiles[2])
    await new Promise((r) => setTimeout(r, 20))
    expect(tiles[2].querySelector('.sb-live')).toBeNull()

    // The count and the pager come from the engine's totals.
    expect(container.querySelector('.sb-count')?.textContent).toContain('698')
    expect(container.querySelector('.sb-pages')?.textContent).toContain('1 / 12')
    // A mixed page is a grid, not the sounds list.
    expect(container.querySelector('.sb-grid.sounds')).toBeNull()
  })

  it('pages, filters by folder, and searches with the same query the agent uses', async () => {
    const { container } = render(StudioBrowser, { kind: '', library: 'lib', onClose: () => {} })
    await waitFor(() => expect(container.querySelectorAll('.sb-tile').length).toBe(4))

    const next = Array.from(container.querySelectorAll('button')).find((b) => b.closest('.sb-pages') && !b.hasAttribute('disabled'))!
    await fireEvent.click(next)
    await waitFor(() => expect((vi.mocked(StudioAssets).mock.calls.at(-1)![0] as any).page).toBe(2))

    const select = container.querySelector('.sb-cat') as HTMLSelectElement
    await fireEvent.change(select, { target: { value: 'FILM BURN' } })
    await waitFor(() => {
      const q = vi.mocked(StudioAssets).mock.calls.at(-1)![0] as any
      expect(q.category).toBe('FILM BURN')
      expect(q.page).toBe(1) // a new filter starts over
      expect(q.library).toBe('lib')
    })

    const input = container.querySelector('.sb-search input') as HTMLInputElement
    await fireEvent.input(input, { target: { value: 'whoosh' } })
    await waitFor(() => expect((vi.mocked(StudioAssets).mock.calls.at(-1)![0] as any).text).toBe('whoosh'))
  })

  // Loading, the two ways it can go wrong: the grid must not flash empty
  // between pages, and a slow answer to an old question must not land on top
  // of the answer to the current one.
  it('keeps the old rows dimmed while the next page loads, and drops stale answers', async () => {
    const page1 = { rows: [row('a', 'wav', 'sfx')], total: 2, page: 1, pages: 2, categories: [] }
    const page2 = { rows: [row('b', 'wav', 'sfx')], total: 2, page: 2, pages: 2, categories: [] }
    let release: (v: any) => void = () => {}
    vi.mocked(StudioAssets).mockReset()
    vi.mocked(StudioAssets)
      .mockResolvedValueOnce(page1 as any)
      .mockImplementationOnce(() => new Promise((res) => { release = res }))
    const { container } = render(StudioBrowser, { kind: 'sfx', library: '', onClose: () => {} })
    await waitFor(() => expect(container.querySelector('.sb-tile .sb-name')?.textContent).toBe('a'))

    const next = Array.from(container.querySelectorAll('button')).find((b) => b.closest('.sb-pages') && !b.hasAttribute('disabled'))!
    await fireEvent.click(next)
    // Still page one on screen, marked busy; not an empty grid.
    await waitFor(() => expect(container.querySelector('.sb-grid')?.classList.contains('busy')).toBe(true))
    expect(container.querySelector('.sb-tile .sb-name')?.textContent).toBe('a')
    release(page2)
    await waitFor(() => expect(container.querySelector('.sb-tile .sb-name')?.textContent).toBe('b'))
    expect(container.querySelector('.sb-grid')?.classList.contains('busy')).toBe(false)

    // Stale: two queries in flight, the older one answering last.
    let releaseOld: (v: any) => void = () => {}
    let releaseNew: (v: any) => void = () => {}
    vi.mocked(StudioAssets)
      .mockImplementationOnce(() => new Promise((res) => { releaseOld = res }))
      .mockImplementationOnce(() => new Promise((res) => { releaseNew = res }))
    const input = container.querySelector('.sb-search input') as HTMLInputElement
    await fireEvent.input(input, { target: { value: 'wh' } })
    await waitFor(() => expect(vi.mocked(StudioAssets).mock.calls.length).toBe(3))
    await fireEvent.input(input, { target: { value: 'whoosh' } })
    await waitFor(() => expect(vi.mocked(StudioAssets).mock.calls.length).toBe(4))
    releaseNew({ ...page1, rows: [row('whoosh', 'wav', 'sfx')] })
    await waitFor(() => expect(container.querySelector('.sb-tile .sb-name')?.textContent).toBe('whoosh'))
    releaseOld({ ...page1, rows: [row('wrong', 'wav', 'sfx')] })
    await new Promise((r) => setTimeout(r, 30))
    expect(container.querySelector('.sb-tile .sb-name')?.textContent).toBe('whoosh')
  })

  // A correction is one press away on every tile, is written to the store,
  // and changes the row on screen without a re-query; hiding takes the row
  // out of the default view and the count with it.
  it('corrects a kind and hides a file from the tile menu', async () => {
    const { container } = render(StudioBrowser, { kind: '', library: '', onClose: () => {} })
    const tiles = await waitFor(() => { const f = container.querySelectorAll('.sb-tile'); expect(f.length).toBe(4); return Array.from(f) })

    await fireEvent.click(tiles[2].querySelector('.sb-more') as HTMLElement)
    const menu = await waitFor(() => { const m = tiles[2].querySelector('.sb-menu'); expect(m).toBeTruthy(); return m! })
    const checked = menu.querySelector('button.on')!
    expect(checked.textContent).toContain('overlay')
    const asIcon = Array.from(menu.querySelectorAll('button')).find((b) => b.textContent?.includes('ไอคอน'))!
    await fireEvent.click(asIcon)
    await waitFor(() => expect(vi.mocked(StudioSetKind)).toHaveBeenCalledWith('burn', 'icon'))
    await waitFor(() => expect(tiles[2].querySelector('.sb-meta')?.textContent).toContain('ไอคอน'))
    expect(tiles[2].querySelector('.sb-menu')).toBeNull()

    await fireEvent.click(tiles[3].querySelector('.sb-more') as HTMLElement)
    const hide = await waitFor(() => Array.from(tiles[3].querySelectorAll('.sb-menu button')).find((b) => b.textContent?.includes('ซ่อนจากเอเจน'))!)
    await fireEvent.click(hide)
    await waitFor(() => expect(vi.mocked(StudioSetHidden)).toHaveBeenCalledWith('rocket', true))
    await waitFor(() => expect(container.querySelectorAll('.sb-tile').length).toBe(3))
    expect(container.querySelector('.sb-count')?.textContent).toContain('697')
  })
})
