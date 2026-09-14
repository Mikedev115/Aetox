<script lang="ts">
  // The studio shelf, page by page — the part of คลังสตูดิโอ a person can
  // actually look at and listen to (docs/architecture/studio-shelf-page-2026-09-12.md §3.3).
  //
  // Opened from a kind tile or a shelf card, and it asks the engine the SAME
  // question the agent's `asset_find query` asks: StudioAssets runs
  // assetlib.Search over the same catalogue. So what a person sees here is
  // what the agent would be handed for the same words, which is the honest
  // way to find out whether the scan understood a folder.
  //
  // Three separate things, kept separate (owner, 12 ก.ย.: "มันควรแยกส่วนเลย
  // แสดงผล เรนเดอร์ และ โหลดไฟล์"):
  //
  //   display — the grid and every word on it come from the catalogue and are
  //             on screen the moment the answer arrives; no file is opened.
  //   render  — a tile's picture is a poster ffmpeg made once, 256px, kept on
  //             disk (studio_thumbs.go). Ones not made yet are asked for in
  //             one call per page and drop in as "studio:thumbs" reports them;
  //             until then the tile shows its kind's icon.
  //   load    — the real file is fetched only when someone hovers a tile, and
  //             only when the webview can decode that format (row.playable).
  //             A ProRes .mov never loads: its poster is all it shows.
  //
  // One sound plays at a time — pressing another stops the first — because
  // two whooshes at once tell you nothing about either.
  import { StudioAssets, StudioThumbs, StudioSetKind, StudioSetHidden, RevealStudioAsset, StudioLibraries } from '../../wailsjs/go/main/App'
  import { EventsOn } from '../../wailsjs/runtime/runtime'
  import { engine } from '../../wailsjs/go/models'
  import type { IconName } from './icons'
  import { t, type TKey } from './i18n.svelte'
  import Icon from './Icon.svelte'

  let {
    kind = '',
    library = '',
    onClose,
    onClearLibrary = () => {},
  }: {
    /** Initial kind filter; '' for every kind. */
    kind?: string
    /** Initial shelf filter (library id); '' for every shelf. */
    library?: string
    onClose: () => void
    /** Pressed on the "every shelf" link when a shelf filter is on. */
    onClearLibrary?: () => void
  } = $props()

  const AUDIO = new Set(['wav', 'mp3', 'ogg', 'm4a', 'aac', 'flac', 'aiff', 'aif'])
  const VIDEO = new Set(['mp4', 'mov', 'webm', 'mkv', 'avi', 'm4v'])

  let text = $state('')
  // Seeded by the effects below from the props, which is where they follow
  // later changes from as well; an initial value here would only be read once.
  let kindPick = $state('')
  let libraryPick = $state('')
  let category = $state('')
  let alphaOnly = $state(false)
  let includeHidden = $state(false)
  let page = $state(1)
  let result = $state<engine.StudioAssetPage | null>(null)
  let loading = $state(false)
  let error = $state('')
  // Posters that arrived after the page did, id → URL ('' = none will come).
  let thumbs = $state<Record<string, string>>({})
  // Which filter the rows on screen answer. A change of filter empties the
  // grid for a skeleton — rows of another tab are not a loading state, they
  // are the wrong rows — while a change of page alone keeps them dimmed.
  let shownFilter = ''

  // The filters a caller opens with may change while the browser is up (a
  // second tile pressed); follow them and start over at page one.
  $effect(() => { kindPick = kind; page = 1 })
  $effect(() => { libraryPick = library; page = 1 })

  // Typing re-queries after a pause, not per keystroke: each query walks the
  // whole catalogue, and "whoosh" is six of them.
  let typeTimer: ReturnType<typeof setTimeout> | undefined
  function onType(v: string) {
    text = v
    clearTimeout(typeTimer)
    typeTimer = setTimeout(() => { page = 1 }, 180)
  }

  // Loading, done properly (owner, 12 ก.ย.: "จัดการการโหลดให้ดี"):
  //
  //  - The first answer gets placeholder tiles, not an empty panel.
  //  - Later answers keep the rows already on screen, dimmed, until the new
  //    ones arrive — paging must not flash the grid empty and then refill it.
  //  - An answer to a question that is no longer the current one is dropped:
  //    type "wh", then "whoosh", and the slower "wh" reply must not land on
  //    top of the right one. The sequence number is that guard.
  //  - Media loads only when its tile is on screen (see onScreen below), so
  //    a page of sixty clips costs the file host the dozen that are visible.
  let seq = 0
  $effect(() => {
    // Read every input here so the effect re-runs when any of them moves.
    const q = { text, kind: kindPick, library: libraryPick, category, alphaOnly, includeHidden, page }
    const mine = ++seq
    void (async () => {
      loading = true
      error = ''
      const filter = JSON.stringify({ ...q, page: 0 })
      if (filter !== shownFilter) { result = null; shownFilter = filter }
      try {
        const got = await StudioAssets(q as engine.StudioAssetQuery)
        if (mine !== seq) return
        result = got
        // Ask once for the posters this page lacks; the answer for the ones
        // already rendered comes back now, the rest arrive by event.
        const missing = got.rows.filter((r) => !r.thumb && !AUDIO.has(r.ext) && !(r.id in thumbs)).map((r) => r.id)
        if (missing.length) {
          const have = await StudioThumbs(missing)
          if (mine !== seq) return
          thumbs = { ...thumbs, ...have }
        }
      } catch (err) {
        if (mine !== seq) return
        error = String(err)
      } finally {
        if (mine === seq) loading = false
      }
    })()
  })

  $effect(() => EventsOn('studio:thumbs', (urls: Record<string, string>) => { thumbs = { ...thumbs, ...urls } }))
  const posterOf = (row: engine.StudioAssetView) => row.thumb || thumbs[row.id] || ''
  const posterPending = (row: engine.StudioAssetView) => !row.thumb && !(row.id in thumbs)
  const KIND_ICON: Record<string, IconName> = { sfx: 'volume2', music: 'headphones', overlay: 'sparkles', background: 'monitor', clip: 'clapperboard', icon: 'puzzle', image: 'image' }

  // Tiles register themselves; the poster inside one gets its src the first
  // time it scrolls into view, and keeps it. One observer for the whole grid.
  let observer: IntersectionObserver | undefined
  function onScreen(node: HTMLElement) {
    if (typeof IntersectionObserver === 'undefined') { reveal(node); return }
    observer ??= new IntersectionObserver((entries) => {
      for (const e of entries) if (e.isIntersecting) { reveal(e.target as HTMLElement); observer!.unobserve(e.target) }
    }, { rootMargin: '200px' })
    observer.observe(node)
    return { destroy() { observer?.unobserve(node) } }
  }
  function reveal(node: HTMLElement) {
    for (const m of node.querySelectorAll<HTMLImageElement>('[data-src]')) {
      m.src = m.dataset.src!
      m.removeAttribute('data-src')
    }
    node.classList.add('seen')
  }
  // A poster that lands after its tile was already seen has a data-src and
  // no observer left to flip it; flip it as it mounts.
  function lateSrc(node: HTMLImageElement) {
    if (node.closest('.sb-tile')?.classList.contains('seen') && node.dataset.src) { node.src = node.dataset.src; node.removeAttribute('data-src') }
  }
  $effect(() => () => observer?.disconnect())


  // ---- shelves whose folder is gone
  // The catalogue outlives the folder — a drive unplugged, a folder moved —
  // and the engine draws none of that shelf's rows (studioShelves), so what
  // this page owes is the reason the rows are not here (owner, 13 ก.ย. 2026:
  // "ทำไมกดฟังไม่ได้", then "ไม่มีก็ไม่ควรแสดงดิ"). Settings marks such a
  // shelf; said here too, because this is where the person is looking for
  // the files.
  let libs = $state<engine.StudioLibraryView[]>([])
  $effect(() => { void StudioLibraries().then((l) => { libs = l ?? [] }).catch(() => { libs = [] }) })
  const goneShown = $derived(libs.filter((l) => l.missing && (!libraryPick || l.id === libraryPick)))

  // ---- one sound at a time
  let audio: HTMLAudioElement | undefined
  let playing = $state('')
  // The row whose play failed, so the tile can say so instead of the button
  // flipping back to ▶ as if nothing had been pressed.
  let failed = $state('')
  function toggle(row: engine.StudioAssetView) {
    if (!audio) {
      audio = new Audio()
      audio.addEventListener('ended', () => { playing = '' })
      audio.addEventListener('error', () => { failed = playing; playing = '' })
    }
    if (playing === row.id) { stop(); return }
    failed = ''
    audio.src = row.url
    audio.currentTime = 0
    void audio.play().catch(() => { failed = row.id; playing = '' })
    playing = row.id
  }
  function stop() {
    audio?.pause()
    playing = ''
  }
  $effect(() => () => stop())

  // The real file exists in the DOM only while a tile is hovered: a <video>
  // over the poster for a clip, the animated file over its still for a GIF.
  // Leaving the tile removes it, so nothing keeps decoding off screen.
  let hovered = $state('')
  function hoverIn(row: engine.StudioAssetView) { if (row.playable && !AUDIO.has(row.ext)) hovered = row.id }
  function hoverOut() { hovered = '' }
  function autoplay(node: HTMLVideoElement) { void node.play()?.catch(() => {}) }

  // ---- corrections (docs/architecture/studio-shelf-page-2026-09-12.md §3.3)
  // One tile's menu open at a time. A correction is written to the store,
  // then the row on screen is patched in place — the agent reads the store
  // on its next call, the page does not need to re-query for one row.
  const KINDS = ['sfx', 'music', 'overlay', 'background', 'clip', 'icon', 'image'] as const
  let menuFor = $state('')
  let correctError = $state('')
  function toggleMenu(id: string) { menuFor = menuFor === id ? '' : id }
  function closeMenu() { menuFor = '' }
  function patch(id: string, change: Partial<engine.StudioAssetView>) {
    if (!result) return
    result = { ...result, rows: result.rows.map((r) => (r.id === id ? { ...r, ...change } : r)) } as engine.StudioAssetPage
  }
  async function setKind(row: engine.StudioAssetView, kind: string) {
    correctError = ''
    try {
      await StudioSetKind(row.id, kind)
      patch(row.id, { kind })
    } catch (err) {
      correctError = String(err)
    }
    closeMenu()
  }
  async function setHidden(row: engine.StudioAssetView, hidden: boolean) {
    correctError = ''
    try {
      await StudioSetHidden(row.id, hidden)
      if (hidden && !includeHidden && result) {
        // Gone from the default view the moment it is hidden; the count
        // follows so the page never claims a row it no longer shows.
        result = { ...result, rows: result.rows.filter((r) => r.id !== row.id), total: result.total - 1 } as engine.StudioAssetPage
      } else {
        patch(row.id, { hidden })
      }
    } catch (err) {
      correctError = String(err)
    }
    closeMenu()
  }
  // Escape closes the menu; a click anywhere outside it does too.
  function onDocKey(e: KeyboardEvent) { if (e.key === 'Escape') closeMenu() }
  function onDocClick(e: MouseEvent) { if (menuFor && !(e.target as HTMLElement).closest('.sb-menu, .sb-more')) closeMenu() }
  $effect(() => {
    document.addEventListener('keydown', onDocKey)
    document.addEventListener('click', onDocClick)
    return () => { document.removeEventListener('keydown', onDocKey); document.removeEventListener('click', onDocClick) }
  })

  const secs = (d: number) => d >= 60 ? `${Math.floor(d / 60)}:${String(Math.round(d % 60)).padStart(2, '0')}` : `${d.toFixed(1)}s`
  const kindLabel = (k: string) => t(`settings.studioKind.${k}` as TKey)
  const first = $derived(result ? (result.page - 1) * 60 + 1 : 0)
  const last = $derived(result ? Math.min(result.page * 60, result.total) : 0)
