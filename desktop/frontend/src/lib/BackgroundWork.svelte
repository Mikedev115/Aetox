<script lang="ts">
  // Work that outlives the turn that started it (§105), drawn as a card at the
  // foot of the conversation.
  //
  // A card rather than the status strip this replaced. The strip was accurate
  // and useless: the owner ran the first build, saw "ใช้ 1 เครื่องมือ · 1s"
  // frozen under a finished turn, and read it as dead work — "ถ้ามันทำงาน
  // ควรจะไม่นิ่งแบบนี้". A number that only changes every few seconds cannot
  // answer "is this alive"; a list of filenames scrolling past can, at a
  // glance, which is why the running card's centre of gravity is its last few
  // steps rather than its counters.
  //
  // Two sources, deliberately, joined on the delegation's id:
  //   - state (running/waiting/done, the counters, the question) from the
  //     ENGINE's register, which is the only thing that knows them — a `task`
  //     tool call completes the moment the work starts, so the event stream
  //     shows every delegation as finished from birth.
  //   - the step lines from the live event feed, because the register does not
  //     keep them and nothing else can make the card look alive.
  //
  // Below the transcript rather than inside it: the whole failure this fixes is
  // work you cannot see, and a card that scrolls away with the history is one
  // more way not to see it.
  import type { BackgroundRun, BackgroundTask, ToolStep } from './types'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import { fold } from './fold'
  import AgentFace from './AgentFace.svelte'
  import { currentStep, tally } from './delegateWork'
  import { compact, hasSpend, spendLabel, spendTitle } from './spend'

  let {
    tasks, runs = [], allTasks = [], steps, onAnswer, onStop, onStopRun, onStopQueue,
  }: {
    tasks: BackgroundTask[]
    /** Declared jobs (internal/subagent/run.go). Each draws one card with its
     *  workers grouped under it, and its phases in the order they were
     *  DECLARED — a phase nobody has worked in yet is the row worth drawing. */
    runs?: BackgroundRun[]
    /** Every delegation, collected ones included. A run's rows are read from
     *  here rather than from `tasks`: inside a group a finished worker is a
     *  line of the record, not a receipt waiting to be cleared. */
    allTasks?: BackgroundTask[]
    steps: ToolStep[]
    /** Answering goes out as an ordinary chat message, the same door the user
     *  would type through — there is no second path into the engine. */
    onAnswer: (id: string, answer: string) => void
    /** The brake, on one delegate. Until this existed the only way to reach a
     *  running sub-agent was the composer's Stop, which is only on screen while
     *  a turn is live — and a delegate deliberately outlives the turn that
     *  started it. So the ordinary case, work dispatched and the turn finished,
     *  left sub-agents looping with no button anywhere that could end them.
     *  This card is on screen for exactly as long as that work is unclaimed. */
    onStop: (id: string) => void
    /** The same brake shaped like the job. Stopping a run one worker at a time
     *  is a race against the phases that are still starting new ones. */
    onStopRun: (runId: string) => void
    /** And shaped like the line. Nothing refuses a fan-out any more, so the
     *  queue behind four running delegates can be any length — and a line that
     *  can only be cancelled a row at a time is not one anybody can stop. */
    onStopQueue: () => void
  } = $props()

  // A collected result is in the conversation now; the card has said all it can.
  // Work inside a run is drawn by its own card instead.
  const shown = $derived(tasks.filter((task) => !task.collected && !task.run))
  // The line, kept out of the list of cards and behind one row of its own.
  //
  // Nothing refuses a fan-out (internal/subagent/runner.go), so this can be any
  // length, and a card each would turn the one place that says what is about to
  // be spent into a wall you scroll past. What a queued row would draw is a name
  // and nothing else — no steps, no spend, no clock — so it costs nothing to
  // fold them into a count and a number the eye can actually take in.
  const queuedShown = $derived(shown.filter((task) => task.state === 'queued'))
  const activeShown = $derived(shown.filter((task) => task.state !== 'queued'))
  let queueOpen = $state(false)

  // A run stays on screen while it is working, and while any of its workers is
  // still owed a collect. Once everything has been read back the whole group
  // goes, for the same reason a single row does: the work is in the chat now.
  const shownRuns = $derived(
    runs.filter((run) =>
      run.running || allTasks.some((task) => task.run === run.id && !task.collected),
    ),
  )

  const workersOf = (run: BackgroundRun, phase: string) =>
    allTasks.filter((task) => task.run === run.id && task.phase === phase)

  // Phases start open and can be folded away. Collapsed rather than expanded is
  // the state worth keeping, so a run with six phases does not need six clicks
  // to look the way it already looked.
  let collapsed = $state<Record<string, boolean>>({})
  const keyOf = (run: BackgroundRun, phase: string) => run.id + '\u0000' + phase
  // A phase opens itself while somebody is working in it, and closes when they
  // are not. Every phase used to be open at once, which on a run that has only
  // reached its first round is three empty rules stacked under one worker.
  // An explicit toggle outranks it: `collapsed` holds only what the reader has
  // decided, so an absent key means "follow the work".
  const phaseOpen = (run: BackgroundRun, phase: { title: string; running: number }) => {
    const said = collapsed[keyOf(run, phase.title)]
    return said === undefined ? phase.running > 0 : !said
  }
  function togglePhase(run: BackgroundRun, phase: string) {
    const key = keyOf(run, phase)
    collapsed[key] = !collapsed[key]
  }

  // compact/hasSpend/spendLabel/spendTitle live in spend.ts now — the
  // transcript draws the same card (§105.6) and had no way to say any of this,
  // which is the two of them drifting apart by omission.

  // Live work, which is the only kind a brake applies to. A delegate parked on
  // a question counts: it is spending nothing this second, but it is holding a
  // slot and the user may simply not want the answer any more. So does one that
  // has not started — changing your mind about a fan-out must not mean waiting
  // for the part you do not want to begin before you can stop it.
  const stoppable = (task: BackgroundTask) =>
    task.state === 'running' || task.state === 'waiting' || task.state === 'queued'

  // The clock ticks client-side off startedAt, so a running second costs no Go
  // call. Armed only while something is actually moving.
  let now = $state(Date.now())
  $effect(() => {
    if (!shown.some((task) => task.state === 'running' || task.state === 'waiting')) return
    const timer = setInterval(() => { now = Date.now() }, 1000)
    return () => clearInterval(timer)
  })

  function elapsed(startedAt: string): string {
    const secs = Math.max(0, Math.round((now - new Date(startedAt).getTime()) / 1000))
    const mins = Math.floor(secs / 60)
    return mins > 0 ? `${mins}m ${String(secs % 60).padStart(2, '0')}s` : `${secs}s`
  }

  // The last few steps of one delegation. Three because the point is movement,
  // not the record: forty rows would bury the conversation the card sits under,
  // and the whole log is already in the answer the delegate hands back.
  const STEP_TAIL = 3
  function tailOf(id: string): ToolStep[] {
    return steps.filter((s) => s.task === id && !s.kind).slice(-STEP_TAIL)
  }
  // The newest row, promoted out of the list and onto the card's headline.
  //
  // It was always the row doing the work of this card — the other two are there
  // so the eye can see that one has moved. Giving it the top line and body size
  // is what the transcript's card was rebuilt around, and this is the same card
  // (§105.6), so it is not a thing only one of them does.
  const nowOf = (id: string): string =>
    currentStep(steps.filter((s) => s.task === id))?.label ?? ''
  // ...and what is left underneath is history: dimmer, because it has already
  // happened, and still there, because a single line cannot show movement to
  // somebody who looked away for a second.
  const pastOf = (id: string): ToolStep[] => tailOf(id).slice(0, -1)
  // What a delegate has touched, told apart by which way (delegateWork.tally).
  // Read off the same event feed the rows come from — the register counts tool
  // CALLS and cannot tell a file that was read from one that was rewritten.
  const touched = (id: string) => tally(steps.filter((s) => s.task === id))

  // One draft per waiting delegate, keyed by id: two of them parked at once
  // must not share a box.
  let drafts = $state<Record<string, string>>({})

  // Whether a whole run is open. Open while it runs and folded once it is over,
  // which is the rule the transcript's own blocks follow: work in flight is
  // what the reader came to watch, work that is over is a record. Only what the
  // reader has decided is held here, same as `collapsed`.
  let runShut = $state<Record<string, boolean>>({})
  const runOpen = (run: BackgroundRun) => {
    const said = runShut[run.id]
    return said === undefined ? run.running : !said
  }
  function toggleRun(run: BackgroundRun) {
    runShut[run.id] = runOpen(run)
  }
  // What a folded run says instead of its phases: how far along it is, then the
  // same three numbers the open head carries. A fold has to be readable without
  // opening it or it is just a hiding place.
  const runDone = (run: BackgroundRun) => run.phases.reduce((n, p) => n + p.done, 0)
  const runPlanned = (run: BackgroundRun) =>
    run.phases.reduce((n, p) => n + (p.planned > 0 ? p.planned : p.done + p.running), 0)
  function send(id: string) {
    const answer = (drafts[id] ?? '').trim()
    if (!answer) return
    drafts[id] = ''
    onAnswer(id, answer)
  }
