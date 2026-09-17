<script lang="ts">
  import Icon from './Icon.svelte'
  import { CONNECTION_MARKS } from './connectionMarks'

  let { id, size = 18 }: { id: string; size?: number } = $props()
  const mark = $derived(CONNECTION_MARKS[id.toLowerCase()])
</script>

<span
  class="connection-mark"
  data-connection-mark={mark ? id.toLowerCase() : 'fallback'}
  style="--connection-mark-size:{size}px{mark?.ink ? `; color:${mark.ink}` : ''}"
>
  {#if mark}
    <svg viewBox={mark.viewBox ?? '0 0 24 24'} fill="currentColor" aria-hidden="true">
      {@html mark.svg}
    </svg>
  {:else}
    <Icon name="plug" size={size} />
  {/if}
</span>
