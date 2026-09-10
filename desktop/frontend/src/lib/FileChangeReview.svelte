<script lang="ts">
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import CodeDiff from './CodeDiff.svelte'
  import { openGitTab, openFileTab } from './stores/workbench.svelte'
  import { extractChangedFiles, type ChangedFile } from './fileChange'
  import type { ToolStep } from './types'

  interface Props {
    steps?: ToolStep[]
    files?: ChangedFile[]
  }

  let { steps = [], files }: Props = $props()

  const fileList = $derived<ChangedFile[]>(files ?? extractChangedFiles(steps))
  const totalAdded = $derived(fileList.reduce((acc, f) => acc + (f.added || 0), 0))
  const totalRemoved = $derived(fileList.reduce((acc, f) => acc + (f.removed || 0), 0))

  let expanded = $state(false)
  let openDiffs = $state<Record<string, boolean>>({})

  function toggleFileDiff(path: string) {
    openDiffs[path] = !openDiffs[path]
  }

  function handleReview(e: MouseEvent) {
    e.stopPropagation()
    openGitTab()
  }

  function handleOpenFile(e: MouseEvent, f: ChangedFile) {
    e.stopPropagation()
    void openFileTab(f.path, f.name)
  }
</script>

{#if fileList.length > 0}
  <div class="file-change-review" class:expanded>
    <div
      class="fcr-head"
      role="button"
      tabindex="0"
      aria-expanded={expanded}
      onclick={() => (expanded = !expanded)}
      onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); expanded = !expanded; } }}
    >
      <div class="fcr-summary">
        <span class="fcr-count">
          {fileList.length === 1
            ? t('chat.fileChanged', { n: fileList.length })
            : t('chat.filesChanged', { n: fileList.length })}
        </span>
        {#if totalAdded > 0 || totalRemoved > 0}
          <span class="fcr-delta">
            {#if totalAdded > 0}<span class="fcr-add">+{totalAdded}</span>{/if}
            {#if totalRemoved > 0}<span class="fcr-del">-{totalRemoved}</span>{/if}
          </span>
        {/if}
        <span class="fcr-chev">
          <Icon name={expanded ? 'chevronDown' : 'chevronRight'} size={13} />
        </span>
      </div>

      <button
        type="button"
        class="fcr-review-btn"
        title={t('chat.review')}
        aria-label={t('chat.review')}
        onclick={handleReview}
      >
        <Icon name="fileCode" size={13} />
        <span>{t('chat.review')}</span>
      </button>
    </div>

    {#if expanded}
      <div class="fcr-list">
        {#each fileList as f (f.path)}
          <div class="fcr-file-block">
            <div
              class="fcr-file-row"
              role="button"
              tabindex="0"
              title={f.path}
              onclick={() => toggleFileDiff(f.path)}
              onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleFileDiff(f.path); } }}
            >
              <span class="fcr-braces" aria-hidden="true">&#123; &#125;</span>
              <span class="fcr-name">{f.name}</span>
              {#if f.dir}
                <span class="fcr-dir">{f.dir}</span>
              {/if}

              <div class="fcr-right">
                {#if f.added > 0 || f.removed > 0}
                  <span class="fcr-stat">
                    {#if f.added > 0}<span class="fcr-add">+{f.added}</span>{/if}
                    {#if f.removed > 0}<span class="fcr-del">-{f.removed}</span>{/if}
                  </span>
                {/if}
                <button
                  type="button"
                  class="fcr-open-btn icobtn tiny"
                  title={t('chat.openInEditor')}
                  aria-label={t('chat.openInEditor')}
                  onclick={(e) => handleOpenFile(e, f)}
                >
                  <Icon name="externalLink" size={11} />
                </button>
              </div>
            </div>

            {#if openDiffs[f.path] && f.diff}
              <div class="fcr-diff">
                <CodeDiff diff={f.diff} />
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}
