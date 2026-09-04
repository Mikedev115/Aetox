/**
 * Binary trail — short runs of 1s and 0s that spawn in the pointer's wake.
 *
 * Glyphs only ever spawn BEHIND the cursor (opposite the direction of travel),
 * so the pointer itself stays clean and the effect reads as exhaust rather
 * than a halo. Faster movement spawns more runs, so a flick scatters and a
 * slow drift leaves a thin thread.
 *
 * Draws to its own 2D canvas layered above the GL composite and below the
 * copy. It runs on every beat, but the light mission/vision sections get an
 * inverted palette — the cyan/white wake is invisible on #EAF2F5, so those
 * beats use ink and a darkened teal/rust instead.
 */

const TRAIL_SECTIONS = new Set([
  'kv', 'works_intro', 'works', 'works_outro', 'mission_in', 'mission',
  'calc', 'vision', 'vision_out', 'service_in', 'service', 'cognexa'
]);

// beats where the scene wipes to #EAF2F5 — the trail flips to a dark palette
const LIGHT_SECTIONS = new Set(['mission_in', 'mission', 'calc', 'vision', 'vision_out']);

const MAX_GLYPHS = 620;
const CELL = 13;          // vertical spacing inside a run
const SIGNAL = '46, 230, 255';
const HEAT = '255, 106, 31';
const LEAD = '242, 245, 247';
// same three roles, re-voiced for the light beats
const SIGNAL_LT = '10, 116, 138';
const HEAT_LT = '188, 68, 12';
const LEAD_LT = '20, 27, 34';

export class BinaryTrail {
  constructor(container) {
    this.canvas = document.createElement('canvas');
    this.canvas.className = 'binary-trail';
    this.canvas.setAttribute('aria-hidden', 'true');
    container.appendChild(this.canvas);
    this.ctx = this.canvas.getContext('2d');

    this.glyphs = [];
    this.alpha = 0;
    this._px = -9999;
    this._py = -9999;
    this._seeded = false;

    this._onMove = (e) => this._track(e.clientX, e.clientY);
    this._onLeave = () => { this._seeded = false; };
    window.addEventListener('pointermove', this._onMove, { passive: true });
    window.addEventListener('pointerleave', this._onLeave, { passive: true });

    this._onResize = () => this.resize();
    window.addEventListener('resize', this._onResize);
    this.resize();
  }

  resize() {
    const dpr = Math.min(2, window.devicePixelRatio || 1);
    this._w = window.innerWidth;
    this._h = window.innerHeight;
    this.canvas.width = Math.floor(this._w * dpr);
    this.canvas.height = Math.floor(this._h * dpr);
    this.ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    this._dpr = dpr;
  }

  _track(x, y) {
    if (!this._seeded) {
      this._px = x;
      this._py = y;
      this._seeded = true;
      return;
    }
    const dx = x - this._px;
    const dy = y - this._py;
    this._px = x;
    this._py = y;

    const speed = Math.hypot(dx, dy);
    if (speed < 0.8 || this.alpha < 0.05) return;

    // unit vector pointing back along the path
    const bx = -dx / speed;
    const by = -dy / speed;
    // perpendicular, for lateral scatter
    const nx = -by;
    const ny = bx;

    const runs = Math.min(4, 1 + Math.floor(speed / 16));
    for (let r = 0; r < runs; r++) {
      const back = 16 + Math.random() * 52;
      const side = (Math.random() - 0.5) * 46;
      this._spawnRun(
        x + bx * back + nx * side,
        y + by * back + ny * side
      );
    }
  }

  _spawnRun(x, y) {
    const len = 4 + Math.floor(Math.random() * 6);
    const heat = Math.random() < 0.12;
    const now = performance.now();
    for (let i = 0; i < len; i++) {
      this.glyphs.push({
        x,
        y: y + i * CELL,
        ch: Math.random() < 0.5 ? '0' : '1',
        born: now,
        delay: i * 26,
        life: 900 + Math.random() * 700,
        lead: i === 0,
        heat
      });
    }
    // oldest-first pool trim keeps the cost flat under a fast scribble
    if (this.glyphs.length > MAX_GLYPHS) {
      this.glyphs.splice(0, this.glyphs.length - MAX_GLYPHS);
    }
  }

  /** @param {string} section current body[data-section] */
  update(section, dt) {
    const want = TRAIL_SECTIONS.has(section) ? 1 : 0;
    const light = LIGHT_SECTIONS.has(section);
    this.alpha += (want - this.alpha) * Math.min(1, dt * 4);

    const ctx = this.ctx;
    ctx.clearRect(0, 0, this._w, this._h);
    if (this.alpha < 0.01 || !this.glyphs.length) {
      if (this.alpha < 0.01) this.glyphs.length = 0;
      return;
    }

    const now = performance.now();
    ctx.font = '600 12px "IBM Plex Mono", ui-monospace, monospace';
    ctx.textBaseline = 'top';

    let write = 0;
    for (let i = 0; i < this.glyphs.length; i++) {
      const g = this.glyphs[i];
      const t = (now - g.born - g.delay) / g.life;
      if (t >= 1) continue;              // expired — drop by not writing back
      this.glyphs[write++] = g;
      if (t < 0) continue;               // not yet revealed

      // near-instant attack, long decay — the wake has to be brightest closest
      // to the cursor, so anything slower than this reads back-to-front
      const a = (t < 0.05 ? t / 0.05 : 1 - (t - 0.05) / 0.95) * this.alpha;
      if (a <= 0) continue;

      // occasional flicker so the run keeps churning while it fades
      if (Math.random() < 0.06) g.ch = g.ch === '0' ? '1' : '0';

      const rgb = light
        ? (g.lead ? LEAD_LT : (g.heat ? HEAT_LT : SIGNAL_LT))
        : (g.lead ? LEAD : (g.heat ? HEAT : SIGNAL));
      ctx.fillStyle = `rgba(${rgb}, ${a.toFixed(3)})`;
      ctx.fillText(g.ch, g.x, g.y + t * 9);
    }
    this.glyphs.length = write;
  }

  destroy() {
    window.removeEventListener('pointermove', this._onMove);
    window.removeEventListener('pointerleave', this._onLeave);
    window.removeEventListener('resize', this._onResize);
    this.canvas.remove();
  }
}
