// The catalogue of everything on the mascot that can be swapped.
//
// Built from the owner's Mascot Technical Blueprint v1.0 (12 ก.ย. 2026): twelve
// parts, of which the anatomy — cap, head shell, face screen, body connector,
// arms, hands, legs, feet — is fixed and lives in rig.ts, and four are slots
// that a role or a user fills by id: the top indicator (1), the ear badges
// (4/6), the light on the face screen (5) and what the hands hold (12).
// Floating cards and marks are a fifth list that only a pose may reach.
//
// Three rules, inherited from the cartoon faces this replaced: a part is named by
// its id and never its position (append, never insert), a row is data with a
// draw function and not a class (adding one is one line, not a file), and
// nothing here reads a file or a store — given the same ids and hue this
// returns the same markup on every machine.
//
// The ear badge deliberately has no list of its own: it is an id from the
// app's ICONS, the same set every button on screen is drawn from, or 'logo'.
// A user who sees `search` on an agent's ear has already seen that glyph on
// the tool that does it — that is the whole point of putting it there.
import { ICONS, type IconName } from '../icons'
import type { Palette } from './palette'

/** Draw function: palette in, SVG markup out. `g` is the gradient id prefix. */
export type Draw = (p: Palette, g: string) => string

// ---------------------------------------------------------------------------
// Icons on the mascot. Lucide paths are 24×24 strokes; the logo is the same
// path Logo.svelte draws (viewBox 804×762), a fill. Both land centred on
// (cx, cy) at `size` user units.
// ---------------------------------------------------------------------------

export type BadgeId = IconName | 'logo'

// Copied from Logo.svelte rather than imported: that component owns the mark
// on the window, this owns the mark on an ear, and neither should have to
// export its path to the other. Keep in step with assets/logo.svg.
const LOGO_PATH =
  'M 116.0,742.5 L 92.0,742.5 L 73.0,737.5 L 53.0,726.5 L 36.5,711.0 L 21.5,680.0 L 20.5,640.0 L 37.5,594.0 L 245.5,120.0 L 265.5,90.0 L 293.0,60.5 L 321.0,40.5 L 347.0,28.5 L 380.0,20.5 L 416.0,19.5 L 452.0,26.5 L 484.0,40.5 L 509.0,57.5 L 533.5,82.0 L 565.5,134.0 L 656.5,345.0 L 658.5,375.0 L 651.5,396.0 L 641.5,410.0 L 622.0,425.5 L 581.0,440.5 L 367.0,442.5 L 339.0,446.5 L 311.0,457.5 L 285.5,479.0 L 275.5,494.0 L 183.5,700.0 L 153.0,729.5 L 137.0,737.5 L 116.0,742.5 Z M 115.5,703.0 L 136.0,694.5 L 154.5,673.0 L 242.5,473.0 L 257.5,452.0 L 276.0,434.5 L 312.0,414.5 L 349.0,405.5 L 494.5,404.0 L 402.0,190.5 L 322.5,371.0 L 313.0,379.5 L 300.0,380.5 L 293.0,377.5 L 285.5,367.0 L 285.5,357.0 L 363.5,179.0 L 373.5,161.0 L 386.0,150.5 L 411.0,147.5 L 425.0,154.5 L 433.5,164.0 L 539.0,404.5 L 567.0,403.5 L 589.0,398.5 L 605.0,390.5 L 616.5,379.0 L 619.5,371.0 L 617.5,350.0 L 530.5,152.0 L 500.5,104.0 L 465.0,74.5 L 443.0,64.5 L 416.0,58.5 L 389.0,58.5 L 357.0,66.5 L 335.0,77.5 L 317.0,91.5 L 294.5,116.0 L 279.5,139.0 L 61.5,637.0 L 58.5,664.0 L 67.5,687.0 L 87.0,701.5 L 115.5,703.0 Z M 711.0,742.5 L 689.0,742.5 L 664.0,735.5 L 642.0,722.5 L 623.5,704.0 L 610.5,683.0 L 553.0,547.5 L 329.0,547.5 L 322.0,544.5 L 314.5,534.0 L 316.5,518.0 L 331.0,508.5 L 573.0,509.5 L 581.5,516.0 L 587.5,527.0 L 652.5,678.0 L 664.0,690.5 L 680.0,700.5 L 710.0,703.5 L 724.0,698.5 L 737.5,686.5 L 745.5,668.0 L 744.5,646.0 L 738.5,627.0 L 711.0,742.5 Z'

