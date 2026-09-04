import * as THREE from 'three';
import {
  buildLogoHalfGeometries, buildLogoOutlineGeometries,
  buildCutPlaneGeometry, cutPerpendicular
} from './logo-geometry.js';
import { GLSL_CONSTANTS, GLSL_HASH, GLSL_ROTATE2 } from './shader-lib.js';

/**
 * The glass "X" prism — built as TWO HALVES sliced corner-to-corner
 * (top-left → bottom-right). Closed, they read as one mark with a faint
 * diagonal seam. Through the vision→service transition the halves pull
 * apart along the cut normal and a prism of light (the cut plane, running
 * the hologram shader) blazes out of the gap; the camera then dives into
 * the light, which dissolves into the service scene.
 *
 * Look: screen-space refraction — the already-rendered scene is copied to
 * a backbuffer texture and the prism bends/chromatically splits it, with a
 * patchy animated frost mask, a GGX highlight, fresnel and a studio cubemap.
 *
 * Feel: no dragging. Pointer movement near screen centre applies small
 * euler impulses; the impulse decays and the quaternion slerps home at a
 * per-section return force.
 */

const SECTION_ROTATION = {
  default: { intensity: 0, baseSpeed: 0, returnForce: 2.0, scrollSpin: 0, hoverMult: 0 },
  kv: { intensity: 0, baseSpeed: 0, returnForce: 2.0, scrollSpin: 0, hoverMult: 1 },
  works_intro: { intensity: 1, baseSpeed: 0.5, returnForce: 0.4, scrollSpin: 0.001, hoverMult: 0.2 },
  works: { intensity: 1, baseSpeed: 0.5, returnForce: 0.1, scrollSpin: 0.001, hoverMult: 0.2 },
  mission: { intensity: 0, baseSpeed: 0, returnForce: 2.0, scrollSpin: 0, hoverMult: 0 },
  vision: { intensity: 0, baseSpeed: 0, returnForce: 1.5, scrollSpin: 0, hoverMult: 0 },
  service: { intensity: 0, baseSpeed: 0, returnForce: 1.8, scrollSpin: 0, hoverMult: 1 }
};

const vert = /* glsl */ `
${GLSL_CONSTANTS}
${GLSL_ROTATE2}
uniform float uVisionRotate;   // vision: lock upright with a slight turn
uniform float uServiceRotate;  // service_in: ease a touch further
uniform float uScreenAspectRatio;
varying vec2 vUv;
varying vec3 vNormal;
varying vec3 vViewPos;

void main() {
  vec3 pos = position;
  vec3 nml = normal;

  // gentle presentation tilt — the cut stays facing the camera so the
  // split and the light inside it read clearly
  float tiltA = (uVisionRotate * 0.12 + uServiceRotate * 0.1) * HPI;
  pos.xz *= rot2(tiltA);
  pos.yz *= rot2(uVisionRotate * -0.08 - uServiceRotate * 0.05);

  vec4 mv = modelViewMatrix * vec4(pos, 1.0);
  gl_Position = projectionMatrix * mv;
  vUv = uv;
  vNormal = normalMatrix * nml;
  vViewPos = -mv.xyz;
}
`;

