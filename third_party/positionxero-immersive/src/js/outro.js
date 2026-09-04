import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

/**
 * Footer outro: a 200vh wrapper with a sticky screen. When the wrapper
 * enters the viewport bottom (+300px of travel) a one-shot sequence runs:
 *   1. the glow canvas fades in over 1 s
 *   2. the triangle A-mark slides from centre to the "A" slot (1 s power2.inOut)
 *   3. 100 ms later the wordmark construction plays (~4 s):
 *      guides → letter outlines → letters fill → guides fade
 * Scrolling back reverses/re-arms. All drawing is original canvas work.
 */

const LETTERS = 'ERO'; // the X arrives as the sliding mark, completing XERO

export class Outro {
  constructor() {
    this.wrapper = document.getElementById('outro');
    this.glow = document.getElementById('outro-glow');
    this.canvas = document.getElementById('outro-canvas');
    this.mark = document.getElementById('outro-mark');
    this.ctx = this.canvas.getContext('2d');
    this.gctx = this.glow.getContext('2d');
    this._t = 0;              // construction timeline 0..1 (0→~4s)
    this._playing = false;
    this._played = false;
    this._glowVis = 0;
    this._glowTarget = 0;
    this._blobs = Array.from({ length: 7 }, (_, i) => ({
      x: 0.2 + Math.random() * 0.6,
      y: 0.25 + Math.random() * 0.5,
      r: 0.18 + Math.random() * 0.22,
      hue: i % 2 ? 189 : 24,
      phase: Math.random() * Math.PI * 2
    }));

    this._resize();
    window.addEventListener('resize', () => this._resize());

    ScrollTrigger.create({
      trigger: this.wrapper,
      start: 'top bottom',
      end: '300px bottom',
      onUpdate: (self) => {
        if (self.progress >= 1 && !this._played) this._play();
      },
      onLeave: () => { if (!this._played) this._play(); },
      onLeaveBack: () => this._reverse()
    });

    this._raf = requestAnimationFrame(this._tick = this._tick.bind(this));
  }

  _resize() {
    const dpr = Math.min(window.devicePixelRatio, 2) * 0.5;
    for (const c of [this.canvas, this.glow]) {
      c.width = Math.max(2, c.clientWidth * dpr);
      c.height = Math.max(2, c.clientHeight * dpr);
    }

    // The lockup spans 36%..95% of the screen and reads X + gap + E R O, which
    // comes to 4.4785 cap heights. Sizing the mark from that here — rather than
    // from a fixed CSS percentage — is what keeps the X at the same cap height
    // as the letters; the two used to be set from different axes (width % vs
    // height %) and drifted apart at every viewport.
    const cw = this.canvas.clientWidth || window.innerWidth;
    const ch = this.canvas.clientHeight || window.innerHeight;
    if (cw > 768) {
      // X + 0.06 gap + 3 letters at 0.95 + 2 gaps at 0.22 = 4.4785 cap heights
      const capH = Math.min((cw * 0.59) / 4.4785, ch * 0.3);
      this.mark.style.width = `${(capH * 237) / 206}px`;
    } else {
      this.mark.style.width = '';      // narrow layout keeps its CSS sizing
    }
  }

  _play() {
    this._played = true;
    this._playing = true;
    this._glowTarget = 1;
    gsap.to(this.mark, { left: '36%', duration: 1, ease: 'power2.inOut' });
    setTimeout(() => { this._t0 = performance.now(); }, 1100);
  }

  _reverse() {
    this._played = false;
    this._playing = false;
    this._t = 0;
    this._t0 = null;
    this._glowTarget = 0;
    gsap.to(this.mark, { left: '50%', duration: 1, ease: 'power2.inOut' });
  }

  _tick(now) {
    this._raf = requestAnimationFrame(this._tick);
    // stop drawing entirely when far off screen
    const rect = this.wrapper.getBoundingClientRect();
    if (rect.top > window.innerHeight * 1.5 || rect.bottom < -window.innerHeight * 0.5) return;

    this._glowVis += (this._glowTarget - this._glowVis) * 0.04;
    this._drawGlow(now / 1000);

    if (this._t0) {
      this._t = Math.min(1, (now - this._t0) / 3970);
    }
    this._drawConstruction();
  }