export function icon(id: BadgeId, cx: number, cy: number, size: number, color = '#fff', width = 2.2): string {
  if (id === 'logo') {
    return `<g transform="translate(${cx} ${cy}) scale(${size / 780}) translate(-402 -381)"><path d="${LOGO_PATH}" fill="${color}" fill-rule="evenodd"/></g>`
  }
  const k = size / 24
  return (
    `<g transform="translate(${cx - 12 * k} ${cy - 12 * k}) scale(${k})" fill="none" stroke="${color}"` +
    ` stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round">${ICONS[id] ?? ''}</g>`
  )
}

/** Is this an id the ear can wear? Unknown ids fall back to the logo. */
export function isBadge(id: string | undefined): id is BadgeId {
  return id === 'logo' || (!!id && id in ICONS)
}

// ---------------------------------------------------------------------------
// 1 · Top indicator
// ---------------------------------------------------------------------------
export type Part = { id: string; label: string; svg: Draw }

// Append only.
export const TOP: Part[] = [
  {
    id: 'orb',
    label: 'ลูกแก้ว LED',
    svg: (p, g) =>
      `<rect x="31" y="5.5" width="2" height="4.5" rx="1" fill="${p.primaryDn}"/>` +
      `<circle class="ms-orb" cx="32" cy="4.4" r="7" fill="url(#${g}halo)"/>` +
      `<circle class="ms-orb" cx="32" cy="4.4" r="3.9" fill="url(#${g}orb)"/>` +
      `<ellipse cx="30.7" cy="3" rx="1.2" ry=".8" fill="#fff" opacity=".85"/>`,
  },
  {
    id: 'chevrons',
    label: '>>',
    svg: (p) =>
      // Each chevron is its own path: mascot.css steps them forward one
      // after the other (ms-chev), the way `>>` reads — owner, 14 ก.ย. 2026:
      // "ตรงลูกศรที่หัวอ่ะครับอยากให้มันขยับได้".
      `<g class="ms-orb ms-chev" fill="none" stroke="${p.primaryUp}" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round">` +
      `<path d="M27 2l3.2 3-3.2 3"/><path d="M33.2 2l3.2 3-3.2 3"/></g>`,
  },
  { id: 'bar', label: 'แถบไฟ', svg: (p) => `<rect class="ms-orb" x="26" y="5" width="12" height="3" rx="1.5" fill="${p.primaryUp}"/>` },
  { id: 'none', label: 'ไม่มี', svg: () => `` },
]

// ---------------------------------------------------------------------------
// 5 · The light on the face screen. One slot, and an expression is which row
// is lit: `identity` rows are what a role rests on, the others are what a pose
// lights over it. One list, because a picker and a renderer must agree.
// ---------------------------------------------------------------------------
export type Face = Part & { identity?: boolean }