const frag = /* glsl */ `
${GLSL_CONSTANTS}
${GLSL_HASH}
uniform sampler2D uBackTex;     // scene rendered behind the prism
uniform vec2 uBackRes;
uniform sampler2D uNoiseTex;    // shared animated flow noise
uniform samplerCube uEnvMap;
uniform float uRoughness;
uniform float uNoiseScale;
uniform vec3 uMaterialColor;    // 0-255 per channel
varying vec2 vUv;
varying vec3 vNormal;
varying vec3 vViewPos;

#define TAPS 8

float ggxSpec(float dNH, float rough) {
  float a2 = rough * rough;
  a2 *= a2;
  float d2 = dNH * dNH;
  if (d2 <= 0.0) return 0.0;
  float denom = d2 * (a2 - 1.0) + 1.0;
  return a2 / (PI * denom * denom);
}

float fresnelSchlick(float d) {
  const float f0 = 0.1;
  return f0 + (1.0 - f0) * pow(1.0 - d, 5.0);
}

void main() {
  vec2 screenUv = gl_FragCoord.xy / uBackRes;

  // patchy frost: warp the noise through itself, threshold the result
  vec4 n1 = texture2D(uNoiseTex, vUv * uNoiseScale);
  vec4 n2 = texture2D(uNoiseTex, vUv + (n1.xy - 0.5) * 2.0);
  float rough = smoothstep(0.3, 0.8, n2.y) * uRoughness;

  vec3 N = normalize(vNormal);
  vec2 bend = N.xy * (1.0 - N.z * 0.7);     // foreshorten the bend

  vec3 refracted = vec3(0.0);
  for (int i = 0; i < TAPS; i++) {
    float fi = float(i);
    float slide = 0.010 + hash21(screenUv + fi * 0.2) * 0.012;
    vec2 scatter = (vec2(
      hash21(screenUv + fi * 0.1),
      hash21(screenUv + fi * 0.2)
    ) - 0.5) * rough * 0.3;
    vec2 uvR = scatter + screenUv - bend * (0.1 + slide);
    vec2 uvG = scatter + screenUv - bend * (0.1 + slide * 2.0);
    vec2 uvB = scatter + screenUv - bend * (0.1 + slide * 4.0);
    refracted += vec3(
      texture2D(uBackTex, uvR).r,
      texture2D(uBackTex, uvG).g,
      texture2D(uBackTex, uvB).b
    ) * 0.9;
  }
  refracted /= float(TAPS);

  vec3 col = refracted;

  vec3 V = normalize(vViewPos);
  vec3 L = normalize(vec3(-1.0, 0.8, -1.0));
  vec3 H = normalize(V + L);
  col += ggxSpec(dot(N, H), 0.003 + rough * 0.4);

  float F = fresnelSchlick(dot(V, N));
  vec3 env = textureCube(uEnvMap, reflect(V, N)).rgb;
  col += mix(col, env, F * 0.9) * (1.0 - F);
  col += env * 0.06;   // faint ambient sparkle so the glass never goes dead

  col *= 1.2;
  col *= uMaterialColor / 255.0;
  gl_FragColor = vec4(col, 1.0);
}
`;

const outlineFrag = /* glsl */ `
void main() {
#ifdef IS_BASE
  // depth mask in the light scene's bg colour, invisible in colour
  gl_FragColor = vec4(0.8431, 0.8588, 0.8627, 0.0);
#else
  gl_FragColor = vec4(1.0);
#endif
}
`;

