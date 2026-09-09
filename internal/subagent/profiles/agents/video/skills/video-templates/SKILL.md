---
name: video-templates
description: ฉากวิดีโอที่ก๊อปไปใช้ได้เลย 75 ฉาก ไม่ใช่คำอธิบายว่าฉากควรหน้าตาแบบไหน แบ่งเป็นฉากเคลื่อนไหว 50 แบบ ครบสามอัตราส่วน — นอน 39 ตั้ง 6 จัตุรัส 5สำหรับเรนเดอร์เป็นคลิป (รวมเด็คนักลงทุนและพิตช์เด็คแบบเต็มเรื่อง กับอีก 9 ฉากในภาษาโฆษณาโปรดักต์ที่ถอดจากสเปกและจากการหยุดเฟรมดูคลิปจริง สี่ในนั้นเคลื่อนด้วยสปริงจริงของ Material 3 Expressive และทั้งแปดเป็นฉากเดียวบนชั้นที่ตัวอักษรไทยมีฟอนต์ที่เลือกไว้) และฉากนิ่ง 25 แบบสำหรับปก อินโฟกราฟิก สไลด์ข้อมูล ไดอะแกรมอธิบาย และหน้าจอจำลอง ส่วนใหญ่เป็น HTML ไฟล์เดียวจบ อีก 13 ฉากเป็นโฟลเดอร์เพราะมีฉากย่อยกับไฟล์ประกอบของตัวเอง ใช้ตอนรู้แล้วว่าจะทำฉากรูปไหน แล้วอยากรู้ว่าในชั้นมีแถวที่เป็นรูปนั้นอยู่แล้วหรือไม่
---

# Video scene templates

Markup, not advice about markup. Every scene here was rendered and watched by
the people who wrote it, so what a row gives you that an empty page cannot is
timing somebody has already seen move.

Read this as a reference, not a menu. Decide the shape of the piece first, then
look for the row that is already that shape. Where one is, take it with
`video new` and change the words. Where none is, the scene is yours to write,
and that is an ordinary answer rather than a fallback.

**All four shelves are reachable and every spelling in these tables works.**
`video new social-cover-editorial`, `video new graphic-scenes/social-cover-editorial.html`
and `video new motion/product-launch-30s/index.html` all name the scene they
look like they name. No name appears on two shelves, so the bare one is never
ambiguous. Until 7 ก.ย. 2569 only the motion shelf could be reached and the
other twenty-five names in these tables were refused, which is worth knowing if
you carry a memory of being told one of them did not exist.

**`video new blank`** is the renderer's own empty composition — a frame, a
registered timeline and nothing else. It is what a scene of your own starts
from, and it is a first-class start: reach for it whenever the piece you are
building is not one of the rows below. Copy a row because it is the shape, not
because copying is the safer move.

**The tables below are the complete inventory.** The listing this tool appends
after the document is capped at 40 entries and the library holds over 100
files, so most of it does not appear there. The tables are what to read; you do
not need to open a scene's files before choosing it — `video new` reports every
file it copies and the text inside each one.

## The rules that bite, one line each

`GUIDE.md` beside this file carries the why and the measurements behind every
line. Read the line here; open the guide's section only when you are about to
work against one. `references/google-register.md` is narrower: the house style
the nine product-ad scenes share, and the only file you need if you are writing
a tenth one in it — and, under "The edit, measured", the only guidance anywhere
here on cutting several of them into one piece.

- **Replace all the sample copy.** No `{{ }}` anywhere — every line is real
  copy, and one left in is the most visible unfinished work. Keep roughly its
  length: a headline three times longer silently overflows the frame.
- **ไทย: only the 13 folder scenes still have no Thai typeface.** Every one of the
  62 single-file scenes now names a Thai face behind its Latin one — `Anuphan`,
  `Noto Serif Thai` or `IBM Plex Sans Thai` depending on what the stack ends in —
  so Thai copy lands somewhere chosen instead of in Leelawadee by accident. For
  the 13 folders, say it before rendering. `caption-bar`, `stat-strap` and
  `lower-third-name` go further and ship Thai sample copy.
- **13 of the 50 motion scenes are folders** (the rows whose path ends
  `/index.html`); the other 37 are one file each.
  `video new` copies the whole folder; by hand, never take the host alone — it
  renders an empty frame that looks like a working render.
