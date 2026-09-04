import * as THREE from 'three';

/**
 * Loading sequence — an original canvas re-creation of the blueprint
 * loader: a construction grid (guide lines, dashed circles, dashed
 * cross-hairs) draws in as progress advances, two long X-blade diagonals
 * strike through it, the X-mark outline traces on once loading completes,
 * and the tagline scramble-types in
 * XERO letters. The finished sheet is also handed to the GL composite
 * as a texture for the reveal crossfade.
 *
 * Timing model:
 *  - guide drawing scrubbed by min(realProgress, elapsed/1s), smoothed 0.1
 *  - logo outline: 1.8 s, cubic-bezier(.53,.25,.3,.99)
 *  - tagline: starts at 400 ms; per-char reveal 300 ms + i*40 ms;
 *    unrevealed chars re-randomise from "XERO" every 32 ms at 30% opacity
 *  - hide: overlay fades 0.8 s, then the GL intro (uLoaded 0→1, 3 s) runs
 */

const TAGLINE = 'We turn ad spend into revenue.';
const SCRAMBLE = ['X', 'E', 'R', 'O'];

// bezier(.53,.25,.3,.99) approximation via sampled cubic
function cubicBezierEase(x1, y1, x2, y2) {
  return (t) => {
    // newton iterations on the x polynomial
    let u = t;
    for (let i = 0; i < 5; i++) {
      const x = 3 * u * (1 - u) * (1 - u) * x1 + 3 * u * u * (1 - u) * x2 + u * u * u - t;
      const dx = 3 * (1 - u) * (1 - u) * x1 + 6 * u * (1 - u) * (x2 - x1) + 3 * u * u * (1 - x2);
      if (Math.abs(dx) < 1e-6) break;
      u -= x / dx;
      u = Math.min(1, Math.max(0, u));
    }
    return 3 * u * (1 - u) * (1 - u) * y1 + 3 * u * u * (1 - u) * y2 + u * u * u;
  };
}
const logoEase = cubicBezierEase(0.53, 0.25, 0.3, 0.99);

/**
 * The blueprint sheet in normalized 0..1 space (drawn square, centred).
 * Triangle: apex (0.5, 0.2625), base corners (0.2937, 0.6194)/(0.7063, 0.6194)
 * — proportions taken from the reference composition.
 */
const APEX = { x: 0.5, y: 0.2625 };
const BL = { x: 0.2937, y: 0.6194 };
const BR = { x: 0.7063, y: 0.6194 };

export class Loader {
  constructor() {
    this.el = document.getElementById('loading');
    this.canvas = document.getElementById('loading-canvas');
    this.textEl = document.getElementById('loading-text');
    this.ctx = this.canvas.getContext('2d');
    this._real = 0;
    this._display = 0;
    this._guides = 0;         // smoothed guide draw progress
    this._logoT = -1;         // -1 = not started
    this._start = performance.now();
    this._textStarted = false;
    this._chars = [];
    this._done = false;
    this._resolve = null;
    this.finished = new Promise((r) => { this._resolve = r; });
    this._resize();
    window.addEventListener('resize', this._resizeBound = () => this._resize());
    this._tick = this._tick.bind(this);
    this._raf = requestAnimationFrame(this._tick);
    setTimeout(() => this._startText(), 400);
  }

  setProgress(p) {
    this._real = Math.max(this._real, Math.min(1, p));
  }

  _resize() {
    const dpr = Math.min(window.devicePixelRatio, 2);
    this.canvas.width = this.canvas.clientWidth * dpr;
    this.canvas.height = this.canvas.clientHeight * dpr;
  }

