<script lang="ts">
  import { cockpit, startPlanRun, stopPlanRun, savePlanText, setPlanStepStop } from '../stores/cockpit.svelte'
  import { renderMarkdown } from '../markdown'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import type { Plan } from '../types'

  const plan = $derived(cockpit.plan)

  let planDraft = $state<string | null>(null)
  let planRefusal = $state('')

  const planHeadingLabel = (heading: string) => {
    const map: Record<string, string> = {
      'What is there now': 'chat.planHead.whatIsThereNow',
      'What to change': 'chat.planHead.whatToChange',
      'What could go wrong': 'chat.planHead.whatCouldGoWrong',
      'How you will know it worked': 'chat.planHead.howYouWillKnowItWorked',
      'What you are unsure of': 'chat.planHead.whatYouAreUnsureOf',
    }
    const key = map[heading]
    return key ? t(key as any) : heading
  }

  const planNow = (p: Plan) => {
    const steps = p.steps ?? []
    return (
      steps.find((st) => st.state === 'doing') ??
      steps.find((st) => st.state !== 'done' && st.state !== 'failed')
    )
  }

  let runTick = $state(0)
  $effect(() => {
    if (!plan?.running) return
    const id = setInterval(() => (runTick += 1), 15000)
    return () => clearInterval(id)
  })

  const planDone = (p: Plan) =>
    (p.steps ?? []).filter((st) => st.state === 'done' || st.state === 'failed').length

  const planAllDone = (p: Plan) => {
    const steps = p.steps ?? []
    return steps.length > 0 && steps.every((st) => st.state === 'done' || st.state === 'failed')
  }

  const planElapsed = (p: Plan) => {
    void runTick
    if (!p.startedAt) return ''
    const began = Date.parse(p.startedAt)
    if (Number.isNaN(began)) return ''
    const mins = Math.max(0, Math.floor((Date.now() - began) / 60000))
    return mins < 1 ? t('chat.planJustNow') : t('chat.planMinutes', { n: String(mins) })
  }

  const planAsMarkdown = (p: Plan) => {
    const title = p.title ? `# ${p.title}\n\n` : ''
    const secs = (p.sections ?? []).map((s) => `**${s.heading}**\n${s.body}`).join('\n\n')
    const steps = (p.steps ?? []).map((s) => `${s.n}. ${s.text}`).join('\n')
    return title + secs + (steps ? `\n\n${steps}` : '')
  }

  function onEditPlan() {
    if (!plan) return
    planDraft = planAsMarkdown(plan)
    planRefusal = ''
  }

  async function onSavePlan() {
    if (planDraft === null) return
    planRefusal = await savePlanText(planDraft)
    if (!planRefusal) planDraft = null
  }

  async function onStartPlanRun() {
    planRefusal = await startPlanRun()
  }

  async function onStopPlanRun() {
    await stopPlanRun()
  }

  function onCopy() {
    if (!plan) return
    navigator.clipboard.writeText(planAsMarkdown(plan))
  }
</script>

