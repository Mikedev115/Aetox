<script lang="ts">
  import { cockpit, startPlanRun, stopPlanRun, savePlanText, setPlanStepStop, sendUserMessage } from '../stores/cockpit.svelte'
  import { openArtifactsTab } from '../stores/workbench.svelte'
  import { renderMarkdown } from '../markdown'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import type { IconName } from '../icons'
  import type { Plan } from '../types'

  let { embedded = false }: { embedded?: boolean } = $props()

  const plan = $derived(cockpit.plan)

  let planDraft = $state<string | null>(null)
  let planRefusal = $state('')
  let copied = $state(false)
  let copyTimer: ReturnType<typeof setTimeout> | undefined

  let feedbackOpen = $state(false)
  let feedbackText = $state('')
  let feedbackSubmitting = $state(false)
  let feedbackSentNotice = $state(false)
  let feedbackNoticeTimer: ReturnType<typeof setTimeout> | undefined

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

  type SecKind = 'scope' | 'risk' | 'success' | 'neutral'

  function sectionKind(heading: string): SecKind {
    const h = heading.toLowerCase()
    if (h.includes('wrong') || h.includes('risk') || h.includes('ปัญหา') || h.includes('เสี่ยง') || h.includes('unsure')) {
      return 'risk'
    }
    if (h.includes('worked') || h.includes('success') || h.includes('สำเร็จ') || h.includes('ตรวจ') || h.includes('verify')) {
      return 'success'
    }
    if (h.includes('change') || h.includes('ขั้นตอน') || h.includes('scope') || h.includes('ดำเนิ') || h.includes('action')) {
      return 'scope'
    }
    return 'neutral'
  }

  function sectionIcon(kind: SecKind): IconName {
    if (kind === 'risk') return 'alertTriangle'
    if (kind === 'success') return 'check'
    if (kind === 'scope') return 'layoutList'
    return 'compass'
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

  $effect(() => {
    function handleOpenFeedback() {
      feedbackOpen = true
      planDraft = null
    }
    window.addEventListener('open-plan-feedback', handleOpenFeedback)
    return () => window.removeEventListener('open-plan-feedback', handleOpenFeedback)
  })

  const planDone = (p: Plan) =>
    (p.steps ?? []).filter((st) => st.state === 'done' || st.state === 'failed').length

  const planAllDone = (p: Plan) => {
    const steps = p.steps ?? []
    return steps.length > 0 && steps.every((st) => st.state === 'done' || st.state === 'failed')
  }

  const planPct = (p: Plan) => {
    const steps = p.steps ?? []
    if (!steps.length) return 0
    return Math.min(100, Math.round((planDone(p) / steps.length) * 100))
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
    feedbackOpen = false
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
    copied = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied = false
    }, 2000)
  }

  const quickFeedbackOptions = $derived([
    {
      icon: 'check' as IconName,
      label: t('chat.planChipVerify'),
      prompt: t('chat.planPromptVerify'),
    },
    {
      icon: 'alertTriangle' as IconName,
      label: t('chat.planChipRisk'),
      prompt: t('chat.planPromptRisk'),
    },
    {
      icon: 'zap' as IconName,
      label: t('chat.planChipPhased'),
      prompt: t('chat.planPromptPhased'),
    },
    {
      icon: 'fileCode' as IconName,
      label: t('chat.planChipDetails'),
      prompt: t('chat.planPromptDetails'),
    },
  ])

  function applyQuickFeedback(promptText: string) {
    if (!feedbackText.trim()) {
      feedbackText = promptText
    } else if (!feedbackText.includes(promptText)) {
      feedbackText = `${feedbackText.trim()}\n- ${promptText}`
    }
  }

  function onFeedbackKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      onSubmitFeedback()
    }
  }

  async function onSubmitFeedback() {
    const trimmed = feedbackText.trim()
    if (!trimmed || !plan || feedbackSubmitting) return
    feedbackSubmitting = true
    try {
      const prompt = `ขอให้ช่วยรีวิวและปรับปรุงแผนงาน "${plan.title || 'แผนงาน'}" (เวอร์ชัน ${plan.version || 1}) ตามข้อเสนอแนะนี้:\n\n${trimmed}`
      await sendUserMessage(prompt)
      feedbackText = ''
      feedbackOpen = false
      feedbackSentNotice = true
      if (feedbackNoticeTimer) clearTimeout(feedbackNoticeTimer)
      feedbackNoticeTimer = setTimeout(() => {
        feedbackSentNotice = false
      }, 5000)
    } finally {
      feedbackSubmitting = false
    }
  }

  function formatStepText(text: string): string {
    if (!text) return ''
    const escaped = text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
    const backticked = escaped.replace(/`([^`]+)`/g, '<code class="step-code">$1</code>')
    return backticked.replace(
      /\b([a-zA-Z0-9_\-\/]+\.(?:csv|xlsx|md|html|json|ts|js|go|py|css|png|jpg))\b(?![^<]*>)/gi,
      '<code class="step-code">$1</code>'
    )
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
    <div class="plan-container" class:running={plan.running} data-plan={planAsMarkdown(plan)}>

      <!-- 1. HERO HEADER CARD -->
      <div class="plan-hero-card">
        <!-- Top Toolbar -->
        <div class="hero-top-row">
          <div class="hero-tags">
            <span class="plan-tag-badge">
              <Icon name="compass" size={12} />
              <span>{t('chat.planCard')}</span>
              {#if plan.version > 1}
                <span class="plan-ver">#{plan.version}</span>
              {/if}
            </span>

            {#if plan.running}
              <span class="plan-status-pill running">
                <span class="status-dot pulse"></span>
                <span>{t('bgw.running')}</span>
              </span>
            {:else if planAllDone(plan)}
              <span class="plan-status-pill done">
                <Icon name="check" size={11} />
                <span>{t('chat.planDoneBadge')}</span>
              </span>
            {:else}
              <span class="plan-status-pill ready">
                <span>{t('chat.planReadyBadge')}</span>
              </span>
            {/if}
          </div>

          <div class="hero-actions">
            {#if !embedded && planDraft === null}
              <button
                class="hero-btn"
                type="button"
                onclick={() => openArtifactsTab('session-plan')}
                title={t('workbench.artifactsTab')}
              >
                <Icon name="package" size={13} />
                <span>{t('workbench.artifactsTab')}</span>
              </button>
            {/if}
            {#if planDraft === null}
              <button
                class="hero-btn feedback-btn"
                class:active={feedbackOpen}
                type="button"
                onclick={() => {
                  feedbackOpen = !feedbackOpen
                  if (feedbackOpen) planDraft = null
                }}
                title={t('chat.planReviewFeedback')}
              >
                <Icon name="messageSquare" size={13} />
                <span>{t('chat.planReviewFeedback')}</span>
              </button>
              <button
                class="hero-btn"
                type="button"
                onclick={onEditPlan}
                title={t('chat.planEditManual')}
              >
                <Icon name="pencil" size={13} />
                <span>{t('chat.planEditManual')}</span>
              </button>
            {/if}
            <button class="hero-btn" class:success={copied} type="button" onclick={onCopy}>
              <Icon name={copied ? 'check' : 'copy'} size={13} />
              <span>{copied ? t('chat.copiedCode') : t('chat.copyCode')}</span>
            </button>
          </div>
        </div>

        <!-- Plan Title -->
        {#if planDraft === null && plan.title}
          <h2 class="hero-title">{plan.title}</h2>
        {/if}

        <!-- Live Progress Metric Bar -->
        {#if (plan.steps ?? []).length > 0}
          <div class="hero-progress-box">
            <div class="progress-info-row">
              <div class="progress-left">
                <span class="progress-label">{t('chat.planExecutionSteps')}</span>
                <span class="progress-dot">·</span>
                <span class="progress-steps-count">
                  {t('chat.planProgress', {
                    done: String(planDone(plan)),
                    total: String((plan.steps ?? []).length),
                  })}
                </span>
              </div>
              <span class="progress-percentage">{planPct(plan)}%</span>
            </div>

            <div class="progress-track">
              <div class="progress-fill" style="width: {planPct(plan)}%"></div>
            </div>

            <div class="progress-meta-row">
              <span>
                {planElapsed(plan) ? `${planElapsed(plan)} · ` : ''}
                {plan.running && planNow(plan) ? planNow(plan)?.text : t('chat.planStepCount', { n: String((plan.steps ?? []).length) })}
              </span>
              {#if plan.sentBack}
                <span class="sent-back-tag">{t('chat.planSentBack', { n: String(plan.sentBack) })}</span>
              {/if}
            </div>
          </div>
        {/if}
      </div>

      <!-- Feedback Sent Notice -->
      {#if feedbackSentNotice}
        <div class="plan-feedback-notice">
          <span class="notice-icon"><Icon name="check" size={13} /></span>
          <span>{t('chat.planFeedbackSentNotice')}</span>
        </div>
      {/if}

      <!-- Interactive Plan Review & Feedback Panel -->
      {#if feedbackOpen && planDraft === null}
        <div class="plan-feedback-card">
          <div class="feedback-card-header">
            <div class="feedback-header-title">
              <span class="feedback-icon"><Icon name="messageSquare" size={14} /></span>
              <span class="feedback-title-text">{t('chat.planFeedbackTitle')}</span>
              <span class="feedback-ver-tag">v{plan.version || 1}</span>
            </div>
            <button
              class="feedback-close-btn"
              type="button"
              onclick={() => (feedbackOpen = false)}
              title={t('chat.planFeedbackCancel')}
            >
              <Icon name="x" size={14} />
            </button>
          </div>

          <p class="feedback-card-desc">{t('chat.planFeedbackDesc')}</p>

          <div class="feedback-chips">
            {#each quickFeedbackOptions as opt}
              <button
                class="feedback-chip"
                type="button"
                onclick={() => applyQuickFeedback(opt.prompt)}
              >
                <span class="chip-icon"><Icon name={opt.icon} size={11} /></span>
                <span>{opt.label}</span>
              </button>
            {/each}
          </div>

          <div class="feedback-input-wrap">
            <textarea
              class="feedback-textarea"
              bind:value={feedbackText}
              placeholder={t('chat.planFeedbackPlaceholder')}
              rows="3"
              onkeydown={onFeedbackKeydown}
            ></textarea>
          </div>

          <div class="feedback-card-footer">
            <span class="feedback-shortcut-hint">
              <span>{t('chat.planFeedbackHint')}</span>
            </span>
            <div class="feedback-actions">
              <button
                class="feedback-btn-cancel"
                type="button"
                onclick={() => {
                  feedbackOpen = false
                  feedbackText = ''
                }}
              >
                {t('chat.planFeedbackCancel')}
              </button>
              <button
                class="feedback-btn-send"
                type="button"
                disabled={!feedbackText.trim() || feedbackSubmitting}
                onclick={onSubmitFeedback}
              >
                {#if feedbackSubmitting}
                  <span class="spin-icon"><Icon name="refreshCw" size={12} /></span>
                  <span>{t('chat.planFeedbackSending')}</span>
                {:else}
                  <Icon name="sendHorizontal" size={13} />
                  <span>{t('chat.planFeedbackSend')}</span>
                {/if}
              </button>
            </div>
          </div>
        </div>
      {/if}

      <!-- EDIT MODE -->
      {#if planDraft !== null}
        <div class="plan-edit-area">
          <textarea class="plan-edit-box" bind:value={planDraft} spellcheck="false"></textarea>
          <div class="plan-foot">
            <span class="plan-edit-hint">{t('chat.planEditHint')}</span>
            <button class="plan-run-stop" type="button" onclick={() => (planDraft = null)}>{t('chat.planEditCancel')}</button>
            <button class="plan-run" type="button" onclick={onSavePlan}>{t('chat.planEditSave')}</button>
          </div>
          {#if planRefusal}<p class="plan-refusal">{planRefusal}</p>{/if}
        </div>
      {:else}

        <!-- 2. STRUCTURED SECTION CALLOUTS -->
        {#if (plan.sections ?? []).length > 0}
          <div class="plan-sections-container">
            {#each plan.sections ?? [] as sec}
              {@const kind = sectionKind(sec.heading)}
              <div
                class="plan-sec-callout kind-{kind}"
                class:plan-changed={(plan.changed ?? []).includes(sec.heading)}
              >
                <div class="sec-callout-head">
                  <span class="sec-callout-icon">
                    <Icon name={sectionIcon(kind)} size={13} />
                  </span>
                  <span class="sec-callout-title">{planHeadingLabel(sec.heading)}</span>
                </div>
                <div class="markdown-body sec-callout-body">
                  {@html renderMarkdown(sec.body)}
                </div>
              </div>
            {/each}
          </div>
        {/if}

        <!-- 3. MODERN VERTICAL EXECUTION STEPPER -->
        {#if (plan.steps ?? []).length > 0}
          <div class="plan-stepper-box">
            <div class="stepper-box-head">
              <div class="stepper-title-wrap">
                <Icon name="layoutList" size={15} />
                <span class="stepper-title">{t('chat.planExecutionSteps')}</span>
                <span class="stepper-badge">{(plan.steps ?? []).length}</span>
              </div>
              <span class="stepper-pct-badge">{planPct(plan)}%</span>
            </div>

            <!-- Vertical Stepper Timeline -->
            <div class="stepper-timeline">
              <div class="stepper-line">
                <div class="stepper-line-fill" style="height: {planPct(plan)}%"></div>
              </div>

              {#each plan.steps ?? [] as st, idx}
                {@const isDone = st.state === 'done'}
                {@const isDoing = st.state === 'doing'}
                {@const isFailed = st.state === 'failed'}
                {@const isTodo = !st.state || st.state === 'todo'}

                <div class="stepper-item-row" data-state={st.state || 'todo'} class:bp={st.stop}>
                  <!-- State Node Icon -->
                  <div
                    class="stepper-node"
                    class:done={isDone}
                    class:doing={isDoing}
                    class:failed={isFailed}
                    class:todo={isTodo}
                  >
                    {#if isDone}
                      <Icon name="check" size={12} />
                    {:else if isDoing}
                      <span class="spin-icon"><Icon name="refreshCw" size={11} /></span>
                    {:else if isFailed}
                      <Icon name="x" size={12} />
                    {:else}
                      <span class="node-num">{String(st.n || idx + 1).padStart(2, '0')}</span>
                    {/if}
                  </div>

                  <!-- Step Card -->
                  <div class="stepper-card" class:active={isDoing} class:failed={isFailed}>
                    <div class="stepper-card-top">
                      <div class="stepper-card-meta">
                        <span class="step-num-pill">STEP {String(st.n || idx + 1).padStart(2, '0')}</span>
                        <span class="step-state-tag {st.state || 'todo'}">
                          {#if isDone}
                            {t('chat.stepDone')}
                          {:else if isDoing}
                            {t('chat.stepDoing')}
                          {:else if isFailed}
                            {t('chat.stepFailed')}
                          {:else}
                            {t('chat.stepTodo')}
                          {/if}
                        </span>
                      </div>

                      {#if !(isDone || isFailed)}
                        <button
                          class="stepper-bp-btn"
                          class:active={st.stop}
                          type="button"
                          title={t('chat.planStopHere')}
                          onclick={() => setPlanStepStop(st.n, !st.stop)}
                        >
                          <span class="bp-circle"></span>
                          <span>{t('chat.planStopHere')}</span>
                        </button>
                      {/if}
                    </div>

                    <div class="stepper-card-text">
                      {@html formatStepText(st.text)}
                    </div>

                    {#if st.note}
                      <div class="stepper-note">
                        <Icon name="alertTriangle" size={12} />
                        <span>{st.note}</span>
                      </div>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>

            <!-- Execution Action Bar Footer -->
            <div class="stepper-actions-foot">
              {#if plan.running}
                <div class="run-status-notice">
                  <span class="pulse-dot"></span>
                  <span>{planNow(plan) ? planNow(plan)?.text : t('chat.planRunning', { done: String(planDone(plan)), total: String((plan.steps ?? []).length) })}</span>
                </div>
                <button class="plan-stop-btn" type="button" onclick={onStopPlanRun}>
                  <Icon name="x" size={13} />
                  <span>{t('chat.planStop')}</span>
                </button>
              {:else if !planAllDone(plan)}
                <button class="plan-start-btn" type="button" onclick={onStartPlanRun}>
                  <Icon name="play" size={13} />
                  <span>{t('chat.planStart')}</span>
                </button>
              {:else}
                <div class="all-done-notice">
                  <Icon name="check" size={14} />
                  <span>{t('chat.planDoneBadge')} — {t('chat.planProgress', { done: String((plan.steps ?? []).length), total: String((plan.steps ?? []).length) })}</span>
                </div>
              {/if}
            </div>
            {#if planRefusal}<p class="plan-refusal">{planRefusal}</p>{/if}
          </div>
        {/if}

      {/if}
    </div>
  {/if}
</div>

<style>
  .plan-pane {
    height: 100%;
    overflow-y: auto;
    padding: 16px;
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

  .plan-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 860px;
    margin: 0 auto;
  }

  /* 1. HERO HEADER CARD */
  .plan-hero-card {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    padding: 16px 18px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .hero-top-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    flex-wrap: wrap;
  }

  .hero-tags {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .plan-tag-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 3px 9px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 500;
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
    border: 1px solid color-mix(in srgb, var(--accent) 25%, transparent);
  }

  .plan-ver {
    font-size: var(--fs-2xs);
    opacity: 0.8;
  }

  .plan-status-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 3px 9px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 600;
  }

  .plan-status-pill.done {
    background: color-mix(in srgb, var(--status-success, #10b981) 14%, transparent);
    color: var(--status-success, #10b981);
    border: 1px solid color-mix(in srgb, var(--status-success, #10b981) 28%, transparent);
  }

  .plan-status-pill.running {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    color: var(--accent);
    border: 1px solid color-mix(in srgb, var(--accent) 35%, transparent);
  }

  .plan-status-pill.ready {
    background: color-mix(in srgb, var(--text-muted) 12%, transparent);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }

  .status-dot.pulse {
    animation: dot-pulse 1.4s infinite ease-in-out;
  }

  @keyframes dot-pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.3; transform: scale(1.2); }
  }

  .hero-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .hero-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 10px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--surface-raised);
    border: 1px solid var(--border-subtle);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .hero-btn:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
    border-color: var(--border-default);
  }

  .hero-btn.success {
    color: var(--status-success, #10b981);
    border-color: var(--status-success, #10b981);
  }

  .hero-btn.feedback-btn {
    color: var(--accent);
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-raised));
    border-color: color-mix(in srgb, var(--accent) 28%, transparent);
    font-weight: 600;
  }

  .hero-btn.feedback-btn:hover,
  .hero-btn.feedback-btn.active {
    background: color-mix(in srgb, var(--accent) 18%, var(--surface-raised));
    border-color: var(--accent);
    color: var(--accent);
  }

  /* Feedback Sent Toast / Notice */
  .plan-feedback-notice {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 14px;
    border-radius: var(--r-md);
    background: color-mix(in srgb, var(--status-success, #10b981) 12%, var(--surface-panel));
    border: 1px solid color-mix(in srgb, var(--status-success, #10b981) 30%, transparent);
    color: var(--status-success, #10b981);
    font-size: var(--fs-xs);
    font-weight: 500;
    animation: feedbackSlideIn 0.2s ease-out;
  }

  .notice-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  /* Interactive Feedback Card */
  .plan-feedback-card {
    background: var(--surface-panel);
    border: 1px solid color-mix(in srgb, var(--accent) 35%, var(--border-subtle));
    border-radius: var(--r-lg);
    padding: 16px 18px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
    display: flex;
    flex-direction: column;
    gap: 12px;
    animation: feedbackSlideIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes feedbackSlideIn {
    from {
      opacity: 0;
      transform: translateY(-6px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .feedback-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .feedback-header-title {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .feedback-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    border-radius: var(--r-sm);
    background: color-mix(in srgb, var(--accent) 15%, transparent);
    color: var(--accent);
  }

  .feedback-title-text {
    font-size: var(--fs-sm);
    font-weight: 600;
    color: var(--text-primary);
  }

  .feedback-ver-tag {
    font-size: var(--fs-2xs);
    font-weight: 600;
    padding: 1px 6px;
    border-radius: var(--r-xs);
    background: var(--surface-raised);
    color: var(--text-muted);
    border: 1px solid var(--border-subtle);
  }

  .feedback-close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
    border-radius: var(--r-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s, background 0.15s;
  }

  .feedback-close-btn:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
  }

  .feedback-card-desc {
    margin: 0;
    font-size: var(--fs-xs);
    color: var(--text-muted);
    line-height: 1.4;
  }

  .feedback-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .feedback-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 10px;
    border-radius: var(--r-full, 9999px);
    font-size: var(--fs-2xs);
    font-weight: 500;
    background: var(--surface-raised);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .feedback-chip:hover {
    background: color-mix(in srgb, var(--accent) 12%, var(--surface-raised));
    border-color: color-mix(in srgb, var(--accent) 40%, transparent);
    color: var(--accent);
    transform: translateY(-1px);
  }

  .chip-icon {
    display: inline-flex;
    align-items: center;
    opacity: 0.85;
  }

  .feedback-input-wrap {
    position: relative;
  }

  .feedback-textarea {
    width: 100%;
    box-sizing: border-box;
    min-height: 84px;
    max-height: 220px;
    resize: vertical;
    padding: 10px 12px;
    border-radius: var(--r-md);
    background: var(--surface-raised);
    border: 1px solid var(--border-default);
    color: var(--text-primary);
    font-size: var(--fs-sm);
    font-family: inherit;
    line-height: 1.5;
    outline: none;
    transition: border-color 0.15s;
  }

  .feedback-textarea:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent) 25%, transparent);
  }

  .feedback-card-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    flex-wrap: wrap;
  }

  .feedback-shortcut-hint {
    font-size: var(--fs-2xs);
    color: var(--text-muted);
  }

  .feedback-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-left: auto;
  }

  .feedback-btn-cancel {
    padding: 6px 12px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 500;
    color: var(--text-muted);
    background: transparent;
    border: 1px solid transparent;
    cursor: pointer;
    transition: all 0.15s;
  }

  .feedback-btn-cancel:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
  }

  .feedback-btn-send {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 14px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 600;
    color: #fff;
    background: var(--accent);
    border: 1px solid var(--accent);
    cursor: pointer;
    transition: all 0.15s;
  }

  .feedback-btn-send:hover:not(:disabled) {
    background: color-mix(in srgb, var(--accent) 85%, white);
    transform: translateY(-1px);
  }

  .feedback-btn-send:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .hero-title {
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: 700;
    line-height: 1.35;
    color: var(--text-primary);
    letter-spacing: -0.01em;
  }

  .hero-progress-box {
    background: var(--surface-raised);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    padding: 10px 12px;
    display: flex;
    flex-direction: column;
    gap: 7px;
  }

  .progress-info-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: var(--fs-xs);
  }

  .progress-left {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--text-secondary);
  }

  .progress-steps-count {
    color: var(--accent);
    font-weight: 600;
  }

  .progress-percentage {
    color: var(--accent);
    font-weight: 700;
    font-family: monospace;
  }

  .progress-track {
    height: 6px;
    background: color-mix(in srgb, var(--border-subtle) 80%, transparent);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(to right, var(--accent), color-mix(in srgb, var(--accent) 80%, white));
    border-radius: 3px;
    transition: width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .progress-meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: var(--fs-2xs);
    color: var(--text-muted);
  }

  .sent-back-tag {
    color: var(--status-warn);
  }

  /* 2. STRUCTURED SECTION CALLOUTS */
  .plan-sections-container {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 12px;
  }

  .plan-sec-callout {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    border-left-width: 4px;
  }

  .plan-sec-callout.kind-scope {
    border-left-color: var(--accent);
  }

  .plan-sec-callout.kind-risk {
    border-left-color: var(--status-warn, #f59e0b);
  }

  .plan-sec-callout.kind-success {
    border-left-color: var(--status-success, #10b981);
  }

  .plan-sec-callout.kind-neutral {
    border-left-color: var(--border-default);
  }

  .sec-callout-head {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: var(--fs-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .plan-sec-callout.kind-scope .sec-callout-head {
    color: var(--accent);
  }

  .plan-sec-callout.kind-risk .sec-callout-head {
    color: var(--status-warn, #f59e0b);
  }

  .plan-sec-callout.kind-success .sec-callout-head {
    color: var(--status-success, #10b981);
  }

  .plan-sec-callout.kind-neutral .sec-callout-head {
    color: var(--text-secondary);
  }

  .sec-callout-body {
    font-size: var(--fs-xs);
    color: var(--text-secondary);
    line-height: 1.5;
  }

  /* 3. MODERN VERTICAL EXECUTION STEPPER */
  .plan-stepper-box {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    padding: 16px 18px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .stepper-box-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .stepper-title-wrap {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: var(--fs-sm);
    font-weight: 600;
    color: var(--text-primary);
  }

  .stepper-badge {
    padding: 1px 7px;
    border-radius: var(--r-sm);
    font-size: var(--fs-2xs);
    font-weight: 500;
    background: var(--surface-raised);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  .stepper-pct-badge {
    font-size: var(--fs-xs);
    font-weight: 600;
    color: var(--accent);
  }

  .stepper-timeline {
    position: relative;
    padding-left: 38px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .stepper-line {
    position: absolute;
    top: 14px;
    bottom: 14px;
    left: 14px;
    width: 2px;
    background: var(--border-subtle);
    border-radius: 1px;
    overflow: hidden;
  }

  .stepper-line-fill {
    width: 100%;
    background: var(--status-success, #10b981);
    transition: height 0.3s ease;
  }

  .stepper-item-row {
    position: relative;
    display: flex;
    flex-direction: column;
  }

  .stepper-node {
    position: absolute;
    left: -38px;
    top: 6px;
    width: 28px;
    height: 28px;
    border-radius: var(--r-md);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--fs-xs);
    font-weight: 700;
    box-sizing: border-box;
    z-index: 2;
    transition: all 0.2s ease;
  }

  .stepper-node.done {
    background: var(--status-success, #10b981);
    color: #fff;
    box-shadow: 0 0 8px color-mix(in srgb, var(--status-success, #10b981) 40%, transparent);
  }

  .stepper-node.doing {
    background: var(--accent);
    color: #fff;
    box-shadow: 0 0 10px color-mix(in srgb, var(--accent) 50%, transparent);
    animation: node-doing-pulse 1.8s infinite ease-in-out;
  }

  @keyframes node-doing-pulse {
    0%, 100% { transform: scale(1); }
    50% { transform: scale(1.08); }
  }

  .stepper-node.failed {
    background: var(--status-danger);
    color: #fff;
  }

  .stepper-node.todo {
    background: var(--surface-raised);
    border: 2px solid var(--border-subtle);
    color: var(--text-muted);
  }

  .node-num {
    font-size: 11px;
    font-family: monospace;
  }

  .spin-icon {
    display: inline-flex;
    animation: spin 1.5s infinite linear;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .stepper-card {
    background: var(--surface-raised);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 7px;
    transition: all 0.15s ease;
  }

  .stepper-card:hover {
    border-color: var(--border-default);
  }

  .stepper-card.active {
    border-color: color-mix(in srgb, var(--accent) 50%, var(--border-default));
    background: color-mix(in srgb, var(--surface-panel) 85%, var(--accent) 15%);
  }

  .stepper-card.failed {
    border-color: color-mix(in srgb, var(--status-danger) 50%, var(--border-default));
  }

  .stepper-card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .stepper-card-meta {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .step-num-pill {
    font-size: var(--fs-2xs);
    font-family: monospace;
    font-weight: 600;
    color: var(--text-muted);
  }

  .step-state-tag {
    font-size: var(--fs-2xs);
    font-weight: 600;
    padding: 1px 6px;
    border-radius: var(--r-sm);
  }

  .step-state-tag.done {
    background: color-mix(in srgb, var(--status-success, #10b981) 15%, transparent);
    color: var(--status-success, #10b981);
  }

  .step-state-tag.doing {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    color: var(--accent);
  }

  .step-state-tag.failed {
    background: color-mix(in srgb, var(--status-danger) 15%, transparent);
    color: var(--status-danger);
  }

  .step-state-tag.todo {
    background: var(--surface-panel);
    color: var(--text-muted);
    border: 1px solid var(--border-subtle);
  }

  .stepper-bp-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 7px;
    border-radius: var(--r-sm);
    font-size: var(--fs-2xs);
    color: var(--text-muted);
    background: transparent;
    border: 1px solid var(--border-subtle);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .stepper-bp-btn:hover {
    color: var(--status-warn);
    border-color: var(--status-warn);
    background: color-mix(in srgb, var(--status-warn) 10%, transparent);
  }

  .stepper-bp-btn.active {
    color: var(--status-warn);
    border-color: var(--status-warn);
    background: color-mix(in srgb, var(--status-warn) 20%, transparent);
  }

  .bp-circle {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }

  .stepper-card-text {
    font-size: var(--fs-sm);
    color: var(--text-primary);
    line-height: 1.45;
  }

  :global(.step-code) {
    font-family: monospace;
    font-size: var(--fs-xs);
    padding: 1px 5px;
    border-radius: 4px;
    background: color-mix(in srgb, var(--accent) 12%, var(--surface-panel));
    color: var(--accent);
    border: 1px solid color-mix(in srgb, var(--accent) 25%, transparent);
  }

  .stepper-note {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--fs-xs);
    color: var(--status-warn);
    padding-top: 4px;
    border-top: 1px dashed color-mix(in srgb, var(--border-subtle) 60%, transparent);
  }

  /* Actions Footer */
  .stepper-actions-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding-top: 10px;
    border-top: 1px solid var(--border-subtle);
    flex-wrap: wrap;
  }

  .plan-start-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: var(--fs-xs);
    font-weight: 600;
    padding: 7px 16px;
    border-radius: var(--r-md);
    background: var(--accent);
    color: var(--text-on-accent);
    border: none;
    cursor: pointer;
    box-shadow: 0 2px 6px color-mix(in srgb, var(--accent) 30%, transparent);
    transition: all 0.15s ease;
  }

  .plan-start-btn:hover {
    background: var(--accent-bright, var(--accent));
    transform: translateY(-1px);
  }

  .plan-stop-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: var(--fs-xs);
    font-weight: 500;
    padding: 6px 14px;
    border-radius: var(--r-md);
    background: transparent;
    color: var(--status-danger);
    border: 1px solid var(--status-danger);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .plan-stop-btn:hover {
    background: color-mix(in srgb, var(--status-danger) 12%, transparent);
  }

  .run-status-notice {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: var(--fs-xs);
    font-weight: 600;
    color: var(--accent);
  }

  .pulse-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    animation: dot-pulse 1.4s infinite ease-in-out;
  }

  .all-done-notice {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--fs-xs);
    font-weight: 600;
    color: var(--status-success, #10b981);
  }

  /* Edit Box */
  .plan-edit-area {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .plan-edit-box {
    width: 100%;
    min-height: 280px;
    padding: 12px;
    box-sizing: border-box;
    border-radius: var(--r-md);
    border: 1px solid var(--border-default);
    background: var(--surface-panel);
    color: var(--text-primary);
    font-family: inherit;
    font-size: var(--fs-sm);
    line-height: 1.5;
    resize: vertical;
  }

  .plan-foot {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .plan-edit-hint {
    margin-right: auto;
    font-size: var(--fs-xs);
    color: var(--text-muted);
  }

  .plan-run {
    font-size: var(--fs-xs);
    padding: 6px 14px;
    border-radius: var(--r-sm);
    cursor: pointer;
    background: var(--accent);
    color: var(--text-on-accent);
    border: none;
    font-weight: 600;
  }

  .plan-run-stop {
    font-size: var(--fs-xs);
    padding: 6px 14px;
    border-radius: var(--r-sm);
    cursor: pointer;
    background: none;
    color: var(--text-muted);
    border: 1px solid var(--border-subtle);
  }

  .plan-refusal {
    margin: 6px 0 0;
    font-size: var(--fs-xs);
    color: var(--status-danger);
  }
</style>
