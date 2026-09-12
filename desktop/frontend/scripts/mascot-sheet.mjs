// Builds docs/mascot.html — the mascot drawn from the app's own code.
//
// GENERATED, never hand-edited, for the same reason agent-face-sheet.mjs is:
// a reference page drawn by hand is a second drawing of the same thing, and
// the day it drifts is the day a screenshot of it is sent back as a bug in
// the app (12 ก.ย. 2026 — three rounds went to a stale lab page). This page
// imports the real rig, the real poses and the real mascot.css, so what it
// shows IS what the app draws, and re-running is the whole update.
//
//   npm run mascots
//
// It also overwrites scratch/mascot/index.html when that folder exists, so
// the :5310 link the owner keeps open shows the same thing — and it writes
// desktop/companion_page.html, the page the Go feed serves to a surface
// outside the window (desktop/companion.go): the same rig, polling the feed.
import { build } from 'esbuild'
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const frontend = resolve(here, '..')
const repo = resolve(frontend, '../..')

const bundled = await build({
  entryPoints: [resolve(frontend, 'src/lib/mascot/rig.ts')],
  bundle: true,
  format: 'esm',
  write: false,
  platform: 'neutral',
})
const mod = await import('data:text/javascript;base64,' + Buffer.from(bundled.outputFiles[0].text).toString('base64'))
const { resolveMascot, mascotSVG, handVars } = mod

const rolesMod = await build({ entryPoints: [resolve(frontend, 'src/lib/mascot/roles.ts')], bundle: true, format: 'esm', write: false, platform: 'neutral' })
const { ROLE, roleOptions } = await import('data:text/javascript;base64,' + Buffer.from(rolesMod.outputFiles[0].text).toString('base64'))
const paletteMod = await build({ entryPoints: [resolve(frontend, 'src/lib/mascot/palette.ts')], bundle: true, format: 'esm', write: false, platform: 'neutral' })
const { SHELL, ACCENT } = await import('data:text/javascript;base64,' + Buffer.from(paletteMod.outputFiles[0].text).toString('base64'))

const posesMod = await build({ entryPoints: [resolve(frontend, 'src/lib/mascot/poses.ts')], bundle: true, format: 'esm', write: false, platform: 'neutral' })
const { POSE } = await import('data:text/javascript;base64,' + Buffer.from(posesMod.outputFiles[0].text).toString('base64'))

const css = readFileSync(resolve(frontend, 'src/lib/mascot/mascot.css'), 'utf8')
const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

function mascot(o, size, turn) {
  const m = resolveMascot({ ...o, size })
  const t = turn ?? m.pose.turn
  return `<span class="mascot pose-${m.poseId} settle" style="--t:${t}deg; ${handVars(m)}; --ms-blink:${4.4 + (m.hue % 9) * 0.3}s; width:${size}px; height:${size}px"><svg viewBox="0 0 64 64">${mascotSVG(m)}</svg></span>`
}
const cell = (o, size, title, sub, turn, dark = false) =>
  `<div class="cell${dark ? ' dark' : ''}">${mascot(o, size, turn)}<b>${esc(title)}</b><i>${esc(sub)}</i></div>`

const ASSISTANT = roleOptions('assistant')
const CODE = roleOptions('code')

// The seven that ship, with the badge each declares in its AGENT.md.
const TEAM = [['deepresearch', 'search'], ['doc', 'fileText'], ['video', 'clapperboard'], ['sheet', 'chartColumn'], ['github', 'gitBranch'], ['automation', 'zap'], ['editor', 'slidersHorizontal']]
const turnaround = [[0, 'Front'], [90, 'Side (L)'], [180, 'Back'], [-90, 'Side (R)'], [30, '30°'], [60, '60°'], [-45, '−45°'], [135, '135°']]
  .map(([t, name]) => cell({ ...ASSISTANT, badgeR: 'terminal', pose: 'idle' }, 150, name, `--t: ${t}deg`, t))
  .join('')

const poses = Object.keys(POSE)
  .map((id) => {
    const role = id === 'coding' || id === 'debugging' ? CODE : ASSISTANT
    return cell({ ...role, pose: id }, 150, id, `turn ${POSE[id].turn}°${POSE[id].look ? ` · look ±${POSE[id].look}°` : ''}`, undefined, role === CODE)
  })
  .join('')

