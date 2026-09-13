<script lang="ts">
  // A face with its rank on it: the mascot the caller draws, and the level's
  // emblem — 3 / 2 / 1 bars in a small disc — at the face's lower-right
  // corner. Style E of the six the owner compared (14 ก.ย. 2026: "ยศเอาไว้
  // ตรง E · ตราที่มุมหน้า แบบนี้ดีกว่า"): the mark travels WITH the face, so a
  // face anywhere — a list, a roster, the editor's stage, a memory group —
  // says its level without a label beside the name, and the name stays
  // clean. The word is the emblem's tooltip; where a page has room to say it
  // in words, RankPip does that beside the name.
  //
  // The emblem is scaled off the face and never off the rig: the mascot
  // stays one body (COMPANY.md §6), and this is a sticker on its corner.
  import type { Snippet } from 'svelte'
  import { t } from './i18n.svelte'

  let { tier, size, children }: { tier: 'head' | 'agent' | 'helper'; size: number; children: Snippet } = $props()
  const bars = $derived(tier === 'head' ? 3 : tier === 'agent' ? 2 : 1)
  const word = $derived(tier === 'head' ? t('rank.head') : tier === 'agent' ? t('rank.agent') : t('rank.helper'))
  // 16px on a 38px face, 28px on the editor's 168px stage, never a dot.
  const px = $derived(Math.round(Math.min(30, Math.max(14, size * 0.42))))
</script>

<span class="ranked" style="width:{size}px;height:{size}px">
  {@render children()}
  <span class="rank-corner rank-{tier}" style="--px:{px}px" title={word} aria-label={word} role="img">
    <span class="rank-bars" aria-hidden="true">{#each Array(bars) as _, i (i)}<i></i>{/each}</span>
  </span>
</span>

<style>
  .ranked { position: relative; display: inline-block; flex: none; line-height: 0; }
  .ranked :global(.mascot) { display: block; }
  .rank-corner {
    position: absolute; right: calc(var(--px) * -.22); bottom: calc(var(--px) * -.1);
    width: var(--px); height: var(--px); border-radius: 50%; box-sizing: border-box;
    display: grid; place-items: center;
    /* a ring in the page's own ground so the disc reads as sitting ON the face */
    border: calc(var(--px) * .09) solid var(--surface-panel);
  }
  .rank-bars { display: inline-flex; gap: calc(var(--px) * .09); align-items: flex-end; height: calc(var(--px) * .4); }
  .rank-bars i { display: block; width: calc(var(--px) * .12); height: 100%; border-radius: 1px; background: currentColor; }
  .rank-head { color: var(--badge-think-text); background: var(--badge-think-bg); outline: 1px solid var(--badge-think-border); }
  .rank-agent { color: var(--badge-cyan-text); background: var(--badge-cyan-bg); outline: 1px solid var(--badge-cyan-border); }
  .rank-helper { color: var(--text-muted); background: var(--surface-sunken); outline: 1px solid var(--border-subtle); }
</style>