const screenFrag = /* glsl */ `
${GLSL_HASH}
uniform sampler2D uSceneTex;       // service scene render target
uniform sampler2D uNoiseTex;
uniform vec2 uScreenResolution;
uniform float uServiceIn;
uniform float uVisionRotate;
uniform float uSplit;              // 0 sealed … 1 fully open
uniform float uCover;              // 0 until the light owns the screen
uniform float uScreenNoiseScale;
varying vec2 vUv;

vec2 lensWarp(vec2 r, float a) { return r * (1.0 - a * dot(r, r)); }

void main() {
  // the light plane spans the whole cut — use its full uv
  vec2 geoUv = vUv;
  vec2 fullUv = gl_FragCoord.xy / uScreenResolution;
  vec2 uv = mix(geoUv, fullUv, pow(uCover, 0.2));

  // holographic shimmer: tri-channel warped noise, brightening as it opens
  vec4 h1 = texture2D(uNoiseTex, geoUv * uScreenNoiseScale);
  vec4 h2 = texture2D(uNoiseTex, geoUv * 0.3 * uScreenNoiseScale + h1.xy * (2.3 + hash21(fullUv) * 0.2));
  vec3 shimmer = pow(h2.xyz, vec3(0.8)) * (1.25 + uSplit * 1.1) + 0.08;

  // NOTHING of the next section leaks until the light covers the screen —
  // only then does the surface dissolve through to the service scene
  vec3 col = shimmer;
  if (uCover > 0.001) {
    float inv = 1.0 - uCover;
    vec3 scene = vec3(0.0);
    for (int i = 0; i < 5; i++) {
      float fi = float(i) / 5.0;
      vec2 wuv = uv + (h2.xy - 0.5) * inv;
      float power = inv * 5.0 + fi * 0.2 * inv;
      scene.r += texture2D(uSceneTex, lensWarp(wuv - 0.5, 1.00 * power) + 0.5).r;
      scene.g += texture2D(uSceneTex, lensWarp(wuv - 0.5, 1.05 * power) + 0.5).g;
      scene.b += texture2D(uSceneTex, lensWarp(wuv - 0.5, 1.10 * power) + 0.5).b;
    }
    scene /= 5.0;
    scene *= 1.0 + inv * 3.0;
    scene *= mix(0.5 + shimmer, vec3(1.0), uCover);

    float w = smoothstep(0.0, smoothstep(0.0, 1.0, h2.z), -h2.y + uCover * 2.0);
    col = mix(shimmer, scene, w);
  }

  // sealed inside the closed prism until the halves part
  float alpha = smoothstep(0.05, 0.35, uSplit);
  gl_FragColor = vec4(col, alpha);
}
`;

export class MainLogo extends THREE.Object3D {
  constructor({ noiseTexture, envMap }) {
    super();

    this.params = {
      roughness: 0.1,
      noiseScale: 9.0,
      screenNoiseScale: 1.0,
      color: { r: 255, g: 255, b: 255 },
      quat: { x: 0, y: 0, z: 0, w: 1 }
    };

    this.uniforms = {
      uBackTex: { value: null },
      uBackRes: { value: new THREE.Vector2(1, 1) },
      uNoiseTex: { value: noiseTexture },
      uEnvMap: { value: envMap },
      uRoughness: { value: this.params.roughness },
      uNoiseScale: { value: this.params.noiseScale },
      uMaterialColor: { value: new THREE.Vector3(255, 255, 255) },
      uVisionRotate: { value: 0 },
      uServiceRotate: { value: 0 },
      uScreenAspectRatio: { value: 16 / 9 }
    };

    this.material = new THREE.ShaderMaterial({
      vertexShader: vert,
      fragmentShader: frag,
      uniforms: this.uniforms,
      transparent: true
    });

    // ---- glass halves (dark scene) ----
    const halves = buildLogoHalfGeometries();
    this._geoUpper = halves.upper;
    this._geoLower = halves.lower;

    this.mesh = new THREE.Group();           // physics target (quaternion)
    this.mesh.scale.setScalar(3);
    this._halfUpper = new THREE.Mesh(this._geoUpper, this.material);
    this._halfLower = new THREE.Mesh(this._geoLower, this.material);
    this._halfUpper.renderOrder = 101;
    this._halfLower.renderOrder = 101;
    this.mesh.add(this._halfUpper, this._halfLower);
    this.add(this.mesh);

    // ---- outline set for the light (mission/vision) scene ----
    this.outlineGroup = new THREE.Group();
    const outlines = buildLogoOutlineGeometries();
    const lineMat = new THREE.LineBasicMaterial({ color: 0xffffff });
    const baseMat = new THREE.ShaderMaterial({
      vertexShader: vert,
      fragmentShader: outlineFrag,
      uniforms: this.uniforms,
      defines: { IS_BASE: 1 },
      transparent: true
    });

    this._outUpper = new THREE.Group();
    this._outUpper.scale.setScalar(3);
    const baseU = new THREE.Mesh(this._geoUpper, baseMat);
    baseU.renderOrder = 10;
    const linesU = new THREE.LineSegments(outlines.upper, lineMat);
    linesU.renderOrder = 11;
    this._outUpper.add(baseU, linesU);

    this._outLower = new THREE.Group();
    this._outLower.scale.setScalar(3);
    const baseL = new THREE.Mesh(this._geoLower, baseMat);
    baseL.renderOrder = 10;
    const linesL = new THREE.LineSegments(outlines.lower, lineMat);
    linesL.renderOrder = 11;
    this._outLower.add(baseL, linesL);

    this.outlineGroup.add(this._outUpper, this._outLower);
    this._baseMat = baseMat;
    this._lineMat = lineMat;
    this._outlineGeos = outlines;

    // ---- the prism of light inside the cut ----
    this.screenUniforms = {
      uSceneTex: { value: null },
      uNoiseTex: { value: noiseTexture },
      uScreenResolution: { value: new THREE.Vector2(1, 1) },
      uServiceIn: { value: 0 },
      uVisionRotate: { value: 0 },
      uSplit: { value: 0 },
      uCover: { value: 0 },
      uScreenNoiseScale: { value: 1 }
    };
    this.screenMesh = new THREE.Mesh(
      buildCutPlaneGeometry(),
      new THREE.ShaderMaterial({
        vertexShader: vert,
        fragmentShader: screenFrag,
        uniforms: { ...this.uniforms, ...this.screenUniforms },
        transparent: true,
        side: THREE.DoubleSide,
        depthWrite: false,
        depthTest: false
      })
    );
    this.screenMesh.renderOrder = 12;
    this.screenMesh.scale.setScalar(3);
    this.outlineGroup.add(this.screenMesh);

    this.scale.setScalar(4.0);
    this.position.set(0, -0.15, 0);

    // rotation state
    this._euler = new THREE.Euler();
    this._impulseQ = new THREE.Quaternion();
    this._identity = new THREE.Quaternion();
    this._spinQ = new THREE.Quaternion();
    this._yAxis = new THREE.Vector3(0, 1, 0);
    this._section = 'kv';
    this._worksRotate = 0;
    this._worksRotateTarget = 0;
    this._pointer = { x: 0, y: 0, px: 0, py: 0, has: false };

    // split / dive state
    this._split = 0;
    this._dive = 0;
    this.planeHalfWidthWorld = 0;
    this.planeUnfold = 0;
    this._perp = cutPerpendicular();         // geometry units, upper side
    // the cut line's direction — the axis the light sheet unfolds around
    this._diagAxis = new THREE.Vector3(this._perp.y, -this._perp.x, 0).normalize();
    this._unfoldQ = new THREE.Quaternion();
  }