- **6 scenes carry baked Tailwind CSS** (`bar-chart-counter`,
  `cinematic-light-leak`, `glitch-title`, `liquid-gradient-hero`, `logo-outro`,
  `typewriter-cursor`): a class not already in use does nothing, silently.
  Write plain CSS for anything new.
- **`Length` is what a real render produced**, every row, safe to quote to a user.
  The 24 written here were rendered 4 ก.ย. 2569 and probed: 24 of 24 came back at
  exactly their stated length, exactly their stated frame, and the frame count the
  duration implies. In the 37 flat scenes the number is `data-duration` on
  `<body>` — change it to change the clip. Header-comment timelines state the
  order, not the length.
- **Everything renders at 30fps by default** and no scene sets `data-fps`, so the
  engine's default is what every clip gets unless `video render` is given one.
  Measured across all 24: one distinct frame rate, 30. `fps` takes an ffmpeg
  rational as text too — `"24000/1001"` for 23.976, `"30000/1001"` for 29.97 —
  which is how a clip cut against filmed footage stops juddering.
- **The frame is declared in every file** and the renderer never guesses it.
  "fluid CSS" rows re-aspect by changing the two `data-` numbers; the rest
  also draw their own 1920x1080 box in CSS that has to move with them.
- **The Frame column is the shape, not the size.** `video render resolution:` can
  deliver any row at 4K — `landscape-4k` 3840x2160, `portrait-4k` 2160x3840,
  `square-4k` 2160x2160 — without touching the scene: it raises the capture
  density and the markup is untouched. What it cannot do is change the aspect
  ratio, so the Frame column still decides the shape and only the shape. Pick the
  row whose shape the piece needs; pick the resolution when you render.
- **The library has no 4:5 scene.** The four shapes in general use are 16:9,
  9:16, 1:1 and 4:5 (1080x1350, the feed on Instagram, Facebook and LinkedIn).
  This shelf covers the first three — 33 rows at 1920x1080, 6 at 1080x1920, 5 at
  1080x1080 — and none of the seventy-five is 4:5. There is no preset for it
  either. A 4:5 piece means a "fluid CSS" row with both `data-` numbers and its
  own CSS box moved to 1080x1350, or `video new blank` and a scene written for
  it. Say which you did.
- **Google Fonts is fetched at render time** (45 of 50 motion scenes). A failed
  fetch is not an error — it is a wrong-looking scene. Check the render.
- **`__VIDEO_SRC__` / `__VIDEO_DURATION__` are engine slots**, not broken
  markup. Leave them in the library; `video new` fills or strips them.
- **Length lives in the markup, and `seconds` reaches four scenes.** The four
  marked *as asked* carry `__VIDEO_DURATION__` and take the number `video new`
  is given; the other 46 have theirs written in by the author, and passing
  `seconds` for one of them changes nothing — `video new` now says so instead of
  going quiet. `video render` has no length option at all and neither does the
  renderer. **To make a scene a different length**, edit `data-duration` on the
  root element *and* the keyframes that run inside it: the two are one number in
  two places, and moving only the first gives a held picture at the end or a
  move cut off in the middle. Changing a length is a real edit, so prefer the
  scene whose length is already near — every row below states one.
- **`CREDITS.md` travels with the work** — licence and lineage of all 75 files
  (Apache-2.0 and MIT for the 51 vendored, ours for the 24 written here;
  attribution required either way). A scene copied elsewhere takes its line
  along.
- **A scene built by hand gets no rewrites.** `video new` is what points GSAP
  at the local `vendor/gsap.min.js`; a project you write from scratch keeps
  whatever CDN address you typed and needs the network at render time. Start
  from `video new` even for a wholly new design, then replace its markup.

## motion/ — 50 animated scenes

These are the ones that become video. Four folders say *as asked*: they carry
`__VIDEO_DURATION__`, so the `seconds` given to `video new` is their length and
the number beside it is the fallback. Every other length is fixed by the scene.

**Read the Length column as part of the choice, not as trivia.** It is the one
column that cannot be changed by editing words, and the piece that ends up four
seconds long is the piece whose scene was four seconds long. No row here is the
default for a kind of brief; pick on shape and on length together, and when two
rows fit, say which two and why you took one — that sentence is the part a user
can correct.