const team = TEAM.map(([name, badge], i) => cell({ ...ASSISTANT, hue: (37 * i + 150) % 360, badge, pose: 'typing' }, 130, name, `icon: ${badge}`, [40, -35, 55, -50, 30, -60, 45][i])).join('')

// Every finish on a spread of accents — the colour dial the owner asked for.
// First the default (ink), then the brand, then round the wheel and the metals.
const finishes = SHELL.map(
  (sh) => `<div class="row"><span class="l">${esc(sh.label)} <code>${esc(sh.id)}</code></span>${['ink', 'brand', 'mint', 'orange', 'magenta', 'gold', 'slate'].map((accent) => mascot({ ...ASSISTANT, accent, shell: sh.id, pose: 'idle' }, 96, 12)).join('')}</div>`
).join('')

const roles = ROLE.map((r) => cell(roleOptions(r.id), 150, r.label, `role: ${r.id}`, 0, r.id === 'code')).join('')

const sizes = [184, 96, 72, 48, 38, 22, 20]
  .map((s) => `<span class="l">${s}px</span>${mascot({ ...ASSISTANT, pose: 'idle' }, s, 0)}${mascot({ ...CODE, pose: 'coding' }, s, 20)}${mascot({ ...ASSISTANT, hue: 150, badge: 'search', pose: 'thinking' }, s, -25)}`)
  .join('')

