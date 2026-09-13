<script lang="ts">
  // ยศ — which level of the company a face belongs to, worn beside its name.
  //
  // Three ranks, fixed (owner, 13 ก.ย. 2026, from three sets offered):
  // หัวหน้า — the main assistant of a desk, the one you talk to;
  // ผู้เชี่ยวชาญ — an agent you may also talk to, that the head may hand
  // work to; ลูกมือ — the built-in helpers the head runs inside its own
  // work. It exists because, once every agent and helper got the same
  // editor as the main one (model, prompt, avatar, think level), a reader
  // of the settings could no longer tell which was the main one and which
  // the hands ("อันไหนคือตัวหลัก อันไหนคือเอเจน อันไหนคือซับเอเจน").
  //
  // Bars and a word: 3 / 2 / 1 bars are the part that needs no language,
  // the word is the part that needs no legend. The mark sits beside the
  // face, never on the rig — the mascot stays one body (COMPANY.md §6).
  // A rank is the level's and is not a setting on anything.
  import { t } from './i18n.svelte'

  // word=false is the bars alone — for a place where the word is already
  // written beside it, like the rail's row named พนักงาน (owner, 14 ก.ย.:
  // "ในหน้าเมนู ทำสัญลักษณ์ยศแปะไว้ด้วย"). The word stays as the tooltip.
  let { tier, size = 'sm', word: showWord = true }: { tier: 'head' | 'agent' | 'helper'; size?: 'sm' | 'md'; word?: boolean } = $props()
  const bars = $derived(tier === 'head' ? 3 : tier === 'agent' ? 2 : 1)
  const word = $derived(tier === 'head' ? t('rank.head') : tier === 'agent' ? t('rank.agent') : t('rank.helper'))
</script>

<span class="rank rank-{size} rank-{tier}" class:rank-bare={!showWord} title={word} aria-label={word} role="img">
  <span class="rank-bars" aria-hidden="true">{#each Array(bars) as _, i (i)}<i></i>{/each}</span>
  {#if showWord}<span class="rank-word">{word}</span>{/if}
</span>

<style>
  .rank {
    display: inline-flex; align-items: center; gap: 5px; vertical-align: middle; flex: none;
    border-radius: 999px; padding: 1px 7px 1px 5px; border: 1px solid transparent;
    font-size: var(--fs-2xs); font-weight: 600; line-height: 1.4; white-space: nowrap;
  }
  .rank-md { font-size: var(--fs-xs); padding: 2px 9px 2px 7px; }
  .rank-bare { padding: 3px 6px; gap: 0; }
  .rank-bars { display: inline-flex; gap: 2px; align-items: flex-end; height: .8em; }
  .rank-bars i { display: block; width: 3px; height: 100%; border-radius: 1px; background: currentColor; }
  .rank-md .rank-bars i { width: 3.5px; }
  /* the app's own badge tokens (palette.css): the head in the think badge's
     colour, the specialist in cyan, the helper in the quiet sunken surface */
  .rank-head { color: var(--badge-think-text); background: var(--badge-think-bg); border-color: var(--badge-think-border); }
  .rank-agent { color: var(--badge-cyan-text); background: var(--badge-cyan-bg); border-color: var(--badge-cyan-border); }
  .rank-helper { color: var(--text-muted); background: var(--surface-sunken); border-color: var(--border-subtle); }
</style>