| File | What it is | Length | Frame |
|:--|:--|:--|:--|
| `motion/statement-title.html` | 1970s editorial poster in motion: giant figure, three tilted headline lines, one red accent | 3.0s | 1920x1080 |
| `motion/minimal-hero.html` | Luxury whitespace hero. Corner marks draw, hero word reveals letter by letter, gold rule extends | 3.8s | 1920x1080 |
| `motion/offset-title.html` | Electric blue and dark panels sliding offset, with a handwritten script stroke | 2.8s | 1920x1080 |
| `motion/bold-portrait-title/index.html` | Six-column grid title in the Vignelli manner. **The one portrait scene** | as asked, 10s | 1080x1920 |
| `motion/glitch-title.html` | Signal-lost title: scanlines, chromatic break, hard cuts. Endless loop, cut to one 4s cycle on `<body>` | 4.0s | 1920x1080, fluid CSS |
| `motion/kinetic-type/index.html` | Words as the subject, moving. Use when the sentence is the whole scene | 15s | 1920x1080 |
| `motion/typewriter-cursor.html` | A statement under a blinking cursor, with directional light rays. **Nothing types** — the cursor is the only motion, held 4s on `<body>` | 4.0s | 1920x1080, fluid CSS |
| `motion/section-marker.html` | A break between parts: section number rolls in, coloured card slides over dark | 2.2s | 1920x1080 |
| `motion/split-quote.html` | Two panels split open, a quote revealed line by line, attribution after | 3.2s | 1920x1080 |
| `motion/stat-anchor.html` | Swiss grid around one giant statistic, bars growing under it | 2.4s | 1920x1080 |
| `motion/bar-chart-counter.html` | Bars growing with the figures counting up beside them | 1.8s | 1920x1080, fluid CSS |
| `motion/editorial-chart/index.html` | Newspaper-style chart, drawn rather than plotted | 15s | 1920x1080 |
| `motion/decision-flowchart/index.html` | A decision tree drawing itself branch by branch | 15s | 1920x1080 |
| `motion/radial-node-diagram.html` | Soft-tech radial node graph on a frosted card, links drawing outward | 3.1s | 1920x1080 |
| `motion/structured-grid/index.html` | Swiss grid frame for A-roll with overlay layers. Use it to hold footage, not to replace it | as asked, 10s | 1920x1080 |
| `motion/product-showcase/index.html` | Product promo: the thing, its claim, its call to action | 20s | 1920x1080 |
| `motion/product-launch-30s/index.html` | A whole 30-second promo: eight mounted scenes and four audio tracks. The shelf’s widest launch piece — right when the brief has eight things to say and 30s to say them, wrong when it has one, when the piece must be shorter, or when it is vertical. Its three product screenshots are 1x1 placeholders: put a `web-scenes` interface in their place | 30s | 1920x1080 |
| `motion/logo-outro.html` | Closing scene: logo with a shimmer sweep | 3.7s | 1920x1080, fluid CSS |
| `motion/grain-texture-hero/index.html` | Warm cream hero with film grain over it | as asked, 10s | 1920x1080 |
| `motion/liquid-gradient-hero.html` | Aurora blobs drifting behind a hero line. Honours `prefers-reduced-motion`. Endless drift, cut to 6s on `<body>` | 6.0s | 1920x1080, fluid CSS |
| `motion/cinematic-light-leak.html` | Film stock: scratches, perforations, vignette, warm leaks. **No `@keyframes` at all** — a painted frame held 5s, a backdrop rather than a shot | 5.0s | 1920x1080, fluid CSS |
| `motion/playful-bounce/index.html` | Bouncy, informal motion. The one register the rest of this library does not have | as asked, 10s | 1920x1080 |
| `motion/airbnb-deck/index.html` | Twelve-beat investor deck in motion, with its own per-beat SFX. The fullest production on the shelf, and 108s is a long way to commit — right for a deck with twelve real beats, wrong as a default for anything deck-shaped. `startup-pitch` is the same shape at 84s and `slideshow-demo` at 32s | 108s | 1920x1080 |
| `motion/startup-pitch/index.html` | Pitch deck in motion: cover, problem, product, market with a drill-down, ask | 84s | 1920x1080 |
| `motion/slideshow-demo/index.html` | Slideshow with branching scenes and a CTA, mounted from one host | 32s | 1920x1080 |
| `motion/motion-blur/index.html` | Title card with real per-frame motion blur; the word drifting through frame IS the design (check reports its overflow on purpose) | 4.0s | 1920x1080 |
| `motion/search-query-reveal.html` | A question typed into a search field, suggestions dropping, then the field itself growing into the answer card. **The one scene where anything types** | 6.0s | 1920x1080 |
| `motion/dot-loader-morph.html` | Four dots bounce, close ranks into one four-coloured bar, and the bar becomes the rule under the headline. A title card wearing a loader's clothes | 4.0s | 1920x1080 |
| `motion/card-expand-hero.html` | Dark stage, a rail of cards lit from behind, and one of them grows into the whole frame. The only true container transform on the shelf | 5.0s | 1920x1080 |
| `motion/caption-montage.html` | Five questions hard-cutting every 1.2s over colour fields, then one answer held for two seconds. Ships footage-free on purpose — the file says why | 8.0s | 1920x1080 |
| `motion/aurora-prompt-box.html` | A prompt box whose border glow rotates hue over the whole scene, a cursor that picks a chip, and the chip becoming a tag inside the box. **The one scene with a pointer in it** | 8.0s | 1920x1080 |
| `motion/ship-it-toolbar.html` | The finished thing under its toolbar, a cursor pressing Install, warm light flooding the frame, then a hard cut to a mark alone on black. Pairs with `aurora-prompt-box` as the closing half | 6.0s | 1920x1080 |
| `motion/spring-stat-tiles.html` | Four figures on tiles that overshoot and settle on real simulated springs, counting up in pure CSS. **The one scene whose motion is physics, not a drawn curve** | 5.0s | 1920x1080 |
| `motion/shared-axis-steps.html` | Four steps handing over along one axis, old leaving left as new arrives from right. The M3 pattern nothing else here could do | 6.4s | 1920x1080 |
| `motion/device-reveal.html` | A portrait screen rising into a near-black frame while the whole stage brightens with it, then stepping aside for a panel under warm light. **The one scene that holds a device** | 7.0s | 1920x1080 |
| `motion/vertical-title.html` | Opening card of the vertical set: kicker, three stacked lines, four-hue rule. **Carries the safe-area numbers the whole set uses** | 4.0s | 1080x1920 |
| `motion/vertical-stat.html` | One figure at 260px counting up, unit beside it, two supporting rows | 4.0s | 1080x1920 |
| `motion/vertical-beats.html` | Five lines cut every 1.2s and one answer held, ticker at the **top** where the platform cannot cover it | 8.0s | 1080x1920 |
| `motion/vertical-showcase.html` | The thing on top, its claim under it, one outlined button last — the reading order vertical is actually better at | 6.0s | 1080x1920 |
| `motion/vertical-outro.html` | Wordmark, one line, four pips. The colour drains before the name lands | 4.0s | 1080x1920 |
| `motion/lower-third-name.html` | Who is speaking. **Overlay: transparent, no background** — mount it over footage | 5.0s | 1920x1080 |
| `motion/caption-bar.html` | What is being said, one line at a time on hard cuts. **Overlay**, and its sample copy is Thai | 6.0s | 1920x1080 |
| `motion/stat-strap.html` | A figure landed top-right while the shot keeps running. **Overlay**, no full-width scrim | 5.0s | 1920x1080 |
| `motion/split-compare-wipe.html` | Before wiped away to reveal after — and the seam **parks at 42%** so the frame ends holding both | 5.5s | 1920x1080 |
| `motion/roadmap-dates.html` | Milestones with real dates, a line drawn as far as today, dashed rail after it | 6.5s | 1920x1080 |
| `motion/square-title.html` | Opening card of the square set, centred because a square has no reading direction. **Carries the safe-area number the set uses** | 4.0s | 1080x1080 |
| `motion/square-quad.html` | A 2x2 of equal cells, one talking and three carrying figures. **The one composition a square does better than either other frame** | 5.0s | 1080x1080 |
| `motion/square-stat.html` | One figure at 300px, centred, sized to survive being scrolled past small | 4.0s | 1080x1080 |
| `motion/square-beats.html` | Five lines cut on the beat, written for a feed card playing **with the sound off** | 8.0s | 1080x1080 |
| `motion/square-outro.html` | Wordmark, one line, an outlined button — a feed card has to say what to do next itself | 4.0s | 1080x1080 |

