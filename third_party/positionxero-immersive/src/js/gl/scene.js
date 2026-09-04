import * as THREE from 'three';
import { Lerper, Animator, easeOutCubic } from '../core/lerper.js';
import { createFlowNoise } from './flow-noise.js';
import { createStudioEnvMap } from './env-map.js';
import { createBrandWordmarkTexture, createWorksAtlasTexture, createServicesTitleTexture } from './wordmark-textures.js';
import { WorksMedia } from './works-media.js';
import { BGWall } from './bg-wall.js';
import { GridOverlay } from './grid-overlay.js';
import { MainLogo } from './main-logo.js';
import { WorksThumbs } from './works-thumbs.js';
import { ServiceScene } from './service-scene.js';
import { PointerTrail } from './pointer-trail.js';
import { GLSL_CONSTANTS, GLSL_QUAD_VERT, GLSL_ROTATE2 } from './shader-lib.js';

/**
 * Scene orchestrator + render pipeline.
 *
 * Scenes:
 *  - main (dark): LED wall, grid, 2D wordmark plane, glass logo, WORKS band
 *  - missionVision (light): #EAF2F5 bg, dark grid, logo outline set
 *  - service (dark): reel wall + SERVICES ghost band
 *  - thumbs (transparent): works screens, drawn over the composite
 *
 * Per frame: main scene renders WITHOUT the logo → copied to a 512 buffer →
 * logo renders on top sampling that copy (screen-space refraction) → the
 * three scene RTs mix (diagonal sweep into light, hard cut into
 * service) → bloom-ish boost → final composite applies the intro zoom /
 * ripple / loader crossfade → works screens render on top.
 */

const mixFrag = /* glsl */ `
${GLSL_CONSTANTS}
uniform sampler2D uMainTex;
uniform sampler2D uMissionTex;
uniform sampler2D uServiceTex;
uniform sampler2D uPointerTex;
uniform vec2 uResolution;
uniform float uVisibleMission;
uniform float uMissionBlur;
uniform float uVisibleService;
varying vec2 vUv;

void main() {
  vec2 cuv = vUv - 0.5;
  float len = length(cuv);
  vec4 pointer = texture2D(uPointerTex, vUv);
  float pl = length(pointer.xy);

  vec3 mainCol = texture2D(uMainTex, vUv + pointer.xy * 0.01).rgb;
  mainCol *= smoothstep(1.2, 0.0, len);
  mainCol *= 1.0 + pl * 0.8;
  vec3 o = mainCol;

  // diagonal sweep into the light scene, bottom-left to top-right;
  // edge softness still tracks wipe velocity through uMissionBlur
  vec3 mission = texture2D(uMissionTex, vUv - pointer.xy * 0.01).rgb;
  float range = 1.0 / uResolution.y + abs(uMissionBlur) * 0.2;
  float d = vUv.y * 0.78 + vUv.x * 0.22;
  float sel = smoothstep(0.0, range, -d + uVisibleMission * (1.0 + range));
  mission *= smoothstep(1.5, 0.3, len);
  mission *= 1.0 + pl * 0.15;   // faint cursor wake on the light scene too
  o = mix(o, mission, sel);

  vec3 service = texture2D(uServiceTex, vUv + pointer.xy * 0.01).rgb;
  service *= 1.0 + pl * 0.8;
  service *= smoothstep(1.5, 0.3, len);
  // quick crossfade — completes while the light blade dominates the frame
  o = mix(o, service, uVisibleService);

  gl_FragColor = vec4(o, 1.0);
}
`;

const bloomThresholdFrag = /* glsl */ `
uniform sampler2D uTex;
varying vec2 vUv;
void main() {
  vec3 c = texture2D(uTex, vUv).rgb;
  float l = max(c.r, max(c.g, c.b));
  gl_FragColor = vec4(c * smoothstep(0.95, 1.2, l), 1.0);
}
`;

const blurFrag = /* glsl */ `
uniform sampler2D uTex;
uniform vec2 uDir;
varying vec2 vUv;
void main() {
  vec3 acc = vec3(0.0);
  float ws[5];
  ws[0] = 0.227; ws[1] = 0.194; ws[2] = 0.121; ws[3] = 0.054; ws[4] = 0.016;
  acc += texture2D(uTex, vUv).rgb * ws[0];
  for (int i = 1; i < 5; i++) {
    vec2 off = uDir * float(i);
    acc += texture2D(uTex, vUv + off).rgb * ws[i];
    acc += texture2D(uTex, vUv - off).rgb * ws[i];
  }
  gl_FragColor = vec4(acc, 1.0);
}
`;

