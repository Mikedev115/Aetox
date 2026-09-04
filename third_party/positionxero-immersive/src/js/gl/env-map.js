import * as THREE from 'three';

/**
 * Procedural photo-studio environment cubemap.
 * Recreates the reflection content the glass logo needs: broad white
 * cyclorama sheets, a near-black ceiling/dark side, and gridded octagonal
 * softboxes that give the sharp cellular highlights. Fully desaturated.
 *
 * Face luminance targets (avg 0–1): px .75 / nx .14 / py .13 / ny .71 /
 * pz .50 / nz .38.
 */

const SIZE = 512;

function makeFace(draw) {
  const c = document.createElement('canvas');
  c.width = c.height = SIZE;
  const ctx = c.getContext('2d');
  draw(ctx);
  // faint photographic noise
  const img = ctx.getImageData(0, 0, SIZE, SIZE);
  for (let i = 0; i < img.data.length; i += 4) {
    const n = (Math.random() - 0.5) * 8;
    img.data[i] += n;
    img.data[i + 1] += n;
    img.data[i + 2] += n;
  }
  ctx.putImageData(img, 0, 0);
  return c;
}

function fillGrey(ctx, v) {
  ctx.fillStyle = `rgb(${v},${v},${v})`;
  ctx.fillRect(0, 0, SIZE, SIZE);
}

function verticalFalloff(ctx, top, bottom) {
  const g = ctx.createLinearGradient(0, 0, 0, SIZE);
  g.addColorStop(0, `rgb(${top},${top},${top})`);
  g.addColorStop(1, `rgb(${bottom},${bottom},${bottom})`);
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, SIZE, SIZE);
}

/** Octagonal gridded softbox: bright octagon + dark grid lines. */
function softbox(ctx, cx, cy, r, cells = 12) {
  ctx.save();
  ctx.translate(cx, cy);
  ctx.beginPath();
  for (let i = 0; i < 8; i++) {
    const a = (i / 8) * Math.PI * 2 + Math.PI / 8;
    const x = Math.cos(a) * r;
    const y = Math.sin(a) * r;
    i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
  }
  ctx.closePath();
  ctx.clip();
  // hot center falloff
  const g = ctx.createRadialGradient(0, 0, 0, 0, 0, r);
  g.addColorStop(0, 'rgb(255,255,255)');
  g.addColorStop(0.85, 'rgb(235,235,235)');
  g.addColorStop(1, 'rgb(200,200,200)');
  ctx.fillStyle = g;
  ctx.fillRect(-r, -r, r * 2, r * 2);
  // honeycomb-ish grid
  ctx.strokeStyle = 'rgba(40,40,40,0.55)';
  ctx.lineWidth = Math.max(1, (r * 2) / cells * 0.15);
  const step = (r * 2) / cells;
  for (let i = 0; i <= cells; i++) {
    const p = -r + i * step;
    ctx.beginPath(); ctx.moveTo(p, -r); ctx.lineTo(p, r); ctx.stroke();
    ctx.beginPath(); ctx.moveTo(-r, p); ctx.lineTo(r, p); ctx.stroke();
  }
  ctx.restore();
}

function bulb(ctx, cx, cy, r) {
  const g = ctx.createRadialGradient(cx, cy, 0, cx, cy, r);
  g.addColorStop(0, 'rgba(255,255,255,1)');
  g.addColorStop(0.4, 'rgba(240,240,240,0.9)');
  g.addColorStop(1, 'rgba(240,240,240,0)');
  ctx.fillStyle = g;
  ctx.beginPath();
  ctx.arc(cx, cy, r, 0, Math.PI * 2);
  ctx.fill();
}

/** Soft curved boundary between white cyclorama and dark studio half. */
function darkHalf(ctx, side /* 'left'|'right' */, dark = 30) {
  ctx.save();
  ctx.beginPath();
  const x0 = side === 'left' ? 0 : SIZE;
  const xm = SIZE / 2 + (side === 'left' ? -20 : 20);
  ctx.moveTo(x0, 0);
  ctx.lineTo(xm + (side === 'left' ? 40 : -40), 0);
  ctx.bezierCurveTo(
    xm - (side === 'left' ? 30 : -30), SIZE * 0.35,
    xm + (side === 'left' ? 20 : -20), SIZE * 0.7,
    xm, SIZE
  );
  ctx.lineTo(x0, SIZE);
  ctx.closePath();
  ctx.fillStyle = `rgb(${dark},${dark},${dark})`;
  ctx.fill();
  ctx.restore();
}