  setSection(name) {
    this._section = SECTION_ROTATION[name] ? name : 'default';
    this._worksRotateTarget = SECTION_ROTATION[this._section].intensity;
  }

  /**
   * Dive progress 0..1. Phase 1 (0→0.45): the seam cracks open.
   * Phase 2 (0.4→1): the halves keep flying apart while the light plane
   * grows until it envelops the screen. `planeHalfWidthWorld` reports the
   * light's world half-width so the scene can verify true coverage.
   */
  setDive(d) {
    this._dive = Math.max(0, Math.min(1, d));
  }

  /** Latest pointer NDC (+y up). Called every frame with the last event. */
  hover(x, y) {
    if (this._pointer.has) {
      const s = SECTION_ROTATION[this._section];
      const dist = Math.min(1, Math.hypot(x, y));
      const gain = 0.01 * Math.max(0, 1 - dist * 1.5) * s.hoverMult;
      const velX = (x - this._pointer.px) * 10;
      const velY = (y - this._pointer.py) * 10;
      this._euler.x -= velY * gain * (1 - this._worksRotate * 0.7);
      this._euler.y += velX * gain;
    }
    this._pointer.px = x;
    this._pointer.py = y;
    this._pointer.has = true;
  }

  resetQuaternion() {
    this.mesh.quaternion.identity();
    this._euler.set(0, 0, 0);
  }