// Coordinates are for the screen at y 17.5–38 (head centre 27).
export const FACE: Face[] = [
  {
    id: 'neutral', label: 'Neutral', identity: true,
    svg: (p) =>
      `<rect class="ms-blink" x="23.6" y="22.6" width="4.8" height="9.6" rx="2.4" fill="${p.eye}"/>` +
      `<rect class="ms-blink" x="35.6" y="22.6" width="4.8" height="9.6" rx="2.4" fill="${p.eye}"/>`,
  },
  {
    id: 'focused', label: 'Focused', identity: true,
    svg: (p) =>
      `<path class="ms-blink" d="M22.6 24.2l6.8 2.2v5.2l-6.8-1.4z" fill="${p.eye}"/>` +
      `<path class="ms-blink" d="M41.4 24.2l-6.8 2.2v5.2l6.8-1.4z" fill="${p.eye}"/>`,
  },
  { id: 'happy', label: 'Happy', svg: (p) => `<g fill="none" stroke="${p.eye}" stroke-width="2.3" stroke-linecap="round"><path d="M23 29.4q3-4.8 6 0"/><path d="M35 29.4q3-4.8 6 0"/></g>` },
  { id: 'thinking', label: 'Thinking', svg: (p) => `<g fill="none" stroke="${p.eye}" stroke-width="2.3" stroke-linecap="round"><path d="M23.4 26.6q2.6 3.6 5.2 0"/><path d="M35.4 26.6q2.6 3.6 5.2 0"/></g>` },
  { id: 'excited', label: 'Excited', svg: (p) => `<g fill="none" stroke="${p.eye}" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"><path d="M23.6 23.8l4.2 3.6-4.2 3.6"/><path d="M40.4 23.8l-4.2 3.6 4.2 3.6"/></g>` },
  { id: 'curious', label: 'Curious', svg: (p) => `<g fill="none" stroke="${p.eye}" stroke-width="2.1" stroke-linecap="round"><ellipse class="ms-blink" cx="26.2" cy="27.6" rx="3" ry="4.2"/><path d="M35.6 24.6c0-2.6 5.2-2.8 5.2-.2 0 1.8-2.6 2-2.6 4.2M38.2 31.4v.3"/></g>` },
  // Appended after the blueprint's six. A picker never offers these; a pose does.
  { id: 'wink', label: 'Wink', svg: (p) => `<g fill="none" stroke="${p.eye}" stroke-width="2.3" stroke-linecap="round"><path d="M23.4 27.6h5.4"/><path d="M40.6 23.8l-4 3.8 4 3.8"/></g>` },
  { id: 'dim', label: 'Dim', svg: (p) => `<g fill="none" stroke="${p.eye}" stroke-width="2" stroke-linecap="round" opacity=".5"><path d="M23.6 28h5"/><path d="M35.6 28h5"/></g>` },
  // Three more resting faces (owner, 12 ก.ย.: "หน้าประจำตัว ค่าเริ่มต้นควรมี 5").
  // Identity rows: a role may rest on them, a picker offers them.
  {
    id: 'round', label: 'Round', identity: true,
    svg: (p) => `<circle class="ms-blink" cx="26" cy="27.4" r="3.4" fill="${p.eye}"/><circle class="ms-blink" cx="38" cy="27.4" r="3.4" fill="${p.eye}"/>`,
  },
  {
    id: 'wide', label: 'Wide', identity: true,
    svg: (p) => `<ellipse class="ms-blink" cx="26" cy="27.4" rx="3.8" ry="5.6" fill="${p.eye}"/><ellipse class="ms-blink" cx="38" cy="27.4" rx="3.8" ry="5.6" fill="${p.eye}"/>`,
  },
  {
    id: 'visor', label: 'Visor', identity: true,
    svg: (p) => `<rect class="ms-blink" x="22.6" y="25.2" width="18.8" height="4.4" rx="2.2" fill="${p.eye}"/>`,
  },
  // Heart eyes (owner, 12 ก.ย.: "เพิ่มตาหัวใจ"). Coordinates baked in rather than
  // a transform attribute: the blink animation owns `transform`.
  {
    id: 'heart', label: 'Heart', identity: true,
    svg: (p) => `<path class="ms-blink" d="M26 30.8C22.2 28.2 22.2 25 24.4 25C25.3 25 26 25.7 26 26.3C26 25.7 26.7 25 27.6 25C29.8 25 29.8 28.2 26 30.8Z" fill="${p.eye}"/><path class="ms-blink" d="M38 30.8C34.2 28.2 34.2 25 36.4 25C37.3 25 38 25.7 38 26.3C38 25.7 38.7 25 39.6 25C41.8 25 41.8 28.2 38 30.8Z" fill="${p.eye}"/>`,
  },
]