### The square set — five scenes for a feed, where the sound is off

`square-title`, `square-quad`, `square-stat`, `square-beats`, `square-outro`.
1080x1080, one margin variable, `--safe: 152px`.

Square is not a crop of either other frame, for three reasons that each change a
layout. Its margin is **equal on all four sides**, because a feed post has almost
no platform furniture over it and what it has is symmetrical — Instagram treats
about 1000x1000 centred as safe, Facebook asks for 14% a side, and this set takes
Facebook's as the cautious one. It has **no reading direction**, so its cards are
centred where the vertical ones are top-weighted. And **the sound is off**: feed
video autoplays muted, so whatever a voice would have said is on the frame or it
is not in the video, which is why `square-beats` is set at 92px with four words to
a line.

`square-quad` is the one that could not be built at any other aspect. Four equal
cells in 16:9 come out as a row of short wide boxes; in 9:16 they come out as a
column, which the eye reads as a ranked list. Only at 1:1 do four things read as
four things **at once**. If the brief is "these four go together", that is the
frame, and this is the scene.

### The vertical set — five scenes, and the only way to cut a phone-shaped piece

`vertical-title`, `vertical-stat`, `vertical-beats`, `vertical-showcase`,
`vertical-outro`. Before these landed the shelf had **one** portrait scene out of
thirty-five, which meant a vertical brief could be opened and not finished. These
five are an open, a figure, a montage, a product beat and a close — enough to cut
a whole piece without turning a landscape scene on its side.

