import * as THREE from 'three';

/**
 * The POSITION XERO "X" prism — same bounding proportions as the previous
 * mark (W 0.4333, H 0.3764, depth 0.106, rounded bevel) so every camera,
 * scale and scroll transform stays valid. A blocky letter X:
 *  - four arm tips landing on the bounding-box corners
 *  - notches at top/bottom/left/right where the inner arm edges cross
 */

const GLB_WIDTH = 0.4333;
const GLB_HEIGHT = 0.3764;
const GLB_DEPTH = 0.1060;

/**
 * The 16-point X outline (counter-clockwise), in local units.
 * tx / ty are the tip widths along the horizontal / vertical box edges.
 */
export function xOutlinePoints(W = GLB_WIDTH, H = GLB_HEIGHT) {
  const hw = W / 2;
  const hh = H / 2;
  const tx = 0.34 * W;
  const ty = 0.34 * H;

  // top notch: inner edges of the two upper arms cross at x = 0
  const sTop = (hw - tx) / (2 * hw - tx);
  const ncY = hh - sTop * (2 * hh - ty);
  // side notch: inner edges of right-hand arms cross at y = 0
  const sSide = (hh - ty) / (2 * hh - ty);
  const ncX = hw - sSide * (2 * hw - tx);

  return [
    [-hw, hh],          // TL tip outer
    [-hw + tx, hh],     // TL tip inner (top edge)
    [0, ncY],           // top notch
    [hw - tx, hh],      // TR tip inner
    [hw, hh],           // TR tip outer
    [hw, hh - ty],      // TR tip inner (side)
    [ncX, 0],           // right notch
    [hw, -hh + ty],     // BR tip inner (side)
    [hw, -hh],          // BR tip outer
    [hw - tx, -hh],     // BR tip inner (bottom)
    [0, -ncY],          // bottom notch
    [-hw + tx, -hh],    // BL tip inner (bottom)
    [-hw, -hh],         // BL tip outer
    [-hw, -hh + ty],    // BL tip inner (side)
    [-ncX, 0],          // left notch
    [-hw, hh - ty]      // TL tip inner (side)
  ];
}

/**
 * The X is built in TWO HALVES, sliced along the corner-to-corner diagonal
 * (top-left tip → bottom-right tip). Closed they read as one mark with a
 * faint diagonal seam; the vision→service transition pulls them apart and
 * light pours out of the cut.
 *
 * Upper half = ring points 0..8 (TR arm + upper bar), closed by the cut.
 * Lower half = ring points 8..15,0 (BL arm + lower bar).
 */
function extrudeHalf(points) {
  const shape = new THREE.Shape();
  shape.moveTo(points[0][0], points[0][1]);
  for (let i = 1; i < points.length; i++) shape.lineTo(points[i][0], points[i][1]);
  shape.closePath();

  const bevel = 0.019 * GLB_WIDTH;
  const geo = new THREE.ExtrudeGeometry(shape, {
    depth: GLB_DEPTH - bevel * 2,
    bevelEnabled: true,
    bevelThickness: bevel,
    bevelSize: bevel * 0.9,
    bevelSegments: 3,
    curveSegments: 8
  });

  // centre on z only — x/y must keep their absolute placement so the two
  // halves assemble into the full mark
  geo.computeBoundingBox();
  const bb = geo.boundingBox;
  geo.translate(0, 0, -(bb.min.z + bb.max.z) / 2);

  // UVs normalised over the FULL X box so the frost noise flows across the cut
  const hw = GLB_WIDTH / 2;
  const hh = GLB_HEIGHT / 2;
  const pos = geo.attributes.position;
  const uv = geo.attributes.uv;
  for (let i = 0; i < pos.count; i++) {
    uv.setXY(
      i,
      (pos.getX(i) + hw) / GLB_WIDTH,
      (pos.getY(i) + hh) / GLB_HEIGHT
    );
  }
  uv.needsUpdate = true;
  return geo;
}

