import * as THREE from 'three';
import { ProceduralTexture } from './procedural-texture.js';
import { GLSL_CONSTANTS, GLSL_HASH, GLSL_ROTATE2, GLSL_HSV } from './shader-lib.js';

/**
 * The curved LED wall: an instanced quadtree of thin box tiles bent onto a
 * half-cylinder wrapping the camera. Three procedural patterns cycle with
 * hard cuts; tiles glitch (blackouts, UV shifts), the XERO wordmark tiles
 * over the wall, the WORKS band burns in during the works intro, and
 * per-work art (blurred + oversaturated) wipes across during the works
 * section.
 */

const WORKS_NUM = 4;

/* ---------------- patterns ---------------- */

const patternClouds = /* glsl */ `
uniform sampler2D uNoiseTex;
varying vec2 vUv;
void main() {
  vec4 n = texture2D(uNoiseTex, vUv);
  vec3 o = vec3(
    smoothstep(0.5, 1.0, n.x),
    smoothstep(0.2, 1.0, n.y),
    smoothstep(0.0, 1.0, n.z)
  );
  o *= vec3(0.10, 0.66, 0.90);
  gl_FragColor = vec4(o, 1.0);
}
`;

const patternZebra = /* glsl */ `
uniform sampler2D uNoiseTex;
varying vec2 vUv;
void main() {
  vec4 n = texture2D(uNoiseTex, vUv);
  float bands = step(0.5, fract(n.w * 9.0));
  gl_FragColor = vec4(vec3(bands), 1.0);
}
`;

const patternColour = /* glsl */ `
uniform sampler2D uNoiseTex;
varying vec2 vUv;
void main() {
  vec4 n = texture2D(uNoiseTex, vUv);
  vec3 c = vec3(0.020, 0.027, 0.039);                                  // ink ground (--bg)
  c = mix(c, vec3(0.180, 0.902, 1.000), smoothstep(0.42, 0.92, n.w));  // signal cyan (--signal)
  c = mix(c, vec3(1.000, 0.416, 0.122), smoothstep(0.82, 1.0, n.y));   // conversion amber (--heat), sparse
  gl_FragColor = vec4(c, 1.0);
}
`;

/* ---------------- wall shaders ---------------- */