  _startText() {
    this._textStarted = true;
    this.textEl.textContent = '';
    this._chars = [...TAGLINE].map((ch, i) => {
      const span = document.createElement('span');
      span.className = 'loading__char';
      if (ch === ' ') {
        span.innerHTML = '&nbsp;';
        span.style.opacity = '0';
      } else {
        span.textContent = SCRAMBLE[(Math.random() * SCRAMBLE.length) | 0];
        span.style.opacity = '0.3';
      }
      this.textEl.appendChild(span);
      return { ch, span, revealAt: 300 + i * 40, revealed: ch === ' ' };
    });
    this._textStart = performance.now();
    this._scrambleTimer = setInterval(() => {
      let allDone = true;
      const el = performance.now() - this._textStart;
      for (const c of this._chars) {
        if (c.revealed) continue;
        if (el >= c.revealAt) {
          c.span.textContent = c.ch;
          c.span.style.opacity = '1';
          c.revealed = true;
        } else {
          c.span.textContent = SCRAMBLE[(Math.random() * SCRAMBLE.length) | 0];
          allDone = false;
        }
      }
      if (allDone) clearInterval(this._scrambleTimer);
    }, 32);
  }

  /* ---------- drawing ---------- */

  _draw() {
    const ctx = this.ctx;
    const W = this.canvas.width;
    const H = this.canvas.height;
    ctx.clearRect(0, 0, W, H);
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, W, H);

    // square sheet centred, sized like the reference (~80% of min dim * 1.6)
    const S = Math.min(W, H) * 1.25;
    const ox = (W - S) / 2;
    const oy = (H - S) / 2;
    const px = (p) => ({ x: ox + p.x * S, y: oy + p.y * S });

    const g = this._guides;                     // 0..1 guide draw progress
    const stage = (a, b) => Math.max(0, Math.min(1, (g - a) / (b - a)));

    // box corners of the X mark: APEX.y is the top edge, BL/BR the bottom
    const tl = px({ x: BL.x, y: APEX.y }), tr = px({ x: BR.x, y: APEX.y });
    const bl = px(BL), br = px(BR);
    const cx = ox + 0.5 * S;
    const cy = oy + (APEX.y + BL.y) / 2 * S + S * 0.02;

    ctx.lineWidth = Math.max(1, S / 2304);
    const grey = 'rgba(84,84,84,0.9)';
    const light = 'rgba(145,145,145,0.9)';

    const line = (x0, y0, x1, y1, t, color = light, dash = null) => {
      if (t <= 0) return;
      ctx.save();
      if (dash) ctx.setLineDash(dash);
      ctx.strokeStyle = color;
      ctx.beginPath();
      ctx.moveTo(x0, y0);
      ctx.lineTo(x0 + (x1 - x0) * t, y0 + (y1 - y0) * t);
      ctx.stroke();
      ctx.restore();
    };

    // 1. verticals (f0-10) — centre line plus the mark's left/right box edges
    const tV = stage(0, 0.11);
    line(cx, oy + S * 0.26, cx, oy + S * 0.74, tV, grey, [4, 4]);
    for (const vx of [BL.x, BR.x]) {
      line(ox + vx * S, oy + S * 0.2, ox + vx * S, oy + S * 0.78, tV, grey, [4, 4]);
    }

    // 2. horizontals (f4-30) — through apex, base, mid guides
    const tH = stage(0.045, 0.33);
    // cap, upper notch shoulder, centre, lower notch shoulder, baseline — the
    // five lines the X is actually set on (the old set was the A's base stack)
    const hy = [APEX.y, 0.3837, 0.441, 0.4983, BL.y, 0.629];
    hy.forEach((y, i) => {
      const yy = oy + y * S;
      const dir = i % 2 ? 1 : -1;
      const x0 = dir > 0 ? ox - S * 0.1 : ox + S * 1.1;
      const x1 = dir > 0 ? ox + S * 1.1 : ox - S * 0.1;
      line(x0, yy, x1, yy, tH, i === 0 ? light : grey);
    });

    // 3. outer dashed circle (f4-40)
    const tC1 = stage(0.045, 0.44);
    if (tC1 > 0) {
      ctx.save();
      ctx.strokeStyle = grey;
      ctx.setLineDash([4, 4]);
      ctx.beginPath();
      ctx.arc(cx, cy, S * 0.2386, -Math.PI / 2, -Math.PI / 2 + tC1 * Math.PI * 2);
      ctx.stroke();
      ctx.restore();
    }