// ---------------------------------------------------------------------------
// 12 · What the hands hold. An object with volume, not a plane: the owner's
// diagnosis of every "the hand goes through it" report was that a flat
// trapezoid gives the eye nothing to put a hand in front OF.
//
// And the right way round (owner, 12 ก.ย.: "ทำไมโน้ตบุ๊กหันออก"): a laptop on a
// lap has its lid at the FAR edge of the deck — nearest the viewer, its back
// cover and the mark facing us, the screen facing the mascot — and the deck
// reaches back from the hinge onto the lap. So from the front the lid is all
// there is; the keyboard shows only as the head turns, the way the sheet's
// side view draws it. Depths: base from 3 (the lap) to 11 (the hinge), lid
// upright at 11 with its 1.2 of thickness towards the viewer; the base is a
// little wider than the lid so its hinge-side lip shows under the lid from
// the front and its end faces have something to attach to. Each plane turns
// with its own matrix in mascot.css.
// ---------------------------------------------------------------------------
function laptop(p: Palette, g: string, glyph: string): string {
  return (
    // Base in (x, depth) space, lap (3) to hinge (11). A little wider than the
    // lid, as the sheet draws it. Behind the lid from the front.
    `<g class="ms-deck"><path d="M21.5 3h21v8h-21z" fill="${p.slabUp}"/><path d="M22.1 3.6h19.8v.9H22.1z" fill="${p.slabDn}" opacity=".35"/>` +
    `<g stroke="${p.slabDn}" stroke-width=".7" opacity=".55" stroke-linecap="round"><path d="M24.5 5.6h15"/><path d="M25.5 7.8h13"/></g></g>` +
    // The base's end faces, (depth, thickness) — seen from the side …
    `<g class="ms-deckL"><path d="M3 0h8v1.8H3z" fill="${p.slab}"/><path d="M3 0h8v.5H3z" fill="#fff" opacity=".22"/></g>` +
    `<g class="ms-deckR"><path d="M3 0h8v1.8H3z" fill="${p.slab}"/><path d="M3 0h8v.5H3z" fill="#fff" opacity=".22"/></g>` +
    // … and its hinge-side edge, the lip that shows under the lid from the front.
    `<g class="ms-deckF"><path d="M21.5 53.4h21v1.8h-21z" fill="${p.slab}"/><path d="M21.5 53.4h21v.5h-21z" fill="#fff" opacity=".28"/></g>` +
    // Lid — 18 × 11.4, back cover towards the viewer with the mark; the screen
    // faces the mascot. Its side faces carry the thickness.
    `<g class="ms-lid"><path d="M24.6 42.4h14.8a1.2 1.2 0 0 1 1.2 1.1l.6 9.9H22.8l.6-9.9a1.2 1.2 0 0 1 1.2-1.1z" fill="url(#${g}slab)"/>` +
    `<path d="M25.2 43.8h13.6l.5 8H24.7z" fill="${p.slabDn}" opacity=".45"/>${glyph}` +
    `<path d="M25.2 43.8h13.6l.5 8H24.7z" fill="url(#${g}gl)"/>` +
    `<path d="M25 43h14" stroke="#fff" stroke-width=".5" opacity=".35" stroke-linecap="round"/></g>` +
    `<g class="ms-lidL"><path d="M11 42h1.2v11.4H11z" fill="${p.slabDn}"/></g>` +
    `<g class="ms-lidR"><path d="M11 42h1.2v11.4H11z" fill="${p.slabDn}"/></g>`
  )
}