They share a stage and three variables so they cut together, and the variables
are the point: `--safe-x: 120px`, `--safe-top: 200px`, `--safe-bottom: 300px`.
A 1080x1920 frame is not 1080x1920 of usable space — every platform draws tabs at
the head, a caption and handle at the foot, and a button column up the right side,
and text outside the intersection is simply covered with nothing in this pipeline
warning you. `vertical-title.html` carries the full reasoning and the sources.
Move content by moving those three variables, never by nudging an element past
them.

### The three overlays — the only scenes here with no background

`lower-third-name`, `caption-bar`, `stat-strap`. Every other scene on this shelf
paints its own background; these must not, because they are meant to be composited
over footage. Their `body` is transparent.

- **Mounted** as a layer through `data-composition-src`, the way
  `grain-texture-hero/compositions/captions.html` is, they sit over the A-roll.
  That is the intended use.
- **Rendered alone** you get the graphic on black. Not a bug, not a useful clip.
  Each file says how to add a backdrop for previewing, and to take it out again.
- Two of the three carry a **scrim** — a soft dark gradient behind the type — and
  it is not decoration: footage changes brightness shot to shot, and white type on
  a bright frame is gone without it. `stat-strap` skips it because its card is its
  own scrim.
- They are positioned not to collide: the lower third takes bottom-left, the strap
  takes top-right. Check that still holds if you move either.

### The nine in the product-ad register

The nine rows above are one house style, written 4 ก.ย. 2569 in three passes.
They are the answer to "make it look like a real product ad", which the rest of
this shelf — editorial, film stock, Swiss grid — has no scene for.

The **first four** are built on published specification: Material 3's easing and
duration tokens, and the cross-fade split from Google's own Flutter code. The
**second four** went further, and the difference is worth knowing:

- Three of them and `spring-stat-tiles` do not use bezier curves at all.
  Material 3 Expressive states its motion as **springs**, and the damping and
  stiffness pairs are in the Android source. Each was integrated as a real
  spring and sampled into a CSS `linear()`. Spatial springs overshoot — the fast
  one by 9.4% — and effects springs never do. Put a spatial spring on an opacity
  and the overshoot is clamped, so the bounce silently becomes a stall.
- `shared-axis-steps` closes the M3 pattern set. Its slide is **160px, not a
  screen width**, because Google's own implementation moves 30dp.
