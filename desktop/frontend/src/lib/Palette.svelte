<script lang="ts">
  // The + menu: one door for everything that goes INTO the message being
  // written (ARCHITECTURE.md §257, superseding §36/§38).
  //
  // It used to be three: "+" attached files, "/" listed prompt presets, and
  // Ctrl+K opened this component in a mode that repeated both and then the
  // model chip beside it (switch model, approval, think level) — a list that
  // promised "search everything" and searched a copy of the row it sat on
  // (owner, 12 ก.ย.: "ปุ่มนี้มันทำงานซ้ำซ้อน"). Everything here now answers one
  // question — what do you want to put into this message — and the rows that
  // answered a different one (how the turn runs) went back to the chip that
  // already owned them.
  //
  // Four groups, one search across all of them:
  //   แนบไฟล์      — the four kinds the app accepts, as tiles; the hint that
  //                  used to be a row of text is the tile's tooltip.
  //   ชุดคำสั่ง    — prompt presets; Enter writes "/name " into the message.
  //   เรียกเอเจน   — the chairs; Enter writes "@name " and addresses the turn.
  //   จากที่เปิดอยู่ — the workbench tabs a drag could already stage as
  //                  context, listed so nobody has to know that a drag works.
  // `focus` narrows to one group on the way in: "/" typed in an empty
  // composer opens on prompts, "@" on agents, Ctrl+K and the button on all.
  // Backspace on an empty search widens it again.
  import { onMount } from 'svelte'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import { ListPromptPresets, ListChairs } from '../../wailsjs/go/main/App'
  import { attachTabContext } from './stores/cockpit.svelte'
  import { workbench, type WorkbenchTab } from './stores/workbench.svelte'
  import { shortcutLabel } from './shortcuts'
  import type { IconName } from './icons'

  export type PaletteFocus = '' | 'prompts' | 'agents'

  let {
    focus = '',
    mentions = true,
    oninsert,
    onmention,
    onattach,
    onmic,
    onclose,
  }: {
    focus?: PaletteFocus
    /** Whether a chair can be addressed right now. Not mid-turn: what is
     *  typed then goes INTO the running turn (Interject), which has no door
     *  to a worker — the "@" roster refuses for the same reason. */
    mentions?: boolean
    /** A preset was picked: write this into the message. */
    oninsert: (text: string) => void
    /** A chair was picked: address the message to it. */
    onmention: (name: string) => void
    /** A file tile was picked: open the native dialog on this filter. */
    onattach: (group: string) => void
    /** The mic row — only drawn while the composer is too narrow to show
     *  the mic button itself (style.css, the composer container query). */
    onmic: () => void
    onclose: () => void
  } = $props()

  type Item = {
    id: string
    group: 'files' | 'mic' | 'prompts' | 'agents' | 'open' | 'keys'
    label: string
    hint?: string
    badge?: string
    icon?: IconName
    /** Everything the search is allowed to match against. */
    text: string
    run?: () => void | Promise<void>
  }

  const fileTiles = [
    { group: 'image', icon: 'image', label: 'chat.attachImages', hint: 'chat.attachImagesHint' },
    { group: 'document', icon: 'fileText', label: 'chat.attachDocs', hint: 'chat.attachDocsHint' },
    { group: 'media', icon: 'clapperboard', label: 'chat.attachMedia', hint: 'chat.attachMediaHint' },
    { group: '', icon: 'paperclip', label: 'chat.attachAny', hint: 'chat.attachAnyHint' },
  ] as const

  let presets = $state<{ name: string; description: string; builtin: boolean }[]>([])
  let chairs = $state<{ name: string; description: string }[]>([])
  let query = $state('')
  // svelte-ignore state_referenced_locally — the initial value is the point:
  // `focus` is how the menu was opened, and Backspace takes it off from here.
  let only = $state<PaletteFocus>(focus)
  let keysOpen = $state(false)
  let cursor = $state(0)
  let inputEl = $state<HTMLInputElement | null>(null)
  let rootEl = $state<HTMLDivElement | null>(null)
  // How tall the menu may be: the room between the button it hangs off and
  // the top of the nearest box that clips (.main is overflow:clip), measured,
  // because a guess in vh cannot know where either edge is — under a tall
  // draft, or in a short window, the guess put the search box above the frame
  // and the frame cut it off.
  let maxH = $state(0)
  function measure() {
    const anchor = rootEl?.parentElement
    if (!anchor) return
    let clip: HTMLElement | null = anchor.parentElement
    while (clip && getComputedStyle(clip).overflow === 'visible') clip = clip.parentElement
    const ceiling = clip ? clip.getBoundingClientRect().top : 0
    maxH = Math.max(220, Math.floor(anchor.getBoundingClientRect().top - ceiling - 26))
  }

  onMount(() => {
    inputEl?.focus()
    measure()
    window.addEventListener('resize', measure)
    // Both lists in flight at once; whichever lands first draws first. A
    // failed roster is an empty group, not a broken menu.
    ListPromptPresets().then((p) => (presets = p)).catch(() => {})
    ListChairs().then((c) => (chairs = c)).catch(() => {})
    return () => window.removeEventListener('resize', measure)
  })

  // The tabs a drag could already stage — Workbench.svelte's canDrag, word for
  // word: a file tab needs a path, a browser tab needs a URL, and a tab that
  // failed to open has neither.
  const openTabs = $derived(
    workbench.tabs.filter((tab: WorkbenchTab) => (tab.kind === 'file' && !!tab.path) || (tab.kind === 'browser' && !!tab.url)),
  )

  const items = $derived.by((): Item[] => {
    const out: Item[] = []
    if (!only) {
      for (const f of fileTiles) {
        out.push({
          id: 'file:' + f.group, group: 'files', icon: f.icon,
          label: t(f.label), hint: t(f.hint), text: t(f.label) + ' ' + t(f.hint),
          run: () => onattach(f.group),
        })
      }
      out.push({
        id: 'mic', group: 'mic', icon: 'mic', label: t('chat.micStart'), hint: t('palette.micHint'),
        text: t('chat.micStart'), run: onmic,
      })
    }
    if (!only || only === 'prompts') {
      for (const p of presets) {
        out.push({
          id: 'preset:' + p.name, group: 'prompts', label: '/' + p.name, hint: p.description,
          badge: p.builtin ? t('settings.promptBuiltin') : undefined,
          text: '/' + p.name + ' ' + p.description,
          run: () => oninsert('/' + p.name + ' '),
        })
      }
    }
    if (mentions && (!only || only === 'agents')) {
      for (const c of chairs) {
        out.push({
          id: 'chair:' + c.name, group: 'agents', icon: 'bot', label: '@' + c.name, hint: c.description,
          text: '@' + c.name + ' ' + c.description,
          run: () => onmention(c.name),
        })
      }
    }
    if (!only) {
      for (const tab of openTabs) {
        const kind = tab.kind === 'file' ? t('palette.tabFile') : t('palette.tabBrowser')
        out.push({
          id: 'tab:' + tab.id, group: 'open', icon: tab.kind === 'file' ? 'fileText' : 'globe',
          label: tab.name, badge: kind, hint: t('palette.attachTab'),
          text: tab.name + ' ' + kind + ' ' + (tab.url ?? tab.path ?? ''),
          run: () => attachTabContext(tab.kind as 'file' | 'browser', tab.kind === 'file' ? tab.path ?? '' : tab.id, tab.name),
        })
      }
      // Read off shortcuts.ts rather than retyped here — this list once said
      // "Ctrl+Shift+G" for a tab that had been Ctrl+P for months, which is
      // worse than saying nothing. Behind the footer link, not always on:
      // they are the one group that does not put anything into the message.
      if (keysOpen) {
        out.push(
          { id: 'k-plus', group: 'keys', label: t('palette.openPlus'), hint: shortcutLabel('palette'), text: '' },
          { id: 'k-new', group: 'keys', label: t('palette.newChat'), hint: shortcutLabel('newSession'), text: '' },
          { id: 'k-settings', group: 'keys', label: t('palette.openSettings'), hint: shortcutLabel('settings'), text: '' },
          { id: 'k-panels', group: 'keys', label: t('palette.togglePanels'), hint: shortcutLabel('toggleSidebar') + ' / ' + shortcutLabel('toggleInspector'), text: '' },
          { id: 'k-wb', group: 'keys', label: t('palette.workbenchTabs'), hint: shortcutLabel('browserTab') + ' · ' + shortcutLabel('filesTab'), text: '' },
        )
      }
    }
    return out
  })

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase()
    if (!q) return items
    return items.filter((r) => r.text.toLowerCase().includes(q))
  })
  // The mic row is drawn only while the composer is narrow (CSS), and a search
  // should not surface a row that is not on screen.
  const tiles = $derived(filtered.filter((r) => r.group === 'files'))
  const rows = $derived(filtered.filter((r) => r.group !== 'files'))
  const groupLabel: Record<Item['group'], string> = {
    files: t('palette.groupFiles'), mic: '', prompts: t('palette.groupPrompts'),
    agents: t('palette.groupAgents'), open: t('palette.groupOpen'), keys: t('palette.groupKeys'),
  }
  const groupKey: Partial<Record<Item['group'], string>> = {
    prompts: t('palette.keyPrompts'), agents: t('palette.keyAgents'),
  }
  const focusLabel: Record<PaletteFocus, string> = { '': '', prompts: t('palette.keyPrompts'), agents: t('palette.keyAgents') }

  // Keep the cursor on a real item while filtering narrows the list.
  $effect(() => {
    if (cursor > filtered.length - 1) cursor = Math.max(0, filtered.length - 1)
  })

  async function pick(row: Item | undefined) {
    if (!row?.run) return // a shortcut row is a readout, not a button
    await row.run()
    onclose()
  }

  function onKeydown(e: KeyboardEvent) {
    // stopPropagation, not just preventDefault: Escape also closes Settings at
    // the window level, and one press must dismiss one layer — the menu
    // opened over Settings, so it is the layer that closes.
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); onclose(); return }
    if (e.key === 'ArrowDown') { e.preventDefault(); cursor = Math.min(cursor + 1, filtered.length - 1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); cursor = Math.max(cursor - 1, 0) }
    else if (e.key === 'Enter') { e.preventDefault(); void pick(filtered[cursor]) }
    // The narrowing "/" or "@" opened with, taken back the way it was typed.
    else if (e.key === 'Backspace' && query === '' && only) { e.preventDefault(); only = '' }
  }
  const at = (row: Item) => filtered.indexOf(row)
