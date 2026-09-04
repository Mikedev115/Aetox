# The long answers

`SKILL.md` states each rule in one line; this file is the why and the
measurements behind them. Open the section you are about to act against,
not the whole file.

## The contract these are written to

**Most of these are CSS `@keyframes` and nothing else** — no library, no build
step, no bundler, which is why they open in a browser as fast as they render.

**32 of the 45 motion scenes hold to that.** The other **13** —
`bold-portrait-title`, `decision-flowchart`, `editorial-chart`,
`grain-texture-hero`, `kinetic-type`, `playful-bounce`, `product-launch-30s`,
`product-showcase`, `structured-grid`, and since 1 ก.ย. `airbnb-deck`,
`startup-pitch`, `slideshow-demo`, `motion-blur` — drive their animation with
**GSAP**.
Checked, not assumed: each of the 9 registers its timeline at
`window.__timelines["..."]`, the exact hook Hyperframes seeks to render
frame-by-frame, so this is not a scene built for the wrong engine. It is the
second of the two things the renderer can drive, CSS keyframes being the first.

The file itself is on this machine, not on a CDN. The library still spells the
`cdn.jsdelivr.net` address, because a scene on the shelf has to open in a
browser as it is — but `video new` rewrites it to `vendor/gsap.min.js` in the
copy it makes, from the pinned copy the งานวิดีโอ page installs. If that has not
been installed the address is left alone, and then those 9 need the network up
at render time or they produce a still picture with nothing saying why.

**6 of the 22** — `bar-chart-counter`, `cinematic-light-leak`, `glitch-title`,
`liquid-gradient-hero`, `logo-outro`, `typewriter-cursor` — lay themselves out
with **Tailwind** utility classes. Upstream fetched the browser JIT from
`cdn.tailwindcss.com` on every render; these six now carry the CSS that script
generated for them, about 5 KB each, in a `<style>` block at the end of
`<head>`. Same pixels (SSIM 1.000000, measured 31 ส.ค.), nothing fetched.

**What that costs you:** the utility set in those six is now fixed to the
classes each scene already uses. A Tailwind class you add that is not in the
block does nothing at all — it will not error, the element simply will not move.
Write plain CSS for anything new, which is what the other 16 scenes do.

## Thirteen of them are folders, not files

The same 13 that drive their motion with GSAP are **compositions**: a host
`index.html` that mounts smaller scenes of its own through
`data-composition-src`, and in three cases plays audio tracks laid out on a
timeline with `data-start`, `data-duration` and `data-volume`. They travel as
folders because they have to:

```
motion/structured-grid/
  index.html                    the host, and the file you open
  compositions/intro.html       the scenes it mounts
  compositions/graphics.html
  compositions/captions.html
  assets/swiss-grid.svg         what those scenes draw
  template.html-video.yaml      its own licence block, default length and frame
  poster.svg
```

Open `index.html`. Change the words there or in the composition it mounts, never
both hoping they agree. Copy the **whole folder** when you copy one of these:
its host alone renders as an empty frame with the parts missing, which is a
failure that looks like a working render.

`product-launch-30s` is the fullest example in the library and worth reading
before writing anything long: eight mounted scenes, four audio tracks, and
thirty seconds of timing somebody already argued about.

**`__VIDEO_SRC__` and `__VIDEO_DURATION__` are not broken markup.** They are the
two slots the engine fills when it scaffolds a project from the template — the
footage you point it at, and the length you asked for. Leave them alone in the
library; they disappear in the copy that renders.

## How the renderer decides a scene's length

**The renderer freezes the page, waits for the fonts, then steps the clock.**
That is what makes a headless machine able to produce smooth motion without
watching it play in real time. It also means the length of a scene is the length
of its keyframes: an animation that finishes in 2 seconds inside a 6 second
scene leaves 4 seconds of a held frame, and one still moving at the end is cut
off mid-move.

**All 32 single-file scenes state their length on `<body>`**, beside the
frame, on the same composition root:

```html
<body data-composition-id="scene" data-no-timeline
      data-width="1920" data-height="1080" data-start="0" data-duration="3.0">
```

