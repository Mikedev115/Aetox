# Where these came from

51 of the 60 scenes here were vendored from somewhere; the other 9 were written
for Aetox and are at the bottom of this file. What follows first is the vendored
haul.

47 scenes in 105 files, renamed from studio/brand names to plain descriptions of
what each one actually is — the same reason `aetox-slide-templates` never named
a slide after where its inventory was cross-checked. Originals are untouched
next to this folder (`../html-video/`, `../huashu-design/`), kept as the
provenance trail; everything here is a renamed copy, not a move.

## motion/ — 22 scenes

Source: [nexu-io/html-video](https://github.com/nexu-io/html-video),
Apache-2.0. The file name changed, from the template's marketing name to what it
does — `frame-vignelli` → `bold-portrait-title`, `frame-pentagram-stat` →
`stat-anchor`, and so on for all 22.

**Three edits to the markup, 31 ส.ค. 2569, and each one closed a defect that was
silent.** The licence permits editing outright; this list exists so that nobody
reading a scene later mistakes our line for upstream's.

- **A composition root on `<body>` in all 13 single-file scenes**, carrying
  `data-composition-id`, `data-width`, `data-height`, `data-duration` and
  `data-no-timeline`. Without the frame the renderer produced 1080x1920 for every
  one of them and cropped designs drawn 1920 wide; without the duration `video
  check` could not sample the layout, and four scenes would not render at all.
  The numbers are measured from real renders, not from the files' own comments,
  which overstate every length they give.
- **Tailwind compiled in, in the 6 scenes that used it.** The
  `cdn.tailwindcss.com` script is gone and the CSS it generated for each file
  sits in a `<style>` block at the end of `<head>` — same position the script
  injected it, same pixels (SSIM 1.000000), 5 KB instead of 282 KB fetched per
  render.
- **Nothing changed for GSAP.** The 9 folders still name `cdn.jsdelivr.net` and
  are byte-for-byte upstream's; `video new` rewrites that address in the copy it
  makes, so the shelf keeps a file that opens in a browser as upstream wrote it.

**Nine of them are copied whole, and were not at first (fixed 30 ส.ค. 2026).**
Upstream ships every template as a folder; the first pass took only its
`index.html`, which is the one file that cannot stand alone in those nine. Each
mounts scenes of its own through `data-composition-src`, three of them play
audio off `assets/`, and every one of those referenced files was missing. A
scene like that does not fail loudly — it renders as a frame with its contents
absent. The folders are now byte-for-byte upstream's, renamed:

| upstream folder | here | what came with it |
|:--|:--|:--|
| `frame-vignelli` | `bold-portrait-title/` | 2 compositions |
| `frame-decision-tree` | `decision-flowchart/` | 1 composition |
| `frame-nyt-graph` | `editorial-chart/` | 1 composition |
| `frame-warm-grain` | `grain-texture-hero/` | 3 compositions |
| `frame-kinetic-type` | `kinetic-type/` | 1 composition |
| `frame-play-mode` | `playful-bounce/` | 3 compositions |
| `frame-product-promo-30s` | `product-launch-30s/` | 8 compositions, 4 audio tracks, 3 screenshots, 2 logos |
| `frame-product-promo` | `product-showcase/` | 3 compositions, 3 svg |
| `frame-swiss-grid` | `structured-grid/` | 3 compositions, 1 svg |

Their `template.html-video.yaml` travels with them rather than being stripped,
and that is deliberate: it carries upstream's own licence block and attribution
alongside the default length and frame size, so the credit for those nine lives
inside the folder as well as in this file.

Six of the 22 are themselves derived by nexu-io from two upstream design
skills, credited in nexu-io's own `ATTRIBUTIONS.md` (copied to
`../html-video/ATTRIBUTIONS.md`) — carried through here rather than dropped:

