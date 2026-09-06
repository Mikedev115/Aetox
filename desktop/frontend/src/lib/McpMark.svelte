<script lang="ts">
  // The face an MCP server wears, in the one place every list reads it from.
  //
  // It began as a snippet inside Capability.svelte, which was fine while the
  // ห้องสมุด was the only room drawing servers. It is not: ตั้งค่า › MCP lists
  // the same servers — the configured register and the แนะนำ shelf — and drew
  // them as bare text, so kinocut had a logo on one page and nothing on the
  // next. The same server with two identities on two pages is two features
  // (DESIGN.md §1, §5), so the drawing moved here and both rooms call it.
  //
  // A real brand mark when we have one, because that is what a person actually
  // recognises: rows of identical text are the hardest kind of list to scan,
  // which is the sentence ProviderMark.svelte was written under and is just as
  // true of this list. Otherwise the app's own lettered tile (coverHue), which
  // every gallery already uses for a named thing with no logo — so a server
  // keeps the same colour it has in โปรเจกต์, ชุดคำสั่ง and the roster.
  //
  // The `.cap-mark` rules live in style.css, not in a <style> here, for the
  // same reason `.reg-head` and `.mcp-badge` do: two rooms share them.
  //
  // @html is safe for exactly the reason it is safe in ProviderMark: `mark` is
  // only ever a value looked up out of a static generated map, and `name`
  // indexes that map — it is never interpolated into markup.
  import { MCP_MARKS } from './mcpMarks'
  import { coverHue } from './coverHue'

  let { name, size = 26 }: { name: string; size?: number } = $props()

  const mark = $derived(MCP_MARKS[name])
</script>

{#if mark}
  <span class="cap-mark logo" style="--px:{size}px{mark.ink ? `; color: ${mark.ink}` : ''}" aria-hidden="true">
    <svg viewBox="0 0 24 24" fill="currentColor" fill-rule="evenodd">{@html mark.svg}</svg>
  </span>
{:else}
  <span class="cap-mark" style="--px:{size}px; --h:{coverHue(name)}" aria-hidden="true">
    {name.slice(0, 2)}
  </span>
{/if}