export function createStudioEnvMap() {
  const px = makeFace((ctx) => {
    verticalFalloff(ctx, 208, 172);           // bright cyclorama
    ctx.fillStyle = 'rgb(24,24,24)';           // ceiling truss corner
    ctx.beginPath();
    ctx.moveTo(SIZE, 0); ctx.lineTo(SIZE * 0.62, 0);
    ctx.lineTo(SIZE, SIZE * 0.3); ctx.closePath(); ctx.fill();
    bulb(ctx, SIZE * 0.94, SIZE * 0.48, 18);
    ctx.fillStyle = 'rgb(150,150,150)';        // grey pillar strip
    ctx.fillRect(SIZE * 0.965, 0, SIZE * 0.035, SIZE);
  });

  const nx = makeFace((ctx) => {
    fillGrey(ctx, 26);
    // faint clutter blocks
    ctx.fillStyle = 'rgb(45,45,45)';
    ctx.fillRect(SIZE * 0.1, SIZE * 0.45, SIZE * 0.1, SIZE * 0.4);
    ctx.fillStyle = 'rgb(60,60,60)';
    ctx.fillRect(SIZE * 0.3, SIZE * 0.55, SIZE * 0.14, SIZE * 0.3);
    // white floor wedge
    ctx.fillStyle = 'rgb(190,190,190)';
    ctx.beginPath();
    ctx.moveTo(SIZE * 0.35, SIZE); ctx.lineTo(SIZE, SIZE);
    ctx.lineTo(SIZE, SIZE * 0.86); ctx.closePath(); ctx.fill();
    // the big gridded softbox, top-right ~35%
    softbox(ctx, SIZE * 0.76, SIZE * 0.22, SIZE * 0.19, 12);
  });

  const py = makeFace((ctx) => {
    fillGrey(ctx, 32);
    // acoustic-panel weave
    ctx.strokeStyle = 'rgba(20,20,20,0.6)';
    ctx.lineWidth = 3;
    for (let i = -SIZE; i < SIZE * 2; i += 42) {
      ctx.beginPath(); ctx.moveTo(i, 0); ctx.lineTo(i + SIZE * 0.5, SIZE); ctx.stroke();
      ctx.beginPath(); ctx.moveTo(i, SIZE); ctx.lineTo(i + SIZE * 0.5, 0); ctx.stroke();
    }
    bulb(ctx, SIZE * 0.52, SIZE * 0.5, 6);
  });

  const ny = makeFace((ctx) => {
    verticalFalloff(ctx, 196, 176);            // white sweep floor
    ctx.fillStyle = 'rgb(28,28,28)';
    ctx.beginPath();
    ctx.moveTo(0, SIZE); ctx.lineTo(0, SIZE * 0.8);
    ctx.lineTo(SIZE * 0.18, SIZE); ctx.closePath(); ctx.fill();
  });

  const pz = makeFace((ctx) => {
    verticalFalloff(ctx, 205, 175);
    darkHalf(ctx, 'left', 28);
    softbox(ctx, SIZE * 0.07, SIZE * 0.3, SIZE * 0.13, 10);
    // free-standing white flat
    ctx.fillStyle = 'rgb(205,205,205)';
    ctx.fillRect(SIZE * 0.38, SIZE * 0.3, SIZE * 0.1, SIZE * 0.42);
    bulb(ctx, SIZE * 0.27, SIZE * 0.42, 12);
  });

  const nz = makeFace((ctx) => {
    fillGrey(ctx, 40);
    // white cyclorama third on the left
    const g = ctx.createLinearGradient(0, 0, SIZE * 0.4, 0);
    g.addColorStop(0, 'rgb(200,200,200)');
    g.addColorStop(1, 'rgb(160,160,160)');
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, SIZE * 0.36, SIZE);
    // white floor bottom
    ctx.fillStyle = 'rgb(185,185,185)';
    ctx.fillRect(0, SIZE * 0.82, SIZE, SIZE * 0.18);
    // dark lounge blocks right
    ctx.fillStyle = 'rgb(22,22,22)';
    ctx.fillRect(SIZE * 0.62, SIZE * 0.55, SIZE * 0.2, SIZE * 0.3);
    ctx.fillStyle = 'rgb(58,40,34)';
    ctx.fillRect(SIZE * 0.84, SIZE * 0.6, SIZE * 0.13, SIZE * 0.25);
    // hero softbox top-centre
    softbox(ctx, SIZE * 0.52, SIZE * 0.16, SIZE * 0.21, 12);
  });

  const tex = new THREE.CubeTexture([px, nx, py, ny, pz, nz]);
  tex.needsUpdate = true;
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}