const finalFrag = /* glsl */ `
${GLSL_CONSTANTS}
uniform sampler2D uSceneTex;
uniform sampler2D uBloomTex;
uniform sampler2D uThumbsTex;
uniform sampler2D uLoaderTex;
uniform float uLoaded;
uniform float uScreenAspectRatio;
varying vec2 vUv;

void main() {
  vec2 uv = vUv;
  vec2 cuv = vUv - 0.5;
  vec3 col;

  if (uLoaded < 0.999) {
    if (uScreenAspectRatio > 1.0) { cuv.y /= uScreenAspectRatio; }
    else { cuv.x *= uScreenAspectRatio; }

    float r = smoothstep(0.0, 0.2 + uLoaded * 0.7, -length(cuv) + uLoaded * 1.4);
    vec2 ripple = sin(r * PI) * normalize(cuv + 1e-6) * 0.1;

    vec2 sceneUv = (uv - 0.5) * (0.5 + uLoaded * 0.5) + 0.5 - ripple;
    vec3 scene = texture2D(uSceneTex, sceneUv).rgb
               + texture2D(uBloomTex, sceneUv).rgb * 0.25;
    vec4 thumbs = texture2D(uThumbsTex, sceneUv);
    scene = mix(scene, thumbs.rgb, thumbs.a);

    vec2 luv = uv - ripple;
    luv -= 0.5;
    float mask = 1.0;
    if (uScreenAspectRatio > 1.0) {
      luv.y /= uScreenAspectRatio;
      luv *= 1.0 / 0.8;
      mask = smoothstep(0.5, 0.4, abs(luv.x));
    } else {
      luv.x *= uScreenAspectRatio;
    }
    luv += 0.5;
    vec3 loader = texture2D(uLoaderTex, luv).rgb * mask;

    col = mix(loader, scene * 1.3, smoothstep(0.0, 0.5, r));
  } else {
    col = texture2D(uSceneTex, uv).rgb + texture2D(uBloomTex, uv).rgb * 0.25;
    vec4 thumbs = texture2D(uThumbsTex, uv);
    col = mix(col, thumbs.rgb, thumbs.a);
    col *= 1.3;
  }

  gl_FragColor = vec4(col, 1.0);
}
`;

const wordmark2DVert = /* glsl */ `
varying vec2 vUv;
void main() {
  vUv = uv;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`;

const wordmark2DFrag = /* glsl */ `
uniform sampler2D uTex;
uniform float uAlpha;
varying vec2 vUv;
void main() {
  vec4 t = texture2D(uTex, vUv);
  gl_FragColor = vec4(vec3(1.5), t.a * uAlpha);
}
`;

function makeQuadScene(fragmentShader, uniforms) {
  const scene = new THREE.Scene();
  const cam = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);
  const geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.BufferAttribute(new Float32Array([-1, -1, 0, 3, -1, 0, -1, 3, 0]), 3));
  geo.setAttribute('uv', new THREE.BufferAttribute(new Float32Array([0, 0, 2, 0, 0, 2]), 2));
  const mesh = new THREE.Mesh(geo, new THREE.ShaderMaterial({
    vertexShader: GLSL_QUAD_VERT,
    fragmentShader,
    uniforms,
    depthTest: false,
    depthWrite: false
  }));
  mesh.frustumCulled = false;
  scene.add(mesh);
  return { scene, cam, mesh };
}