export const PROP: Part[] = [
  { id: 'laptopA', label: 'แล็ปท็อป (โลโก้)', svg: (p, g) => laptop(p, g, icon('logo', 32, 47.7, 4.6)) },
  { id: 'laptopTerm', label: 'แล็ปท็อป >_', svg: (p, g) => laptop(p, g, icon('terminal', 32, 47.7, 5, p.eyeGlow, 2.6)) },
  {
    id: 'doc', label: 'เอกสาร',
    svg: (p) =>
      `<g class="ms-sheet"><path d="M24.5 43.5h13l2 12.5H23z" fill="${p.card}" stroke="${p.cardLine}" stroke-width=".8"/>` +
      `<g stroke="${p.cardLine}" stroke-width=".9" stroke-linecap="round"><path d="M27 47.5h8"/><path d="M27.4 50h8"/><path d="M27.8 52.5h5.5"/></g></g>`,
  },
  { id: 'none', label: 'มือเปล่า', svg: () => `` },
]

// ---------------------------------------------------------------------------
// Floating cards beside the head: what the mascot is doing, said with the
// app's own icon and no words, because the words change with the UI language
// and the icon does not. Screen-space — they neither turn nor breathe, they
// are UI rather than body. Only a pose may name one.
// ---------------------------------------------------------------------------
function card(p: Palette, iconId: IconName, tone: 'card' | 'alert' | 'dark' = 'card'): string {
  const fill = tone === 'alert' ? '#ffe9e9' : tone === 'dark' ? p.screen : p.card
  const line = tone === 'alert' ? '#e5484d' : tone === 'dark' ? p.primary : p.cardLine
  const ink = tone === 'alert' ? '#e5484d' : tone === 'dark' ? p.eyeGlow : p.primary
  return (
    `<g class="ms-float"><path d="M47 1h13a2 2 0 0 1 2 2v9.5a2 2 0 0 1-2 2h-8.5l-2.8 2.4V14.5H47a2 2 0 0 1-2-2V3a2 2 0 0 1 2-2z"` +
    ` fill="${fill}" stroke="${line}" stroke-width=".9"/>${icon(iconId, 53.5, 7.7, 8.5, ink, 2.4)}</g>`
  )
}

