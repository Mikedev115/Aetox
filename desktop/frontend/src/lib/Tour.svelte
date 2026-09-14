<script lang="ts">
  // รู้จักกับ Aetox — the tour a first run plays after the language is picked
  // and before the wizard asks about a model, and the one ตั้งค่า › เกี่ยวกับ
  // replays. Nine scenes, one idea each, every one drawn on the real rig and
  // the real stylesheets (DECISIONS §279; the words are docs/FIRST-RUN-TOUR.md).
  //
  // Each scene plays its beats on a timer; ถัดไป / ← → / the dots move
  // between scenes, and ข้าม lands on the last one. Nothing is a video: it is
  // the mascot and the app's own CSS, so it follows the theme, the language and
  // the name the person types in scene 2 — which is saved for real, because
  // this is the one moment a new user names the assistant without hunting a
  // menu for it.
  import { onMount } from 'svelte'
  import { fly, fade, scale } from 'svelte/transition'
  import { cubicOut, backOut } from 'svelte/easing'
  import Icon from './Icon.svelte'
  import Mascot from './mascot/Mascot.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import RankedFace from './RankedFace.svelte'
  import RankPip from './RankPip.svelte'
  import { headOptions } from './mascot/avatarPrefs.svelte'
  import { t, setLocale, localeNames, i18n, type Locale } from './i18n.svelte'
  import { profile, loadProfileName, saveProfileName } from './stores/profile.svelte'
  import { SetHeadName, HeadName } from '../../wailsjs/go/main/App'

  // flow: 'setup' is the wizard's — the last scene hands over to the three
  // setup steps still ahead rather than saying "ready" a screen too early
  // (owner, 14 ก.ย.: "ก่อนจะไปหน้าตั้งค่า ทำหน้าเพิ่ม … ตั้งค่าให้ระบบเราก่อน").
  // 'replay' is the footer menu's and เกี่ยวกับ's: the app is already set up,
  // so the last scene closes.
  let { onDone, flow = 'replay' }: { onDone: () => void; flow?: 'setup' | 'replay' } = $props()

  const A = headOptions('assistant')
  const C = headOptions('coding')

  const SCENE_COUNT = 9
  // How many beats each scene plays, and the gap between them. Slow on
  // purpose (owner, 14 ก.ย.: "เอาช้านิดนึง").
  const BEATS = [4, 4, 3, 11, 7, 5, 8, 5, 2]
  const GAP = 1400

  let scene = $state(0)
  let beat = $state(0)
  let timer: ReturnType<typeof setTimeout> | undefined
  let botName = $state('')
  let youName = $state('')
  const shown = $derived(botName.trim() || 'Aetox')

  function play() {
    clearTimeout(timer)
    if (beat < BEATS[scene]) timer = setTimeout(() => { beat++; play() }, GAP)
  }
  // Which way the next scene slides in from: forward from the right, back
  // from the left — the direction the person pressed.
  let dir = $state(1)
  function go(n: number) {
    if (n < 0 || n >= SCENE_COUNT) return
    dir = n > scene ? 1 : -1
    scene = n; beat = 0; play()
  }
  // What scene 2 typed, kept: the head's name through the same call ตัวหลัก ›
  // ตัวตน makes, the person's through the footer's store. Blank changes
  // nothing — a name the person did not type is not a name.
  function commitNames() {
    const b = botName.trim()
    if (b) { try { void SetHeadName('assistant', b).catch(() => {}) } catch { /* no engine: nothing to save to */ } }
    if (youName.trim()) saveProfileName(youName)
  }
  function finish() {
    commitNames()
    clearTimeout(timer)
    onDone()
  }
  onMount(() => {
    play()
    void loadProfileName().then(() => { if (!youName) youName = profile.name })
    try { void HeadName('assistant').then((n) => { if (!botName && n) botName = n }).catch(() => {}) } catch { /* no engine yet: the field opens blank */ }
    const key = (e: KeyboardEvent) => {
      if ((e.target as HTMLElement | null)?.tagName === 'INPUT') return
      if (e.key === 'ArrowRight' || e.key === ' ') go(scene + 1)
      if (e.key === 'ArrowLeft') go(scene - 1)
    }
    window.addEventListener('keydown', key)
    return () => { window.removeEventListener('keydown', key); clearTimeout(timer) }
  })

  const ROOMS = [
    { icon: 'monitor', key: 'tour.roomSlides' }, { icon: 'globe', key: 'tour.roomBrowser' },
    { icon: 'folder', key: 'tour.roomFiles' }, { icon: 'terminal', key: 'tour.roomTerminal' },
  ] as const
  const CODE_ROOMS = [{ icon: 'gitBranch', key: 'tour.roomGit' }, { icon: 'clock', key: 'tour.roomTimeline' }] as const
  const AGENTS = [
    { name: 'doc', icon: 'fileText', hue: 280 }, { name: 'sheet', icon: 'chartColumn', hue: 95 },
    { name: 'github', icon: 'gitBranch', hue: 330 }, { name: 'automation', icon: 'zap', hue: 40 },
    { name: 'deepresearch', icon: 'search', hue: 150 }, { name: 'editor', icon: 'slidersHorizontal', hue: 195 },
    { name: 'video', icon: 'clapperboard', hue: 235 },
  ]
  const HELPERS = [
    { name: 'explore', icon: 'search' }, { name: 'general', icon: 'wrench' },
    { name: 'reviewer', icon: 'eye', ro: true }, { name: 'tester', icon: 'check', ro: true },
  ]

  // ---- scene 3: three cards, where each goes ------------------------------
  type Card = { key: 'tour.card1' | 'tour.card2' | 'tour.card3'; scopeKey: 'settings.you' | 'tour.scopeBot'; to: 'you' | 'bot' | 'no' }
  const CARDS: Card[] = [
    { key: 'tour.card1', scopeKey: 'settings.you', to: 'you' },
    { key: 'tour.card2', scopeKey: 'tour.scopeBot', to: 'bot' },
    { key: 'tour.card3', scopeKey: 'tour.scopeBot', to: 'no' },
  ]
  // beats: 0 card1 up · 1 fly · 2 card2 up · 3 fly · 4 card3 up · 5 reject · 6..9 agents grow · 10 caption
  const cardAt = $derived(scene !== 3 ? -1 : beat < 2 ? 0 : beat < 4 ? 1 : beat < 6 ? 2 : -1)
  const flying = $derived(scene === 3 && beat % 2 === 1 && beat < 6)
  let cardEl = $state<HTMLElement | null>(null)
  let boxEl: Record<string, HTMLElement | null> = { you: null, bot: null }
  let flyStyle = $state('')
  let landed = $state({ you: false, bot: false })
  // The flight: the button presses, then the card itself travels to its box
  // (one CSS transition on the real element), and the line appears in the box
  // the moment it lands.
  $effect(() => {
    if (scene !== 3) { landed = { you: false, bot: false }; flyStyle = ''; return }
    if (!flying || cardAt < 0) { flyStyle = ''; return }
    const c = CARDS[cardAt]
    const t1 = setTimeout(() => {
      const from = cardEl?.getBoundingClientRect(); const to = c.to === 'no' ? null : boxEl[c.to]?.getBoundingClientRect()
      if (!from || !to) { flyStyle = 'transform:scale(.92) translateY(18px); opacity:0'; return }
      const dx = (to.left + to.width / 2) - (from.left + from.width / 2)
      const dy = (to.top + to.height * .62) - (from.top + from.height / 2)
      flyStyle = `transform:translate(${dx}px, ${dy}px) scale(.18); opacity:.15`
    }, 220)
    const t2 = setTimeout(() => { if (c.to !== 'no') landed = { ...landed, [c.to]: true } }, 1000)
    return () => { clearTimeout(t1); clearTimeout(t2) }
  })
  const inBox = $derived({ you: scene === 3 && landed.you, bot: scene === 3 && landed.bot })
  const GROW = [
    { name: 'doc', icon: 'fileText', hue: 280, key: 'tour.grow1' },
    { name: 'sheet', icon: 'chartColumn', hue: 95, key: 'tour.grow2' },
    { name: 'deepresearch', icon: 'search', hue: 150, key: 'tour.grow3' },
    { name: 'github', icon: 'gitBranch', hue: 330, key: 'tour.grow4' },
  ] as const
  const grown = $derived(scene === 3 ? Math.max(0, beat - 5) : 0)

  // ---- scene 4: cloud or local -------------------------------------------
  const ITEMS = [
    { icon: 'messageSquare', key: 'tour.itemChats' }, { icon: 'brain', key: 'tour.itemMemory' },
    { icon: 'settings', key: 'tour.itemSettings' }, { icon: 'shield', key: 'tour.itemKeys' },
  ] as const
  let local = $state(false)
  $effect(() => { if (scene === 4) local = beat >= 6 })

  // ---- scene 6: a team, then one job handed to it --------------------------
  const TEAM = [
    { name: 'deepresearch', icon: 'search', hue: 150, job: 'tour.job1', out: 'tour.out1' },
    { name: 'sheet', icon: 'chartColumn', hue: 95, job: 'tour.job2', out: 'tour.out2' },
    { name: 'doc', icon: 'fileText', hue: 280, job: 'tour.job3', out: 'tour.out3' },
  ] as const
  // beats: 0 ask · 1 panel · 2 ticked · 3 jobs out · 4 working · 5 results back · 6 done · 7 caption
  const tPhase = $derived(scene !== 6 ? 0 : beat)
  const memberState = $derived(tPhase >= 6 ? 'done' : tPhase >= 4 ? 'work' : tPhase >= 3 ? 'think' : '')

  // ---- scene 7: RAM while working ------------------------------------------
  // Private bytes, every process of the app counted. Aetox: measured on the
  // owner's 32 GB machine — 455 MB chat only (8 ก.ย. 2026, BENCHMARK.md §6),
  // 706 MB chat + browser tab + code desk (14 ก.ย.). The others: 'own' = the
  // same machine, 'net' = a figure reported publicly (the source is in the
  // comment), never one we made up. BENCHMARK.md §6 marks a mixed-state table
  // "ห้ามขึ้นเว็บ": before a release this scene's rows get one same-state
  // round of bench.ps1, and the footnote says which is which until then.
  const AETOX = [
    { k: 'app', key: 'tour.segApp', mb: 66 },
    { k: 'ui', key: 'tour.segUI', mb: 127 },
    { k: 'web', key: 'tour.segWeb', mb: 113 },
    { k: 'wv', key: 'tour.segWV', mb: 400 },
  ] as const
  const AETOX_WORK = 455, AETOX_HEAVY = 706
  const RAM: { name: string; kind: string; work: number; heavy?: number; src: 'own' | 'net' }[] = [
    { name: 'OpenClaw', kind: 'tour.kNodeGateway', work: 600, heavy: 1200, src: 'net' },   // sfailabs.com: gateway 400–800 MB idle + Playwright 200–400 MB
    { name: 'Hermes Agent', kind: 'tour.kPython', work: 282, heavy: 1100, src: 'net' },    // mindstudio.ai: 282 MB median of 31 live containers, 1.1 GB mid-response
    { name: 'ChatGPT', kind: 'tour.kDesktop', work: 1285, heavy: 2048, src: 'own' },       // own 1.3 GB; 2 GB+ in long chats (lencx/ChatGPT#1306)
    { name: 'Windsurf', kind: 'tour.kIDE', work: 1500, heavy: 10000, src: 'net' },         // boostdevspeed: 1–2 GB, 10 GB+ leak
    { name: 'Cursor', kind: 'tour.kIDE', work: 1750, heavy: 22000, src: 'own' },           // own 1.75 GB; forum.cursor.com #158844: 22 GB
    { name: 'Antigravity', kind: 'tour.kIDE', work: 2608, src: 'own' },
    { name: 'Claude Desktop', kind: 'tour.kHarnessDesktop', work: 3000, heavy: 5500, src: 'net' }, // anthropics/claude-code#40633 + 1.8 GB VM; #43310
    { name: 'OpenCode', kind: 'tour.kCLI', work: 3000, heavy: 15000, src: 'net' },         // anomalyco/opencode#11399
    { name: 'VS Code + Claude Code', kind: 'tour.kIDEHarness', work: 4814, heavy: 23278, src: 'own' },
    { name: 'Codex', kind: 'tour.kHarnessDesktop', work: 7000, heavy: 12500, src: 'net' }, // openai/codex#36971: app-server 7.0–7.3 GB + task worker 5.5 GB
  ]
  // log scale, 100 MB → 30 GB: on a linear one every bar but the last two is a dot.
  const LOG_MIN = Math.log10(100), LOG_MAX = Math.log10(30000)
  const pct = (mb: number) => Math.max(1, (Math.log10(mb) - LOG_MIN) / (LOG_MAX - LOG_MIN) * 100)
  const gb = (mb: number) => mb >= 1000 ? (mb / 1024).toFixed(1) + ' GB' : mb + ' MB'