</script>

<section class="sb" data-guide="studio.browser" aria-label={t('settings.studioBrowse')}>
  <div class="sb-bar">
    <label class="sb-search">
      <Icon name="search" size={14} />
      <input type="search" data-guide="studio.search" placeholder={t('settings.studioSearch')} value={text} oninput={(e) => onType(e.currentTarget.value)} />
    </label>
    {#if libraryPick && result?.rows[0]}
      <!-- Opened from one shelf's card: say so, and let it go back to all. -->
      <span class="sb-scope">{t('settings.studioScope', { name: result.rows[0].library })}<button class="linklike" onclick={onClearLibrary}>{t('settings.studioScopeAll')}</button></span>
    {/if}
    {#if result && result.categories.length > 1}
      <select class="ctrl sb-cat" data-guide="studio.filter_tab" value={category} onchange={(e) => { category = e.currentTarget.value; page = 1 }}>
        <option value="">{t('settings.studioAllFolders')}</option>
        {#each result.categories as c (c.name)}<option value={c.name}>{c.name} · {c.count.toLocaleString()}</option>{/each}
      </select>
    {/if}
    <label class="sb-alpha"><input type="checkbox" checked={alphaOnly} onchange={(e) => { alphaOnly = e.currentTarget.checked; page = 1 }} /> {t('settings.studioAlphaOnly')}</label>
    <label class="sb-alpha"><input type="checkbox" checked={includeHidden} onchange={(e) => { includeHidden = e.currentTarget.checked; page = 1 }} /> {t('settings.studioShowHidden')}</label>
    <button class="ctrl ctrl-icon sb-close" title={t('settings.studioClose')} aria-label={t('settings.studioClose')} onclick={onClose}><Icon name="x" size={14} /></button>
  </div>

  {#if error}<div class="mset-error">{error}</div>{/if}
  {#if correctError}<div class="mset-error">{correctError}</div>{/if}

  {#if !result && loading}
    <!-- The very first answer: the panel's shape before its content. -->
    <div class="sb-count"><span class="sk sk-line" style:width="120px"></span></div>
    <div class="sb-grid" aria-busy="true">
      {#each Array.from({ length: 12 }) as _, i (i)}
        <article class="sb-tile skeleton" aria-hidden="true"><div class="sb-media sk"></div><span class="sk sk-line"></span><span class="sk sk-line" style:width="60%"></span></article>
      {/each}
    </div>
  {/if}

  {#each goneShown as lib (lib.id)}
    <div class="mset-error sb-gone">{t('settings.studioGoneHere', { name: lib.name, root: lib.root })}</div>
  {/each}

  {#if result}
    <div class="sb-count">
      {#if result.total === 0}{t('settings.studioNoMatch')}{:else}{t('settings.studioShowing', { a: first.toLocaleString(), b: last.toLocaleString(), n: result.total.toLocaleString() })}{/if}
      {#if loading}<span class="sb-busy"><Icon name="loaderCircle" size={12} /></span>{/if}
    </div>

    <!-- Sounds have nothing to show, so they take a row each instead of a
         square: forty-eight blank tiles are a page of nothing. -->
    <div class="sb-grid" class:busy={loading} class:sounds={result.rows.length > 0 && result.rows.every((r) => AUDIO.has(r.ext))} aria-busy={loading}>
      {#each result.rows as row (row.id)}
        <article class="sb-tile" data-guide="studio.preview_item" class:sound={AUDIO.has(row.ext)} class:playing={playing === row.id} class:alpha={row.alpha} class:hidden-row={row.hidden} class:menu-open={menuFor === row.id} class:playable={row.playable && !AUDIO.has(row.ext)} use:onScreen onmouseenter={() => hoverIn(row)} onmouseleave={hoverOut}>
          <div class="sb-media">
            {#if AUDIO.has(row.ext)}
              <button class="sb-play" onclick={() => toggle(row)} aria-label={playing === row.id ? t('settings.studioStop') : t('settings.studioPlay')}>
                <Icon name={playing === row.id ? 'square' : 'play'} size={22} />
              </button>
            {:else}
              {#if posterOf(row)}
                <img class="sb-poster" data-src={posterOf(row)} alt="" decoding="async" use:lateSrc />
              {:else}
                <span class="sb-kind-mark" class:pending={posterPending(row)}><Icon name={KIND_ICON[row.kind] ?? 'image'} size={26} /></span>
              {/if}
              {#if hovered === row.id}
                {#if VIDEO.has(row.ext)}
                  <!-- svelte-ignore a11y_media_has_caption -->
                  <video class="sb-live" src={row.url} muted playsinline loop use:autoplay></video>
                {:else if row.ext === 'gif'}
                  <img class="sb-live" src={row.url} alt="" />
                {/if}
              {/if}
            {/if}
            {#if row.duration > 0 && !AUDIO.has(row.ext)}<span class="sb-dur">{secs(row.duration)}</span>{/if}
            {#if row.alpha}<span class="sb-badge">alpha</span>{/if}
          </div>
          <div class="sb-text">
            <div class="sb-name" title={row.path}>{row.name}</div>
            <div class="sb-meta">
              {#if failed === row.id}<span class="sb-fail">{t('settings.studioPlayFailed')}</span>{/if}
              <span>{kindLabel(row.kind)}</span>
              {#if row.width > 0}<span>{row.width}×{row.height}</span>{/if}
              {#if row.category}<span class="sb-cat-name" title={row.category}>{row.category}</span>{/if}
            </div>
          </div>
          {#if AUDIO.has(row.ext) && row.duration > 0}<span class="sb-dur">{secs(row.duration)}</span>{/if}
          <!-- The tile's own menu: correct what the scan guessed, keep the
               file from the agent, or go and look at it. -->
          <button class="sb-more" aria-label={t('settings.studioMore')} aria-expanded={menuFor === row.id} onclick={() => toggleMenu(row.id)}><Icon name="ellipsisVertical" size={14} /></button>
          {#if menuFor === row.id}
            <div class="sb-menu" role="menu">
              <div class="sb-menu-lab">{t('settings.studioMenuKind')}</div>
              {#each KINDS as k (k)}
                <button role="menuitemradio" aria-checked={row.kind === k} class:on={row.kind === k} onclick={() => setKind(row, k)}>
                  <Icon name={KIND_ICON[k]} size={13} /> {kindLabel(k)}
                </button>
              {/each}
              <div class="sb-menu-sep"></div>
              {#if row.hidden}
                <button role="menuitem" onclick={() => setHidden(row, false)}><Icon name="eye" size={13} /> {t('settings.studioUnhide')}</button>
              {:else}
                <button role="menuitem" data-guide="studio.delete_btn" onclick={() => setHidden(row, true)}><Icon name="x" size={13} /> {t('settings.studioHide')}</button>
              {/if}
              <button role="menuitem" onclick={() => { void RevealStudioAsset(row.id); closeMenu() }}><Icon name="folderOpen" size={13} /> {t('settings.studioReveal')}</button>
            </div>
          {/if}
          {#if row.hidden}<span class="sb-hidden-tag">{t('settings.studioHiddenTag')}</span>{/if}
        </article>
      {/each}
    </div>

    {#if result.pages > 1}
      <nav class="sb-pages" aria-label={t('settings.studioPages')}>
        <button class="ctrl" disabled={loading || result.page <= 1} onclick={() => { page = result!.page - 1; stop() }}><Icon name="chevronLeft" size={14} /></button>
        <span>{t('settings.studioPageOf', { p: result.page, n: result.pages })}</span>
        <button class="ctrl" disabled={loading || result.page >= result.pages} onclick={() => { page = result!.page + 1; stop() }}><Icon name="chevronRight" size={14} /></button>
      </nav>
    {/if}
  {/if}
</section>
