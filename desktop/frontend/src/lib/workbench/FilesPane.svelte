<script lang="ts">
  import { browseFolder, cockpit, openFolder, stopBrowsing, toggleNode, visibleTree } from '../stores/cockpit.svelte'
  import { workbench, openFileTab, setTabDragPayload } from '../stores/workbench.svelte'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'

  const rows = $derived(visibleTree(cockpit.tree))
</script>

{#if cockpit.browseRoot}
  <!-- A tree with no project behind it has to say so, or it is indistinguishable
       from one that has quietly adopted a folder — which is the confusion this
       whole thing came from. The way out sits beside the way in. -->
  <div class="browsing">
    <span class="ic"><Icon name="folderOpen" size={13} /></span>
    <span class="bt" title={cockpit.browseRoot}>
      <strong>{cockpit.browseRoot.split(/[\\/]/).filter(Boolean).pop()}</strong>
      <span class="bn">{t('workbench.browseNote')}</span>
    </span>
    <button
      type="button" class="bx" onclick={stopBrowsing}
      aria-label={t('workbench.stopBrowsing')} title={t('workbench.stopBrowsing')}
    ><Icon name="x" size={13} /></button>
  </div>
{/if}

<div class="insp-scroll">
  <div class="proj">
    {#each rows as node (node.label + node.depth)}
      <!-- Draggable, because this is where a file actually lives: the tab strip
           and the reply's file cards could both be dragged to the composer or
           the desk, and the tree — the one place you go to *find* a file —
           could not. A folder is not: there is nothing to attach. -->
      <button
        type="button" class="row" class:active={workbench.activeId === 'file-' + node.path}
        style="padding-left:{6 + node.depth * 14}px"
        onclick={() => (node.kind === 'dir' ? toggleNode(node) : openFileTab(node.path))}
        draggable={node.kind !== 'dir'}
        ondragstart={(e) => setTabDragPayload(e, 'file', node.path, node.label)}
      >
        {#if node.kind === 'dir'}
          <span class="tw" class:open={node.open}></span>
        {/if}
        <span class="ic"><Icon name={node.kind === 'dir' ? (node.open ? 'folderOpen' : 'folder') : 'fileText'} size={13} /></span>
        <!-- The name gives ground, not the mark. A bare text node could not:
             `incident-report-db-deletion-2026-07-23.pdf` pushed its own status
             past the edge of the panel, so the rows that most needed reading
             were the ones whose answer was off screen. -->
        <span class="lbl {node.status ? node.status.toLowerCase() : ''}">{node.label}</span>
        <!-- A folder says something inside it changed and which kind; a file
             says what happened to it and how much. Both come stamped from
             ProjectTree, so the tree and the git room never disagree about a
             file. Counts only where git had something to compare — a new file
             wears its letter alone rather than a `+0 −0` that means nothing. -->
        {#if node.status}
          <span class="gitmark">
            {#if node.kind === 'dir'}
              <!-- A folder cannot say how much, so it says how loud: a dot in
                   the colour of the strongest thing inside it. The letter is
                   still there for anything reading the row out. -->
              <span class="dot {node.status.toLowerCase()}" title={node.status}></span>
            {:else}
              {#if node.added}<span class="add">+{node.added}</span>{/if}
              {#if node.removed}<span class="rem">−{node.removed}</span>{/if}
              <span class="st {node.status.toLowerCase()}">{node.status}</span>
            {/if}
          </span>
        {/if}
      </button>
    {/each}
    {#if cockpit.tree.length === 0}
      <div class="pane-empty">
        <!-- Two different empty panels, because they are two different
             situations. With a project focused this is a project with nothing
             in it, and the way out is another project. Without one, the panel is
             empty *by design* — the assistant is tied to no folder — and saying
             "no files yet" there reads as something being broken. -->
        <p>{cockpit.project.focused ? t('sidebar.noFiles') : t('workbench.noProjectHere')}</p>
        {#if cockpit.project.focused}
          <button type="button" class="proj-add" onclick={openFolder}>
            <span class="ic"><Icon name="folderOpen" size={14} /></span> {t('topbar.openFolder')}
          </button>
        {:else}
          <!-- Looking, not moving in: browseFolder points the tree at a folder
               and stops there. เปิดโฟลเดอร์ (the โค้ด door's button) would focus
               the engine on it, retarget every new chat, and remember it — which
               is what happened to the owner on 7 ก.ย. -->
          <button type="button" class="proj-add" onclick={browseFolder}>
            <span class="ic"><Icon name="folderOpen" size={14} /></span> {t('workbench.browseFolder')}
          </button>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  /* Sticky to the panel's own scroller (.insp-slot), so the answer to "whose
     files are these" stays on screen while the tree scrolls under it. */
  .browsing {
    position: sticky; top: 0; z-index: 2;
    display: flex; align-items: center; gap: 8px; min-width: 0;
    padding: 7px 10px; font-size: var(--fs-sm);
    background: var(--surface-sunken); border-bottom: 1px solid var(--border-default);
  }
  .browsing .ic { flex: none; display: inline-flex; color: var(--text-muted); }
  .bt { min-width: 0; display: flex; flex-direction: column; gap: 1px; }
  .bt strong {
    font-weight: 600; color: var(--text-primary);
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .bn { font-size: var(--fs-xs, 11px); color: var(--text-muted); }
  .bx {
    margin-left: auto; flex: none; appearance: none; background: none; border: 0;
    padding: 2px; color: var(--text-muted); cursor: pointer; display: inline-flex;
  }
  .bx:hover { color: var(--text-primary); }
</style>