export class PositionXeroGL {
  init(container) {
    this.container = container;
    try {
      this.renderer = new THREE.WebGLRenderer({ antialias: false, powerPreference: 'high-performance' });
    } catch {
      return false;
    }
    const dpr = Math.min(window.devicePixelRatio, 1.5);
    this.renderer.setPixelRatio(dpr);
    this.renderer.setSize(container.clientWidth, container.clientHeight);
    this.renderer.autoClear = true;
    container.appendChild(this.renderer.domElement);
    this.renderer.domElement.setAttribute('aria-hidden', 'true');

    this.lerper = new Lerper();
    this.animator = new Animator();
    this.animator.register('loaded', 0);
    this.animator.register('kvZoom', 1);
    this.animator.register('wordmark2D', 1);

    /* ---- shared resources ---- */
    this.noise = createFlowNoise(this.renderer);
    this.envMap = createStudioEnvMap();
    this.media = new WorksMedia();
    this.pointerTrail = new PointerTrail(this.renderer);
    this._wordmarkTex = createBrandWordmarkTexture();
    // the WORKS field tiled across the wall — one typeface per atlas cell
    this._worksTitleTex = createWorksAtlasTexture();
    this._servicesTex = createServicesTitleTexture();
    this._loaderTex = null; // set by the loader once its canvas is final

    /* ---- main (dark) scene ---- */
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color('#000');
    this.camera = new THREE.PerspectiveCamera(45, 1, 0.1, 1000);
    this.camera.position.set(0, 0, 10);

    this.wall = new BGWall(this.renderer, {
      noiseTexture: this.noise.texture,
      logoTexture: this._wordmarkTex,
      worksTitleTexture: this._worksTitleTex,
      pointerTexture: this.pointerTrail.texture,
      media: this.media
    });
    this.scene.add(this.wall.mesh);

    this.grid = new GridOverlay({ dark: false, cylinder: true });
    this.scene.add(this.grid.group);

    // 2D wordmark plane behind the glass (kv only)
    this.wordmark2D = new THREE.Mesh(
      new THREE.PlaneGeometry(28, 28 * (250 / 1204)),
      new THREE.ShaderMaterial({
        vertexShader: wordmark2DVert,
        fragmentShader: wordmark2DFrag,
        uniforms: { uTex: { value: this._wordmarkTex }, uAlpha: { value: 1 } },
        transparent: true,
        depthWrite: false,
        depthTest: false          // sits past the wall cylinder; draw over it
      })
    );
    this.wordmark2D.position.set(0, 1, -15);
    this.wordmark2D.renderOrder = -900;
    this.scene.add(this.wordmark2D);

    this.logo = new MainLogo({ noiseTexture: this.noise.texture, envMap: this.envMap });
    this.scene.add(this.logo);

    /* ---- light mission/vision scene ---- */
    this.missionScene = new THREE.Scene();
    this.missionScene.background = new THREE.Color('#EAF2F5');
    this.missionCamera = new THREE.PerspectiveCamera(60, 1, 0.1, 100);
    this.missionCamera.position.set(0, 0, 8);
    this.orthoCamera = new THREE.OrthographicCamera(-4, 4, 4, -4, 0.1, 100);
    this.orthoCamera.position.set(0, 0, 8);
    // the dive gets its own camera so the works→mission projection lerp
    // (which copies orthoCamera) never inherits the dive zoom
    this.diveCamera = this.orthoCamera.clone();
    this.missionGrid = new GridOverlay({ dark: true, cylinder: false });
    this.missionScene.add(this.missionGrid.group);
    this.missionScene.add(this.logo.outlineGroup);

    /* ---- service scene ---- */
    this.service = new ServiceScene(this.media, { servicesTitleTexture: this._servicesTex });

    /* ---- thumbs scene (works screens over everything) ---- */
    this.thumbsScene = new THREE.Scene();
    this.thumbs = new WorksThumbs(this.media, {});
    this.thumbsScene.add(this.thumbs.group);

    /* ---- render targets ---- */
    const rt = (opts = {}) => new THREE.WebGLRenderTarget(1, 1, {
      minFilter: THREE.LinearFilter,
      magFilter: THREE.LinearFilter,
      type: THREE.HalfFloatType,
      ...opts
    });
    this.rtMain = rt({ depthBuffer: true });
    this.rtMission = rt({ depthBuffer: true });
    this.rtService = rt({ depthBuffer: true });
    this.rtThumbs = rt({ depthBuffer: true });
    this.rtMixed = rt();
    this.rtBack = rt();          // 512 copy the glass refracts
    this.rtBloomA = rt();
    this.rtBloomB = rt();
    this.rtBack.setSize(512, 512);

    this.logo.uniforms.uBackTex.value = this.rtBack.texture;
    this.logo.screenUniforms.uSceneTex.value = this.rtService.texture;

    /* ---- post passes ---- */
    this.mixPass = makeQuadScene(mixFrag, {
      uMainTex: { value: this.rtMain.texture },
      uMissionTex: { value: this.rtMission.texture },
      uServiceTex: { value: this.rtService.texture },
      uPointerTex: { value: this.pointerTrail.texture },
      uResolution: { value: new THREE.Vector2(1, 1) },
      uVisibleMission: { value: 0 },
      uMissionBlur: { value: 0 },
      uVisibleService: { value: 0 }
    });
    this.thresholdPass = makeQuadScene(bloomThresholdFrag, {
      uTex: { value: this.rtMixed.texture }
    });
    this.blurPassH = makeQuadScene(blurFrag, {
      uTex: { value: this.rtBloomA.texture },
      uDir: { value: new THREE.Vector2(1 / 256, 0) }
    });
    this.blurPassV = makeQuadScene(blurFrag, {
      uTex: { value: this.rtBloomB.texture },
      uDir: { value: new THREE.Vector2(0, 1 / 256) }
    });
    this.finalPass = makeQuadScene(finalFrag, {
      uSceneTex: { value: this.rtMixed.texture },
      uBloomTex: { value: this.rtBloomA.texture },
      uThumbsTex: { value: this.rtThumbs.texture },
      uLoaderTex: { value: new THREE.Texture() },
      uLoaded: { value: 0 },
      uScreenAspectRatio: { value: 16 / 9 }
    });

    // copy pass for the 512 backbuffer
    this.copyPass = makeQuadScene(
      'uniform sampler2D uTex; varying vec2 vUv; void main(){ gl_FragColor = texture2D(uTex, vUv); }',
      { uTex: { value: this.rtMain.texture } }
    );

    /* ---- state ---- */
    this._pointerNdc = new THREE.Vector2();
    this._cameraOffset = new THREE.Vector2();
    this._missionVisible = 0;
    this._missionBlur = 0;
    this._section = 'kv';
    this._clock = new THREE.Clock();
    this._running = true;
    this._raf = 0;
    this._scroll = {
      worksTitle: 0, worksProgress: 0, worksOutro: 0, missionIn: 0,
      vision: 0, serviceIn: 0, serviceProgress: 0, cognexaIn: 0, page: 0
    };
    this._projA = new THREE.Matrix4();

    this._bindEvents();
    this.resize();
    this._loop();
    return true;
  }