  update(dt, scrollVelocity) {
    const s = SECTION_ROTATION[this._section];

    this._worksRotate += (this._worksRotateTarget - this._worksRotate) * Math.min(1, dt * 3);

    // pointer impulse acts as decaying angular velocity
    this._euler.x *= 1 - dt;
    this._euler.y *= 1 - dt;
    this._impulseQ.setFromEuler(this._euler);
    this.mesh.quaternion.premultiply(this._impulseQ);

    // works auto-spin + scroll-speed spin
    const spin = -dt * s.baseSpeed * this._worksRotate - scrollVelocity * s.scrollSpin;
    if (spin !== 0) {
      this._spinQ.setFromAxisAngle(this._yAxis, spin);
      this.mesh.quaternion.premultiply(this._spinQ);
    }

    // spring home
    this.mesh.quaternion.slerp(
      this._identity,
      Math.min(1, dt * s.returnForce * (1 - this._worksRotate * 0.8))
    );

    // halves part along the cut normal — a crack first, then they keep
    // separating right off the screen while the light grows
    const d = this._dive;
    const sstep = (a, b, x) => {
      const u = Math.max(0, Math.min(1, (x - a) / (b - a)));
      return u * u * (3 - 2 * u);
    };
    const seam = sstep(0, 0.45, d) * 0.085 + Math.pow(sstep(0.4, 1, d), 1.5) * 1.15;
    const gap = seam;                        // geometry units
    this._split = sstep(0, 0.45, d);         // drives the light's alpha/brightness
    this._halfUpper.position.set(this._perp.x * gap, this._perp.y * gap, 0);
    this._halfLower.position.set(-this._perp.x * gap, -this._perp.y * gap, 0);

    // the prism of light swells with the separation AND unfolds out of the
    // cut to face the camera — its width swings into the screen plane, so
    // it visibly grows until it envelops the view
    const grow = sstep(0.35, 1, d);
    const planeScaleZ = 3 * (1 + grow * 40);         // width (across the cut)
    const planeScaleXY = 3 * (1 + grow * 1.5);       // length (along the cut)
    this.screenMesh.scale.set(planeScaleXY, planeScaleXY, planeScaleZ);
    this.planeUnfold = sstep(0.45, 0.85, d);
    this._unfoldQ.setFromAxisAngle(this._diagAxis, this.planeUnfold * Math.PI / 2);
    // world half-width of the light band (0.0716 = plane z half-extent)
    this.planeHalfWidthWorld = 0.0716 * planeScaleZ * this.scale.x;

    // sync the light-scene set to the glass pose + split
    this.outlineGroup.position.copy(this.position);
    this.outlineGroup.scale.copy(this.scale);
    for (const g of [this._outUpper, this._outLower]) {
      g.quaternion.copy(this.mesh.quaternion);
    }
    this.screenMesh.quaternion.copy(this.mesh.quaternion).multiply(this._unfoldQ);
    this._outUpper.position.set(this._perp.x * gap * 3, this._perp.y * gap * 3, 0);
    this._outLower.position.set(-this._perp.x * gap * 3, -this._perp.y * gap * 3, 0);

    // HUD params → uniforms
    this.uniforms.uRoughness.value = this.params.roughness;
    this.uniforms.uNoiseScale.value = this.params.noiseScale;
    this.uniforms.uMaterialColor.value.set(
      this.params.color.r, this.params.color.g, this.params.color.b
    );
    this.screenUniforms.uScreenNoiseScale.value = this.params.screenNoiseScale;
    this.screenUniforms.uSplit.value = this._split;

    const q = this.mesh.quaternion;
    if (q.w < 0) q.set(-q.x, -q.y, -q.z, -q.w);
    this.params.quat.x = q.x;
    this.params.quat.y = q.y;
    this.params.quat.z = q.z;
    this.params.quat.w = q.w;
  }

  dispose() {
    this._geoUpper.dispose();
    this._geoLower.dispose();
    this.material.dispose();
    this._baseMat.dispose();
    this._lineMat.dispose();
    this._outlineGeos.upper.dispose();
    this._outlineGeos.lower.dispose();
    this.screenMesh.geometry.dispose();
    this.screenMesh.material.dispose();
  }
}