    // 4. X-blade diagonals (f9-40) — the two long strokes the mark is built
    // from, corner to corner, three parallels each. They run well past the
    // sheet so the pair reads as an X being struck out, not as the mark.
    const tD1 = stage(0.1, 0.27);
    const tD2 = stage(0.24, 0.44);
    const EXT = 1.9;               // overshoot past each corner, in blade lengths
    const blades = [
      { a: tl, b: br, t: tD2 },    // top-left -> bottom-right
      { a: tr, b: bl, t: tD1 }     // top-right -> bottom-left
    ];
    for (const bd of blades) {
      const dx = bd.b.x - bd.a.x;
      const dy = bd.b.y - bd.a.y;
      const len = Math.hypot(dx, dy) || 1;
      const nx = -dy / len, ny = dx / len;      // unit perpendicular
      for (let i = -1; i <= 1; i++) {
        const off = i * S * 0.008;
        line(
          bd.a.x - dx * EXT + nx * off, bd.a.y - dy * EXT + ny * off,
          bd.b.x + dx * EXT + nx * off, bd.b.y + dy * EXT + ny * off,
          bd.t, light
        );
      }
    }

    // 5. inner dashed circle (f15-40)
    const tC2 = stage(0.17, 0.44);
    if (tC2 > 0) {
      ctx.save();
      ctx.strokeStyle = grey;
      ctx.setLineDash([4, 4]);
      ctx.lineWidth = Math.max(1, S / 1600);
      ctx.beginPath();
      ctx.arc(cx, cy, S * 0.1186, -Math.PI / 2, -Math.PI / 2 + tC2 * Math.PI * 2);
      ctx.stroke();
      ctx.restore();
    }

    // 6. dashed cross-hairs (f17-37)
    const tX = stage(0.19, 0.41);
    line(cx - S * 0.24, cy - S * 0.24, cx + S * 0.24, cy + S * 0.24, tX, grey, [4, 4]);
    line(cx + S * 0.24, cy - S * 0.24, cx - S * 0.24, cy + S * 0.24, tX, grey, [4, 4]);

    // 7. logo outline draw-on (double stroke: outer + inner counter + feet)
    if (this._logoT >= 0) {
      const lt = logoEase(Math.min(1, this._logoT));
      this._drawLogoOutline(ctx, S, ox, oy, lt);
    }