  /* ---------- external API ---------- */

  onLoadingComplete() {
    this.animator.animate('loaded', 1, 3, easeOutCubic);
  }

  setLoaderTexture(tex) {
    this.finalPass.mesh.material.uniforms.uLoaderTex.value = tex;
  }

  /** name ∈ section list; drives logo mode, kv zoom, wordmark visibility */
  changeSection(name) {
    if (this._section === name) return;
    this._section = name;
    const logoSection =
      name === 'kv' ? 'kv'
      : name === 'works_intro' ? 'works_intro'
      : name === 'works' || name === 'works_outro' ? 'works'
      : name === 'mission_in' || name === 'mission' || name === 'calc' ? 'mission'
      // stay locked (no hover wobble) through the dive into the side wall
      : name === 'vision' || name === 'vision_out' || name === 'service_in' ? 'vision'
      : name === 'service' || name === 'cognexa' ? 'service'
      : 'default';
    this.logo.setSection(logoSection);
    this.animator.animate('kvZoom', name === 'kv' ? 1 : 0, 1);
    this.animator.animate('wordmark2D', name === 'kv' ? 1 : 0, 0.5);
  }

  /** Raw trigger progress, every frame from the scroll manager. */
  setScrollState(s) {
    Object.assign(this._scroll, s);
  }

  /** 1 only when the light band fully covers the screen. */
  get cover() {
    return this._cover ?? 0;
  }

  setPointer(clientX, clientY) {
    this._pointerNdc.set(
      (clientX / window.innerWidth) * 2 - 1,
      -(clientY / window.innerHeight) * 2 + 1
    );
    this.pointerTrail.setPointer(clientX / window.innerWidth, 1 - clientY / window.innerHeight);
  }

  /* ---------- internals ---------- */

