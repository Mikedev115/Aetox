# three.js r186, the snippets a page needs

Checked against mrdoob/three.js at tag r186 (package `three@0.186.0`,
8 ก.ย. 2569): `exports` carries `./addons/*`, `./webgpu`, `./tsl`;
`WebGLRenderer` defaults `outputColorSpace` to sRGB and `toneMapping` to
none; `examples/jsm/capabilities/WebGL.js` has `isWebGL2Available()` and
nothing for WebGL1.

## Vendoring (ชั้น B, no build)

The build is split in two: `build/three.module.js` imports
`./three.core.js` beside it. Copy both into `assets/`, and the addon files
you use under `assets/addons/...`, then an import map:

```html
<script type="importmap">
{ "imports": { "three": "./assets/three.module.js",
               "three/addons/": "./assets/addons/" } }
</script>
```

662 KB + 1.4 MB uncompressed (measured on 0.186.0); say so in the header,
it is the price of road 3.

## Gate, then build

```js
import * as THREE from "three";
import WebGL from "three/addons/capabilities/WebGL.js";

const host = document.querySelector(".w-hero-scene");   // has the fallback inside
const forced = new URLSearchParams(location.search).has("nogl");

if (!forced && WebGL.isWebGL2Available()) {
  host.querySelector(".w-hero-still")?.remove();       // the still stands down
  mount(host);
}
// else: the still image already in the markup is the hero. Nothing to do.
```

Markup for the host, so the no-GPU page is complete before any script:

```html
<div class="w-hero-scene" aria-hidden="true">
  <img class="w-hero-still" src="assets/hero-still.webp" width="1600" height="900" alt="">
</div>
```

## Renderer, camera, size

```js
function mount(host) {
  const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true,
                                             powerPreference: "high-performance" });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  host.appendChild(renderer.domElement);

  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(35, 1, 0.1, 100);
  camera.position.set(0, 0, 8);

  const ro = new ResizeObserver(([e]) => {
    const { width, height } = e.contentRect;
    camera.aspect = width / height;
    camera.updateProjectionMatrix();
    renderer.setSize(width, height, false);   // false: CSS owns the box
    render();
  });
  ro.observe(host);
  ...
}
```

## Lights, physical units

```js
scene.add(new THREE.AmbientLight(0xffffff, 0.4));
const key = new THREE.DirectionalLight(0xffffff, 2.5);   // lux-ish; 1 is dim
key.position.set(3, 4, 5);
scene.add(key);
```

`PointLight` intensity is candela; a small scene wants 3 to 20, not 1.

## Render on demand vs. a loop

```js
let dirty = true;
function render() { dirty = true; }

const reduce = matchMedia("(prefers-reduced-motion: reduce)");
const io = new IntersectionObserver(([e]) => { visible = e.isIntersecting; tick(); });
io.observe(renderer.domElement);
document.addEventListener("visibilitychange", tick);

const timer = new THREE.Timer();      // Clock is deprecated in r186
function tick() {
  if (document.hidden || !visible) { renderer.setAnimationLoop(null); return; }
  renderer.setAnimationLoop(() => {
    timer.update();
    const dt = timer.getDelta();
    if (!reduce.matches) { mesh.rotation.y += dt * 0.3; dirty = true; }
    if (dirty) { renderer.render(scene, camera); dirty = false; }
  });
}
tick();
reduce.addEventListener("change", () => { dirty = true; });
```

With reduced motion on, the loop still runs but only renders when something
external (resize, scroll, pointer) marked it dirty. A scene that never
breathes drops `setAnimationLoop` and calls `renderer.render` from those
events directly.

## Scroll drives the camera (with aetox-motion)

```js
gsap.to(camera.position, {
  z: 3, y: 1, ease: "none",
  scrollTrigger: { trigger: ".w-story", start: "top top", end: "bottom bottom",
                   scrub: 1 },
  onUpdate: render,
});
```

The tween marks the frame dirty; the loop draws it. Under reduced motion
the tween is created with `duration: 0` inside `gsap.matchMedia()` and
the camera simply sits at its end pose.

## Numbers to report

```js
const { calls, triangles } = renderer.info.render;   // after a render
```

Frame time: DevTools Performance with CPU 4x slowdown, or
`performance.now()` around `renderer.render` averaged over 60 frames at the
heaviest scroll position. Blank-canvas check: read back a downscaled copy of
the canvas and confirm the pixel values vary (a solid colour means the scene
did not draw).

## Leaving

```js
function unmount() {
  renderer.setAnimationLoop(null);
  ro.disconnect(); io.disconnect();
  scene.traverse((o) => {
    o.geometry?.dispose();
    const mats = Array.isArray(o.material) ? o.material : [o.material];
    mats.forEach((m) => { if (!m) return;
      for (const v of Object.values(m)) v?.isTexture && v.dispose();
      m.dispose(); });
  });
  renderer.dispose();
  renderer.domElement.remove();
}
```

## Road 2, one shader, no library

```js
const gl = canvas.getContext("webgl2", { antialias: false, alpha: true });
if (!gl) { /* the still stays */ } else { /* compile, one quad, uniforms:
  u_time, u_res, u_pointer; requestAnimationFrame only while visible;
  under reduced motion draw once with u_time frozen */ }
```

Same gate, same still, same `?nogl=1`, same DPR cap
(`canvas.width = w * Math.min(devicePixelRatio, 2)`).

## WebGPU, when the brief names it

```js
import * as THREE from "three/webgpu";
import { color, time, oscSine } from "three/tsl";
const renderer = new THREE.WebGPURenderer();
await renderer.init();          // falls back to a WebGL2 backend by itself
```

Materials become `Mesh*NodeMaterial` and shader logic is TSL, not GLSL.
Not the page default.
