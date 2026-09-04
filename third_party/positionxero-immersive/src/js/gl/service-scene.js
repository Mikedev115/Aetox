import * as THREE from 'three';
import { GLSL_CONSTANTS, GLSL_ROTATE2 } from './shader-lib.js';
import { GridOverlay } from './grid-overlay.js';

/**
 * The service scene: three reel panels riding a sine-wave wall that
 * un-curls as the section enters, a ghost "SERVICES" title band scrolling
 * behind them with a dim projection of the current reel, and the third
 * panel (cognexa) peeling off the wall into a fullscreen cover-fit quad.
 */

const panelVert = /* glsl */ `
${GLSL_CONSTANTS}
${GLSL_ROTATE2}
uniform float uInstanceId;
uniform float uThumbnailScroll;  // 0.5..4 smoothed
uniform float uServiceIn;        // wall un-curl
uniform float uCognexaIn;        // cognexa section entering
uniform float uCognexaView;      // 0..1 peel to fullscreen
uniform vec3 uScale;             // frustum extents at z=0
uniform float uCanvasAspect;
uniform float uVideoAspect;
varying vec2 vUv;
varying vec2 vLocal;

vec2 coverUv(vec2 uv, float canvasA, float videoA) {
  vec2 t = uv - 0.5;
  if (canvasA > videoA) {
    t.y *= videoA / canvasA;
  } else {
    t.x *= canvasA / videoA;
  }
  return t + 0.5;
}

void main() {
  float x = uInstanceId - (uThumbnailScroll - 1.0);

  // Geometry is a unit quad; the reel's 16:9 is applied here, once. uScale is
  // isotropic (see resize), so x and y take the same world scale.
  vec2 local = position.xy * 0.3;
  local.x *= (16.0 / 9.0) * 0.85;
  local.x += x * 0.95;
  local.x += -0.05 * (1.0 - uCognexaIn);
#ifndef IS_COGNEXA
  local.x -= uCognexaView * 0.5;
#endif

  // one panel spans a full sine period, so neighbours sit a phase apart
  float theta = local.x * PI + (1.0 - uServiceIn) * 8.0 - uCognexaIn * 2.0;
  float wave = 4.0 * uServiceIn * (1.0 - uCognexaIn);

  vec3 pos = vec3(local.x * uScale.x, local.y * uScale.y, -sin(theta) * wave);
  vec4 clip = projectionMatrix * modelViewMatrix * vec4(pos, 1.0);

  vUv = uv;
  vLocal = uv - 0.5;

#ifdef IS_COGNEXA
  // left-to-right per-vertex peel into a screen-covering quad
  float curve = 0.3;
  float w = smoothstep(0.0, 1.0, -uv.x * curve + uCognexaView * (1.0 + curve));
  clip = mix(clip, vec4(position.xy * 2.0, 0.0, 1.0), w);
  vUv = mix(vUv, coverUv(uv, uCanvasAspect, uVideoAspect), w);
#endif

  gl_Position = clip;
}
`;

const panelFrag = /* glsl */ `
uniform sampler2D uTex;
uniform float uServiceIn;
varying vec2 vUv;
varying vec2 vLocal;
void main() {
  vec3 col = texture2D(uTex, vUv).rgb;
#ifndef IS_COGNEXA
  col *= smoothstep(1.0, 0.3, length(vLocal));
#endif
  col *= 0.7;
  gl_FragColor = vec4(col, smoothstep(0.5, 1.0, uServiceIn));
}
`;

const titleVert = /* glsl */ `
${GLSL_CONSTANTS}
uniform vec3 uScale;
uniform float uServiceIn;
uniform float uCognexaIn;
varying vec2 vUv;
void main() {
  // the ghost wall rides the same sine bend as the panels, 1.4x wider and 1.7x
  // taller than a unit quad, which is what makes it full-bleed at every phase —
  // a flat quad at a fixed z left a black frame on all four sides.
  vec2 local = vec2(position.x * 1.4, position.y);
  float theta = local.x * PI + (1.0 - uServiceIn) * 8.0 - uCognexaIn * 2.0;
  float wave = 4.0 * uServiceIn * (1.0 - uCognexaIn);
  vec3 pos = vec3(local.x * uScale.x, local.y * uScale.y * 1.7, -sin(theta) * wave);
  gl_Position = projectionMatrix * modelViewMatrix * vec4(pos, 1.0);
  vUv = uv;
}
`;

const titleFrag = /* glsl */ `
uniform sampler2D uTitleTex;
uniform sampler2D uVideoTex;
uniform float uTime;
uniform float uThumbnailScroll;
uniform float uServiceIn;
uniform float uCognexaIn;
varying vec2 vUv;
void main() {
  vec2 t = vUv;
  t.y *= 4.5;   // many small letter rows, not a few huge ones
  t.x *= 1.1;
  t.x += uThumbnailScroll * 0.2 + uTime * 0.015;
  t.x = fract(t.x) * 1.1;
  t.y = fract(t.y);
  // the band is an ENTRANCE effect: it burns across the wall while the tunnel
  // un-curls and is gone by the time a card settles, which is why the live
  // service section reads as near-black behind the reel.
  float glyph = texture2D(uTitleTex, t).a * mix(0.8, 0.0, uServiceIn);

  // blurred echo of the active reel, widened 1/0.7 and ramped to black rightward
  vec2 texUv = vec2((vUv.x - 0.5) * 0.7 + 0.5, vUv.y);
  vec3 col = texture2D(uVideoTex, texUv).rgb * (1.0 - vUv.x) * 0.4;
  col += glyph;
  // the ghost wall clears out as cognexa takes over
  col *= 1.0 - uCognexaIn * 0.92;
  gl_FragColor = vec4(col, 1.0);
}
`;