const html = `<!doctype html>
<html lang="th">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Aetox mascot — จากโค้ดจริง</title>
<style>
${css}
:root { --bg:#0b1020; --ink:#e8edf7; --mute:#8b95ab; --card:#121a2e; --line:#243051; }
body { margin:0; font:14px/1.5 system-ui, "Segoe UI", sans-serif; color:var(--ink); background:var(--bg); }
main { max-width:1240px; margin:0 auto; padding:24px 24px 60px; }
h1 { font-size:20px; margin:0 0 4px; }
h2 { font-size:14px; margin:28px 0 10px; color:var(--mute); font-weight:600; }
p.note { color:var(--mute); margin:0 0 12px; max-width:820px; }
.sheet { display:flex; flex-wrap:wrap; gap:12px; }
.cell { background:var(--card); border:1px solid var(--line); border-radius:14px; padding:12px 8px 8px; text-align:center; width:166px; }
.cell.dark { background:#05080f; }
.cell .mascot { margin:0 auto; }
.cell b { display:block; font-size:12.5px; margin-top:6px; }
.cell i { display:block; font-style:normal; color:var(--mute); font-size:11.5px; }
.strip { display:flex; align-items:flex-end; gap:10px; background:var(--card); border:1px solid var(--line); border-radius:12px; padding:12px 14px; flex-wrap:wrap; }
.strip .l { color:var(--mute); font-size:12px; margin:0 6px 0 14px; align-self:center; }
.live { display:grid; grid-template-columns:460px 1fr; gap:20px; background:var(--card); border:1px solid var(--line); border-radius:18px; padding:20px; align-items:center; }
.live .mascot { cursor:pointer; }
.ctl { display:flex; flex-wrap:wrap; gap:6px; margin-top:10px; }
button { font:inherit; font-size:12px; padding:5px 10px; border-radius:8px; border:1px solid var(--line); background:#1a2440; color:var(--ink); cursor:pointer; }
button.on { background:#2f6fe4; border-color:#2f6fe4; }
input[type=range] { width:300px; }
.st { color:var(--mute); font-size:12px; margin-top:8px; }
.finishes { display:flex; flex-direction:column; gap:8px; }
.row { display:flex; align-items:center; gap:8px; background:var(--card); border:1px solid var(--line); border-radius:12px; padding:8px 14px; }
.row .l { width:120px; color:var(--mute); font-size:12px; }
</style>
</head>
<body>
<main>
  <h1>Aetox mascot — วาดจากโค้ดในแอป (<code>src/lib/mascot</code>)</h1>
  <p class="note">หน้านี้เจนด้วย <code>npm run mascots</code> จาก rig.ts / poses.ts / mascot.css ตัวเดียวกับที่แอปใช้ — สิ่งที่เห็นที่นี่คือสิ่งที่แอปวาด ไม่มีสำเนาที่สอง</p>

  <div class="live">
    <div id="liveBox">${mascot({ ...ASSISTANT, pose: 'idle' }, 440, 0)}</div>
    <div>
      <b>ลากแถบเพื่อหัน · กดท่า</b>
      <div class="ctl"><input id="turn" type="range" min="-180" max="180" value="0"> <span id="tv">0°</span></div>
      <div class="ctl" id="poseBtns"></div>
      <div class="ctl"><button data-role="assistant" class="on">assistant</button><button data-role="code">code</button></div>
      <div class="ctl" id="shellBtns"></div>
      <div class="ctl" id="accentBtns"></div>
      <div class="ctl"><span class="st">hue (เอเจน)</span> <input id="hue" type="range" min="-1" max="359" value="-1"> <span id="hv">—</span></div>
    </div>
  </div>

  <h2>Roles — เทมเพลตของทั้งตัว (roles.ts) · เอเจน = assistant + icon ของตัวเอง</h2>
  <div class="sheet">${roles}</div>

  <h2>Finishes — ตัวเดียวกัน เปลี่ยนสีตัวและสี accent (palette.ts SHELL × ACCENT: ink · brand · mint · orange · magenta · gold · slate)</h2>
  <div class="finishes">${finishes}</div>

  <h2>Turnaround</h2>
  <div class="sheet">${turnaround}</div>

  <h2>Poses — ${Object.keys(POSE).length} ท่า</h2>
  <div class="sheet">${poses}</div>

  <h2>ทีมเอเจน — badge = icon ใน AGENT.md</h2>
  <div class="sheet">${team}</div>

  <h2>ขนาด — 184 (หน้าต้อนรับ) · 96 (เริ่มมี glow) · 72 (ใน bubble) · 48 (เริ่ม LOD ต่ำ) · 38 · 22 · 20</h2>
  <div class="strip">${sizes}</div>
</main>
<script type="module">
const bundle = ${JSON.stringify(bundled.outputFiles[0].text)}
const mod = await import('data:text/javascript;base64,' + btoa(unescape(encodeURIComponent(bundle))))
const { resolveMascot, mascotSVG, handVars } = mod
const POSE = ${JSON.stringify(Object.fromEntries(Object.entries(POSE).map(([k, v]) => [k, { turn: v.turn }])))}
const ROLES = { assistant: ${JSON.stringify(ASSISTANT)}, code: ${JSON.stringify(CODE)} }
const SHELLS = ${JSON.stringify(SHELL.map((s) => s.id))}
const ACCENTS = ${JSON.stringify(ACCENT.map((a) => a.id))}
let pose = 'idle', role = 'assistant', turn = 0, shell = 'white', accent = 'ink', hue = -1
const box = document.getElementById('liveBox')
function paint() {
  const m = resolveMascot({ ...ROLES[role], shell, accent, ...(hue >= 0 ? { hue } : {}), pose, size: 440 })
  box.innerHTML = '<span class="mascot pose-' + m.poseId + ' settle" style="--t:' + turn + 'deg; ' + handVars(m) + '; --ms-blink:5.1s; width:440px; height:440px"><svg viewBox="0 0 64 64">' + mascotSVG(m) + '</svg></span>'
  for (const b of document.querySelectorAll('#poseBtns button')) b.classList.toggle('on', b.dataset.pose === pose)
  for (const b of document.querySelectorAll('[data-role]')) b.classList.toggle('on', b.dataset.role === role)
  for (const b of document.querySelectorAll('[data-shell]')) b.classList.toggle('on', b.dataset.shell === shell)
  for (const b of document.querySelectorAll('[data-accent]')) b.classList.toggle('on', hue < 0 && b.dataset.accent === accent)
}
const AB = document.getElementById('accentBtns')
for (const id of ACCENTS) { const b = document.createElement('button'); b.textContent = id; b.dataset.accent = id; b.onclick = () => { accent = id; hue = -1; document.getElementById('hue').value = -1; document.getElementById('hv').textContent = '—'; paint() }; AB.append(b) }
const SB = document.getElementById('shellBtns')
for (const id of SHELLS) { const b = document.createElement('button'); b.textContent = id; b.dataset.shell = id; b.onclick = () => { shell = id; paint() }; SB.append(b) }
document.getElementById('hue').oninput = (e) => { hue = Number(e.target.value); document.getElementById('hv').textContent = hue < 0 ? '—' : hue + '°'; paint() }
const PB = document.getElementById('poseBtns')
for (const id of Object.keys(POSE)) { const b = document.createElement('button'); b.textContent = id; b.dataset.pose = id; b.onclick = () => { pose = id; turn = POSE[id].turn; document.getElementById('turn').value = turn; document.getElementById('tv').textContent = turn + '°'; paint() }; PB.append(b) }
for (const b of document.querySelectorAll('[data-role]')) b.onclick = () => { role = b.dataset.role; paint() }
document.getElementById('turn').oninput = (e) => { turn = Number(e.target.value); document.getElementById('tv').textContent = turn + '°'; box.firstElementChild.style.setProperty('--t', turn + 'deg') }
paint()
</script>
</body>
</html>
`