That is the one place the length lives in those 32; change the number to change
the clip. It is stated rather than derived for two reasons. Four of them —
`glitch-title`, `typewriter-cursor`, `liquid-gradient-hero` and
`cinematic-light-leak` — have no end to derive: the first three loop for ever
and the fourth has no `@keyframes` at all, and the renderer refuses all four
with "Composition has zero duration". The other nine derive fine at render time
and still could not be **checked**: `video check` reports "Could not determine
composition duration" without it, so the cheap half of the loop was returning
nothing. `data-no-timeline` is the third piece, and it stops the renderer
waiting 45 seconds for a GSAP timeline that is never coming.

The numbers in those 13 are measured, not intended — each is what a real render
produced on the owner's machine, rounded up to a tenth of a second so nothing is
cut mid-move.

**The nine added 4 ก.ย. 2569 are the exception, and SKILL.md marks them
*designed* rather than quoting them flat.** `search-query-reveal`,
`dot-loader-morph`, `card-expand-hero`, `caption-montage`, `aurora-prompt-box`,
`ship-it-toolbar`, `spring-stat-tiles`, `shared-axis-steps` and `device-reveal`
were written the other way round — the last keyframe was placed *on* the number
the scene wanted, so 6.0 / 4.0 / 5.0 / 8.0 / 8.0 / 6.0 / 5.0 / 6.4 / 7.0 are where
the animation ends by construction rather than where a render was observed to
stop. `aurora-prompt-box`
is the strictest of the eight: its 8.0s is exactly one turn of the border glow,
so shortening it leaves the hue stopped mid-rotation. Each was stepped frame by frame in a browser
at several times on the clock and looks right at each, which is not the same
evidence and should not be quoted as if it were. First person to render one:
replace the word *designed* in the table with what came back.

Seven of the motion files write their timeline into a header comment
(`Timeline (5s @ 60fps): corner marks draw -> eyebrow fades -> ...`). **Read it
for the order, not for the length.** Every one of those seven numbers is the
length its author meant; every one of them is longer than what the renderer
actually produces, because the last keyframe lands earlier than the comment
says. Measured on this machine, 31 ส.ค.: `minimal-hero` says 5s and renders
3.8s, `offset-title` says 4.5s and renders 2.8s, `section-marker` says 4s and
renders 2.2s. The `Length` column in SKILL.md's table is the measured number,
and it is the one to quote to a user.

## Google Fonts is the one thing still fetched at render time

40 of the 45 motion scenes pull typefaces over a `<link>` — every one except
`bold-portrait-title`, `structured-grid`, `startup-pitch`, `slideshow-demo`
and `motion-blur`; counted 31 ส.ค., recounted 1 ก.ย. when four scenes
arrived from the hyperframes registry (only `airbnb-deck` among them fetches),
and again 4 ก.ย. when the nine product-ad scenes landed, all of which fetch,
rather than estimated, because the number here said 17 for a while and was
wrong. Unlike
Tailwind and GSAP this one cannot be cured by putting a file on the machine: the
renderer asks `fonts.googleapis.com` for a stylesheet on every render, subsetted
to the exact characters on the page, and only the font binaries it names are
cached afterwards. Change the words and it asks again.

If the request fails the frames record the fallback face. That is not a crash
and not an error message: it is a scene that came out looking wrong for a reason
nothing in the output explains. Check the render, do not assume it.

## ไทย: 13 of the 70 scenes still have no Thai typeface, and they are all folders

Counted 31 ส.ค. over the original 47 scenes: 26 font families between them and
not one covers Thai; the four scenes vendored 1 ก.ย. add only `Nunito Sans`
(airbnb-deck), which does not either. Two cover Chinese (`Noto Sans SC`, `Noto Serif SC`) and
the rest are Latin. There is no `IBM Plex Sans Thai`, no `Sarabun`, no `Noto Sans
Thai`, nowhere.

