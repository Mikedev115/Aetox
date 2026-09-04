import * as THREE from 'three';
import { buildThumbnailScreenGeometry } from './logo-geometry.js';
import { GLSL_CONSTANTS } from './shader-lib.js';

/**
 * The six works "screens": rounded 16:9 slabs flying a spiral carousel in
 * their own transparent scene, drawn over the composited main scene.
 * Per screen: 4-tap chromatic lens distortion, vignette, subtle env tint.
 * Layout (k = index + 1 - smoothed position):
 *   x = sin(k)*11, z = cos(k)*5 - 6, y = -k, rotY = k*0.6,
 *   scale = 0.9 + 0.2*(1 - min(1,|k|)),
 *   alpha = (1 - smoothstep(|k|, 0.8, 2.5)) * edge fades.
 */

const vert = /* glsl */ `
${GLSL_CONSTANTS}
uniform float uTexAspect;
varying vec2 vUv;
varying vec2 vMeshUv;
varying vec3 vNormal;
varying vec3 vViewDir;

void main() {
  vec3 pos = position;
  vec3 nrm = normal;
  // thin LED sheet, wrapped onto a vertical cylinder (arc-length preserving)
  // so the panel silhouette itself arcs — edges swing back, centre bows
  // toward the camera, and side-on panels read as curved displays
  pos.z *= 0.25;
  // R is measured, not chosen: on the live site a face-on panel's top edge bows
  // 1.14% of its own width, which is this radius at HALF_W. Arc-length
  // preserving, so the wrap costs the panel no apparent width.
  const float R = 21.1;
  const float HALF_W = 3.9731;
  float ang = pos.x / R;
  float ca = cos(ang), sa = sin(ang);
  float rr = R + pos.z;
  pos = vec3(sa * rr, pos.y, ca * rr - R);
  pos.z += R * (1.0 - cos(HALF_W / R)) * 0.55; // recentre the bow depth
  nrm = vec3(ca * nrm.x + sa * nrm.z, nrm.y, ca * nrm.z - sa * nrm.x);

  vec4 mv = modelViewMatrix * vec4(pos, 1.0);
  gl_Position = projectionMatrix * mv;

  vMeshUv = uv;
  // contain-fit the artwork into the 16:9 face
  const float MESH_ASPECT = 16.0 / 9.0;
  vec2 t = uv;
  if (uTexAspect > MESH_ASPECT) {
    t.y = (t.y - 0.5) * (uTexAspect / MESH_ASPECT) + 0.5;
  } else {
    t.x = (t.x - 0.5) / (uTexAspect / MESH_ASPECT) + 0.5;
  }
  vUv = t;
  vNormal = normalize(normalMatrix * nrm);
  vViewDir = normalize(-mv.xyz);
}
`;

const frag = /* glsl */ `
uniform sampler2D uTex;
uniform float uAlpha;
uniform float uLoaded;
uniform float uScrollVelocity;
uniform float uPhoto;      // 1 once a real screenshot replaced the generated art
uniform float uParallax;   // how far behind the front glass the content sits
varying vec2 vUv;
varying vec2 vMeshUv;
varying vec3 vNormal;
varying vec3 vViewDir;

vec2 lensWarp(vec2 r, float a) { return r * (1.0 - a * dot(r, r)); }

void main() {
  float facing = abs(dot(vNormal, vViewDir));
  vec2 normalOffset = -vNormal.xy * 0.5 * smoothstep(0.8, 1.0, 1.0 - facing);

  // --- depth -------------------------------------------------------------
  // Tangential view direction: which way we are looking ACROSS the panel face.
  // Zero head-on, and it grows as the carousel rotates the screen away.
  vec3 N = normalize(vNormal);
  vec3 V = normalize(vViewDir);
  vec2 across = (V - N * dot(V, N)).xy;

  vec2 cuv = (vUv - 0.5);
  // The abstract art is happy overscanned and mirrored at the seam; a
  // screenshot is not. Photo panels sample INSIDE the image (0.88) rather than
  // past its edge, which leaves real content for the parallax to slide into —
  // sampling past 1.0 would just smear the clamped edge pixels instead.
  cuv *= mix(1.3, 0.88, uPhoto);
  cuv.x *= mix(0.9, 1.0, uPhoto);
  cuv.x += uScrollVelocity * 0.15;

  // Content is behind the glass, so it slides against the view the way
  // anything seen through a window does.
  cuv -= across * uParallax * uPhoto;

  vec3 col = vec3(0.0);
  for (int i = 0; i < 4; i++) {
    float base = 0.1 + float(i) / 4.0 * 0.03;
    col.r += texture2D(uTex, lensWarp(cuv, base + 0.10) + 0.5 + normalOffset * 1.00).r;
    col.g += texture2D(uTex, lensWarp(cuv, base + 0.12) + 0.5 + normalOffset * 1.01).g;
    col.b += texture2D(uTex, lensWarp(cuv, base + 0.14) + 0.5 + normalOffset * 1.02).b;
  }
  col /= 4.0;

  col *= smoothstep(0.9, 0.49, length(vMeshUv - 0.5));
  col = mix(col, vec3(smoothstep(0.0, 0.2, vViewDir.x)), 0.05);

  // Recess wall. Off-axis you see the near inside edge of the housing, and it
  // falls on the side you are looking in from — this reads as depth far more
  // strongly than the parallax shift on its own does.
  vec2 e = vMeshUv - 0.5;
  vec2 q = abs(e + across * 0.42);
  float inner = smoothstep(0.5, 0.28, max(q.x, q.y));
  col *= mix(1.0, 0.58 + 0.42 * inner, uPhoto);

  // Front-surface sheen: a soft highlight that sweeps across the glass as the
  // panel turns, so the surface reads as being in front of the content.
  float sweep = dot(normalize(e + 1e-5), normalize(across + 1e-5));
  col += pow(1.0 - facing, 2.5) * smoothstep(0.15, 1.0, sweep) * 0.11 * uPhoto;

  // the slab was a rounded rect before it needed segmenting for the cylinder
  // bend; the box that replaced it has square corners, so mask the radius back
  const vec2 HALF = vec2(3.9731, 2.2349);
  const float CORNER = 0.05;
  vec2 d = abs((vMeshUv - 0.5) * vec2(7.9462, 4.4698)) - (HALF - CORNER);
  float sd = length(max(d, vec2(0.0))) + min(max(d.x, d.y), 0.0) - CORNER;

  gl_FragColor = vec4(col, uAlpha * uLoaded * smoothstep(0.012, -0.012, sd));
}
`;

