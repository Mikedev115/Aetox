# POSITION XERO — immersive site (design study rebuild)

A from-scratch rebuild of the interaction design of [alche.studio](https://alche.studio/),
rebranded to **Position Xero**: the screen-space-refraction glass "X", the Tweakpane debug
HUD, the instanced LED-wall quadtree, and the fully scroll-driven top page — as an
editable, hostable Vite project.

**Origin note.** The interaction design is studied from alche.studio; all code is original
and every asset (geometry, textures, environment map, artwork, loader/outro animations) is
generated procedurally at runtime — nothing from the reference site ships here. Branding
and copy are Position Xero's own. Before deploying publicly, run the full SEO engagement
process (titles, URLs, schema, content structure) — this build is an interaction demo, not
a launch-ready site.

Reference material used during development (screen recordings, scroll-ladder screenshots,
extracted parameter notes) lives in `reference/` and is gitignored.

## Architecture

| Piece | Where | What it does |
| --- | --- | --- |
| Boot + loader | `src/js/main.js`, `src/js/loader.js` | Canvas blueprint loader (guide grid draw-in, triangle outline trace, ALCHE-charset scramble tagline), then a composite-shader handoff: scene starts 2× zoomed and pulls back over 3 s with a radial ripple. |
| Render pipeline | `src/js/gl/scene.js` | Three scenes to render targets (dark main / light mission-vision / dark service) + a transparent works-screens scene, mixed with a bottom-up arced wipe and a hard service cut, then bloom and the final composite. |
| Glass logo | `src/js/gl/main-logo.js`, `logo-geometry.js` | Procedural A-prism (hole ≈ 43.7 % of side, foot slot, rounded bevel). Custom refraction shader: 8-tap chromatic samples of a backbuffer copy, GGX highlight, fresnel + procedural studio cubemap. Pointer *movement* (no drag) applies euler impulses; quaternion slerps home per-section. |
| LED wall | `src/js/gl/bg-wall.js` | Instanced quadtree of box tiles bent onto a half-cylinder. Three flow-noise patterns (blue clouds / B-W zebra / purple-cyan-orange) hard-cut every 1–2 s, tile blackouts + UV-shift glitches, ALCHE wordmark tiling in three modes, WORKS band burn-in, per-work blurred art wipes. |
| Works carousel | `src/js/gl/works-thumbs.js` | Six rounded 16:9 slabs on a spiral (x=sin·11, z=cos·5−6, y=−k, rotY=0.6k) with chromatic lens distortion. |
| Service + stellla | `src/js/gl/service-scene.js` | Reel panels riding a sine wave that un-curls on entry, ghost SERVICES band, and the third panel peeling left-to-right into a fullscreen cover-fit quad. |
| Scroll | `src/js/scroll-manager.js` | Lenis (defaults) + plain ScrollTriggers as progress trackers — no scrub; the scene smooths every channel through a 10/s exponential lerper. Snap arrays per section. |
| DOM sections | `src/js/dom-sections.js` | Works item swap (SplitText chars), mission/vision draw-on with black marker swipes + rising mask, service items, stellla frame, left rail indicator. |
| HUD | `src/js/panels.js` | Real Tweakpane panes: MainLogo Material / Screen / Quaternion (rotation-gizmo plugin) + Reset. |
| Outro | `src/js/outro.js` | 200 vh sticky screen: glow canvas, guides → letter outlines → fill wordmark construction, A-mark slide. |

Scroll proportions match the reference exactly: kv 100svh+100px, works_intro 100lvh,
works 6×100lvh, works_outro 150lvh, mission_in 100lvh, mission 160vh, vision 200vh,
vision_out 150lvh, service_in 100lvh, service 3×170lvh, stellla 150lvh, outro 200vh
(27,316 px total at 1920×1080).

## Develop

```bash
npm install
npm run dev        # http://localhost:5173
```

Pages: `/` (top), `/works/`, `/about/`, `/news/`, `/stellla/`, `/contact/`.
Subpages are static HTML with a shared header.

## Build & host

```bash
npm run build      # outputs static site to dist/
npm run preview    # sanity-check the build locally
```

## Capture / comparison tooling

`tools/capture/capture-live.mjs` — Playwright rig that screenshots a fixed scroll ladder
(and mirrors assets) from any URL; used to diff this rebuild against the reference
frame-by-frame. `tools/capture/record-screen.ps1` — 60 fps ffmpeg desktop capture.

## Before any real use

- Replace the ALCHE name/mark, all Japanese copy, news items, and work titles
- The sound toggle is visual-only (no audio is shipped)
- Works/reel artwork is placeholder — swap in real key visuals