  /* ---- soft blue noise-cloud glow, radial falloff, faint grain ---- */
  _drawGlow(t) {
    const ctx = this.gctx;
    const W = this.glow.width, H = this.glow.height;
    ctx.clearRect(0, 0, W, H);
    if (this._glowVis < 0.01) return;
    ctx.save();
    ctx.globalAlpha = this._glowVis;
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, W, H);
    ctx.globalCompositeOperation = 'lighter';
    for (const b of this._blobs) {
      const bx = (b.x + Math.sin(t * 0.23 + b.phase) * 0.06) * W;
      const by = (b.y + Math.cos(t * 0.19 + b.phase * 1.7) * 0.05) * H;
      const br = b.r * Math.min(W, H) * (1 + Math.sin(t * 0.31 + b.phase) * 0.15);
      const g = ctx.createRadialGradient(bx, by, 0, bx, by, br);
      g.addColorStop(0, `hsla(${b.hue}, 85%, 42%, ${0.26 * this._glowVis})`);
      g.addColorStop(1, 'hsla(196, 90%, 18%, 0)');
      ctx.fillStyle = g;
      ctx.beginPath();
      ctx.arc(bx, by, br, 0, Math.PI * 2);
      ctx.fill();
    }
    // radial falloff toward edges
    const vg = ctx.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.1, W / 2, H / 2, Math.max(W, H) * 0.7);
    vg.addColorStop(0, 'rgba(0,0,0,0)');
    vg.addColorStop(1, 'rgba(0,0,0,0.9)');
    ctx.globalCompositeOperation = 'source-over';
    ctx.fillStyle = vg;
    ctx.fillRect(0, 0, W, H);
    ctx.restore();
  }

  /* ---- wordmark construction ---- */
  _drawConstruction() {
    const ctx = this.ctx;
    const W = this.canvas.width, H = this.canvas.height;
    ctx.clearRect(0, 0, W, H);
    const t = this._t;
    if (t <= 0) return;

    // timeline stages (of ~4 s comp)
    const stage = (a, b) => Math.max(0, Math.min(1, (t - a) / (b - a)));
    const tGuidesV = stage(0, 0.19);      // vertical guides
    const tGuidesH = stage(0.1, 0.30);    // horizontals
    const tCircle = stage(0.17, 0.30);    // dashed circle
    const tOutline = stage(0.31, 0.46);   // letter outlines draw
    const tFill = stage(0.48, 0.66);      // letters fill white
    const guideDim = 1 - stage(0.6, 0.78); // guides fade out

    // The SVG mark is the single size reference: measure it and hang E R O off
    // it, so the letters share the X's cap height and baseline exactly and sit
    // beside it instead of under it. The lockup is held clear of the offer copy
    // column on the left rather than running through it.
    const cssW = this.canvas.clientWidth || 1;
    const k = W / cssW;                       // canvas px per CSS px
    const mr = this.mark.getBoundingClientRect();
    const markW = mr.width * k;
    const letterH = mr.height * k;
    const markCx = mr.left * k;
    const y0 = mr.top * k;
    const x0 = markCx + markW + letterH * 0.06;
    // Letters are proportioned off the cap height, never off "space that is
    // left" — the mark slides in on a tween, and deriving widths from the gap
    // to the right edge made the letters resize for the whole animation.
    const letterW = letterH * 0.95;
    const gap = letterW * 0.22;

    ctx.save();
    ctx.globalAlpha = Math.min(1, guideDim + 0.001) * 0.9;
    ctx.strokeStyle = 'rgba(145,145,145,0.8)';
    ctx.lineWidth = Math.max(0.5, W / 2200);

    // vertical guides through letter boundaries
    if (tGuidesV > 0) {
      for (let i = 0; i <= 11; i++) {
        const gx = W * 0.06 + (W * 0.88) * (i / 11);
        const len = H * tGuidesV;
        ctx.beginPath();
        ctx.moveTo(gx, (H - len) / 2);
        ctx.lineTo(gx, (H + len) / 2);
        ctx.stroke();
      }
    }
    // horizontals: top/mid/bottom of letters
    if (tGuidesH > 0) {
      for (const gy of [y0, y0 + letterH / 2, y0 + letterH, y0 - letterH * 0.3, y0 + letterH * 1.3, H * 0.5]) {
        const len = W * tGuidesH;
        ctx.beginPath();
        ctx.moveTo((W - len) / 2, gy);
        ctx.lineTo((W + len) / 2, gy);
        ctx.stroke();
      }
    }
    if (tCircle > 0) {
      ctx.setLineDash([5, 5]);
      ctx.lineWidth = Math.max(1, W / 1100);
      ctx.beginPath();
      ctx.arc(markCx + markW / 2, H / 2, H * 0.42, -Math.PI / 2, -Math.PI / 2 + tCircle * Math.PI * 2);
      ctx.stroke();
      ctx.setLineDash([]);
    }
    ctx.restore();

    // letters: outline then fill
    for (let i = 0; i < LETTERS.length; i++) {
      const lx = x0 + i * (letterW + gap);
      const local = Math.max(0, Math.min(1, tOutline * (LETTERS.length + 0.8) - i * 0.9));
      if (local > 0) {
        this._letterPath(ctx, LETTERS[i], lx, y0, letterW, letterH);
        ctx.save();
        ctx.strokeStyle = `rgba(255,255,255,${0.9 * local})`;
        ctx.lineWidth = Math.max(1, W / 1500);
        ctx.stroke();
        ctx.restore();
      }
      const fillLocal = Math.max(0, Math.min(1, tFill * (LETTERS.length + 0.8) - i * 0.9));
      if (fillLocal > 0) {
        this._letterPath(ctx, LETTERS[i], lx, y0, letterW, letterH);
        ctx.save();
        ctx.fillStyle = `rgba(255,255,255,${fillLocal})`;
        ctx.fill('nonzero');   // bars overlap — even-odd would punch holes
        ctx.restore();
      }
    }
  }

  _letterPath(ctx, ch, x, y, w, h) {
    const s = w * 0.26;
    ctx.beginPath();
    switch (ch) {
      case 'E':
        ctx.rect(x, y, s, h);
        ctx.rect(x, y, w, s);
        ctx.rect(x, y + (h - s) / 2, w * 0.82, s);
        ctx.rect(x, y + h - s, w, s);
        break;
      case 'R': {
        ctx.rect(x, y, s, h);
        ctx.rect(x, y, w, s);
        ctx.rect(x + w - s, y, s, h * 0.5);
        ctx.rect(x, y + h * 0.5 - s / 2, w, s);
        // diagonal leg
        ctx.moveTo(x + w * 0.45, y + h * 0.5 + s / 2);
        ctx.lineTo(x + w * 0.45 + s, y + h * 0.5 + s / 2);
        ctx.lineTo(x + w, y + h);
        ctx.lineTo(x + w - s, y + h);
        ctx.closePath();
        break;
      }
      case 'O':
        ctx.rect(x, y, w, s);
        ctx.rect(x, y + h - s, w, s);
        ctx.rect(x, y, s, h);
        ctx.rect(x + w - s, y, s, h);
        break;
      default:
        break;
    }
  }

  destroy() {
    cancelAnimationFrame(this._raf);
  }
}