  _bindEvents() {
    this._onMove = (e) => this.setPointer(e.clientX, e.clientY);
    window.addEventListener('pointermove', this._onMove, { passive: true });

    this._onVis = () => {
      this._running = !document.hidden;
      if (this._running) {
        this._clock.getDelta();
        this._loop();
      } else {
        cancelAnimationFrame(this._raf);
      }
    };
    document.addEventListener('visibilitychange', this._onVis);

    this._onLost = (e) => e.preventDefault();
    this.renderer.domElement.addEventListener('webglcontextlost', this._onLost);
  }

  resize() {
    const w = this.container.clientWidth;
    const h = this.container.clientHeight;
    const aspect = w / h;
    this.renderer.setSize(w, h);

    this.camera.aspect = aspect;
    this._baseFov = 35 + (1 / aspect) * 18;
    this.camera.fov = this._baseFov;
    this.camera.updateProjectionMatrix();

    this.missionCamera.aspect = aspect;
    this.missionCamera.updateProjectionMatrix();
    const halfH = 4;
    this.orthoCamera.left = -halfH * aspect;
    this.orthoCamera.right = halfH * aspect;
    this.orthoCamera.top = halfH;
    this.orthoCamera.bottom = -halfH;
    this.orthoCamera.updateProjectionMatrix();

    const pw = Math.round(w * Math.min(window.devicePixelRatio, 1.5));
    const ph = Math.round(h * Math.min(window.devicePixelRatio, 1.5));
    for (const rt of [this.rtMain, this.rtMission, this.rtService, this.rtThumbs, this.rtMixed]) {
      rt.setSize(pw, ph);
    }
    this.rtBloomA.setSize(pw / 4, ph / 4);
    this.rtBloomB.setSize(pw / 4, ph / 4);

    this.wall.resize(pw, ph, this.camera);
    this.grid.resize(this.camera);
    this.service.resize(w, h);
    this.mixPass.mesh.material.uniforms.uResolution.value.set(pw, ph);
    this.finalPass.mesh.material.uniforms.uScreenAspectRatio.value = aspect;
    // refraction samples are addressed by gl_FragCoord over the main RT
    this.logo.uniforms.uBackRes.value.set(pw, ph);
    this.logo.uniforms.uScreenAspectRatio.value = aspect;
    this.logo.screenUniforms.uScreenResolution.value.set(pw, ph);
    this.pointerTrail.setAspect(aspect);
    this.noise.uniforms.uScreenAspectRatio.value = aspect;
  }

  _renderQuad(pass, target) {
    this.renderer.setRenderTarget(target);
    this.renderer.render(pass.scene, pass.cam);
  }

