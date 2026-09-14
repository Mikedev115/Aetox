---
name: aetox-web-3d
before: putting a canvas, WebGL, a shader or a three.js scene on a page, before choosing the road
description: ตอนงานต้องมีฉากสามมิติ shader หรือ canvas บนหน้าเว็บ ไม่ว่าจะเป็น hero ที่หมุนได้ พื้นหลังที่เคลื่อน โมเดลสินค้า หรือหน้าที่ web-templates ตัดสินว่าเป็นชั้น C
source: written here; the measure-then-change discipline is adapted from https://github.com/majidmanzarpour/threejs-game-skills (threejs-debug-profiler), MIT
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Web 3D

Three.js r186 (8 ก.ย. 2569). Snippets in `references/three-r186.md`. Every
public three.js skill read before writing this one stops at the API; none
of them says what the page does when the GPU is not there. That is the part
this file exists for.

The choices are yours: the scene, the material, the light, the motion and
how far to take them. The bar is not, and it is the same bar whichever model
opens this file: a piece ships when it looks finished, runs inside its
frame on a slow machine, and is still a page without the GPU. Nothing below
is a ceiling on the work; all of it is the floor the work stands on.

## Four roads; name the one you take

One line, before any code, saying which and why:

1. **CSS 3D.** `perspective`, `rotateX/Y`, `transform-style: preserve-3d`.
   A tilt, a flip, a stacked-card depth. No canvas, no script that matters.
2. **One shader in one file.** Raw WebGL2 and GLSL, no library, ชั้น A.
   A hero background, a gradient that breathes, noise, a field of points.
   Usually a hundred-odd lines. When it wants a camera you steer, that is
   road 3.
3. **three.js.** Real geometry, lights, a loaded model, post-processing.
   Vendored beside the page (ชั้น B) or npm (ชั้น C);
   `import * as THREE from "three"`, addons from `three/addons/...`.
4. **React Three Fiber.** The same engine in components; at home when the
   page is already React.

Any road is right when the piece calls for it. The habit to watch is taking
road 3 for something a shader draws in a hundred lines, or road 2 for
something that wants a camera; either way, the header says which road and
why, and a change of road mid-build is fine when it is said out loud.

## The contract, before the first line

- **No GPU is still a page.** `WebGL.isWebGL2Available()` from
  `three/addons/capabilities/WebGL.js`, or `canvas.getContext("webgl2")`
  on road 2. When it is false, the canvas is removed and what stands in its
  place is a real element already in the markup: a still image with
  `width`/`height`, a gradient, or nothing, because the canvas was
  decoration. The page's text, layout and every control are the same
  either way.
- **Reduced motion is a still scene, not a missing one.** Render one frame
  and stop the loop. The object is there; it does not turn.
- **The rest state is the finished page** (STANDARD.md ข้อ 2). Nothing
  behind the canvas is hidden until the scene "arrives".
- **A `?nogl=1` query param forces the fallback path.** It is how the
  fallback is tested and how it is shown in the proof.

## The bar, in numbers

- A frame under 8 ms with the CPU throttled 4x, at the heaviest moment on
  the page. Over it, the piece is not done: classify first (CPU: many
  objects, per-frame allocation; GPU fill: resolution, post passes,
  overdraw; GPU vertex: triangle count; memory: textures), change one
  thing, measure again.
- `renderer.setPixelRatio(Math.min(devicePixelRatio, 2))`.
- **Render on demand.** A scene that only moves on scroll or pointer renders
  when those change, not in a `requestAnimationFrame` loop. A scene that
  breathes uses `renderer.setAnimationLoop` and stops it when the tab is
  hidden (`visibilitychange`) or the canvas is out of view
  (`IntersectionObserver`).
- One loop per page. Two `setAnimationLoop`s is the first thing to check
  when the frame time doubles.
- Read `renderer.info.render.calls` and say the number, and what each
  call is for. A hero lands under ~100; past that, instancing and merged
  static meshes come before any other change.
- Textures: say the total bytes. 2048 on the long side; KTX2 or WebP for a
  photo. A texture the eye cannot resolve at the size it is drawn is cost
  without a picture.
- Shadows and post-processing are each a full extra draw of the scene or
  the screen. Take them when they are the point of the piece, say what
  they buy, and they count inside the 8 ms.

## The bar, in craft

- Light is placed, not defaulted: a key with a direction and a reason, a
  fill that says where the room is, nothing at intensity 1 because the
  tutorial had it. Materials read as a material (paper, glass, brushed
  metal), not as the constructor's grey.
- Colour comes from the page's tokens, so the scene belongs to the page in
  both themes and does not float over it.
- The object is composed with the copy: beside it on a wide screen, above
  it on a narrow one, never behind the words.
- The still image for the no-GPU page is drawn from the same scene (a
  render, or an SVG of its silhouette in the same colours), not a
  placeholder gradient. Someone on that page gets the same idea.
- Motion has a reason the header can name: a turn that shows the form, a
  camera walk that follows the copy. Ambient motion with no reason reads as
  a screensaver.

## r186 facts that old snippets get wrong

- Import paths: `three`, `three/addons/<folder>/<File>.js`. There is no
  `three/examples/jsm` in an app import.
- Colour: `renderer.outputColorSpace` is sRGB by default; colour textures
  need `texture.colorSpace = THREE.SRGBColorSpace`, data textures do not.
- Lights are physical since r155. A tutorial `PointLight(0xfff, 1)` is
  near black now; intensities are candela and lux, so start around 3 for a
  point light in a small scene and scale to taste.
- WebGL1 is gone (r163). `isWebGL2Available()` is the only capability
  question left.
- `three/webgpu` and `three/tsl` exist and the WebGPU renderer falls back
  to WebGL2 on its own. For a page, WebGL is still the road unless the
  brief names WebGPU; TSL is a different language to learn.

## Lifecycle

- Resize with `ResizeObserver` on the canvas's box, not `window.resize`:
  set `camera.aspect`, `camera.updateProjectionMatrix()`,
  `renderer.setSize(w, h, false)`.
- Leaving the scene (route change, component destroy): dispose every
  geometry, material and texture you created, then `renderer.dispose()`,
  then stop the loop. The GPU holds what JS forgot.
- `THREE.Timer` (`update()` then `getDelta()`) for anything time-based;
  `Clock` is deprecated in r186. Never `+= 0.01` per frame.

## Prove it, then say what was measured

Before saying it works: the canvas is not blank (pixels vary), draw calls
and frame time at the heaviest moment under 4x CPU throttle, the `?nogl=1`
page, the `prefers-reduced-motion: reduce` page, and a narrow viewport.
Five screenshots or it is a guess, and a guess does not ship. The numbers
go in the file header next to the road and its reason (STANDARD.md ข้อ 6);
a header without them is the header of a page that was not measured.
`aetox-verify` before the sentence "it runs".

## Hand-offs

- Scroll or a timeline driving the camera, pinned copy beside the canvas:
  `aetox-motion`.
- The page around the canvas: `aetox-web-templates`. The look:
  `aetox-frontend-design`.
- A game, physics, XR, IFC viewers: out of scope for a page skill; say so
  and reach for the project's own dependencies with `aetox-brainstorm`.

## What is not an exit

"Every modern device has WebGL" is true of the devices on the desk and
false of the one the client will open the file on, in a webview, behind a
GPU blocklist. "The fallback can come later" is a page that is blank
today. "It runs at 60 on my machine" is a number from a machine nobody
else has; throttle it.
