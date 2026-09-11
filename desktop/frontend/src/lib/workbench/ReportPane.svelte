<script lang="ts">
  // The closing report of one round of the plan (desktop/plan_report.go),
  // drawn the way the plan card is drawn — the same hero, the same callouts —
  // on purpose: it is the "after" of the same document, and a reader who has
  // learned the one has learned the other.
  //
  // Everything in the hero is the engine's: which revision of the plan, how
  // many steps settled and how, how long the round took, how often the turn
  // was sent back, why it stopped if it stopped short. Only the callouts are
  // the model's words. That split is what lets the card say "6/6 · 14 min"
  // over a report that claims less, or more.
  import { renderMarkdown } from '../markdown'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import type { IconName } from '../icons'
  import type { PlanReport } from '../types'

  let { report }: { report: PlanReport } = $props()

  // The report's headings are mode.ReportHeadings() (internal/mode/stance.go),
  // English on the wire like the plan's, and labelled here like the plan's
  // (PlanPane.planHeadingLabel). A heading this build does not know is drawn
  // as written.
  const headingLabel = (heading: string) => {
    const map: Record<string, string> = {
      'What was done': 'artifactsPane.reportHead.whatWasDone',
      'How it was checked': 'artifactsPane.reportHead.howItWasChecked',
      'What is left': 'artifactsPane.reportHead.whatIsLeft',
    }
    const key = map[heading]
    return key ? t(key as any) : heading
  }

  type SecKind = 'scope' | 'success' | 'risk'
  const sectionKind = (heading: string): SecKind => {
    if (heading === 'How it was checked') return 'success'
    if (heading === 'What is left') return 'risk'
    return 'scope'
  }
  const sectionIcon = (kind: SecKind): IconName =>
    kind === 'success' ? 'check' : kind === 'risk' ? 'alertTriangle' : 'pencil'

  const settled = $derived(report.done + report.failed)
  // Finished means every step DONE — a round with a failed step settled its
  // checklist and did not finish the plan, and the badge says which.
  const finished = $derived(!report.stopped && report.total > 0 && report.failed === 0 && report.done >= report.total)
  const minutes = $derived(Math.floor((report.elapsedSecs ?? 0) / 60))
  const when = $derived.by(() => {
    const d = new Date(report.at)
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleString(undefined, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
  })
</script>

<div class="report-pane">
  <div class="report-hero">
    <div class="hero-tags">
      <span class="tag-badge">
        <Icon name="fileText" size={12} />
        <span>{t('artifactsPane.groupReports')}</span>
        <span class="tag-round">{t('artifactsPane.round', { n: String(report.run) })}</span>
      </span>
      {#if finished}
        <span class="status-pill done"><Icon name="check" size={11} /><span>{t('artifactsPane.reportFinished')}</span></span>
      {:else if report.stopped}
        <span class="status-pill held"><span>{report.stopped}</span></span>
      {:else}
        <span class="status-pill partial"><span>{t('artifactsPane.reportPartial', { done: String(settled), total: String(report.total) })}</span></span>
      {/if}
    </div>
    {#if report.title}
      <h2 class="hero-title">{report.title}</h2>
    {/if}
    <div class="report-facts">
      <span>{t('artifactsPane.reportPlanVersion', { n: String(report.planVersion) })}</span>
      <span class="dot">·</span>
      <span>{t('artifactsPane.stepsShort', { done: String(settled), total: String(report.total) })}</span>
      {#if report.failed > 0}
        <span class="dot">·</span>
        <span class="failed">{t('artifactsPane.reportFailed', { n: String(report.failed) })}</span>
      {/if}
      {#if minutes > 0}
        <span class="dot">·</span>
        <span>{t('chat.planMinutes', { n: String(minutes) })}</span>
      {/if}
      {#if report.sentBack}
        <span class="dot">·</span>
        <span class="sent-back">{t('chat.planSentBack', { n: String(report.sentBack) })}</span>
      {/if}
      {#if when}
        <span class="dot">·</span>
        <span>{when}</span>
      {/if}
    </div>
  </div>

  <div class="report-sections">
    {#each report.sections ?? [] as sec (sec.heading)}
      {@const kind = sectionKind(sec.heading)}
      <div class="sec-callout kind-{kind}">
        <div class="sec-head">
          <Icon name={sectionIcon(kind)} size={13} />
          <span>{headingLabel(sec.heading)}</span>
        </div>
        <div class="markdown-body sec-body">{@html renderMarkdown(sec.body)}</div>
      </div>
    {/each}
  </div>
</div>

<style>
  .report-pane {
    display: flex;
    flex-direction: column;
    gap: 12px;
    max-width: 860px;
    margin: 0 auto;
    padding: 16px;
    box-sizing: border-box;
    width: 100%;
  }

  .report-hero {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    padding: 16px 18px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .hero-tags {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .tag-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 3px 9px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 500;
    background: color-mix(in srgb, var(--status-warn, #f59e0b) 12%, transparent);
    color: var(--status-warn, #f59e0b);
    border: 1px solid color-mix(in srgb, var(--status-warn, #f59e0b) 25%, transparent);
  }
  .tag-round {
    font-size: var(--fs-2xs);
    opacity: 0.85;
  }
  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 3px 9px;
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .status-pill.done {
    background: color-mix(in srgb, var(--status-success, #10b981) 14%, transparent);
    color: var(--status-success, #10b981);
    border: 1px solid color-mix(in srgb, var(--status-success, #10b981) 28%, transparent);
  }
  /* A round that stopped short is a finding, not a failure: the badge is the
     hold's own words in the warn colour, readable — never the faintest ink. */
  .status-pill.held,
  .status-pill.partial {
    background: color-mix(in srgb, var(--status-warn, #f59e0b) 12%, transparent);
    color: var(--status-warn, #f59e0b);
    border: 1px solid color-mix(in srgb, var(--status-warn, #f59e0b) 28%, transparent);
  }
  .hero-title {
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: 700;
    line-height: 1.35;
    color: var(--text-primary);
    letter-spacing: -0.01em;
  }
  .report-facts {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    font-size: var(--fs-xs);
    color: var(--text-muted);
  }
  .report-facts .dot {
    color: var(--text-dim);
  }
  .report-facts .failed,
  .report-facts .sent-back {
    color: var(--status-warn, #f59e0b);
  }

  .report-sections {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .sec-callout {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    border-left-width: 4px;
  }
  .sec-callout.kind-scope {
    border-left-color: var(--accent);
  }
  .sec-callout.kind-success {
    border-left-color: var(--status-success, #10b981);
  }
  .sec-callout.kind-risk {
    border-left-color: var(--status-warn, #f59e0b);
  }
  .sec-head {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: var(--fs-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }
  .sec-callout.kind-scope .sec-head {
    color: var(--accent);
  }
  .sec-callout.kind-success .sec-head {
    color: var(--status-success, #10b981);
  }
  .sec-callout.kind-risk .sec-head {
    color: var(--status-warn, #f59e0b);
  }
  .sec-body {
    font-size: var(--fs-xs);
    color: var(--text-secondary);
    line-height: 1.5;
  }
</style>