</script>

<div class="tour">
  <div class="tour-top">
    <div class="tour-dots" role="tablist">
      {#each Array(SCENE_COUNT) as _, i}
        <button type="button" class="tour-dot" class:on={i === scene} class:done={i < scene} role="tab" aria-selected={i === scene}
          aria-label={t('tour.sceneN', { n: String(i + 1) })} onclick={() => go(i)}></button>
      {/each}
    </div>
    <button type="button" class="ob-link tour-skip" onclick={() => go(SCENE_COUNT - 1)}>{t('tour.skip')}</button>
  </div>
  <!-- The language, switchable mid-tour, in the bottom corner where it is out
       of the scene's way (owner: "ภาษาควรเอาไว้ด้านข้างล่าง"): every word here
       is a key, so a person who picked the wrong one a screen ago does not
       restart. -->
  <div class="tour-lang" role="group">
    {#each Object.entries(localeNames) as [code, name]}
      <button type="button" class:on={i18n.locale === code} onclick={() => setLocale(code as Locale)}>{name}</button>
    {/each}
  </div>

  <div class="ob-screen tour-screen">
    {#key scene}
    <div class="tour-scene" in:fly={{ x: dir * 36, duration: 460, delay: 160, easing: cubicOut }} out:fade={{ duration: 200 }}>

    {#if scene === 0}
      <div class="stage t-open">
        <div class="win" in:scale={{ start: .92, duration: 600, easing: cubicOut }}>
          <div class="win-bar"><i></i><i></i><i></i><span>Aetox</span></div>
          <div class="win-grid">
            <!-- Each room moves the way it does in the app, with real content
                 rather than grey bars (owner, 14 ก.ย.: "ยังไม่ชัดพอที่จะเห็นภาพ
                 เบราว์เซอร์ก็ควรจะเป็นเมาส์จริง โค้ดก็เอาโค้ดในโปรเจกต์เราไปแสดง"):
                 a deck with titles, a page with a real arrow cursor clicking a
                 numbered link, prompt.go's own person() being edited, a shell
                 running. All CSS loops, so it follows the theme and the
                 language and adds nothing to the installer. -->
            <div class="pane" class:on={beat >= 1}>
              <div class="deck">
                <div class="deck-strip">
                  <div class="sl"><h5>{t('tour.demoSlide1')}</h5><p>{t('tour.demoSlide1a')}</p><p>{t('tour.demoSlide1b')}</p><div class="chart"><i style="height:40%"></i><i style="height:75%"></i><i style="height:55%"></i><i style="height:90%"></i></div></div>
                  <div class="sl"><h5>{t('tour.demoSlide2')}</h5><div class="img"></div><p>{t('tour.demoSlide2a')}</p></div>
                  <div class="sl"><h5>{t('tour.demoSlide3')}</h5><p>· {t('tour.demoSlide3a')}</p><p>· {t('tour.demoSlide3b')}</p><p>· {t('tour.demoSlide3c')}</p></div>
                </div>
                <div class="deck-dots"><i></i><i></i><i></i></div>
              </div>
              <small><Icon name="monitor" size={11} /> {t('tour.roomSlides')}</small>
            </div>
            <div class="pane" class:on={beat >= 2}>
              <div class="web">
                <div class="chrome"><i></i><i></i><i></i><span class="addr">aetox.dev/docs</span></div>
                <div class="site">
                  <div class="site-nav"><span class="logo">A</span><span class="nl b1">{t('tour.demoNav1')}<em>1</em></span><span class="nl b2">{t('tour.demoNav2')}<em>2</em></span><span class="nl b3">{t('tour.demoNav3')}<em>3</em></span></div>
                  <div class="site-body page-a"><h5>{t('tour.demoPageA')}</h5><p>{t('tour.demoPageAText')}</p><p class="dim">{t('tour.demoPageAText2')}</p></div>
                  <div class="site-body page-b"><h5>{t('tour.demoNav2')}</h5><span class="dl">Windows x64 · 34 MB</span><p class="dim">{t('tour.demoPageBText')}</p></div>
                </div>
                <svg class="mouse" viewBox="0 0 16 22" width="14" height="19" aria-hidden="true"><path d="M1 1l0 16 4-4 3 7 3-1-3-7 5 0z" fill="#fff" stroke="#000" stroke-width="1.2" stroke-linejoin="round"/></svg>
              </div>
              <small><Icon name="globe" size={11} /> {t('tour.roomBrowser')}</small>
            </div>
            <div class="pane" class:on={beat >= 3}>
              <div class="files">
                <div class="tree">
                  <div><Icon name="folderOpen" size={10} /> internal</div>
                  <div class="in"><Icon name="folderOpen" size={10} /> prompt</div>
                  <div class="in2 sel"><Icon name="fileCode" size={10} /> prompt.go</div>
                  <div class="in2"><Icon name="fileCode" size={10} /> prompt_test.go</div>
                  <div class="in"><Icon name="folder" size={10} /> learned</div>
                </div>
                <div class="editor">
                  <div><span class="kw">func</span> <span class="fn">person</span>(name <span class="ty">string</span>) <span class="ty">string</span> {'{'}</div>
                  <div>  name = strings.<span class="fn">Join</span>(strings.<span class="fn">Fields</span>(name), <span class="st">" "</span>)</div>
                  <div>  <span class="kw">if</span> name == <span class="st">""</span> {'{'} <span class="kw">return</span> <span class="st">""</span> {'}'}</div>
                  <div class="del">  <span class="kw">return</span> <span class="st">"The person is "</span> + name</div>
                  <div class="add"><span class="typed">  <span class="kw">return</span> <span class="st">"The person you are working with calls themselves "</span> + name</span></div>
                  <div>{'}'}</div>
                </div>
              </div>
              <small><Icon name="folder" size={11} /> {t('tour.roomFiles')}</small>
            </div>
            <div class="pane" class:on={beat >= 4}>
              <div class="term">
                <span class="l1">$ go test ./internal/prompt/</span>
                <span class="l2 ok">ok  internal/prompt  16.5s</span>
                <span class="l3">$ git commit -m "prompt: the name is on file"</span>
                <span class="l4 ok">[main 9f2a1c] 1 file changed, 4 insertions(+)</span>
                <span class="cur">$ <i></i></span>
              </div>
              <small><Icon name="terminal" size={11} /> {t('tour.roomTerminal')}</small>
            </div>
          </div>
        </div>
        <div class="open-bot" in:fly={{ x: -40, duration: 600, easing: backOut }}>
          <Mascot {...A} pose={beat >= 1 ? 'presenting' : 'greeting'} size={130} />
        </div>
      </div>
      <h2>{t('tour.s0Title')}</h2>
      <p class="ob-sub">{t('tour.s0Sub')}</p>

    {:else if scene === 1}
      <div class="stage split" class:apart={beat >= 1}>
        <div class="twho">
          <Mascot {...A} pose="idle" size={120} />
          {#if beat >= 2}<div class="who-name" in:fade>{t('desk.assistant')}</div>
            <div class="who-rooms" in:fly={{ y: 8 }}>{#each ROOMS as r}<span class="room sm"><Icon name={r.icon} size={13} /></span>{/each}</div>{/if}
        </div>
        <div class="twho">
          <Mascot {...C} pose={beat >= 3 ? 'coding' : 'idle'} size={120} />
          {#if beat >= 2}<div class="who-name" in:fade>{t('desk.coding')}</div>
            <div class="who-rooms" in:fly={{ y: 8 }}>{#each ROOMS as r}<span class="room sm"><Icon name={r.icon} size={13} /></span>{/each}{#each CODE_ROOMS as r}<span class="room sm t-plus"><Icon name={r.icon} size={13} /></span>{/each}</div>{/if}
        </div>
      </div>
      <h2>{t('tour.s1Title')}</h2>
      <p class="ob-sub"><b>{t('desk.assistant')}</b> {t('tour.s1A')}<br /><b>{t('desk.coding')}</b> {t('tour.s1C')}<br />{#if beat >= 4}<span in:fade>{t('tour.s1Own')}</span>{/if}</p>

    {:else if scene === 2}
      <div class="stage">
        <div class="hero"><Mascot {...A} pose={youName.trim() ? 'greeting' : 'asking'} size={130} look /></div>
        {#if youName.trim() || botName.trim()}
          <div class="tbubble" in:scale={{ start: .8, duration: 300, easing: backOut }}>
            {youName.trim() ? t('tour.s2Hi', { you: youName.trim(), bot: shown }) : t('tour.s2Me', { bot: shown })}
          </div>
        {/if}
      </div>
      <h2>{t('tour.s2Title')}</h2>
      <div class="names">
        <label><span>{t('tour.s2Bot')}</span><input placeholder="Aetox" bind:value={botName} maxlength="40" /></label>
        <label><span>{t('tour.s2You')}</span><input placeholder={t('tour.s2YouPh')} bind:value={youName} maxlength="40" /></label>
      </div>
      <p class="ob-sub small">{t('tour.s2Note')}</p>

    {:else if scene === 3}
      <div class="stage mem">
        <div class="boxes">
          <div class="tbox you" class:lit={inBox.you} bind:this={boxEl.you}><Icon name="circleUser" size={16} /><b>{t('settings.you')}</b><small>{t('tour.boxYouSub')}</small>{#if inBox.you}<i class="line" in:scale={{ start: .6, duration: 420, easing: backOut }}>{t('tour.card1')}</i>{/if}</div>
          <div class="tbox bot" class:lit={inBox.bot} bind:this={boxEl.bot}><Icon name="brain" size={16} /><b>{t('tour.boxBot', { name: shown })}</b><small>{t('tour.boxBotSub', { name: shown })}</small>{#if inBox.bot}<i class="line" in:scale={{ start: .6, duration: 420, easing: backOut }}>{t('tour.card2')}</i>{/if}</div>
          <div class="tbox ag" class:lit={grown > 0}><Icon name="bot" size={16} /><b>{t('tour.boxAgents')}</b><small>{t('tour.boxAgentsSub')}</small>
            <div class="grow">
              {#each GROW as g, i}
                <div class="grow-one" class:up={grown > i}>
                  <span class="t-ring"></span>
                  <AgentMascot name={g.name} icon={g.icon} hue={g.hue} size={grown > i ? 40 : 32} state={grown > i ? 'done' : ''} />
                  {#if grown > i}<span class="t-plus" in:fly={{ y: 10, duration: 500, easing: backOut }}><Icon name="sparkles" size={10} /> +1</span>{/if}
                </div>
              {/each}
            </div>
            {#if grown > 0}<i class="line" in:fade>{t(GROW[Math.min(grown, GROW.length) - 1].key)}</i>{/if}
          </div>
        </div>
        {#if cardAt >= 0}
          {#key cardAt}
          {@const c = CARDS[cardAt]}
          <div class="memcard tour-card" bind:this={cardEl} style={flyStyle} in:fly={{ y: 40, duration: 480, easing: cubicOut }}>
            <div class="memcard-head"><span class="ic"><Icon name="brain" size={14} /></span><span class="memcard-kind">{t('chat.memoryAsk')}</span><span class="memcard-scope">{t(c.scopeKey)}</span></div>
            <div class="memcard-line">{t(c.key)}</div>
            <div class="memcard-foot"><span class="memcard-note">{t('chat.memoryNextChat')}</span><button type="button" class="memcard-no" class:pressed={flying && c.to === 'no'} tabindex="-1">{t('settings.learningReject')}</button><button type="button" class="memcard-yes" class:pressed={flying && c.to !== 'no'} tabindex="-1">{t('settings.learningApprove')}</button></div>
          </div>
          {/key}
        {/if}
        {#if scene === 3 && beat === 6}<div class="t-hint" in:fade>{t('tour.s3Rejected')}</div>{/if}
        {#if scene === 3 && beat >= 10}<div class="t-hint" in:fade>{t('tour.s3Grow')}</div>{/if}
      </div>
      <h2>{t('tour.s3Title')}</h2>
      <p class="ob-sub">{t('tour.s3Sub')}</p>

    {:else if scene === 4}
      <div class="stage data">
        <div class="pc" in:scale={{ start: .94, duration: 500, easing: cubicOut }}>
          <div class="screen" class:sealed={local}>
            <div class="screen-bar"><Icon name="monitor" size={11} /> {t('tour.s4Machine')} {#if local}<span class="localchip" in:fade><Icon name="zap" size={10} /> Ollama · LM Studio</span>{/if}</div>
            <div class="tiles">
              <div class="t-tile home" class:on={beat >= 1}>
                <div class="t-tile-t"><Icon name="folderOpen" size={13} /> Aetox</div>
                <div class="items">
                  {#each ITEMS as it, i}
                    {#if beat >= 1}<span in:fly={{ y: 8, duration: 350, delay: i * 90 }}><Icon name={it.icon} size={11} /> {t(it.key)}</span>{/if}
                  {/each}
                </div>
              </div>
              <div class="t-tile proj" class:on={beat >= 2}>
                <div class="t-tile-t"><Icon name="folder" size={13} /> {t('tour.s4Proj')}</div>
                <small>{t('tour.s4ProjSub')}</small>
                {#if beat >= 3}
                  <div class="t-ask" in:fly={{ y: 8, duration: 350 }}>
                    <span>{t('tour.s4Ask')} <code>D:\{t('tour.s4OtherFolder')}</code></span>
                    <b>{t('tour.s4Allow')}</b><i>{t('tour.s4Deny')}</i>
                  </div>
                {/if}
              </div>
              <div class="t-tile vault" class:on={beat >= 4}>
                <div class="t-tile-t"><Icon name="shield" size={13} /> {t('tour.s4Vault')}</div>
                <small>.ssh · Chrome · Edge · Windows</small>
                {#if beat >= 4}<em in:fade>{t('tour.s4VaultNo')}</em>{/if}
              </div>
            </div>
          </div>
          <div class="stand"></div>
        </div>
        <div class="link" class:cut={local} class:live={beat >= 5}>
          <span class="dash"></span>
          {#if local}<span class="stop" in:scale={{ start: .5, duration: 350, easing: backOut }}><Icon name="check" size={13} /></span>{/if}
        </div>
        <div class="cloud" class:off={local}>
          <Icon name="globe" size={22} />
          <b>{t('tour.s4Cloud')}</b>
          <small>{t('tour.s4CloudSub')}</small>
        </div>
        <div class="verdict">
          {#if local}<span class="good" in:fade>{t('tour.s4Local')}</span>
          {:else}<span in:fade>{t('tour.s4OutOne')}</span>{/if}
        </div>
        <div class="mode">
          <button type="button" class:on={!local} onclick={() => (local = false)}><Icon name="globe" size={12} /> {t('tour.s4ModeCloud')}</button>
          <button type="button" class:on={local} onclick={() => (local = true)}><Icon name="monitor" size={12} /> {t('tour.s4ModeLocal')}</button>
        </div>
      </div>
      <h2>{t('tour.s4Title')}</h2>
      <p class="ob-sub">{t('tour.s4Sub')}</p>

    {:else if scene === 5}
      <div class="stage org">
        <div class="chart">
          <div class="band b-head">
            <div class="band-k"><RankPip tier="head" size="md" /></div>
            <div class="t-orow">
              <div class="tnode top"><RankedFace tier="head" size={66}><Mascot {...A} pose="idle" size={66} still /></RankedFace><small>{shown}</small></div>
              <div class="tnode top"><RankedFace tier="head" size={66}><Mascot {...C} pose="idle" size={66} still /></RankedFace><small>{t('desk.coding')}</small></div>
            </div>
          </div>
          <div class="band b-agent" class:on={beat >= 1}>
            <div class="band-k"><RankPip tier="agent" size="md" /><em>{t('tour.s5AgentsSub')}</em></div>
            <div class="t-orow bus">
              {#each AGENTS as a, i}
                <div class="tnode leaf" class:on={beat >= 1} style="transition-delay:{i * 70}ms"><AgentMascot name={a.name} icon={a.icon} hue={a.hue} size={38} /><small>{a.name}</small></div>
              {/each}
            </div>
          </div>
          <div class="band b-helper" class:on={beat >= 2}>
            <div class="band-k"><RankPip tier="helper" size="md" /><em>{t('tour.s5HelpersSub')}</em></div>
            <div class="t-orow bus narrow">
              {#each HELPERS as h, i}
                <div class="tnode leaf" class:on={beat >= 2} class:ro={h.ro} style="transition-delay:{i * 70}ms"><AgentMascot name={h.name} icon={h.icon} size={30} /><small>{h.name}{#if h.ro} <Icon name="eye" size={9} />{/if}</small></div>
              {/each}
            </div>
          </div>
          <div class="trunk" class:on={beat >= 1}></div>
        </div>
      </div>
      <h2>{t('tour.s5Title')}</h2>
      <p class="ob-sub">
        {#if beat >= 3}<span in:fade>{t('tour.s5Agents')}</span><br />{/if}
        {#if beat >= 4}<span in:fade>{t('tour.s5Helpers')}</span>{/if}
      </p>

    {:else if scene === 6}
      <div class="stage team">
        <div class="t-head">
          <Mascot {...A} pose={tPhase >= 6 ? 'success' : tPhase >= 3 ? 'planning' : 'thinking'} size={110} />
          <div class="t-name">{shown}</div>
          {#if tPhase === 0}<div class="t-say" in:scale={{ start: .8, duration: 300, easing: backOut }}>{t('tour.s6Ask')}</div>{/if}
          {#if tPhase >= 6}<div class="t-say done" in:scale={{ start: .8, duration: 300, easing: backOut }}><Icon name="check" size={12} /> {t('tour.s6Done')}</div>{/if}
        </div>
        <div class="t-links">
          {#each TEAM as m, i}
            <div class="t-link" class:on={tPhase >= 2 && tPhase < 5} class:back={tPhase >= 5}>
              <span class="dash"></span>
              {#if tPhase >= 3 && tPhase < 5}<span class="job" in:fly={{ x: -20, duration: 350, delay: i * 120 }}>{t(m.job)}</span>{/if}
              {#if tPhase >= 5}<span class="job back" in:fly={{ x: 20, duration: 350, delay: i * 120 }}><Icon name="check" size={10} /> {t(m.out)}</span>{/if}
            </div>
          {/each}
        </div>
        <div class="t-panel" class:on={tPhase >= 1}>
          <div class="t-panel-h"><Icon name="users" size={13} /> {t('tour.s6Setup')} <span>{t('tour.s6TeamName')}</span></div>
          {#each TEAM as m, i}
            <div class="t-row" class:in={tPhase >= 2}>
              <span class="t-tick" class:on={tPhase >= 2} style="transition-delay:{i * 120}ms"><Icon name="check" size={10} /></span>
              <AgentMascot name={m.name} icon={m.icon} hue={m.hue} size={34} state={memberState} />
              <b>{m.name}</b>
              {#if tPhase >= 4 && tPhase < 6}<i class="prog" in:fade><u style="transition-delay:{i * 150}ms"></u></i>{/if}
              {#if tPhase >= 6}<em in:fade>{t('tour.s6Finished')}</em>{/if}
            </div>
          {/each}
          {#if tPhase >= 1 && tPhase < 2}<div class="t-foot" in:fade>{t('tour.s6Pick')}</div>{/if}
          {#if tPhase >= 4 && tPhase < 6}<div class="t-foot" in:fade>{t('tour.s6Parallel')}</div>{/if}
        </div>
      </div>
      <h2>{t('tour.s6Title', { name: shown })}</h2>
      <p class="ob-sub">{t('tour.s6Sub', { name: shown })}<br />{#if tPhase >= 7}<span in:fade>{t('tour.s6Sub2', { name: shown })}</span>{/if}</p>

    {:else if scene === 7}
      <div class="stage perf">
        <div class="bars">
          <div class="bars-h"><Icon name="zap" size={12} /> {t('tour.s7Head')} <small>{t('tour.s7HeadSub')}</small></div>
          <div class="bar me">
            <span class="bar-name">Aetox <em>{t('tour.kHarnessDesktop')} · {t('tour.s7Measured')}</em></span>
            <span class="bar-track">
              <i class="heavy" style="width:{beat >= 1 ? pct(AETOX_HEAVY) : 0}%"></i>
              <i class="stack" style="width:{beat >= 1 ? pct(AETOX_WORK) : 0}%">{#each AETOX as seg}<b class="seg-{seg.k}" style="flex:{seg.mb}" title="{t(seg.key)} {seg.mb} MB"></b>{/each}</i>
            </span>
            <span class="bar-val"><b>{AETOX_WORK} MB</b> - {AETOX_HEAVY} MB</span>
          </div>
          {#if beat >= 2}
            <div class="legend" in:fade>
              {#each AETOX as seg}<span><i class="seg-{seg.k}"></i>{t(seg.key)} <b>{seg.mb}</b></span>{/each}
              <span class="legend-n">{t('tour.s7Note')}</span>
            </div>
          {/if}
          {#each RAM as r, i}
            <div class="bar">
              <span class="bar-name">{r.name} <em>{t(r.kind as 'tour.kIDE')}</em></span>
              <span class="bar-track">
                {#if r.heavy}<i class="heavy" style="width:{beat >= 1 ? pct(r.heavy) : 0}%; transition-delay:{i * 70 + 300}ms"></i>{/if}
                <i style="width:{beat >= 1 ? pct(r.work) : 0}%; transition-delay:{i * 70 + 200}ms"></i>
              </span>
              <span class="bar-val">{gb(r.work)}{#if r.heavy} - {gb(r.heavy)}{/if}{#if r.src === 'net'}<sup>*</sup>{/if}</span>
            </div>
          {/each}
          <div class="bars-f">{t('tour.s7Foot')}</div>
        </div>
      </div>
      <h2>{t('tour.s7Title')}</h2>
      <p class="ob-sub">{t('tour.s7Sub')}<br />{#if beat >= 3}<span in:fade>{t('tour.s7Sub2')}</span>{/if}</p>

    {:else if flow === 'setup'}
      <div class="stage split apart">
        <div class="twho"><Mascot {...A} pose={beat >= 1 ? 'presenting' : 'greeting'} size={120} /><div class="who-name">{shown}</div></div>
        <div class="twho"><Mascot {...C} pose="idle" size={120} still /><div class="who-name">{t('desk.coding')}</div></div>
      </div>
      <h2>{t('tour.s8SetupTitle')}</h2>
      <p class="ob-sub">{t('tour.s8SetupSub', { name: shown })}</p>
      <div class="steps-ahead">
        {#each [['plug', 'onboard.intentTitle'], ['palette', 'onboard.themeTitle'], ['hand', 'onboard.approvalTitle']] as [ic, k], i}
          <span class="step-ahead" in:fly={{ y: 10, duration: 400, delay: 200 + i * 140 }}><i>{i + 1}</i><Icon name={ic as 'plug'} size={13} /> {t(k as 'onboard.intentTitle')}</span>
        {/each}
      </div>
      <div class="ob-stack tight"><button type="button" class="ob-big primary tour-start" onclick={finish}><span class="ob-bigt">{t('tour.goSetup')}</span></button></div>
    {:else}
      <div class="stage split apart">
        <div class="twho"><Mascot {...A} pose={beat >= 1 ? 'success' : 'greeting'} size={120} /><div class="who-name">{shown}</div></div>
        <div class="twho"><Mascot {...C} pose="idle" size={120} /><div class="who-name">{t('desk.coding')}</div></div>
      </div>
      <h2>{t('tour.s8Title', { name: shown })}</h2>
      <p class="ob-sub">{t('tour.s8Sub')}</p>
      <div class="ob-stack tight"><button type="button" class="ob-big primary tour-start" onclick={finish}><span class="ob-bigt">{t('tour.close')}</span></button></div>
    {/if}

    </div>
    {/key}

    <div class="tour-nav">
      <button type="button" class="ob-link" disabled={scene === 0} onclick={() => go(scene - 1)}>← {t('tour.prev')}</button>
      <span class="tour-n">{scene + 1} / {SCENE_COUNT}</span>
      {#if scene < SCENE_COUNT - 1}
        <button type="button" class="ctrl primary tour-next" onclick={() => go(scene + 1)}>{t('tour.next')} →</button>
      {:else}
        <span class="tour-nav-gap"></span>
      {/if}
    </div>
  </div>
</div>