const out = resolve(repo, 'docs/mascot.html')
writeFileSync(out, html)
console.log('wrote', out)

// ---- the companion page: what a window outside the app shows -----------------
// Polls ./state (relative, so the token in the path comes along) and draws the
// mascot the feed describes. Transparent ground, because the window it is
// meant for is transparent; a plain browser tab shows it on white.
const companionHtml = `<!doctype html>
<html lang="th">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Aetox companion</title>
<style>
${css}
html, body { margin: 0; height: 100%; background: transparent; overflow: hidden; font: 13px/1.4 system-ui, "Segoe UI", sans-serif; }
.wrap { position: fixed; inset: 0; display: flex; align-items: flex-end; justify-content: flex-end; padding: 12px; gap: 10px; }
.say { position: relative; max-width: 260px; background: #f5f8ff; color: #1a2233; border-radius: 12px; padding: 8px 11px; box-shadow: 0 6px 18px rgb(0 0 0 / .35); margin-bottom: 30px; overflow-wrap: anywhere; }
.say::after { content: ''; position: absolute; right: -6px; bottom: 12px; width: 12px; height: 12px; background: #f5f8ff; transform: rotate(45deg); border-radius: 2px; }
.say[hidden] { display: none; }
.off { opacity: .35; }
</style>
</head>
<body>
<div class="wrap"><div class="say" id="say" hidden></div><div id="fig"></div></div>
<script type="module">
const mod = await import('data:text/javascript;base64,' + btoa(unescape(encodeURIComponent(${JSON.stringify(bundled.outputFiles[0].text)}))))
const { resolveMascot, mascotSVG, handVars } = mod
const SIZE = 104
const fig = document.getElementById('fig'), say = document.getElementById('say')
let seq = -1, drawn = ''
function draw(s) {
  const key = JSON.stringify([s.pose, s.prefs, s.on])
  if (key !== drawn) {
    drawn = key
    const p = s.prefs ?? {}
    const m = resolveMascot({ badge: 'logo', shell: p.shell, accent: p.accent, top: p.top, face: p.face, pose: s.pose || 'idle', size: SIZE })
    fig.innerHTML = '<span class="mascot pose-' + m.poseId + ' sway settle' + (s.on ? '' : ' off') + '" style="--t:' + m.pose.turn + 'deg; ' + handVars(m) + '; --ms-blink:5.1s; width:' + SIZE + 'px; height:' + SIZE + 'px"><svg viewBox="0 0 64 64">' + mascotSVG(m) + '</svg></span>'
  }
  say.textContent = s.report || ''
  say.hidden = !s.report
}
async function poll() {
  try {
    const r = await fetch('state', { cache: 'no-store' })
    if (r.ok) { const s = await r.json(); if (s.seq !== seq) { seq = s.seq; draw(s) } }
  } catch { /* the app is away; try again */ }
  setTimeout(poll, 500)
}
draw({ pose: 'idle', prefs: {}, on: true, report: '' })
poll()
</script>
</body>
</html>
`
const page = resolve(repo, 'desktop/companion_page.html')
writeFileSync(page, companionHtml)
console.log('wrote', page)
const lab = resolve(repo, 'scratch/mascot/index.html')
if (existsSync(dirname(lab))) {
  writeFileSync(lab, html)
  console.log('wrote', lab)
}