const wallVert = /* glsl */ `
${GLSL_CONSTANTS}
${GLSL_HASH}
${GLSL_ROTATE2}
// atlas cell is 512x176. 6 cells across fills the grid without shrinking the
// word past reading size (~65 words on a 16:9 screen); at 4 it read as a few
// big fragments cropped by the frame rather than a field.
#define WORKS_CELL_ASPECT 2.909
#define WORKS_COLS_ON_SCREEN 6.0
attribute vec3 instancePosition;
attribute vec2 instanceScale;
attribute vec4 instanceID;
attribute float instanceDepth;
uniform vec3 uScale;
uniform sampler2D uPointerTex;
uniform float uUVShift;
uniform float uUVShiftPower;
uniform float uUVShiftHash;
uniform float uBlackOut;
uniform float uBlackOutHash;
uniform float uPatternSelect;
uniform float uScroll;          // raw works progress
uniform float uScrollPage;      // fract of smoothed works page
uniform float uWorks1Aspect;
uniform float uWorks2Aspect;
uniform float uScreenAspectRatio;
uniform float uWorksTitleProgress;

varying vec2 vUv;
varying vec2 vGlobalUv;
varying vec2 vScreenUv;
varying vec2 vWorksUv1;
varying vec2 vWorksUv2;
varying vec2 vWorksTitleUv;
varying float vDisplayWorks;
varying float vBlackOut;
varying float vPatternSelect;
varying float vSideFace;
varying float vEmitSide;
varying vec4 vInstanceID;

void main() {
  vec3 pos = position;

  // seams: shrink tiles by depth so black shows through between panels
  pos.xy *= 1.0 - 0.003 * pow(2.0, instanceDepth);
  pos.xy *= instanceScale;
  vec3 flat_ = pos;

  // where is this tile on screen? (used for pointer wake + works sweep)
  vec4 offMv = modelViewMatrix * vec4(instancePosition * uScale, 1.0);
  vec4 offClip = projectionMatrix * offMv;
  offClip.xyz /= offClip.w;
  vec2 tileScreenUv = offClip.xy * 0.5 + 0.5;
  vec4 pointer = texture2D(uPointerTex, tileScreenUv);

  // works art sweeps in from the left as the section scrolls
  float sweep = smoothstep(0.0, 0.3, -tileScreenUv.x + uScroll * float(${WORKS_NUM}) * 1.6);
  float fadeOut = smoothstep(1.0, 0.9, uScroll);
  vDisplayWorks = sweep * fadeOut;

  vPatternSelect = step(0.5, uPatternSelect);

  // bend the flat -0.5..0.5 layout onto a half cylinder
  pos += instancePosition;
  vec3 flatWorld = flat_ + instancePosition;
  flatWorld.x *= uScale.x;
  flatWorld.y *= uScale.y;

  float theta = pos.x * PI;
  vec3 bent = pos;
  bent.x = 0.0;
  bent.xz *= rot2(HPI);
  bent.z -= uScale.x / 2.0;
  bent.y *= uScale.y * 1.5;
  bent.xz *= rot2(-theta);

  vec4 mv = modelViewMatrix * vec4(bent, 1.0);
  gl_Position = projectionMatrix * mv;

  vSideFace = abs(normal.z);
  vEmitSide = length(pointer.xy);
  vUv = uv;
  vGlobalUv = instancePosition.xy + 0.5 + (uv - 0.5) * instanceScale;
  vInstanceID = instanceID;

  // screen-space uv of the *unbent* layout — patterns read as flat images
  vec4 suvMv = modelViewMatrix * vec4(flatWorld * 0.5, 1.0);
  vec4 suvClip = projectionMatrix * suvMv;
  vScreenUv = suvClip.xy / suvClip.w * 0.5 + 0.5;

  // per-tile uv-shift glitch (quantised hash states)
  float shiftHash = floor(uUVShiftHash * 5.0) / 5.0;
  vec2 us = vec2(
    hash21(instanceID.xy + shiftHash),
    hash21(instanceID.xy + shiftHash + 10.0)
  ) - 0.5;
  us *= uUVShiftPower * step(hash21(instanceID.xy + shiftHash), uUVShift);
  vScreenUv += us;

  float boHash = floor(uBlackOutHash * 5.0) / 5.0;
  vBlackOut = step(hash21(instanceID.xy + boHash), uBlackOut);

  // WORKS field: the 4x4 atlas tiled across the whole wall, so the word reads
  // everywhere in the background instead of one band sweeping past.
  //
  // Aspect has to be unwound before the rotation, or the non-uniform scale
  // after it shears the letterforms. Work in units of screen height, rotate
  // there, then divide by the on-screen size of one whole atlas.
  vec2 wt = vScreenUv - 0.5;
  wt.x *= uScreenAspectRatio;
  wt *= rot2(-0.05);
  float cellW = uScreenAspectRatio / WORKS_COLS_ON_SCREEN;
  float cellH = cellW / WORKS_CELL_ASPECT;
  wt /= vec2(cellW * 4.0, cellH * 4.0);   // 4x4 cells per atlas
  wt.x += uWorksTitleProgress * 0.12;     // gentle drift, not a spin
  vWorksTitleUv = wt + 0.5;

  // adjacent works art slide (outgoing left, incoming from the right)
  vec2 wShift = vec2(0.0, hash21(instanceID.yz) - 0.5);
  float slide = 0.5;
  vWorksUv1 = vScreenUv - us - wShift * pow(uScrollPage, 2.0);
  vWorksUv1.x -= uScrollPage * slide;
  vWorksUv2 = vScreenUv - us - wShift * pow(1.0 - uScrollPage, 2.0);
  vWorksUv2.x += (1.0 - uScrollPage) * slide;

  // cover-fit both works uvs
  if (uScreenAspectRatio < uWorks1Aspect) {
    vWorksUv1.x = (vWorksUv1.x - 0.5) * (uScreenAspectRatio / uWorks1Aspect) + 0.5;
  } else {
    vWorksUv1.y = (vWorksUv1.y - 0.5) / (uScreenAspectRatio / uWorks1Aspect) + 0.5;
  }
  if (uScreenAspectRatio < uWorks2Aspect) {
    vWorksUv2.x = (vWorksUv2.x - 0.5) * (uScreenAspectRatio / uWorks2Aspect) + 0.5;
  } else {
    vWorksUv2.y = (vWorksUv2.y - 0.5) / (uScreenAspectRatio / uWorks2Aspect) + 0.5;
  }
}
`;

