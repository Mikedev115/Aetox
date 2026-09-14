<script lang="ts">
  /**
   * One thing the agent asked to remember, offered under the answer that asked.
   *
   * Memory is the only tool whose work does not happen when it runs: it queues a
   * proposal and waits for a person (internal/learned/tool.go). Before this card
   * the whole of that showed up in the chat as a row reading "memory" inside a
   * collapsed panel, and the decision lived on the Settings page — so the user
   * saw neither what was proposed nor that anything was waiting, unless they
   * went looking. The work suggested it here; it is decided here.
   *
   * The row is read from the queue by id rather than carried in the transcript,
   * so a proposal decided anywhere else comes back saying which way it went
   * instead of asking a second time.
   *
   * Two shapes, because it is two different things (owner's pick, 16 ส.ค.):
   *
   *  - **Waiting** is a question, and gets a card. Open, always: the first cut
   *    collapsed it behind a chevron, which asked the user to click before they
   *    could see what they were being asked. Nothing is hidden, so there is no
   *    chevron either.
   *  - **Decided** is history, and gets one quiet line. A conversation that
   *    remembered five things would otherwise be a wall of cards, each one
   *    asking about a decision already made.
   */
  import Icon from './Icon.svelte'
  import ScopeMark from './ScopeMark.svelte'
  import { t } from './i18n.svelte'
  import { cockpit } from './stores/cockpit.svelte'
  import { scopeMeta, USER_SCOPE, type ScopeMeta } from './memoryScope'
  import { PendingChangeByID, ApprovePendingChange, ApprovePendingChangeTo, RejectPendingChange, LearnedScopeInfos } from '../../wailsjs/go/main/App'
  import type { engine } from '../../wailsjs/go/models'

  let { id }: { id: number } = $props()

  let change = $state<engine.PendingChange | null>(null)
  /** Only ever the decided line's disclosure — a waiting card has nothing shut. */
  let detail = $state(false)
  let busy = $state(false)
  let error = $state('')

  async function load(): Promise<void> {
    try {
      const row = await PendingChangeByID(id)
      // The queue is the truth: a row that is no longer there (the history was
      // cleared, the database was moved) draws nothing at all, rather than a
      // card offering a decision on something that cannot be decided.
      change = row?.id ? row : null
    } catch {
      change = null
    }
  }

  // Loads on mount, and again whenever the badge moves: any decision anywhere
  // emits learning:changed, and this card may be about the row that moved it.
  // Cheaper than an event listener per card, and it covers the case the card
  // cannot see for itself — the same proposal approved from Settings.
  $effect(() => {
    void cockpit.pendingLearned
    void load()
  })

  // "เก็บที่อื่น" (11 ก.ย.): the second half of the decision. A person judging
  // a line decides whether it is true AND who should know it, and the card
  // asked only the first — the only way to correct a destination was to refuse
  // the line and hope the right desk proposed it again. The list of files is
  // asked for when the menu opens, not carried on every card: most cards are
  // decided with one press. Only a new line can be redirected (a replace names
  // a line in one file), which ApprovePendingChangeTo also refuses.
  let elsewhere = $state(false)
  let targets = $state<ScopeMeta[]>([])
  async function openElsewhere(): Promise<void> {
    if (elsewhere) { elsewhere = false; return }
    try {
      const infos = await LearnedScopeInfos()
      targets = infos
        .filter((s) => !s.orphan && s.scope !== change?.scope)
        .map((s) => scopeMeta(s.scope))
        .filter((m) => m.tone !== 'agent')
    } catch {
      targets = [scopeMeta(USER_SCOPE), scopeMeta('')].filter((m) => m.scope !== change?.scope)
    }
    elsewhere = true
  }

  async function decide(approve: boolean, to = ''): Promise<void> {
    if (busy) return
    busy = true
    error = ''
    elsewhere = false
    try {
      if (approve && to) await ApprovePendingChangeTo(id, to)
      else if (approve) await ApprovePendingChange(id)
      else await RejectPendingChange(id)
      await load()
    } catch (err) {
      // Shown, not swallowed. An approval that could not be applied leaves the
      // proposal exactly where it was, and a button that seems to do nothing is
      // how a person decides the feature is broken.
      error = String(err)
    } finally {
      busy = false
    }
  }

  const pending = $derived(change?.state === 'pending')
  const isSkill = $derived(change?.kind === 'skill')
  const meta = $derived(scopeMeta(change?.scope ?? ''))
  const scope = $derived(isSkill ? (change?.scope || 'skill') : meta.label)
  const canRedirect = $derived(!isSkill && change?.op === 'add')
  // The verb goes in the heading, which is why no `op` badge is drawn: "ADD"
  // was the database's own enum, in English, in the middle of a Thai sentence.
  // The one operation the heading cannot fully carry is a replacement, and that
  // one shows the line it overwrites instead.
  const asking = $derived(
    isSkill
      ? (change?.op === 'create' ? t('chat.skillCreateAsk') : t('chat.skillTuneAsk'))
      : change?.op === 'remove' ? t('chat.memoryForgetAsk')
      : change?.op === 'replace' ? t('chat.memoryReplaceAsk')
      : t('chat.memoryAsk'),
  )
  const askIcon = $derived(
    isSkill ? 'sparkles'
      : change?.op === 'remove' ? 'x' : change?.op === 'replace' ? 'refreshCw' : 'brain',
  )
  const settled = $derived(
    isSkill
      ? (change?.state === 'approved' ? (change?.op === 'create' ? t('chat.skillCreated') : t('chat.skillTuned')) : t('chat.skillDropped'))
      : (change?.state === 'approved' ? t('chat.memoryKept') : t('chat.memoryDropped'))
  )
  // A removal has no new text, so the line it is about is the one to show.
  const line = $derived(change?.body || change?.before || '')