export class ServiceScene {
  constructor(media, { servicesTitleTexture }) {
    this.media = media;
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color('#000000');
    this.camera = new THREE.PerspectiveCamera(50, 1, 0.1, 100);
    this.camera.position.set(0, 0, 10);

    this.uniformsShared = {
      uThumbnailScroll: { value: 0.5 },
      uServiceIn: { value: 0 },
      uCognexaIn: { value: 0 },
      uCognexaView: { value: 0 },
      uScale: { value: new THREE.Vector3(14.8, 8.3, 1) }
    };

    this.panels = [];
    for (let i = 0; i < 3; i++) {
      const isCognexa = i === 2;
      const uniforms = {
        // 0-based: reel i sits centred when uThumbnailScroll = i + 1
        uInstanceId: { value: i },
        uTex: { value: media.serviceTexture(i) },
        uCanvasAspect: { value: 16 / 9 },
        uVideoAspect: { value: 16 / 9 },
        ...this.uniformsShared
      };
      const mesh = new THREE.Mesh(
        // unit quad: the peel target below is `position.xy * 2.0`, which only
        // lands on the +/-1 NDC box when position is +/-0.5
        new THREE.PlaneGeometry(1, 1, 16, 8),
        new THREE.ShaderMaterial({
          vertexShader: panelVert,
          fragmentShader: panelFrag,
          uniforms,
          transparent: true,
          side: THREE.DoubleSide,
          defines: isCognexa ? { IS_COGNEXA: 1 } : {},
          depthWrite: false
        })
      );
      mesh.renderOrder = isCognexa ? 5 : 2;
      mesh.frustumCulled = false;
      this.scene.add(mesh);
      this.panels.push(mesh);
    }

    // ghost SERVICES band + reel projection behind the panels
    this.titleUniforms = {
      uTitleTex: { value: servicesTitleTexture },
      uVideoTex: { value: media.serviceTexture(0) },
      uTime: { value: 0 },
      uThumbnailScroll: this.uniformsShared.uThumbnailScroll,
      uServiceIn: this.uniformsShared.uServiceIn,
      uCognexaIn: this.uniformsShared.uCognexaIn,
      uScale: this.uniformsShared.uScale
    };
    this.titleMesh = new THREE.Mesh(
      // segmented across x so the sine bend in titleVert has vertices to act on
      new THREE.PlaneGeometry(1, 1, 64, 1),
      new THREE.ShaderMaterial({
        vertexShader: titleVert,
        fragmentShader: titleFrag,
        uniforms: this.titleUniforms,
        depthWrite: false
      })
    );
    this.titleMesh.renderOrder = 0;
    this.titleMesh.frustumCulled = false;
    this.scene.add(this.titleMesh);

    this.grid = new GridOverlay({ dark: false, cylinder: true });
    this.grid.uniforms.uGrid.value.set(75, 50);
    this.scene.add(this.grid.group);
  }

  resize(w, h) {
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
    const fh = 2 * Math.abs(this.camera.position.z) * Math.tan(THREE.MathUtils.degToRad(this.camera.fov) / 2);
    const fw = fh * this.camera.aspect;
    // isotropic: the shaders apply each mesh's own aspect themselves, so feeding
    // a non-square uScale would apply the viewport's aspect a second time
    const s = Math.max(fw, fh);
    this.uniformsShared.uScale.value.set(s, s, 1);
    this.grid.resize(this.camera);
    for (const p of this.panels) {
      p.material.uniforms.uCanvasAspect.value = w / h;
    }
  }

  /**
   * @param {number} thumbScroll 0.5..4 smoothed reel position
   * @param {number} serviceIn   0..1 wall un-curl
   * @param {number} cognexaIn   0..1 cognexa enter
   * @param {number} cognexaView 0..1 fullscreen peel
   */
  update(t, thumbScroll, serviceIn, cognexaIn, cognexaView) {
    this.uniformsShared.uThumbnailScroll.value = Math.min(3, thumbScroll);
    this.uniformsShared.uServiceIn.value = serviceIn;
    this.uniformsShared.uCognexaIn.value = cognexaIn;
    this.uniformsShared.uCognexaView.value = cognexaView;
    this.titleUniforms.uTime.value = t;
    const reel = Math.max(0, Math.min(2, Math.round(thumbScroll - 1)));
    this.titleUniforms.uVideoTex.value = this.media.serviceTexture(reel);
    this.grid.setScroll(t * 0.01 + thumbScroll * 0.3);
  }

  dispose() {
    for (const p of this.panels) {
      p.geometry.dispose();
      p.material.dispose();
    }
    this.titleMesh.geometry.dispose();
    this.titleMesh.material.dispose();
    this.grid.dispose();
  }
}