const wallFrag = /* glsl */ `
${GLSL_CONSTANTS}
${GLSL_HASH}
${GLSL_HSV}
uniform float uTime;
uniform vec2 uScreenResolution;
uniform sampler2D uLogoTex;
uniform sampler2D uPatternCurrent;
uniform sampler2D uPatternNext;
uniform float uLogoDisplayType;
uniform float uScrollPage;
uniform sampler2D uWorks1Tex;
uniform float uWorks1Loaded;
uniform sampler2D uWorks2Tex;
uniform float uWorks2Loaded;
uniform sampler2D uWorksTitleTex;
uniform float uWorksTitleProgress;

varying vec2 vUv;
varying vec2 vGlobalUv;
varying vec2 vScreenUv;
varying vec2 vWorksUv1;
varying vec2 vWorksUv2;
varying vec2 vWorksTitleUv;
varying float vDisplayWorks;
varying float vBlackOut;
varying float vPatternSelect;
varying float vSideFace;
varying float vEmitSide;
varying vec4 vInstanceID;

void main() {
  vec4 o = vec4(0.0, 0.0, 0.0, 1.0);
  vec2 fragUv = gl_FragCoord.xy / uScreenResolution;

  float bandVis = smoothstep(0.1, 0.3, uWorksTitleProgress) * smoothstep(1.0, 0.9, uWorksTitleProgress);

  vec3 pat1 = texture2D(uPatternCurrent, vScreenUv).rgb;
  vec3 pat2 = texture2D(uPatternNext, vScreenUv).rgb;
  o.rgb = mix(pat1, pat2, vPatternSelect);
  o.rgb *= 1.0 - bandVis * 0.5;
  o.rgb *= 1.0 - vBlackOut;

  // XERO wordmark tiling
  vec2 logoUv = vScreenUv;
  if (uLogoDisplayType < 0.5) {
    logoUv = vGlobalUv - 0.5;
    logoUv.y *= 1204.0 / 250.0;
    logoUv += 0.5;
    vec2 tile = logoUv * 2.0;
    tile.x += sin(floor(tile.y) * 3.0 + uTime) * 0.1;
    logoUv = fract(tile) * 1.3;
  } else if (uLogoDisplayType < 1.5) {
    logoUv = vGlobalUv - 0.5;
    logoUv.y *= 1204.0 / 250.0;
    logoUv *= 1.1;
    logoUv += 0.5;
    vec2 tile = logoUv;
    tile.x += uTime * 0.05 * sign(floor(tile.y));
    logoUv = fract(tile);
    if (abs(floor(tile.y)) < 0.5) logoUv = vec2(0.0);
  } else {
    logoUv = vUv - 0.5;
    logoUv.x /= 1204.0 / 250.0;
    logoUv += 0.5;
    logoUv.y -= uTime * 0.5 * vInstanceID.x;
    logoUv.y = fract(logoUv.y);
    logoUv = (logoUv - 0.5) * (1.3 + vInstanceID.z * 5.0) + 0.5;
    logoUv.x -= 0.38;
    if (logoUv.x > 0.23 || logoUv.x < 0.0 || logoUv.y > 1.0 || logoUv.y < 0.0 || vInstanceID.y < 0.0) {
      logoUv = vec2(0.0);
    }
  }
  vec4 logo = texture2D(uLogoTex, logoUv);
  float logoMask = step(0.5, logo.a) * step(0.0, logoUv.y) * step(logoUv.y, 1.0);
  o.rgb += logoMask * 0.2 * (1.0 - bandVis);

  // WORKS band burned in
  // alpha carries the per-cell weighting painted into the atlas
  vec3 band = vec3(texture2D(uWorksTitleTex, vWorksTitleUv).a) * bandVis;

  // works art wipe between adjacent works
  vec3 t1 = texture2D(uWorks1Tex, vWorksUv1).rgb * uWorks1Loaded;
  vec3 t2 = texture2D(uWorks2Tex, vWorksUv2).rgb * uWorks2Loaded;
  float blurW = 0.03;
  vec3 workCol = mix(t1, t2, smoothstep(fragUv.x - blurW, fragUv.x + blurW, uScrollPage * (1.0 + blurW * 2.0) - blurW));
  workCol *= 0.85;  // the wall carries a glow of the art, not a flat wash
  workCol *= mix(1.0, hash21(gl_FragCoord.xy / 1000.0), 0.1);
  vec3 hsv = rgb2hsv(workCol);
  workCol = hsv2rgb(vec3(hsv.x, hsv.y * 2.0, hsv.z));
  o.rgb = mix(o.rgb, workCol, vDisplayWorks);
  o.a = min(o.a + vDisplayWorks, 1.0);

  // per-tile falloff, band overlay, dot matrix, wall vignette
  o.rgb *= smoothstep(1.9, 0.1, length(vUv - 0.5));
  // Background, not foreground: at 0.62 the field competed with the hero copy
  // and washed the refraction out of the glass mark in front of it.
  o.rgb += band * 0.45;
  float dotw = smoothstep(0.5, 0.2, length(fract(vGlobalUv * 1800.0 * 0.23) - 0.5));
  dotw = mix(dotw, 1.0, 0.6) * 0.9;
  o.rgb *= dotw;
  o.rgb *= smoothstep(0.55, 0.05, length(vGlobalUv - 0.5));

  // front faces show the pattern; box sides glow near the pointer,
  // and the whole tile patch brightens inside the cursor wake
  o.rgb *= vSideFace;
  o.rgb *= 1.0 + min(vEmitSide * 2.2, 1.4);
  o.rgb += (1.0 - vSideFace) * 0.9 * vEmitSide * (0.08 + vDisplayWorks);

  o.rgb *= 0.5;
  gl_FragColor = o;
}
`;