</script>

{#if change && pending}
  <div class="memcard" data-guide="memory.card">
    <div class="memcard-head">
      <span class="ic"><Icon name={askIcon} size={15} /></span>
      <span class="memcard-kind">{asking}</span>
      <!-- Whose memory this lands in, at the top right, because it is the
           question being decided: a line true everywhere costs every session,
           and one true only here costs nothing anywhere else. -->
      {#if isSkill}
        <span class="memcard-scope" data-guide="memory.card.scope">{scope}</span>
      {:else}
        <span class="memcard-scope learn-scope mem-tone-{meta.tone}" data-guide="memory.card.scope"><ScopeMark {meta} size={11} face={16} /> {meta.label}</span>
      {/if}
    </div>
    {#if !isSkill}
      <!-- Who will read it, in words, under the name — the sentence the label
           alone never said (memoryScope.ts). -->
      <div class="memcard-aud">{meta.audience}</div>
    {/if}
    {#if change.before}
      <!-- What it overwrites, struck through above what it becomes: approving a
           change without seeing what it replaces is not a decision. -->
      <div class="memcard-was" class:learn-doc={change.kind === 'skill'}>{change.before}</div>
    {/if}
    {#if change.body}<div class="memcard-line" class:learn-doc={change.kind === 'skill'}>{change.body}</div>{/if}
    {#if change.reason}
      <div class="memcard-why">{t('chat.memoryBecause')} {change.reason}</div>
    {/if}
    {#if error}<div class="memcard-error">{error}</div>{/if}
    <div class="memcard-foot">
      <!-- Its own room in the footer rather than crammed beside the buttons:
           it is the one thing about this card a person assumes wrongly. -->
      <span class="memcard-note">{t('chat.memoryNextChat')}</span>
      {#if canRedirect}
        <div class="mem-move">
          <button data-guide="memory.card.edit" type="button" class="memcard-no" disabled={busy} aria-expanded={elsewhere}
            onclick={openElsewhere}>{t('settings.learningKeepElsewhere')} <Icon name="chevronDown" size={11} /></button>
          {#if elsewhere}
            <div class="mem-menu" role="menu">
              <div class="mem-menu-h">{t('settings.learningKeepIn')}</div>
              {#each targets as m (m.scope)}
                <button type="button" class="mem-menu-i" role="menuitem" onclick={() => decide(true, m.scope)}>
                  <span class="learn-scope mem-tone-{m.tone}"><ScopeMark meta={m} size={11} face={16} /> {m.label}</span>
                  <small>{m.audience}</small>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
      <button data-guide="memory.card.reject" type="button" class="memcard-no" disabled={busy}
        onclick={() => decide(false)}>{t('settings.learningReject')}</button>
      <button data-guide="memory.card.accept" type="button" class="memcard-yes" disabled={busy}
        onclick={() => decide(true)}>{t('settings.learningApprove')}</button>
    </div>
  </div>
{:else if change}
  <div class="memdone" class:open={detail}>
    <button type="button" class="memdone-head" onclick={() => (detail = !detail)} aria-expanded={detail}>
      <span class="ic"><Icon name={change.state === 'approved' ? 'check' : 'x'} size={13} /></span>
      <span class="memdone-what">{settled}</span>
      {#if isSkill}
        <span class="memdone-scope">{scope}</span>
      {:else}
        <span class="memdone-scope learn-scope mem-tone-{meta.tone}"><ScopeMark {meta} size={10} face={15} /> {meta.label}</span>
      {/if}
      <span class="chev"><Icon name={detail ? 'chevronDown' : 'chevronRight'} size={12} /></span>
    </button>
    {#if detail}
      <div class="memdone-body">
        {#if change.before}<div class="memcard-was" class:learn-doc={change.kind === 'skill'}>{change.before}</div>{/if}
        {#if line}<div class="memdone-line" class:learn-doc={change.kind === 'skill'}>{line}</div>{/if}
        {#if change.reason}
          <div class="memcard-why">{t('chat.memoryBecause')} {change.reason}</div>
        {/if}
      </div>
    {/if}
  </div>
{/if}