export class WorksThumbs {
  constructor(media, { onPick } = {}) {
    this.media = media;
    this.count = media.count;
    this.group = new THREE.Group();
    this.group.position.set(0, 0.05, 0);
    // 0.6 is the portrait/mobile layout value — desktop runs full size so the
    // active screen dominates (~half the viewport) and neighbours clip at the edges
    this.group.scale.setScalar(1.0);
    this.onPick = onPick;

    const geo = buildThumbnailScreenGeometry();
    this.screens = [];
    for (let i = 0; i < this.count; i++) {
      const uniforms = {
        uTex: { value: media.workTexture(i) },
        uTexAspect: { value: media.workAspect(i) },
        uAlpha: { value: 0 },
        uLoaded: { value: 1 },
        uScrollVelocity: { value: 0 },
        uPhoto: { value: 0 },
        uParallax: { value: 0.055 }
      };
      const mesh = new THREE.Mesh(geo, new THREE.ShaderMaterial({
        vertexShader: vert,
        fragmentShader: frag,
        uniforms,
        transparent: true,
        side: THREE.FrontSide,
        depthWrite: false
      }));
      mesh.userData.workIndex = i;
      mesh.visible = false;
      this.group.add(mesh);
      this.screens.push(mesh);
    }
  }

  /** u: smoothed carousel position 0..count+1; vel: smoothed velocity. */
  update(u, vel) {
    const fadeIn = Math.min(1, u * 2);
    const fadeOut = Math.min(1, (this.count + 1 - u) * 2);
    for (let i = 0; i < this.count; i++) {
      const mesh = this.screens[i];
      const k = i + 1 - u;
      const ak = Math.abs(k);
      const alpha = (1 - THREE.MathUtils.smoothstep(ak, 0.8, 2.5)) * fadeIn * fadeOut;
      mesh.visible = alpha > 0.004;
      if (!mesh.visible) continue;
      mesh.position.set(Math.sin(k) * 11, -k, Math.cos(k) * 5 - 6);
      mesh.rotation.y = k * 0.6;
      mesh.scale.setScalar(0.9 + 0.2 * (1 - Math.min(1, ak)));
      mesh.material.uniforms.uAlpha.value = alpha;
      mesh.material.uniforms.uScrollVelocity.value = vel;
      // screenshots arrive async, so the depth treatment switches on when they do
      mesh.material.uniforms.uPhoto.value = this.media.workHasImage(i) ? 1 : 0;
    }
  }

  pickAt(raycaster) {
    const hits = raycaster.intersectObjects(this.screens.filter((s) => s.visible), false);
    return hits.length ? hits[0].object.userData.workIndex : -1;
  }

  dispose() {
    for (const s of this.screens) s.material.dispose();
    this.screens[0]?.geometry.dispose();
  }
}