/* ---------------- quadtree geometry ---------------- */

function createQuadTreeGeometry() {
  const base = new THREE.BoxGeometry(1, 1, 0.005, 2, 2, 1);
  const geo = new THREE.InstancedBufferGeometry();
  geo.index = base.index;
  geo.attributes.position = base.attributes.position;
  geo.attributes.normal = base.attributes.normal;
  geo.attributes.uv = base.attributes.uv;

  const positions = [];
  const scales = [];
  const ids = [];
  const depths = [];

  const emit = (x, y, scale, depth) => {
    positions.push(x, y, 0);
    scales.push(scale, scale);
    ids.push(Math.random(), Math.random(), Math.random(), Math.random());
    depths.push(depth);
  };

  const subdivide = (x, y, size, depth) => {
    if (depth >= 4 || (depth === 3 && Math.random() < 0.5)) {
      emit(x, y, size, depth);
      return;
    }
    const q = size / 4;
    subdivide(x - q, y - q, size / 2, depth + 1);
    subdivide(x + q, y - q, size / 2, depth + 1);
    subdivide(x - q, y + q, size / 2, depth + 1);
    subdivide(x + q, y + q, size / 2, depth + 1);
  };

  // start below leaf size so leaves land at 1/8 or 1/16
  subdivide(0, 0, 1, 1);

  geo.setAttribute('instancePosition', new THREE.InstancedBufferAttribute(new Float32Array(positions), 3));
  geo.setAttribute('instanceScale', new THREE.InstancedBufferAttribute(new Float32Array(scales), 2));
  geo.setAttribute('instanceID', new THREE.InstancedBufferAttribute(new Float32Array(ids), 4));
  geo.setAttribute('instanceDepth', new THREE.InstancedBufferAttribute(new Float32Array(depths), 1));
  geo.instanceCount = depths.length;
  return geo;
}

