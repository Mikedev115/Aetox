import * as THREE from 'three';

/**
 * Each case gets a key visual: a real screenshot where we have one (`img`),
 * otherwise a procedurally drawn stand-in in the Position Xero palette —
 * amber for the ads-led work, cyan for the search/AI-led work.
 *
 * Order here MUST match the .works-item order in index.html or the wall art
 * desyncs from the card copy.
 *
 * `img` is drawn over the procedural art once it loads, into the same canvas,
 * so the texture object never changes identity and nothing has to re-bind
 * uniforms. The generated art is what shows while the file is in flight.
 */

// public/ assets are served from the configured base
const asset = (p) => `${import.meta.env.BASE_URL}${p}`.replace(/([^:])\/{2,}/g, '$1/');

const WORKS = [
  // Cognexa leads the wall — our own AI company, and the proof behind the
  // AI-search work.
  { title: 'COGNEXA', sub: 'AI SYSTEMS + AUTOMATION', pal: ['#2EE6FF', '#0E5C77', '#05070A'], img: asset('img/works/cognexa.jpg') },
  { title: 'TINY HOMES SA', sub: 'CONFIGURATOR + PAID SEARCH', pal: ['#FF6A1F', '#7A2E0C', '#05070A'], img: asset('img/works/kv-01.jpg') },
  { title: 'OTAKU KULTURE', sub: 'ANIME + CAR CULTURE MERCH', pal: ['#2EE6FF', '#0E5C77', '#05070A'], img: asset('img/works/otaku-kulture.jpg') },
  { title: 'CAJEE BOTES', sub: 'BOOKINGS + LOCAL SEARCH', pal: ['#2EE6FF', '#0E5C77', '#05070A'], img: asset('img/works/cajee-botes.jpg') }
];

const SERVICE = [
  { title: 'ADS', sub: 'GOOGLE + META', pal: ['#FF6A1F', '#7A2E0C', '#05070A'] },
  { title: 'SEARCH', sub: 'AI + TRADITIONAL', pal: ['#2EE6FF', '#0E5C77', '#05070A'] },
  // The Cognexa reel peels to a fullscreen cover-fit quad, so it carries the
  // real site rather than generated art: this is the one panel a visitor reads
  // as a preview. 2560x1440 because it fills the viewport on a retina display.
  // The bright palette still paints the field while the image is in flight.
  { title: '', sub: '', pal: ['#EAF2F5', '#BFE9F5', '#8FD8EE'], bright: true,
    img: asset('img/works/cognexa-reel.jpg') }
];

function seeded(seed) {
  let s = (seed * 2654435761 + 1013904223) >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 4294967296;
  };
}

function paintArt(ctx, W, H, spec, seed, t = 0) {
  const rand = seeded(seed);
  const [c0, c1, c2] = spec.pal;

  const g = ctx.createLinearGradient(0, 0, W, H);
  g.addColorStop(0, c0);
  g.addColorStop(0.55, c1);
  g.addColorStop(1, c2);
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, W, H);

  // drifting light beams (bright specs stay airy — no dark beams)
  for (let i = 0; i < 8; i++) {
    ctx.save();
    ctx.translate(W * rand(), H * rand());
    ctx.rotate(rand() * Math.PI + t * 0.1 * (i % 2 ? 1 : -1));
    ctx.globalAlpha = spec.bright ? 0.08 + rand() * 0.1 : 0.10 + rand() * 0.18;
    ctx.fillStyle = spec.bright || i % 2 ? '#ffffff' : '#000000';
    const bw = W * (0.04 + rand() * 0.08);
    ctx.fillRect(-W, -bw / 2, W * 2, bw);
    ctx.restore();
  }

  // floating shards
  for (let i = 0; i < 12; i++) {
    ctx.save();
    const px = W * rand() + Math.sin(t * 0.7 + i) * 14;
    const py = H * rand() + Math.cos(t * 0.5 + i * 1.7) * 10;
    ctx.translate(px, py);
    ctx.rotate(rand() * Math.PI + t * 0.15);
    ctx.globalAlpha = 0.16 + rand() * 0.22;
    ctx.fillStyle = '#ffffff';
    const s = 22 + rand() * 90;
    ctx.beginPath();
    ctx.moveTo(0, -s / 2);
    ctx.lineTo(s / 2, s / 2);
    ctx.lineTo(-s / 2, s / 2);
    ctx.closePath();
    ctx.fill();
    ctx.restore();
  }

  // title block
  ctx.save();
  ctx.textAlign = 'center';
  ctx.fillStyle = '#ffffff';
  ctx.shadowColor = 'rgba(0,0,0,0.5)';
  ctx.shadowBlur = 26;
  ctx.font = `700 ${Math.round(H * 0.18)}px "Archivo", "Helvetica Neue", sans-serif`;
  ctx.fillText(spec.title, W / 2, H * 0.52);
  ctx.globalAlpha = 0.85;
  ctx.font = `400 ${Math.round(H * 0.055)}px "IBM Plex Mono", monospace`;
  ctx.fillText(spec.sub ?? '', W / 2, H * 0.62);
  ctx.restore();

  // scanlines
  ctx.globalAlpha = 1;
  ctx.fillStyle = 'rgba(0,0,0,0.06)';
  for (let y = 0; y < H; y += 4) ctx.fillRect(0, y, W, 1.2);
}