export function buildLogoHalfGeometries() {
  const pts = xOutlinePoints();
  const upper = pts.slice(0, 9);                    // 0..8, cut closes 8→0
  const lower = [...pts.slice(8), pts[0]];          // 8..15 + 0, cut closes 0→8
  return {
    upper: extrudeHalf(upper),
    lower: extrudeHalf(lower)
  };
}

/** Unit vector perpendicular to the cut, pointing into the upper half. */
export function cutPerpendicular() {
  const hw = GLB_WIDTH / 2;
  const hh = GLB_HEIGHT / 2;
  const l = Math.hypot(hh, hw);
  return new THREE.Vector2(hh / l, hw / l);
}

/**
 * Clean hidden-line outlines for the light scene, one per half — front and
 * back loops (including the cut edge) plus corner connectors.
 */
function halfOutline(ring) {
  const z = GLB_DEPTH / 2;
  const pts = [];
  const loop = (zz) => {
    for (let i = 0; i < ring.length; i++) {
      const a = ring[i];
      const b = ring[(i + 1) % ring.length];
      pts.push(a[0], a[1], zz, b[0], b[1], zz);
    }
  };
  loop(z);
  loop(-z);
  for (const p of ring) {
    pts.push(p[0], p[1], z, p[0], p[1], -z);
  }
  const geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.BufferAttribute(new Float32Array(pts), 3));
  return geo;
}

export function buildLogoOutlineGeometries() {
  const pts = xOutlinePoints();
  return {
    upper: halfOutline(pts.slice(0, 9)),
    lower: halfOutline([...pts.slice(8), pts[0]])
  };
}

/**
 * The prism of light living inside the cut: a plane spanning the full
 * corner-to-corner diagonal and the prism depth (slightly oversized so it
 * blazes past the faces once the halves open). u runs across depth,
 * v runs along the diagonal (bottom-right → top-left).
 */
export function buildCutPlaneGeometry() {
  const hw = GLB_WIDTH / 2;
  const hh = GLB_HEIGHT / 2;
  const zHalf = (GLB_DEPTH / 2) * 1.35;
  const over = 1.15;                         // reach a touch past the tips

  const a = [hw * over, -hh * over];         // bottom-right tip
  const b = [-hw * over, hh * over];         // top-left tip

  const geo = new THREE.BufferGeometry();
  const verts = new Float32Array([
    a[0], a[1], -zHalf,
    a[0], a[1], zHalf,
    b[0], b[1], -zHalf,
    b[0], b[1], zHalf
  ]);
  const uvs = new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]);
  geo.setAttribute('position', new THREE.BufferAttribute(verts, 3));
  geo.setAttribute('uv', new THREE.BufferAttribute(uvs, 2));
  geo.setIndex([0, 1, 2, 2, 1, 3]);
  const p = cutPerpendicular();
  const normals = new Float32Array(12);
  for (let i = 0; i < 4; i++) {
    normals[i * 3] = p.x;
    normals[i * 3 + 1] = p.y;
  }
  geo.setAttribute('normal', new THREE.BufferAttribute(normals, 3));
  return geo;
}

/**
 * 16:9 slab for the works screens.
 *
 * Segmented across the face because works-thumbs.js bends it onto a cylinder in
 * the vertex shader: an ExtrudeGeometry face is triangulated from its outline
 * only, so a bend would move the corners and leave the span between them a
 * straight line — the panel would render flat.
 */
export function buildThumbnailScreenGeometry() {
  const W = 7.9462, H = 4.4698, D = 0.1947;
  const geo = new THREE.BoxGeometry(W, H, D, 56, 32, 1);
  geo.computeBoundingBox();
  const bb = geo.boundingBox;
  const pos = geo.attributes.position;
  const uv = geo.attributes.uv;
  for (let i = 0; i < pos.count; i++) {
    uv.setXY(
      i,
      (pos.getX(i) - bb.min.x) / (bb.max.x - bb.min.x),
      (pos.getY(i) - bb.min.y) / (bb.max.y - bb.min.y)
    );
  }
  uv.needsUpdate = true;
  return geo;
}