/* ---------------- the wall ---------------- */

export class BGWall {
  constructor(renderer, { noiseTexture, logoTexture, worksTitleTexture, pointerTexture, media }) {
    this.renderer = renderer;
    this.media = media;

    const mkPattern = (frag, ratio) => {
      const p = new ProceduralTexture(renderer, {
        width: 256,
        height: 256,
        fragmentShader: frag,
        uniforms: { uNoiseTex: { value: noiseTexture } },
        wrap: THREE.RepeatWrapping
      });
      p.ratio = ratio;
      return p;
    };
    this.patterns = [
      mkPattern(patternClouds, 0.3),
      mkPattern(patternZebra, 0.8),
      mkPattern(patternColour, 0.8)
    ];
    this._patternParams = [
      { blackOut: 0, blackOutRandom: 0, uvShift: 0 },
      { blackOut: 0, blackOutRandom: 0.5, uvShift: 0.5 },
      { blackOut: 0, blackOutRandom: 0.5, uvShift: 0.5 }
    ];
    this._current = 2;
    this._next = 1;

    this.uniforms = {
      uTime: { value: 0 },
      uScale: { value: new THREE.Vector3(14.8, 14.8, 1) },
      uScreenResolution: { value: new THREE.Vector2(1, 1) },
      uScreenAspectRatio: { value: 16 / 9 },
      uPointerTex: { value: pointerTexture },
      uPatternCurrent: { value: this.patterns[this._current].texture },
      uPatternNext: { value: this.patterns[this._next].texture },
      uPatternSelect: { value: 0 },
      uUVShift: { value: 0 },
      uUVShiftPower: { value: 0 },
      uUVShiftHash: { value: 0 },
      uBlackOut: { value: 0 },
      uBlackOutHash: { value: 0 },
      uLogoTex: { value: logoTexture },
      uLogoDisplayType: { value: 0 },
      uWorksTitleTex: { value: worksTitleTexture },
      uWorksTitleProgress: { value: 0 },
      uScroll: { value: 0 },
      uScrollPage: { value: 0 },
      uWorks1Tex: { value: media.workTexture(0) },
      uWorks2Tex: { value: media.workTexture(0) },
      uWorks1Loaded: { value: 0 },
      uWorks2Loaded: { value: 0 },
      uWorks1Aspect: { value: 16 / 9 },
      uWorks2Aspect: { value: 16 / 9 }
    };

    this.mesh = new THREE.Mesh(createQuadTreeGeometry(), new THREE.ShaderMaterial({
      vertexShader: wallVert,
      fragmentShader: wallFrag,
      uniforms: this.uniforms,
      side: THREE.DoubleSide,
      transparent: true
    }));
    this.mesh.frustumCulled = false;
    this.mesh.renderOrder = -1000;

    // glitch scheduling
    this._nextPatternAt = 1.5;
    this._nextBlackoutAt = 1;
    this._nextUvShiftAt = 0.5;
    this._uvShiftAnim = { from: 0, to: 0, t0: 0, dur: 0.8 };
    this._hashAnimBO = { from: 0, to: 0, t0: 0, dur: 0.3 };
    this._hashAnimUV = { from: 0, to: 0, t0: 0, dur: 0.3 };
  }