    // edge fade mask (10% / 90% horizontal)
    const fadeW = W * 0.1;
    let grad = ctx.createLinearGradient(0, 0, fadeW, 0);
    grad.addColorStop(0, 'rgba(0,0,0,1)');
    grad.addColorStop(1, 'rgba(0,0,0,0)');
    ctx.fillStyle = grad;
    ctx.fillRect(0, 0, fadeW, H);
    grad = ctx.createLinearGradient(W, 0, W - fadeW, 0);
    grad.addColorStop(0, 'rgba(0,0,0,1)');
    grad.addColorStop(1, 'rgba(0,0,0,0)');
    ctx.fillStyle = grad;
    ctx.fillRect(W - fadeW, 0, fadeW, H);
  }

  _drawLogoOutline(ctx, S, ox, oy, t) {
    if (t <= 0) return;

    // X-mark outline in sheet space: box matching the old mark's footprint
    const bx0 = 0.2937, bx1 = 0.7063;
    const W = bx1 - bx0;
    const H = W * 0.8687;
    const cx = 0.5;
    const cy = 0.441;
    const tx = 0.34 * W;
    const ty = 0.34 * H;
    const hw = W / 2, hh = H / 2;
    const sTop = (hw - tx) / (2 * hw - tx);
    const ncY = hh - sTop * (2 * hh - ty);
    const sSide = (hh - ty) / (2 * hh - ty);
    const ncX = hw - sSide * (2 * hw - tx);
    const P = (x, y) => [cx + x, cy - y];
    const path = [
      P(-hw, hh), P(-hw + tx, hh), P(0, ncY), P(hw - tx, hh), P(hw, hh),
      P(hw, hh - ty), P(ncX, 0), P(hw, -hh + ty), P(hw, -hh),
      P(hw - tx, -hh), P(0, -ncY), P(-hw + tx, -hh), P(-hw, -hh),
      P(-hw, -hh + ty), P(-ncX, 0), P(-hw, hh - ty), P(-hw, hh)
    ];
    // centre cross-hair accents inside the notches
    const inner = [P(-ncX * 0.55, 0), P(ncX * 0.55, 0)];

    const drawPath = (pts, tt) => {
      let total = 0;
      const segs = [];
      for (let i = 1; i < pts.length; i++) {
        const dx = (pts[i][0] - pts[i - 1][0]) * S;
        const dy = (pts[i][1] - pts[i - 1][1]) * S;
        const len = Math.hypot(dx, dy);
        segs.push(len);
        total += len;
      }
      let remaining = total * tt;
      ctx.beginPath();
      ctx.moveTo(ox + pts[0][0] * S, oy + pts[0][1] * S);
      for (let i = 1; i < pts.length && remaining > 0; i++) {
        const len = segs[i - 1];
        const f = Math.min(1, remaining / len);
        ctx.lineTo(
          ox + (pts[i - 1][0] + (pts[i][0] - pts[i - 1][0]) * f) * S,
          oy + (pts[i - 1][1] + (pts[i][1] - pts[i - 1][1]) * f) * S
        );
        remaining -= len;
      }
      ctx.stroke();
    };

    ctx.save();
    ctx.strokeStyle = '#bbb';
    ctx.lineWidth = Math.max(1.4, S / 1150);
    drawPath(path, t);
    drawPath(inner, Math.max(0, (t - 0.25) / 0.75));
    ctx.restore();
  }

  /* ---------- lifecycle ---------- */

  _tick(now) {
    if (this._done) return;
    this._raf = requestAnimationFrame(this._tick);

    const elapsed = (now - this._start) / 1000;
    const timeCap = Math.min(1, elapsed / 1);
    this._display = Math.min(this._real, timeCap);
    this._guides += (this._display * 1 - this._guides) * 0.1;

    if (this._logoT < 0 &&
        this._real >= 0.999 && timeCap >= 0.999 && Math.abs(this._display - this._guides) < 0.01) {
      this._logoStart = now;
      this._logoT = 0;
    }
    if (this._logoT >= 0 && this._logoT < 1) {
      this._logoT = (now - this._logoStart) / 1800;
      if (this._logoT >= 1) {
        this._logoT = 1;
        this._finish();
      }
    }

    this._draw();
  }

  async _finish() {
    // final frame → GL texture for the composite crossfade
    this._draw();
    const snap = document.createElement('canvas');
    const size = 2048;
    snap.width = snap.height = size;
    const sctx = snap.getContext('2d');
    const s = Math.min(this.canvas.width, this.canvas.height);
    sctx.drawImage(
      this.canvas,
      (this.canvas.width - s) / 2, (this.canvas.height - s) / 2, s, s,
      0, 0, size, size
    );
    this.texture = new THREE.CanvasTexture(snap);
    this.texture.colorSpace = THREE.SRGBColorSpace;
    this.texture.flipY = true;

    this._resolve();
  }

  /** Fade the DOM overlay out (0.8 s) — call after wiring the GL handoff. */
  hide() {
    return new Promise((res) => {
      this.el.setAttribute('data-done', 'true');
      setTimeout(() => {
        this._done = true;
        cancelAnimationFrame(this._raf);
        clearInterval(this._scrambleTimer);
        window.removeEventListener('resize', this._resizeBound);
        res();
      }, 800);
    });
  }
}