**What that means for a Thai brief, which is most of them.** Thai text does not
come out blank — Chrome falls back to whatever Thai face the machine has, usually
Leelawadee UI or Tahoma. So the render succeeds, the words are readable, and the
typography is somebody else's: a system UI face sitting inside a layout drawn
around Shrikhand or Libre Baskerville, at a weight and a rhythm that were never
chosen. It looks wrong in a way that is obvious to a person and invisible to
every check in this pipeline. `video check` passes it. `video_ocr` reads the
letters back correctly and says nothing.

**The nine product-ad scenes added 4 ก.ย. 2569 are the exception**, and they are
the first on this shelf that are. `search-query-reveal`, `dot-loader-morph`,
`card-expand-hero`, `caption-montage`, `aurora-prompt-box`, `ship-it-toolbar`,
`spring-stat-tiles`, `shared-axis-steps` and `device-reveal` each name `Anuphan` — Thai and Latin,
geometric — immediately after their Latin display face, with `IBM Plex Sans Thai`
behind it. Per-glyph fallback does the rest: Latin resolves to Google Sans, Thai
resolves to Anuphan, and nothing lands in Leelawadee by accident. Nothing needs
saying before rendering those four. `references/google-register.md` has the
reasoning and the reason their `<link>` elements are one per family.

**For the other 51, say it before the render.** When the copy is Thai, tell the
user that the scene's own typeface does not cover Thai and that the headline will
land in a system font. Then either agree on that, or add a Thai family to the scene's
`<link>` and set it first in the `font-family` stack for the Thai elements —
`Noto Sans Thai` and `IBM Plex Sans Thai` sit closest to what this library
already uses. Do not quietly render and hand it over.

## Frame size is stated in every file, and has to be

Every scene declares its frame on its composition root: on `<body>` for the 32
single-file scenes, on the root `<div>` for the 13 folders. 39 of the 45 say
`data-width="1920" data-height="1080"`; the other **six are portrait, 1080x1920**:
`bold-portrait-title`, which was the only one for a long time, and the five-scene
vertical set added 4 ก.ย. 2569.

**Stated because the renderer does not guess it.** A scene with no declared
frame renders at 1080x1920 whatever its CSS says — it crops a design drawn 1920
wide, and nothing in the output says so. Measured 31 ส.ค. on the owner's
machine: that is how all 13 single-file scenes came out until the frame was
written into them.

Changing the aspect is those two numbers, plus whatever the scene's own CSS
pins. Six of them — `glitch-title`, `typewriter-cursor`, `bar-chart-counter`,
`logo-outro`, `liquid-gradient-hero`, `cinematic-light-leak` — lay themselves
out fluidly, so for those it is only the two numbers. The other 34 draw their own
1920x1080 box in CSS as well, and moving the frame without editing that rule
leaves the scene drawing its old box inside the new one. The `Frame` column in
SKILL.md's table says which is which.

## The sample copy is real copy, not placeholders

There is no `{{ }}` in any of these. Every headline, figure and label is a
finished sentence somebody wrote to show the layout working, which means two
things.

Replace all of it. A scene that ships with one line of the sample copy still in
it is the most visible kind of unfinished work, and it is always the line nobody
looked at because it was small.

And respect the length it was drawn for. These layouts were built around text of
roughly the length that is in them. A headline three times longer does not wrap,
it overflows the frame, and the markup will not tell you: only opening it will.
Preview before you render, and read the render back — `video render` with
`proof: true` OCRs its own output in the same reply.

## The explainer diagram's empty middle

The explainer diagram's middle is empty on purpose — no `{{ }}` placeholder,
just a box sized to whatever fills it. Its two small brand marks (icon, logo)
are the kind of image `media_fetch` saves straight off a real site; the
diagram itself is the one thing in this whole library nothing here draws for
you; either find one that already exists for the exact topic, or draw it as
inline SVG the way `decision-flowchart` and `radial-node-diagram` do. Aetox
generates no image via any paid API — that choice was made once, generally,
and applies here rather than being decided again per scene.

## Where these came from

`CREDITS.md` beside this file records the source repository, the licence and the
upstream design lineage of every one of the 70 files, including six that were
themselves derived from other design skills and nineteen that were written here. The licences are Apache-2.0 and
MIT. Both allow commercial use and both require the attribution to travel with
the work.