  _loop = () => {
    if (!this._running) return;
    this._raf = requestAnimationFrame(this._loop);

    const dt = Math.min(this._clock.getDelta(), 1 / 30);
    const t = this._clock.elapsedTime;
    const s = this._scroll;
    const lp = this.lerper;

    this.animator.update(dt);
    lp.update(dt);

    /* --- smoothed scroll channels --- */
    const worksTitle = lp.set('worksTitle', s.worksTitle, 0.3);
    const worksProg = lp.set('worksProgress', s.worksProgress, 1);
    const carouselO = worksProg * (this.media.count + 1);
    const magnet = Math.round(carouselO) - (Math.round(carouselO) - carouselO) * 0.4;
    const carouselU = lp.set('worksIndex', magnet, 0.5);
    const carouselVel = lp.velocity('worksProgress') * 8;
    const bgU = lp.set('bgQuadProgress', s.worksProgress, 0.7) * (this.media.count + 1);
    const worksOutro = lp.set('worksOutro', s.worksOutro, 1);
    const parallaxKill = 1 - lp.set('camWorksOutro', s.worksOutro, 0.5);
    const missionVis = lp.set('missionVis', s.missionIn, 1.5);
    const visionRot = lp.set('visionRot', s.vision, 0.5);
    const serviceRot = lp.set('serviceRot', s.serviceIn, 1);
    const serviceIn = lp.set('serviceListIn', s.serviceIn, 0.6);
    const thumbO = (lp.set('thumbScroll', s.serviceProgress, 1) * 7 + 1) / 2;
    const thumbMag = Math.round(thumbO) - (Math.round(thumbO) - thumbO) * 0.4;
    const thumbU = Math.min(4, lp.set('thumbIndex', thumbMag, 0.5));
    const cognexaIn = lp.set('cognexaIn', s.cognexaIn, 0.7);
    // thumbU must be clamped to the reel count first: the two terms are designed
    // to sum to exactly 1, so feeding the unclamped value drove the peel to 1.5
    // and finished it a full viewport before the cognexa runway starts
    const cognexaView = lp.set('cognexaView', Math.max(0, Math.min(1, Math.min(3, thumbU) - 2.5)), 0.7) + cognexaIn * 0.5;
    const scrollVel = lp.set('lenisVel', Math.max(-30, Math.min(30, s.lenisVelocity ?? 0)), 0.5);
    const missionScroll = lp.set('missionScroll', s.pageScroll ?? 0, 0.5);

    /* --- procedural textures --- */
    this.noise.render(t);
    this.pointerTrail.update();

    /* --- camera --- */
    const kvZoom = this.animator.get('kvZoom');
    this._cameraOffset.lerp(
      new THREE.Vector2(
        this._pointerNdc.x * 0.5 * parallaxKill,
        this._pointerNdc.y * 0.5 * parallaxKill
      ),
      Math.min(3 * dt, 1)
    );
    this.camera.position.set(this._cameraOffset.x, this._cameraOffset.y, 10 - kvZoom * 0.5);
    this.camera.fov = this._baseFov - kvZoom * 4;
    this.camera.updateProjectionMatrix();
    this.camera.lookAt(0, 0, 0);

    // flatten perspective → ortho during works_outro
    if (worksOutro > 0.001) {
      const a = this.camera.projectionMatrix.elements;
      const b = this.orthoCamera.projectionMatrix.elements;
      const m = this._projA.elements;
      for (let i = 0; i < 16; i++) m[i] = a[i] + (b[i] - a[i]) * worksOutro;
      this.camera.projectionMatrix.copy(this._projA);
      this.camera.projectionMatrixInverse.copy(this._projA).invert();
    }

    /* --- logo + hover --- */
    this.logo.hover(this._pointerNdc.x, this._pointerNdc.y);
    this.logo.update(dt, scrollVel);
    this.logo.uniforms.uVisionRotate.value = visionRot;
    this.logo.uniforms.uServiceRotate.value = serviceRot;
    this.logo.screenUniforms.uServiceIn.value = serviceIn;
    this.logo.screenUniforms.uVisionRotate.value = visionRot;
    // dive: seam cracks, halves fly apart, the light swells
    this.logo.setDive(serviceRot);

    // TRUE-coverage gate: the next section exists only once the light band
    // is geometrically wider than everything the camera can see
    const diveZoom = 1 + Math.pow(serviceRot, 1.2) * 7;
    const visHalfW = (4 * this.camera.aspect) / diveZoom;
    const visHalfH = 4 / diveZoom;
    const perp = this.logo._perp;
    const required = visHalfW * Math.abs(perp.x) + visHalfH * Math.abs(perp.y);
    // only the width PROJECTED into the screen plane counts — edge-on the
    // sheet is a sliver; as it unfolds toward the camera it truly widens
    const projected = this.logo.planeHalfWidthWorld *
      Math.max(0.2, Math.sin(this.logo.planeUnfold * Math.PI / 2));
    const ratio = required > 0 ? projected / required : 0;
    // geometric coverage is the hard precondition; the time term holds the
    // pure-light moment for a beat before the dissolve-through begins
    const coverGeom = THREE.MathUtils.smoothstep(ratio, 1.05, 1.35);
    const cover = Math.min(coverGeom, THREE.MathUtils.smoothstep(serviceRot, 0.65, 0.85));
    this._cover = cover;
    this.logo.screenUniforms.uCover.value = cover;
    const serviceVis = cover;

    /* --- scene pieces --- */
    this.media.update(t);
    this.wall.setWorksTitleProgress(worksTitle);
    this.wall.setWorksProgress(worksProg, bgU);
    this.wall.update(t);
    this.grid.setScroll(missionScroll * 0.1);
    this.wordmark2D.material.uniforms.uAlpha.value = this.animator.get('wordmark2D');
    this.missionGrid.setScroll(missionScroll);
    this.thumbs.update(carouselU, carouselVel);
    this.service.update(t, thumbU, serviceIn, cognexaIn, cognexaView);

    /* --- mission wipe smoothing (edge arc from wipe velocity) --- */
    const dv = missionVis - this._missionVisible;
    this._missionBlur += (dv - this._missionBlur) * 0.5;
    this._missionVisible += dv * 0.5;
    this.mixPass.mesh.material.uniforms.uVisibleMission.value = this._missionVisible;
    this.mixPass.mesh.material.uniforms.uMissionBlur.value = this._missionBlur;
    this.mixPass.mesh.material.uniforms.uVisibleService.value = serviceVis;

    /* ============ render ============ */
    const r = this.renderer;

    // 1. main scene without the logo
    this.logo.mesh.visible = false;
    r.setRenderTarget(this.rtMain);
    r.render(this.scene, this.camera);

    // 2. 512 copy for refraction
    this.copyPass.mesh.material.uniforms.uTex.value = this.rtMain.texture;
    this._renderQuad(this.copyPass, this.rtBack);

    // 3. logo over the main scene (no clear)
    this.logo.mesh.visible = true;
    r.setRenderTarget(this.rtMain);
    r.autoClear = false;
    r.render(this.logo, this.camera);
    r.autoClear = true;

    // 4. light scene (outline follows the logo pose) — ortho matches the
    // flattened cam; during service_in the X splits open and the camera
    // dives INTO the light in the cut, carrying the whole mark with it
    if (this._missionVisible > 0.002 || visionRot > 0.002) {
      // moderate dive — the growing light does the enveloping, the camera
      // just leans in so the move still reads as travelling into it
      this.diveCamera.copy(this.orthoCamera);
      this.diveCamera.zoom = 1 + Math.pow(serviceRot, 1.2) * 7;
      this.diveCamera.position.y = -0.15 * serviceRot;
      this.diveCamera.updateProjectionMatrix();
      r.setRenderTarget(this.rtMission);
      r.render(this.missionScene, worksOutro > 0.5 ? this.diveCamera : this.camera);
    }

    // 5. service scene
    if (serviceVis > 0.002 || serviceRot > 0.002 || visionRot > 0.3) {
      r.setRenderTarget(this.rtService);
      r.render(this.service.scene, this.service.camera);
    }

    // 6. thumbs scene — only while the carousel is actually in range;
    // outside it the RT stays cleared so nothing can leak into the dive
    const thumbsActive = carouselU > 0.05 && carouselU < this.media.count + 0.95;
    if (thumbsActive || !this._thumbsCleared) {
      r.setRenderTarget(this.rtThumbs);
      r.setClearColor(0x000000, 0);
      r.clear();
      if (thumbsActive) r.render(this.thumbsScene, this.camera);
      r.setClearColor(0x000000, 1);
      this._thumbsCleared = !thumbsActive;
    }

    // 7. mix the three scenes
    this._renderQuad(this.mixPass, this.rtMixed);

    // 8. bloom
    this.thresholdPass.mesh.material.uniforms.uTex.value = this.rtMixed.texture;
    this._renderQuad(this.thresholdPass, this.rtBloomA);
    this.blurPassH.mesh.material.uniforms.uTex.value = this.rtBloomA.texture;
    this._renderQuad(this.blurPassH, this.rtBloomB);
    this.blurPassV.mesh.material.uniforms.uTex.value = this.rtBloomB.texture;
    this._renderQuad(this.blurPassV, this.rtBloomA);

    // 9. final composite to screen (intro zoom + loader crossfade)
    this.finalPass.mesh.material.uniforms.uLoaded.value = this.animator.get('loaded');
    r.setRenderTarget(null);
    r.render(this.finalPass.scene, this.finalPass.cam);
  };

  destroy() {
    this._running = false;
    cancelAnimationFrame(this._raf);
    window.removeEventListener('pointermove', this._onMove);
    document.removeEventListener('visibilitychange', this._onVis);
    this.renderer.domElement.removeEventListener('webglcontextlost', this._onLost);
    this.logo.dispose();
    this.wall.dispose();
    this.grid.dispose();
    this.missionGrid.dispose();
    this.thumbs.dispose();
    this.service.dispose();
    this.media.dispose();
    this.noise.dispose();
    this.pointerTrail.dispose();
    for (const rt of [this.rtMain, this.rtMission, this.rtService, this.rtThumbs, this.rtMixed, this.rtBack, this.rtBloomA, this.rtBloomB]) rt.dispose();
    this.renderer.dispose();
    this.renderer.domElement.remove();
  }
}