export const PANEL: Part[] = [
  { id: 'hi', label: 'ทักทาย', svg: (p) => card(p, 'hand') },
  {
    id: 'dots', label: 'คิด',
    svg: (p) =>
      `<g class="ms-float"><path d="M47 1h13a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-8.5l-2.8 2.4V13H47a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2z" fill="${p.card}" stroke="${p.cardLine}" stroke-width=".9"/>` +
      `<circle cx="50" cy="7" r="1.1" fill="${p.primary}"/><circle cx="53.5" cy="7" r="1.1" fill="${p.primary}"/><circle cx="57" cy="7" r="1.1" fill="${p.primary}"/></g>`,
  },
  { id: 'search', label: 'ค้นข้อมูล', svg: (p) => card(p, 'search') },
  { id: 'web', label: 'ค้นเว็บ', svg: (p) => card(p, 'globe') },
  { id: 'files', label: 'ค้นไฟล์', svg: (p) => card(p, 'folderOpen') },
  {
    id: 'docsearch', label: 'ค้นในเอกสาร',
    svg: (p) =>
      `<g class="ms-float"><path d="M47 1h13a2 2 0 0 1 2 2v9.5a2 2 0 0 1-2 2h-8.5l-2.8 2.4V14.5H47a2 2 0 0 1-2-2V3a2 2 0 0 1 2-2z" fill="${p.card}" stroke="${p.cardLine}" stroke-width=".9"/>` +
      `${icon('fileText', 52.4, 7.5, 8, p.primary, 2.4)}<circle cx="57.6" cy="10.1" r="2.6" fill="${p.card}"/>${icon('search', 57.8, 10.3, 5.2, p.primaryDn, 3)}</g>`,
  },
  { id: 'answer', label: 'ตอบ', svg: (p) => card(p, 'messageSquare') },
  { id: 'plan', label: 'วางแผน', svg: (p) => card(p, 'layoutList') },
  { id: 'code', label: 'โค้ด', svg: (p) => card(p, 'fileCode', 'dark') },
  { id: 'alert', label: 'ดีบัก', svg: (p) => card(p, 'alertTriangle', 'alert') },
  { id: 'chart', label: 'นำเสนอ', svg: (p) => card(p, 'chartColumn') },
  { id: 'heart', label: 'ช่วย', svg: (p) => card(p, 'heart') },
  { id: 'done', label: 'เสร็จ', svg: (p) => card(p, 'check') },
  { id: 'mic', label: 'ฟัง', svg: (p) => card(p, 'mic') },
  {
    id: 'zzz', label: 'พัก',
    svg: (p) =>
      `<g class="ms-float" fill="none" stroke="${p.primaryUp}" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">` +
      `<path d="M44 16h3.5l-3.5 4h3.5"/><path d="M50 10h4.5l-4.5 5h4.5"/><path d="M57 3h5.5l-5.5 6h5.5"/></g>`,
  },
  // On the ground beside a resting mascot, not beside its head — a pose names
  // it through `ground` rather than `panel` so it is drawn under the body.
  {
    id: 'charger', label: 'แท่นชาร์จ',
    svg: (p) =>
      `<ellipse cx="52" cy="57.5" rx="8" ry="3.2" fill="${p.ink}"/><path d="M44 54.5a8 3.2 0 0 1 16 0v3a8 3.2 0 0 1-16 0z" fill="${p.joint}"/>` +
      icon('zap', 52, 54, 5, p.eyeGlow, 2.4),
  },
  // A pillow under the head of the sleeper (owner, 12 ก.ย.: "เอาแบตออก เอาหมอน
  // มาให้น้องนอน"). On the ground layer, so the head lies on it; it sits where
  // the head comes to rest once the body leans back (mascot.css pose-recharge)
  // — low and to the left, past the viewBox's edge (the svg overflows) — and
  // big ("หมอนอันโต ๆ"): the whole sleeper lies on it, it shows all round the
  // head. Flared at the corners the way a pillow is, a seam across.
  {
    id: 'pillow', label: 'หมอน',
    svg: (p) =>
      `<ellipse cx="11" cy="64" rx="27" ry="2.8" fill="${p.ink}" opacity=".3"/>` +
      `<path d="M-14 48c0-3.4 4.4-4.4 7.6-3.4h30c3.2-1 7.6 0 7.6 3.4v11.6c0 3.4-4.4 4.4-7.6 3.4h-30c-3.2 1-7.6 0-7.6-3.4z" fill="${p.card}" stroke="${p.cardLine}" stroke-width=".6"/>` +
      `<path d="M-8 58q19 3 38 0" fill="none" stroke="${p.cardLine}" stroke-width=".6" opacity=".5" stroke-linecap="round"/>` +
      `<path d="M-11 46.6c5-1.4 10-1.7 15-1.7" fill="none" stroke="#fff" stroke-width=".9" opacity=".8" stroke-linecap="round"/>`,
  },
]

export const MARK: Record<string, (p: Palette) => string> = {
  flick: (p) => `<g stroke="${p.primaryUp}" stroke-width="1.6" stroke-linecap="round"><path d="M50 12l3-3.6"/><path d="M54 16l4.4-1.6"/></g>`,
  sparkle: () =>
    `<g class="ms-float" fill="#f3c34a"><path d="M8 6l1.2 3 3 1.2-3 1.2L8 14.4l-1.2-3-3-1.2 3-1.2z"/>` +
    `<path d="M57 20l.8 1.8 1.8.8-1.8.8-.8 1.8-.8-1.8-1.8-.8 1.8-.8z"/><path d="M12 18l.7 1.6 1.6.7-1.6.7-.7 1.6-.7-1.6-1.6-.7 1.6-.7z"/></g>`,
}

/** Find a row by id, or the fallback row. Unknown ids are the ordinary case
 *  for a file somebody wrote by hand, so they land on a part, never on an error. */
export function row<T extends { id: string }>(list: T[], id: string | undefined, fallback: string): T {
  return list.find((r) => r.id === id) ?? (list.find((r) => r.id === fallback) as T)
}