So the credit file is not paperwork, it is part of the library. If a scene is
copied out into somewhere else, its line in `CREDITS.md` goes with it.

## Three scenes have no background, and that is the point

`lower-third-name`, `caption-bar` and `stat-strap` are **overlays**. Their `body`
declares no background at all, so everything outside the graphic is transparent.
No other scene on this shelf does that, and it changes two things.

**Mounted, they work as intended.** A host `index.html` pointing at one with
`data-composition-src` composites it over the A-roll underneath, the same
mechanism `grain-texture-hero` uses for its own caption layer. That is what they
are for.

**Rendered on their own, you get the graphic on black.** The renderer fills what
is not painted, and black is what it fills with. That is not a failure and it is
also not a clip anybody wants. Each of the three carries the one line to paste
into `body` to preview it against a still, and a reminder to take it out again —
left in, it is the overlay equivalent of shipping the sample copy.

**Two of them carry a scrim and it is load-bearing.** Footage changes brightness
between shots; white type over a bright frame is not dim, it is gone. The soft
dark gradient behind the type in `lower-third-name` and `caption-bar` is the only
thing standing between the graphic and that, and it is the first thing somebody
will try to delete to "clean it up". `stat-strap` has none because its card is
already opaque enough to be its own.

## Vertical is not landscape rotated

Six scenes are 1080x1920 and five of them arrived together as a set. Turning a
landscape scene on its side does not produce any of them, for one reason:

**A 1080x1920 frame is not 1080x1920 of usable space.** Every platform draws its
own interface over the video — tabs and search at the head, caption, handle and
audio credit at the foot, a column of buttons up the right. Anything outside the
intersection of those is covered, the render is still correct, and nothing in this
pipeline will tell you. Checked 4 ก.ย. 2569 against published guides
(kreatli.com/guides/safe-zone-guide and adaptlypost.com's 2026 round-up), which
agree: head 150-200px, foot 250-300px (Reels takes up to ~500, Shorts ~320), right
120px, and 900x1400 centred is safe on all three.

The set takes the cautious end — head 200, foot 300, sides 120 — and holds every
readable element inside it, in three variables named `--safe-x`, `--safe-top` and
`--safe-bottom`. The background deliberately runs past them so the frame still
fills a phone edge to edge. Two consequences worth knowing before editing one:

- `vertical-beats` puts its beat ticker at the **top**. Its landscape cousin
  `caption-montage` puts the same ticker at the foot, which in a vertical frame is
  underneath the platform's caption row.
- Captions are set over three or four short lines. At 1080 wide minus 240 of
  margin, a sentence that fits a landscape card wraps to five lines. Keep each one
  under about six words.

The same applies to the overlays if you take them vertical: each says which
number to move and to what.

## Every scene that animates now has a reduced-motion fallback

Three vendored scenes did not, until 4 ก.ย. 2569: `bar-chart-counter`,
`logo-outro` and `typewriter-cursor` all animated and carried no
`@media (prefers-reduced-motion: reduce)` block at all.
`aetox-design-system`'s `references/motion.md` lists "motion with no
`prefers-reduced-motion` fallback" as something to flag on sight, so the shelf was
failing its own rule. Each now has one, pinning the state the scene was heading
for rather than hiding anything.

**`cinematic-light-leak` is the exception and does not need one.** Its only
`@keyframes` lives inside the baked Tailwind block and no element in it declares
`animation` at all — it is a painted frame held for five seconds. A grep for
`@keyframes` flags it; a grep for `animation:` does not, and the second one is the
question worth asking.

**One of the three was a trap worth writing down.** `logo-outro`'s wordmark is
painted with `background-clip: text` over `color: transparent`, so switching its
animation off and stopping there parks the gradient where it is mostly
transparent and the 84px wordmark **disappears**. Its block therefore undoes the
clip and gives the colour back as well; measured in a browser, `.shimmer` goes
from `rgba(0,0,0,0)` to `rgb(245,245,247)` under the override. Any scene that
paints text with a moving gradient has this same hole in it — check the text is
still there, not just that the motion stopped.

All nineteen scenes written here shipped with the block from the start.