</script>

{#if shown.length > 0 || shownRuns.length > 0}
  <div class="bgw">
    <!-- Declared jobs first: a run is the frame the loose rows below it are
         exceptions to, and reading the exceptions first makes the group look
         like an afterthought. -->
    {#each shownRuns as run (run.id)}
      <div class="bgw-item" transition:fold>
        <div class="bgw-card" class:run={run.running}>
          <div class="bgw-head">
            <!-- The whole run folds, and its head is the control — the summary
                 is already there, and a separate row saying "N phases" would
                 put the same count in two places (the argument the transcript's
                 phase headers settled). -->
            <button
              class="bgw-runfold" type="button"
              aria-expanded={runOpen(run)} onclick={() => toggleRun(run)}
              title={runOpen(run) ? t('bgw.foldRun') : t('bgw.openRun')}
            >
              <Icon name={runOpen(run) ? 'chevronDown' : 'chevronRight'} size={14} />
            </button>
            <span class="bgw-mark" class:run={run.running} class:ok={!run.running}>
              <Icon name={run.running ? 'loaderCircle' : 'check'} size={15} />
            </span>
            <b class="bgw-agent">{run.name}</b>
            <!-- No state pill. §105.5 took one off the delegation card for the
                 reason that applies here word for word: a spinning mark, a name,
                 a pill naming the mark's state again and a clock is four things
                 saying two, and the only one of them that ever moved was the
                 clock. The mark spins or it does not; the clock counts or it
                 does not; a run that is over says so by both. -->
            <span class="bgw-meta">
              <!-- Folded, the progress leads: it is the one fact the phases
                   below were carrying and the only one a fold could lose. -->
              {#if !runOpen(run) && runPlanned(run) > 0}{runDone(run)}/{runPlanned(run)} · {/if}
              {elapsed(run.startedAt)} ·
              {t('bgw.runWorkers', { n: allTasks.filter((task) => task.run === run.id).length })}
              {#if run.tokens > 0} · {t('bgw.runTokens', { n: compact(run.tokens) })}{/if}
            </span>
            <!-- One brake for the whole job. Stopping a run worker by worker is
                 a race against its own phases, which are still starting more. -->
            {#if run.running}
              <button
                class="bgw-stop" type="button"
                title={t('bgw.stopRun')} aria-label={t('bgw.stopRun')}
                onclick={() => onStopRun(run.id)}
              >
                <Icon name="square" size={10} />
                <span>{t('bgw.stop')}</span>
              </button>
            {/if}
          </div>
          {#if runOpen(run)}
          <div class="bgw-runbody" transition:fold>
          {#if run.brief}<div class="bgw-brief">{run.brief}</div>{/if}

          {#each run.phases as phase (phase.title)}
            <!-- The phase header. Its count is the whole argument for runs:
                 `planned` was written down before any of this ran, so a round
                 that was promised and skipped sits here at 0 of it. -->
            <button
              class="bgw-phase"
              onclick={() => togglePhase(run, phase.title)}
              aria-expanded={phaseOpen(run, phase)}
            >
              <Icon name={phaseOpen(run, phase) ? 'chevronDown' : 'chevronRight'} size={14} />
              <span class="bgw-phase-title" class:idle={phase.done + phase.running === 0}>{phase.title}</span>
              <!-- One pip per worker the phase was PROMISED, filled as they
                   land. The bar it replaces stretched the full width of the
                   row, so at 0/1 it was an empty rule running across the panel
                   — read as a divider rather than as progress, and unable to
                   say the one thing a plan can say that a percentage cannot:
                   how many. Two of three lit is a fact you take in without
                   reading a number. -->
              <span class="bgw-pips" aria-hidden="true">
                {#each { length: Math.min(12, phase.planned > 0 ? phase.planned : phase.done + phase.running) } as _, i}
                  <span class="bgw-pip" class:on={i < phase.done} class:live={i >= phase.done && i < phase.done + phase.running}></span>
                {/each}
              </span>
              <span class="bgw-phase-count">
                {#if phase.planned > 0}{phase.done}/{phase.planned}{:else if phase.done + phase.running > 0}{phase.done}{:else}{t('bgw.phaseWaiting')}{/if}
              </span>
            </button>

            {#if phaseOpen(run, phase)}
              {#each workersOf(run, phase.title) as task (task.id)}
                <div class="bgw-worker">
                  <!-- The worker's own face, not a glyph. Every other surface in
                       the app that names a delegate draws the person (the
                       transcript's card, the roster, the mention menu); this one
                       was left behind on a tick-or-spinner, so the same worker
                       was a portrait in one panel and an anonymous mark in the
                       next (owner, 7 ก.ย.: "อันนี้อีก UI พัง ล้าหลังไปแล้วมั้ง").
                       One mark, two facts: WHO, from the wardrobe, and WHAT IS
                       HAPPENING, from the ring and the movement inside it —
                       which is the argument §105.5 made when it put the face on
                       the card and dropped the glyph beside it. -->
                  <span class="bgw-worker-face">
                    <AgentFace
                      name={task.agent ?? ''}
                      size={20}
                      state={task.state === 'running' ? 'work' : task.state === 'failed' ? 'err' : task.state === 'waiting' || task.state === 'queued' ? '' : 'done'}
                    />
                  </span>
                  <span class="bgw-worker-label">{task.label}</span>
                  <span class="bgw-worker-agent">{task.agent}</span>
                  <span class="bgw-meta" title={hasSpend(task) ? spendTitle(task) : undefined}>
                    {#if task.state === 'waiting'}
                      {t('bgw.waiting')}
                    {:else if task.state === 'queued'}
                      {t('bgw.queued')}
                    {:else}
                      {#if hasSpend(task)}{spendLabel(task)} · {/if}
                      {task.state === 'running' ? elapsed(task.startedAt) : Math.round((task.elapsedMs ?? 0) / 1000) + 's'}
                    {/if}
                  </span>
                  {#if stoppable(task)}
                    <button
                      class="bgw-stop bgw-stop-worker" type="button"
                      title={t('bgw.stopTask', { agent: task.agent })}
                      aria-label={t('bgw.stopTask', { agent: task.agent })}
                      onclick={() => onStop(task.id)}
                    >
                      <Icon name="square" size={10} />
                    </button>
                  {/if}
                </div>
                {#if task.state === 'running'}
                  {#each tailOf(task.id).slice(-1) as step (step.ref ?? step.label)}
                    <div class="bgw-worker-step">{step.label}</div>
                  {/each}
                {/if}
                <!-- A worker parked on a question, answered where it stands.
                     The alternative is a second card for the same worker, and
                     the whole point of the group is that it is one job. -->
                {#if task.state === 'waiting'}
                  <div class="bgw-worker-ask">
                    {#if task.question}<div class="bgw-question">{task.question}</div>{/if}
                    <div class="bgw-answer">
                      <input
                        class="ctrl" type="text"
                        placeholder={t('bgw.answerPlaceholder', { agent: task.agent })}
                        bind:value={drafts[task.id]}
                        onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); send(task.id) } }}
                      />
                      <button class="ctrl" onclick={() => send(task.id)}>{t('bgw.send')}</button>
                    </div>
                  </div>
                {/if}
              {/each}
            {/if}
          {/each}
          </div>
          {/if}
        </div>
      </div>
    {/each}

    {#each activeShown as task (task.id)}
      <!-- One wrapper per delegation, and the fold lives on it rather than on
           the three cards inside. A row entering or leaving the tray is a
           delegation starting or being collected, which is worth easing; the
           running→done swap within it is the same row changing its mind, and
           folding that would read as one card leaving and another arriving. -->
      <div class="bgw-item" transition:fold>
      {#if task.state === 'running'}
        <!-- The state class the turn timeline's card carries too: the running
             beam is styled on `.run`, and these two are deliberately one card
             (§105.5), so it cannot be a thing only one of them does. -->
        <div class="bgw-card run">
          <!-- A face and the sentence beside it — the same top row the
               transcript's card draws, because they are one card at two
               moments (§105.6). The spinner that used to sit here said
               "working" a third time, after the beam and the pill had already
               said it; what the top line owes the reader is WHAT is being
               worked on. -->
          <div class="bgw-top">
            <span class="bgw-face"><AgentFace name={task.agent} size={34} state={steps.some((s) => s.task === task.id && s.state === 'run') ? 'work' : 'think'} /></span>
            <div class="bgw-said">
              {#key nowOf(task.id)}
                <div class="bgw-now" title={nowOf(task.id)}>
                  <span class="bgw-now-in">{nowOf(task.id) || t('bgw.running')}</span>
                </div>
              {/key}
              <div class="bgw-who">
                <b class="bgw-agent">{task.agent}</b>
                <span class="bgw-dot">·</span><span>{elapsed(task.startedAt)}</span>
                <span class="bgw-dot">·</span><span>{t('bgw.tools', { n: task.toolCalls })}</span>
                {#if hasSpend(task)}
                  <span class="bgw-meta" title={spendTitle(task)}>{spendLabel(task)}</span>
                {/if}
              </div>
            </div>
            <!-- The brake, on the row it belongs to. The composer's Stop is
                 gone by now on the ordinary path: it only draws while a turn is
                 live, and this delegate is here precisely because it outlived
                 the one that started it.
                 It keeps its word here and is icon-only in the transcript: this
                 card stands on its own with room for a label, that one sits
                 inside a conversation where every delegation of a long session
                 would be shouting it. -->
            <button
              class="bgw-stop" type="button"
              title={t('bgw.stopTask', { agent: task.agent })}
              aria-label={t('bgw.stopTask', { agent: task.agent })}
              onclick={() => onStop(task.id)}
            >
              <Icon name="square" size={10} />
              <span>{t('bgw.stop')}</span>
            </button>
          </div>
          <div class="bgw-told"><div class="bgw-brief">{task.label}</div></div>
          {#if touched(task.id).read > 0 || touched(task.id).wrote > 0}
            <div class="bgw-foot">
              {#if touched(task.id).read > 0}<span class="bgw-tal">{t('bgw.tallyRead', { n: touched(task.id).read })}</span>{/if}
              {#if touched(task.id).wrote > 0}<span class="bgw-tal">{t('bgw.tallyWrote', { n: touched(task.id).wrote })}</span>{/if}
            </div>
          {/if}
          <!-- What has already happened, under the line that is happening.
               Dimmer than the headline and still present: one line cannot show
               movement to somebody who looked away for a second, which is the
               whole reason §105.5 put a list here instead of a counter. -->
          <div class="bgw-steps bgw-past">
            {#each pastOf(task.id) as step (step.ref ?? step.label)}
              <!-- The same row the turn timeline draws (.tool-step), so a
                   delegate's step looks identical whether its turn is still
                   open or long finished. -->
              <div class="tool-step {step.state}">
                {#if step.state === 'run'}
                  <span class="glyph spin"></span>
                {:else}
                  <span class="glyph"><Icon name={step.state === 'done' ? 'check' : 'x'} size={12} /></span>
                {/if}
                <span class="lbl">{step.label}</span>
              </div>
            {/each}
          </div>
        </div>
      {:else if task.state === 'waiting'}
        <div class="bgw-card is-waiting">
          <div class="bgw-top">
            <span class="bgw-face"><AgentFace name={task.agent} size={34} /></span>
            <div class="bgw-said">
              <div class="bgw-now" title={task.label}>{task.label}</div>
              <div class="bgw-who">
                <span class="bgw-mark wait"><Icon name="hand" size={12} /></span>
                <b class="bgw-agent">{task.agent}</b>
                <span class="bgw-dot">·</span><span>{t('bgw.waiting')}</span>
                <span class="bgw-dot">·</span><span>{elapsed(task.startedAt)}</span>
                <span class="bgw-dot">·</span><span>{t('bgw.tools', { n: task.toolCalls })}</span>
                {#if hasSpend(task)}
                  <span class="bgw-meta" title={spendTitle(task)}>{spendLabel(task)}</span>
                {/if}
              </div>
            </div>
            <!-- Stoppable while parked too: it is spending nothing this second,
                 but it holds one of the four slots and the user may simply not
                 want the answer any more. -->
            <button
              class="bgw-stop" type="button"
              title={t('bgw.stopTask', { agent: task.agent })}
              aria-label={t('bgw.stopTask', { agent: task.agent })}
              onclick={() => onStop(task.id)}
            >
              <Icon name="square" size={10} />
              <span>{t('bgw.stop')}</span>
            </button>
          </div>
          {#if task.question}<div class="bgw-question">{task.question}</div>{/if}
          <div class="bgw-answer">
            <input
              class="ctrl" type="text"
              placeholder={t('bgw.answerPlaceholder', { agent: task.agent })}
              bind:value={drafts[task.id]}
              onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); send(task.id) } }}
            />
            <button class="ctrl" onclick={() => send(task.id)}>{t('bgw.send')}</button>
          </div>
        </div>
      {:else}
        <!-- Finished, and nobody has to do anything about it: the moment the
             poll sees this state it sends the turn that collects the result
             (cockpit's autoCollectFinished), so this row is a receipt, not a
             control. It disappears when that turn redeems it. -->
        <!-- 'stopped' shares this branch and is none of the other three. It is
             not success, it is not a failure, and unlike both it is NOT read
             back by anyone: auto-collect skips it on purpose (cockpit), because
             sending the model to report on work somebody just paid a click to
             end spends another turn and invites it to start the job again.
             What is left is a receipt, and the counts on it are the point of
             one: how far it had got, and what it had already cost. -->
        <div class="bgw-card is-done" class:is-stopped={task.state === 'stopped'}>
          <div class="bgw-top">
            <span class="bgw-face"><AgentFace name={task.agent} size={34} state={task.state === 'failed' ? 'err' : task.state === 'stopped' ? '' : 'done'} /></span>
            <div class="bgw-said">
              <!-- A receipt leads with how it ended, where a running card leads
                   with what it is doing. Same line, same size — the card does
                   not change shape when the work stops, which is the whole of
                   "one card, two lives" (§105.6). -->
              <div class="bgw-now"><span class="bgw-done-note">
                {task.state === 'failed' ? t('bgw.failed') : task.state === 'stopped' ? t('bgw.stopped') : t('bgw.finished')}
              </span></div>
              <div class="bgw-who">
                <span class="bgw-mark {task.state === 'failed' ? 'fail' : task.state === 'stopped' ? 'off' : 'ok'}">
                  <Icon name={task.state === 'failed' ? 'x' : task.state === 'stopped' ? 'square' : 'check'} size={12} />
                </span>
                <b class="bgw-agent">{task.agent}</b>
                <span class="bgw-dot">·</span><span>{t('bgw.tools', { n: task.toolCalls })}</span>
                {#if hasSpend(task)}
                  <span class="bgw-meta" title={spendTitle(task)}>{spendLabel(task)}</span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {/if}
      </div>
    {/each}

    <!-- The line, as one row. It is the tray's answer to "what is this about to
         cost me": a count you can read at a glance, and a single brake that ends
         all of it. The four already working are not in it and are not touched by
         that brake — work already begun has been paid for, and throwing it away
         is StopAll's decision with its own button. -->
    {#if queuedShown.length > 0}
      <div class="bgw-queue" transition:fold>
        <button
          class="bgw-queue-head" type="button"
          onclick={() => (queueOpen = !queueOpen)}
          aria-expanded={queueOpen}
        >
          <Icon name={queueOpen ? 'chevronDown' : 'chevronRight'} size={14} />
          <span class="bgw-mark queue"><Icon name="clock" size={13} /></span>
          <span class="bgw-queue-count">{t('bgw.queueCount', { n: queuedShown.length })}</span>
        </button>
        <button
          class="bgw-stop" type="button"
          title={t('bgw.stopQueue')}
          aria-label={t('bgw.stopQueue')}
          onclick={onStopQueue}
        >
          <Icon name="square" size={10} />
          <span>{t('bgw.stopQueue')}</span>
        </button>
      </div>
      {#if queueOpen}
        {#each queuedShown as task (task.id)}
          <!-- Opened, a queued job is a name and a brief and nothing else.
               There is no clock on it and no spend under it, because neither
               has started. -->
          <div class="bgw-item" transition:fold>
            <div class="bgw-card is-queued">
              <div class="bgw-top">
                <span class="bgw-face"><AgentFace name={task.agent} size={34} /></span>
                <div class="bgw-said">
                  <div class="bgw-now" title={task.label}>{task.label}</div>
                  <div class="bgw-who">
                    <span class="bgw-mark queue"><Icon name="clock" size={12} /></span>
                    <b class="bgw-agent">{task.agent}</b>
                    <!-- No number: how many run at once is the engine's
                         (maxConcurrent), and a second copy of it here would be
                         right until the day it is changed in one place. -->
                    <span class="bgw-dot">·</span><span>{t('bgw.queuedNote')}</span>
                  </div>
                </div>
                <button
                  class="bgw-stop" type="button"
                  title={t('bgw.stopTask', { agent: task.agent })}
                  aria-label={t('bgw.stopTask', { agent: task.agent })}
                  onclick={() => onStop(task.id)}
                >
                  <Icon name="square" size={10} />
                  <span>{t('bgw.stop')}</span>
                </button>
              </div>
            </div>
          </div>
        {/each}
      {/if}
    {/if}
  </div>
{/if}