export class WorksMedia {
  constructor() {
    this.count = WORKS.length;
    this._works = WORKS.map((spec, i) => this._make(spec, i, 1024, 576));
    this._service = SERVICE.map((spec, i) => this._make(spec, 100 + i, 1024, 576));
  }

  _make(spec, seed, W, H) {
    const c = document.createElement('canvas');
    c.width = W;
    c.height = H;
    const ctx = c.getContext('2d');
    paintArt(ctx, W, H, spec, seed);
    const tex = new THREE.CanvasTexture(c);
    tex.colorSpace = THREE.SRGBColorSpace;
    tex.wrapS = tex.wrapT = THREE.MirroredRepeatWrapping;
    tex.anisotropy = 4;

    // tiny copy for the LED wall — linear magnification acts as the blur
    const small = document.createElement('canvas');
    small.width = 64;
    small.height = 36;
    small.getContext('2d').drawImage(c, 0, 0, 64, 36);
    const blurTex = new THREE.CanvasTexture(small);
    blurTex.colorSpace = THREE.SRGBColorSpace;
    blurTex.wrapS = blurTex.wrapT = THREE.MirroredRepeatWrapping;

    const entry = { canvas: c, ctx, tex, blurTex, spec, seed, aspect: W / H, hasImage: false };

    if (spec.img) {
      const im = new Image();
      im.crossOrigin = 'anonymous';
      im.onload = () => {
        // cover-fit: the screenshot and the panel are both 16:9, but do not
        // assume it — a mismatched capture should crop, never letterbox
        const s = Math.max(W / im.width, H / im.height);
        const dw = im.width * s;
        const dh = im.height * s;
        ctx.clearRect(0, 0, W, H);
        ctx.drawImage(im, (W - dw) / 2, (H - dh) / 2, dw, dh);
        // the generated art tiles happily; a screenshot flipped at the seam
        // is instantly readable as wrong, so clamp instead of mirror
        tex.wrapS = tex.wrapT = THREE.ClampToEdgeWrapping;
        tex.needsUpdate = true;
        small.getContext('2d').drawImage(c, 0, 0, 64, 36);
        blurTex.needsUpdate = true;
        entry.hasImage = true;
      };
      im.onerror = () => {
        console.warn('[px] works art failed to load, keeping generated art:', spec.img);
      };
      im.src = spec.img;
    }

    return entry;
  }

  workTexture(i) {
    return this._works[i]?.tex ?? null;
  }

  workBlurTexture(i) {
    return this._works[i]?.blurTex ?? null;
  }

  workAspect(i) {
    return this._works[i]?.aspect ?? 16 / 9;
  }

  /** true once a real screenshot has replaced the generated stand-in */
  workHasImage(i) {
    return this._works[i]?.hasImage === true;
  }

  serviceTexture(i) {
    return this._service[i]?.tex ?? null;
  }

  /** Animate the "video" reels: cheap repaint a few times a second. */
  update(t) {
    if (!this._last) this._last = 0;
    if (t - this._last < 0.12) return;    // ~8fps repaint is plenty
    this._last = t;
    for (const m of this._service) {
      // A panel carrying a real screenshot must not be repainted. This loop
      // redraws the generated art onto the same canvas the image was drawn
      // into, so without this guard the Cognexa reel loaded its image and then
      // had it painted over 8 times a second. `hasImage` was already set for
      // exactly this reason; nothing had ever needed to read it, because until
      // the Cognexa panel got a screenshot no SERVICE entry carried an image.
      if (m.hasImage) continue;
      paintArt(m.ctx, m.canvas.width, m.canvas.height, m.spec, m.seed, t);
      m.tex.needsUpdate = true;
    }
  }

  dispose() {
    for (const m of [...this._works, ...this._service]) {
      m.tex.dispose();
      m.blurTex?.dispose();
    }
  }
}