- **statement-title, section-marker, offset-title, split-quote** — style
  presets from [frontend-slides](https://github.com/zarazhangrui/frontend-slides)
  by Zara Zhang (MIT).
- **stat-anchor, radial-node-diagram** — design philosophies from
  [huashu-design](https://github.com/alchaincyf/huashu-design) by alchaincyf
  (MIT), themselves stylistic homages to real studios (Pentagram/Michael
  Bierut, Takram) that nexu-io names as inspiration only, not affiliation.

Not included: `frame-data-rollup` (would have been `number-rollup`) — its
`source_entry` is a Remotion `.ts` component, not an HTML file, so it needs
the Remotion toolchain rather than a browser. Left in `../html-video/` only.

## graphic-scenes/explainer-diagram-poster.html — 1 file, adapted not just copied

Source: [amigoscode/infographics-skill](https://github.com/amigoscode/infographics-skill),
MIT, Copyright (c) 2026 Amigoscode. The source skill's workflow needs a
`GEMINI_API_KEY` (calls Gemini to draw the diagram) plus Playwright
(screenshots the result) — both wrong for Aetox, which calls no paid image
API and already has its own screenshot path (the WebView2 + `CapturePreview`
capture the slide renderer uses). That coupling lives in the *workflow*, not
the template file: `assets/template.html` itself has zero API calls, zero
Gemini/Playwright references — 140 lines of plain HTML with five
`{{PLACEHOLDER}}` tokens.

**Kept as-is:** the frame — fixed 1200×1581 poster, top-left tech icon,
top-right ALL-CAPS title, full-bleed content area between them, bottom bar
with footer text + logo wordmark. One Google Fonts `@import` (fonts-only,
degrades to a fallback face).

**Cut:** `generate-diagram.ts`, `install.sh`, `config.example.json`, the
Amigoscode branding assets, and every Gemini/Playwright step of its
`SKILL.md` workflow — none of that is in the file kept here.

**Adapted — how the placeholders get filled now:** `{{ICON_PATH}}` /
`{{LOGO_PATH}}` are small brand marks, fetched from the real source's own
site via `web_fetch` — the same mechanism `DECISIONS.md` already documents as
how Aetox gets a picture at all ("there is no image search in Aetox and
there never was; what there was is a page fetch that hands back the picture
URLs it saw"). `{{DIAGRAM_SRC}}`, the main illustration, has no Aetox-native
equivalent to Gemini's generation — either `web_fetch` finds an image that
already exists for the exact topic, or the `<img>` is replaced with inline
SVG the model draws directly, consistent with `motion/decision-flowchart`
and `motion/radial-node-diagram` in this same library.

## slide-scenes/, graphic-scenes/, web-scenes/ — 24 files

Source: [huashu-design](https://github.com/alchaincyf/huashu-design) by
alchaincyf (MIT), `assets/showcases/`. These are **static** — no GSAP, no
`@keyframes`, checked directly in the markup — so they sit next to
`aetox-slide-templates`/`aetox-web-templates` in spirit, not next to
`motion/`. Split by medium for the same reason those two skills are separate
skills: a deck, a web page and a standalone graphic don't share a contract.

The source names each scene `{scenario}-{studio}` (e.g. `ppt-pentagram`,
`ainav-takram`) — three real design studios' names (Pentagram, Build, Takram)
used as style labels. Renamed here by what the style actually looks like
instead, since none of those names mean anything on sight otherwise:

| Studio label (source) | Renamed to | Look |
|---|---|---|
| Pentagram | `-editorial` | black/white, Swiss grid, red accent, high contrast |
| Build | `-minimal` | 70%+ whitespace, ultra-thin type, warm-gold hairlines |
| Takram | `-organic` | rounded, soft-tech, natural tones, curved connectors |

Scenario names renamed the same way — `ainav` → `directory-nav`,
`aiwriting` → `writing-tool`, `homepage` → `portfolio-home` — describing the
page's purpose rather than the fictional product huashu-design demoed it with.

Studio names are recorded here as the original design inspiration, exactly as
huashu-design's own license requires attribution for — not a claim that
Aetox is affiliated with Pentagram, Build, or Takram.

## From the hyperframes registry itself (1 ก.ย. 2569)

Source: [heygen-com/hyperframes](https://github.com/heygen-com/hyperframes)
(Apache-2.0), `registry/examples/`. Four full productions vendored whole —
folder shape, GSAP timelines already registered at `window.__timelines`,
already pinned to gsap@3.14.2, engine-checked with 0 errors before landing:

| Scene | Source example | Notes |
|---|---|---|
| `motion/airbnb-deck/` | `registry/examples/airbnb-deck` | 108s, 12 beats, ships its own per-beat SFX (`sfx/*.mp3`, same licence) and a `DESIGN.md` |
| `motion/startup-pitch/` | `registry/examples/startup-pitch` | 84s pitch deck |
| `motion/slideshow-demo/` | `registry/examples/slideshow-demo` | 32s branching slideshow |
| `motion/motion-blur/` | `registry/examples/motion-blur` | 4s title; its "DRIFT" overflow is the design and `check` reports it on purpose |

Their upstream `registry-item.json` metadata was folded into each folder's
`template.html-video.yaml` rather than carried as a second metadata file. Two
more registry examples were inspected and NOT taken: `product-promo` and
`play-mode` are the upstream originals of `product-showcase` and
`playful-bounce` already on this shelf (via hyperframes-student-kit), and
`vscode-theme-visualizer` was deferred because it needs its own build scripts,
which breaks this shelf's copy-and-edit contract.

## motion/ — 9 scenes written here, not vendored (4 ก.ย. 2569)

`search-query-reveal`, `dot-loader-morph`, `card-expand-hero`, `caption-montage`,
`aurora-prompt-box`, `ship-it-toolbar`, `spring-stat-tiles`, `shared-axis-steps`,
`device-reveal`.
No upstream repository, no licence to carry, no attribution owed to anyone —
these are Aetox's own files, and they are recorded here because a credits file
that lists only what was borrowed leaves a reader guessing about the rest.

They are written in a house style derived from Google's, and the whole point of
that sentence is which half of "derived" applies. Nothing was copied from any
advertisement — no shot, no line of copy, no edit. What was used:

| What | From | How it was obtained |
|:--|:--|:--|
| Easing tokens, duration tokens | Material Design 3 | Read off [material-components-android `docs/theming/Motion.md`](https://github.com/material-components/material-components-android/blob/master/docs/theming/Motion.md) (Apache-2.0), cross-checked against [mdui](https://www.mdui.org/en/docs/2/styles/design-tokens), which agrees value for value |
| The Emphasized curve, sampled to CSS `linear()` | The same spec | The spec states it as a two-segment path; it was sampled at 25 even points because no `cubic-bezier()` can express it |
| Cross-fade split (6/20 out, 14/20 in, scale 0.92) | Google's Flutter `animations` package | Read off [`fade_through_transition.dart`](https://github.com/flutter/packages/blob/main/packages/animations/lib/src/fade_through_transition.dart) (BSD-3-Clause) |
| Dark surface values in `card-expand-hero` | `aistudio.google.com` | Measured in a browser on 4 ก.ย. 2569 — computed styles, not sampled pixels: stage `rgb(18,19,23)`, card `linear-gradient(135deg, rgb(28,30,35), rgb(42,45,53))` at 20px, white pill CTA at 12px, h1 Google Sans Flex weight 450 at -1.44px/64px |
| `Google Sans`, `Google Sans Flex` | Google, SIL Open Font License since 10 ธ.ค. 2025 | Fetched at render time from `fonts.googleapis.com`, like every other face on this shelf |
| `Anuphan` | Cadson Demak, SIL OFL, via Google Fonts | The Thai coverage; see GUIDE.md, "ไทย" |

Specifications and computed styles are facts about published work, and an OFL
face is licensed for exactly this. A company's identity is neither, so none of it
is here: no mark, no wordmark, no product name, and not the four brand hues — the
palette is indigo `#3B4AE0`, coral `#E8543F`, amber `#F2B33D`, teal `#1F9D74`.

One line of that deserves singling out, because the first draft of these scenes
got it wrong. The **sequence** blue → red → yellow → green identifies a company
on sight regardless of the hex values underneath, and `dot-loader-morph` had it
before anyone looked twice. It now runs indigo → teal → amber → coral, and the
file says why in a comment so the next person does not helpfully "fix" it back.

Full reasoning, and every number above with its date, in
`references/google-register.md`.

### The second four, and two sources the first four did not use

`aurora-prompt-box`, `ship-it-toolbar`, `spring-stat-tiles` and
`shared-axis-steps` were written later the same day and reach further back into
Google's published work. Same licence position: specifications and source code
are facts about published work, an OFL face is licensed for use, and an identity
is neither.

| What | From | How it was obtained |
|:--|:--|:--|
| Spring tokens — `SpringDefaultSpatial` 0.8/380, `SpringFastSpatial` 0.6/800, `SpringSlowSpatial` 0.8/200, and the effects springs at damping 1.0 / 1600, 3800, 800 | Material 3 Expressive, via AOSP (Apache-2.0) | Read off `compose/material3/.../tokens/ExpressiveMotionTokens.kt` and `StandardMotionTokens.kt`, version v0_14_0, on 4 ก.ย. 2569 |
| The CSS `linear()` curves those became | — | Ours. Each token integrated as a unit step response with mass 1, which is what androidx's `SpringSimulation` uses, and sampled at 33 even points |
| Shared axis: 30dp translate, opacity split `Interval(0.0,0.3)` out and `Interval(0.3,1.0)` in, legacy easing set | Google's Flutter `animations` package (BSD-3-Clause) | Read off `shared_axis_transition.dart`, cross-checked against Flutter's `Easing` class in `material/motion.dart` |
| The observation that a film's border glow rotates hue, that the shipping beat is lit warm, and that the outro is a mark alone on black | "Build Android Apps From A Text Prompt with Google AI Studio", Google for Developers, 31.4s | Opened in a browser 4 ก.ย. 2569, paused and stepped frame by frame; frames drawn to a canvas and saturated pixels bucketed by hue for the colour readings. Not read from its title or description |

**The colour readings are recorded and not used.** The sampled hues were
`#0AC67C` at t=3s, `#347C4A`/`#827A27` at t=12s, `#75A6C5`/`#658AB5` at t=18s,
`#977017`/`#9C4326` across ~48,000px at t=21s, and `#36A853` at t=25s. That last
one is within rounding of a brand colour and the first is a platform green. What
was taken is the *behaviour* those numbers reveal — that the hue travels, and
that the payoff beat is lit warmer than the work — not the values. The scenes
run indigo / teal / amber / coral like the other four.

Nothing from the film's audio, script, edit or footage is reproduced anywhere in
these files, and no frame of it was saved.

### The third pass: every frame, not a sample

`device-reveal` came from going back to the same film a third time and taking
**all** of it rather than thirteen stills. Method, 4 ก.ย. 2569: the video was
played at 1.5x with each frame drawn to a 64x36 canvas on `requestAnimationFrame`
and reduced to a mean RGB plus a per-pixel difference against the frame before
it. That produced **1,250 samples, one every 25ms of runtime**, from which the
cut list and the luminance curve in `references/google-register.md` were
computed.

Nothing was saved. The samples are 64x36 averages held in memory for the length
of one analysis, no frame was written to disk, and the only things kept are the
derived numbers — ten cut timestamps, nine gap lengths, and a per-second
luminance and hue table. Those numbers describe the film; they do not reproduce
it, and no image, audio, or line of its script exists anywhere in this library.