  resize(width, height, camera) {
    const h = 2 * Math.abs(camera.position.z) * Math.tan(THREE.MathUtils.degToRad(camera.fov) / 2);
    const w = Math.max(h, h * camera.aspect);
    this.uniforms.uScale.value.set(w, w, 1);
    this.uniforms.uScreenResolution.value.set(width, height);
    this.uniforms.uScreenAspectRatio.value = width / height;
    const base = 0.5;
    for (const p of this.patterns) {
      p.setSize(width * base * p.ratio, height * base * p.ratio);
    }
  }

  setWorksProgress(raw, pageU) {
    this.uniforms.uScroll.value = raw;
    const page = Math.floor(pageU);
    const frac = pageU - page;
    this.uniforms.uScrollPage.value = frac;
    const i1 = Math.min(this.media.count - 1, Math.max(0, page - 1));
    const i2 = Math.min(this.media.count - 1, Math.max(0, page));
    this.uniforms.uWorks1Tex.value = this.media.workBlurTexture(i1);
    this.uniforms.uWorks2Tex.value = this.media.workBlurTexture(i2);
    this.uniforms.uWorks1Loaded.value = page >= 1 ? 1 : 0;
    this.uniforms.uWorks2Loaded.value = pageU >= 0.001 ? 1 : 0;
    this.uniforms.uWorks1Aspect.value = this.media.workAspect(i1);
    this.uniforms.uWorks2Aspect.value = this.media.workAspect(i2);
  }

  setWorksTitleProgress(p) {
    this.uniforms.uWorksTitleProgress.value = p;
  }

  update(t) {
    this.uniforms.uTime.value = t;

    // only the two active patterns re-render
    this.patterns[this._current].render(t);
    this.patterns[this._next].render(t);

    if (t >= this._nextPatternAt) {
      this._current = this._next;
      let n = Math.floor(Math.random() * this.patterns.length);
      if (n === this._current) n = (n + 1) % this.patterns.length;
      this._next = n;
      this.uniforms.uPatternCurrent.value = this.patterns[this._current].texture;
      this.uniforms.uPatternNext.value = this.patterns[this._next].texture;
      // hard cut: uPatternSelect snaps 0 → 1 (we simply swap + reset)
      this.uniforms.uPatternSelect.value = 0;
      setTimeout(() => { this.uniforms.uPatternSelect.value = 1; }, 16);
      this._nextPatternAt = t + 1 + Math.random();
    }

    const par = this._patternParams[this._next];
    if (t >= this._nextBlackoutAt) {
      this.uniforms.uBlackOut.value = par.blackOut + Math.random() * par.blackOutRandom;
      this._hashAnimBO = { from: this.uniforms.uBlackOutHash.value, to: Math.random(), t0: t, dur: 0.3 };
      this.uniforms.uLogoDisplayType.value = Math.floor(Math.random() * 3);
      this._nextBlackoutAt = t + 1 + Math.random();
    }
    if (t >= this._nextUvShiftAt) {
      this.uniforms.uUVShift.value = par.uvShift;
      this._uvShiftAnim = { from: this.uniforms.uUVShiftPower.value, to: Math.random(), t0: t, dur: 0.8 };
      this._hashAnimUV = { from: this.uniforms.uUVShiftHash.value, to: Math.random(), t0: t, dur: 0.3 };
      this._nextUvShiftAt = t + 0.1 + Math.random() * 5;
    }

    const ease = (a) => {
      const u = Math.min(1, (t - a.t0) / a.dur);
      return a.from + (a.to - a.from) * (1 - Math.pow(1 - u, 3));
    };
    this.uniforms.uUVShiftPower.value = ease(this._uvShiftAnim);
    this.uniforms.uUVShiftHash.value = ease(this._hashAnimUV);
    this.uniforms.uBlackOutHash.value = ease(this._hashAnimBO);
  }

  dispose() {
    this.mesh.geometry.dispose();
    this.mesh.material.dispose();
    for (const p of this.patterns) p.dispose();
  }
}