</script>

<div class="palette" role="dialog" aria-label={t('palette.title')} bind:this={rootEl} style:max-height={maxH ? maxH + 'px' : undefined}>
  <div class="pal-head">
    <span class="pal-ic"><Icon name="search" size={14} /></span>
    <input
      class="pal-search"
      bind:this={inputEl}
      bind:value={query}
      placeholder={t('palette.search')}
      onkeydown={onKeydown}
    />
    {#if only}<span class="pal-mode">{focusLabel[only]}</span>{/if}
  </div>
  <div class="pal-list">
    {#if filtered.length === 0}
      <div class="pal-empty">{t('palette.noMatch')}</div>
    {/if}
    {#if tiles.length > 0}
      <div class="pal-group">{groupLabel.files}</div>
      <div class="pal-tiles">
        {#each tiles as row (row.id)}
          <button
            type="button" class="pal-tile" class:on={at(row) === cursor}
            title={row.hint} aria-label={row.label + ' — ' + row.hint}
            onmouseenter={() => (cursor = at(row))}
            onclick={() => pick(row)}
          >
            <span class="ic"><Icon name={row.icon ?? 'paperclip'} size={18} /></span>
            <span class="nm">{row.label}</span>
          </button>
        {/each}
      </div>
    {/if}
    {#each rows as row, i (row.id)}
      {#if row.group !== 'mic' && (i === 0 || rows[i - 1].group !== row.group)}
        <div class="pal-group">
          {groupLabel[row.group]}
          {#if groupKey[row.group]}<span class="k">{groupKey[row.group]}</span>{/if}
        </div>
      {/if}
      <button
        type="button"
        class="pal-row"
        class:mic-row={row.group === 'mic'}
        class:on={at(row) === cursor}
        class:static={!row.run}
        onmouseenter={() => (cursor = at(row))}
        onclick={() => pick(row)}
      >
        {#if row.icon}<span class="ic"><Icon name={row.icon} size={14} /></span>{/if}
        <span class="pal-label">{row.label}</span>
        {#if row.badge}<span class="pal-badge">{row.badge}</span>{/if}
        {#if row.hint}<span class="pal-hint">{row.hint}</span>{/if}
      </button>
    {/each}
  </div>
  <div class="pal-foot">
    <span>{t('chat.attachNote')}</span>
    <button type="button" class="pal-keys" class:on={keysOpen} onclick={() => (keysOpen = !keysOpen)}>{t('palette.allKeys')}</button>
  </div>
</div>