<div class="plan-pane">
  {#if !plan}
    <div class="plan-pane-empty">
      <span class="empty-icon"><Icon name="compass" size={36} /></span>
      <h3>{t('chat.planCard')}</h3>
      <p>{t('chat.planEmptyHint')}</p>
    </div>
  {:else}
    <div class="plan-card pane-card" class:running={plan.running} data-plan={planAsMarkdown(plan)}>
      {#if plan.running}
        {@const now = planNow(plan)}
        <div class="runbar">
          <div class="runbar-top">
            <span class="runbar-now">
              <span class="livedot"></span>{now ? now.text : t('chat.planRunning', { done: String(planDone(plan)), total: String((plan.steps ?? []).length) })}
            </span>
            <span class="runbar-meta">
              {planDone(plan)}/{(plan.steps ?? []).length}{planElapsed(plan) ? ' · ' + planElapsed(plan) : ''}{plan.sentBack ? ' · ' + t('chat.planSentBack', { n: String(plan.sentBack) }) : ''}
            </span>
            <button class="plan-run-stop" type="button" onclick={onStopPlanRun}>{t('chat.planStop')}</button>
          </div>
          <div class="runbar-track"><i style="width:{(plan.steps ?? []).length ? Math.round((planDone(plan) / (plan.steps ?? []).length) * 100) : 0}%"></i></div>
        </div>
      {/if}

      <div class="plan-head">
        <span class="plan-kind"><Icon name="compass" size={13} />{t('chat.planCard')}</span>
        {#if plan.version > 1}
          <span class="plan-rev">{t('chat.planRevision', { n: String(plan.version) })}</span>
        {/if}
        {#if planAllDone(plan)}
          <span class="plan-chip-badge done">{t('chat.planDoneBadge')}</span>
        {/if}
        {#if planDraft === null}
          <button class="plan-edit" type="button" onclick={onEditPlan}>{t('chat.planEdit')}</button>
        {/if}
        <button class="plan-copy" type="button" onclick={onCopy}>{t('chat.copyCode')}</button>
      </div>

      {#if planDraft !== null}
        <textarea class="plan-edit-box" bind:value={planDraft} spellcheck="false"></textarea>
        <div class="plan-foot">
          <span class="plan-edit-hint">{t('chat.planEditHint')}</span>
          <button class="plan-run-stop" type="button" onclick={() => (planDraft = null)}>{t('chat.planEditCancel')}</button>
          <button class="plan-run" type="button" onclick={onSavePlan}>{t('chat.planEditSave')}</button>
        </div>
        {#if planRefusal}<p class="plan-refusal">{planRefusal}</p>{/if}
      {:else}
        {#if plan.title}<h3 class="plan-title">{plan.title}</h3>{/if}
        <div class="plan-body">
          {#each plan.sections ?? [] as sec}
            <h4 class="plan-heading" class:plan-changed={(plan.changed ?? []).includes(sec.heading)}>
              {planHeadingLabel(sec.heading)}
            </h4>
            <div class="markdown-body">{@html renderMarkdown(sec.body)}</div>
          {/each}

          {#if (plan.steps ?? []).length > 0}
            <ol class="plan-steps">
              {#each plan.steps ?? [] as st}
                <li class="plan-step" data-state={st.state || 'todo'} class:bp={st.stop}>
                  <span class="plan-step-mark" aria-hidden="true"></span>
                  <span class="plan-step-text">{st.text}</span>
                  {#if !(st.state === 'done' || st.state === 'failed')}
                    <button
                      class="plan-step-bp"
                      class:on={st.stop}
                      type="button"
                      title={t('chat.planStopHere')}
                      onclick={() => setPlanStepStop(st.n, !st.stop)}
                    >{t('chat.planStopHere')}</button>
                  {/if}
                  {#if st.note}<span class="plan-step-note">{st.note}</span>{/if}
                </li>
              {/each}
            </ol>
          {/if}
        </div>

        {#if (plan.steps ?? []).length > 0 && !plan.running && !planAllDone(plan)}
          <div class="plan-foot">
            <button class="plan-run" type="button" onclick={onStartPlanRun}>
              <Icon name="play" size={13} />
              {t('chat.planStart')}
            </button>
          </div>
          {#if planRefusal}<p class="plan-refusal">{planRefusal}</p>{/if}
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .plan-pane {
    height: 100%;
    overflow-y: auto;
    padding: 14px;
    box-sizing: border-box;
  }
  .plan-pane-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    height: 100%;
    min-height: 280px;
    color: var(--text-dim);
    gap: 10px;
    padding: 30px 20px;
  }
  .empty-icon {
    opacity: 0.4;
    color: var(--text-dim);
  }
  .plan-pane-empty h3 {
    margin: 0;
    font-size: var(--fs-md);
    color: var(--text-muted);
  }
  .plan-pane-empty p {
    margin: 0;
    font-size: var(--fs-xs);
    max-width: 260px;
    line-height: 1.5;
  }
  .pane-card {
    margin: 0;
    background: transparent;
    border: none;
    padding: 0;
  }
</style>
