<script lang="ts">
  // The mark of a memory scope: the head's own face for the two desks, the
  // scope's icon for everything else (you, a project, a delegate).
  //
  // Owner, 14 ก.ย. 2026, pointing at ความจำของผู้ช่วยและโค้ด: "จุดไหนที่เป็นแบบนี้
  // เอารูปอวตารไปแปะเลย" — an icon of a sparkle next to the word ผู้ช่วย is a
  // second way of drawing someone the app draws as a face everywhere else,
  // and the face is the one the user dressed (avatarPrefs, per desk). The
  // face is still (a list of these breathing together stutters, MASCOT.md
  // §3.7) and a little larger than the icon it replaces, because a face
  // at icon size is a smudge.
  import Icon from './Icon.svelte'
  import Mascot from './mascot/Mascot.svelte'
  import { headOptions } from './mascot/avatarPrefs.svelte'
  import type { ScopeMeta } from './memoryScope'
  import RankedFace from './RankedFace.svelte'

  let { meta, size = 14, face = size * 2 }: { meta: ScopeMeta; size?: number; face?: number } = $props()
</script>

{#if meta.head && face >= 24}
  <!-- big enough for the rank's emblem on its corner (RankedFace) -->
  <span class="scope-face" style="width:{face}px;height:{face}px"><RankedFace tier="head" size={face}><Mascot {...headOptions(meta.head)} size={face} still /></RankedFace></span>
{:else if meta.head}
  <span class="scope-face" style="width:{face}px;height:{face}px"><Mascot {...headOptions(meta.head)} size={face} still /></span>
{:else}
  <Icon name={meta.icon} {size} />
{/if}

<style>
  .scope-face { display: inline-block; flex: none; line-height: 0; vertical-align: middle; }
  .scope-face :global(.mascot) { display: block; }
</style>