- `aurora-prompt-box` and `ship-it-toolbar` came from watching an actual film
  frame by frame and sampling its colours off a canvas, not from reading about
  it. That produced: a border glow whose **hue rotates across the length of the
  piece**, a chip that becomes a tag inside the input, a cursor used as a
  character, and the finding that the shipping beat is **lit warmer than every
  beat before it**.

`device-reveal` came last and from a third pass over the same film, this time
capturing **every** frame rather than a handful — 1,250 samples at 25ms apart —
and differencing them. That gave the film's actual edit: **no cut at all for the
first 12.5 seconds**, then ten cuts at a median 1.6s apart, then a 3.06s held
ending. It also gave a luminance curve showing the frame sitting between L2 and
L15 out of 255 for fourteen seconds before jumping to L53 on the shot where the
built thing is finally shown. `device-reveal` is that jump. The full numbers,
and what they mean for assembling a piece out of these nine, are under
"The edit, measured" in `references/google-register.md` — read that before
cutting several of these together.

They are not an impression of anybody's film. Their motion is Material Design
3's published easing and duration tokens; their cross-fades use the 6/20 – 14/20
– scale-0.92 split from Google's own Flutter implementation; `card-expand-hero`'s
dark surface is measured off a public page on a stated date. What is *not* taken
— the marks, the brand hues, and the blue-red-yellow-green sequence, which reads
as a trademark whatever the hex values under it are — is listed in
`references/google-register.md` along with every number above, so a fifth scene
in this register can be written without re-deriving any of it.

Two of them are light (`search-query-reveal`, `dot-loader-morph`), one is dark
(`card-expand-hero`), and `caption-montage` is neither — it is six colour fields.
Pick a surface and stay on it across a piece; the light and dark halves of this
register do not cut together.

## The 25 still scenes

These do not move. They are for the frames around a video, and for the stills a
video job also needs: a cover, an infographic, a data slide, or a believable
interface on screen where a real screenshot would leak real data.

Each subject comes in the same three voices. Pick the voice once and stay in it
across a whole piece: *editorial* is print-like and typographic, *minimal* is
white space and restraint, *organic* is soft shapes and warmth.

| Subject | Files | Use when |
|:--|:--|:--|
| Social cover | `graphic-scenes/social-cover-editorial.html` and `-minimal` `-organic` | The still image a post needs beside the clip |
| Vertical infographic | `graphic-scenes/vertical-infographic-editorial.html` and `-minimal` `-organic` | Several facts read in order, on a phone |
| Data slide | `slide-scenes/data-slide-editorial.html` and `-minimal` `-organic` | A number and its context, held still long enough to read |
| SaaS landing page | `web-scenes/saas-landing-editorial.html` and `-minimal` `-organic` | A product page on screen inside a scene |
| Portfolio home | `web-scenes/portfolio-home-editorial.html` and `-minimal` `-organic` | A personal or studio site on screen |
| API docs | `web-scenes/api-docs-editorial.html` and `-minimal` `-organic` | Developer documentation on screen |
| Directory nav | `web-scenes/directory-nav-editorial.html` and `-minimal` `-organic` | A listing or catalogue interface on screen |
| Writing tool | `web-scenes/writing-tool-editorial.html` and `-minimal` `-organic` | An editor or writing app on screen |
| Explainer diagram | `graphic-scenes/explainer-diagram-poster.html` | A single branded "how X works" poster — icon, title, one diagram, footer |

`video new` takes these like any other scene and gives the copy the composition
root the renderer needs, at the size the scene already draws itself — a cover
comes out 1200x510, an infographic 1080x1920, a slide 1920x1080, an interface
1440x900, the poster 1200x1581. Without that frame the renderer makes every
project 1080x1920 and crops the rest away silently, which is why these were held
back before it existed.

**What that does not change is what they are for.** One of these rendered on its
own is a video of a photograph. Use them as the material a moving scene mounts
or screenshots, or as the still the job also needs — a cover beside the clip, a
poster, an interface on screen where a real screenshot would leak real data.
`motion/product-launch-30s` ships three 1x1 transparent PNGs where its product
screenshots go, and the five `web-scenes` subjects are what belongs there.

The three-voice subjects share one file name with `-editorial`, `-minimal` or
`-organic` on the end; the explainer diagram is one design, not three, and its
empty middle is on purpose — the diagram is the one thing this library does not
draw for you (GUIDE.md, "The explainer diagram's empty middle").
