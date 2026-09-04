---
name: video-templates
description: ฉากวิดีโอที่ก๊อปไปใช้ได้เลย 60 ฉาก ไม่ใช่คำอธิบายว่าฉากควรหน้าตาแบบไหน แบ่งเป็นฉากเคลื่อนไหว 35 แบบสำหรับเรนเดอร์เป็นคลิป (รวมเด็คนักลงทุนและพิตช์เด็คแบบเต็มเรื่อง กับอีก 9 ฉากในภาษาโฆษณาโปรดักต์ที่ถอดจากสเปกและจากการหยุดเฟรมดูคลิปจริง สี่ในนั้นเคลื่อนด้วยสปริงจริงของ Material 3 Expressive และทั้งแปดเป็นฉากเดียวบนชั้นที่ตัวอักษรไทยมีฟอนต์ที่เลือกไว้) และฉากนิ่ง 25 แบบสำหรับปก อินโฟกราฟิก สไลด์ข้อมูล ไดอะแกรมอธิบาย และหน้าจอจำลอง ส่วนใหญ่เป็น HTML ไฟล์เดียวจบ อีก 13 ฉากเป็นโฟลเดอร์เพราะมีฉากย่อยกับไฟล์ประกอบของตัวเอง ใช้ตอนกำลังจะทำฉากขึ้นใหม่แล้วไม่อยากประดิษฐ์เองทุกครั้ง
---

# Video scene templates

Markup that gets copied, not advice about markup. Pick from the tables, take it
with `video new`, change the words. A scene written from nothing is a scene
whose timing nobody has tested; every one of these has been rendered by the
people who wrote it.

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
- **ไทย: 51 of the 60 scenes have no Thai typeface.** Thai copy renders in a
  system font — readable, chosen by nobody. Say so before rendering, or put a
  Thai family (`Noto Sans Thai`, `IBM Plex Sans Thai`) first in the stack. The
  nine exceptions are the product-ad scenes, which carry `Anuphan` behind their
  Latin face and need nothing said.
- **13 of the 35 motion scenes are folders** (the rows whose path ends
  `/index.html`); the other 22 are one file each.
  `video new` copies the whole folder; by hand, never take the host alone — it
  renders an empty frame that looks like a working render.
- **6 scenes carry baked Tailwind CSS** (`bar-chart-counter`,
  `cinematic-light-leak`, `glitch-title`, `liquid-gradient-hero`, `logo-outro`,
  `typewriter-cursor`): a class not already in use does nothing, silently.
  Write plain CSS for anything new.
- **`Length` is what a real render produced**, safe to quote to a user — with
  nine exceptions, marked *designed* in the table: the product-ad scenes were
  built so the last keyframe lands on the stated number, but nobody has rendered
  one yet, so quote those as intended rather than as measured until someone has.
  In the 22 flat scenes the number is `data-duration` on `<body>` — change it to
  change the clip. Header-comment timelines state the order, not the length.
- **The frame is declared in every file** and the renderer never guesses it.
  "fluid CSS" rows re-aspect by changing the two `data-` numbers; the rest
  also draw their own 1920x1080 box in CSS that has to move with them.
- **Google Fonts is fetched at render time** (30 of 35 motion scenes). A failed
  fetch is not an error — it is a wrong-looking scene. Check the render.
- **`__VIDEO_SRC__` / `__VIDEO_DURATION__` are engine slots**, not broken
  markup. Leave them in the library; `video new` fills or strips them.
- **`CREDITS.md` travels with the work** — licence and lineage of all 60 files
  (Apache-2.0 and MIT for the 51 vendored, ours for the 9 written here;
  attribution required either way). A scene copied elsewhere takes its line
  along.
- **A scene built by hand gets no rewrites.** `video new` is what points GSAP
  at the local `vendor/gsap.min.js`; a project you write from scratch keeps
  whatever CDN address you typed and needs the network at render time. Start
  from `video new` even for a wholly new design, then replace its markup.

## motion/ — 35 animated scenes

These are the ones that become video. Four folders say *as asked*: they carry
`__VIDEO_DURATION__`, so the `seconds` given to `video new` is their length and
the number beside it is the fallback. Every other length is fixed by the scene.

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
| `motion/product-launch-30s/index.html` | A whole 30-second promo: eight mounted scenes and four audio tracks. Start here when the brief is a launch clip | 30s | 1920x1080 |
| `motion/logo-outro.html` | Closing scene: logo with a shimmer sweep | 3.7s | 1920x1080, fluid CSS |
| `motion/grain-texture-hero/index.html` | Warm cream hero with film grain over it | as asked, 10s | 1920x1080 |
| `motion/liquid-gradient-hero.html` | Aurora blobs drifting behind a hero line. Honours `prefers-reduced-motion`. Endless drift, cut to 6s on `<body>` | 6.0s | 1920x1080, fluid CSS |
| `motion/cinematic-light-leak.html` | Film stock: scratches, perforations, vignette, warm leaks. **No `@keyframes` at all** — a painted frame held 5s, a backdrop rather than a shot | 5.0s | 1920x1080, fluid CSS |
| `motion/playful-bounce/index.html` | Bouncy, informal motion. The one register the rest of this library does not have | as asked, 10s | 1920x1080 |
| `motion/airbnb-deck/index.html` | Twelve-beat investor deck in motion, with its own per-beat SFX. The fullest production on the shelf — start here for a deck-shaped video | 108s | 1920x1080 |
| `motion/startup-pitch/index.html` | Pitch deck in motion: cover, problem, product, market with a drill-down, ask | 84s | 1920x1080 |
| `motion/slideshow-demo/index.html` | Slideshow with branching scenes and a CTA, mounted from one host | 32s | 1920x1080 |
| `motion/motion-blur/index.html` | Title card with real per-frame motion blur; the word drifting through frame IS the design (check reports its overflow on purpose) | 4.0s | 1920x1080 |
| `motion/search-query-reveal.html` | A question typed into a search field, suggestions dropping, then the field itself growing into the answer card. **The one scene where anything types** | 6.0s, designed | 1920x1080 |
| `motion/dot-loader-morph.html` | Four dots bounce, close ranks into one four-coloured bar, and the bar becomes the rule under the headline. A title card wearing a loader's clothes | 4.0s, designed | 1920x1080 |
| `motion/card-expand-hero.html` | Dark stage, a rail of cards lit from behind, and one of them grows into the whole frame. The only true container transform on the shelf | 5.0s, designed | 1920x1080 |
| `motion/caption-montage.html` | Five questions hard-cutting every 1.2s over colour fields, then one answer held for two seconds. Ships footage-free on purpose — the file says why | 8.0s, designed | 1920x1080 |
| `motion/aurora-prompt-box.html` | A prompt box whose border glow rotates hue over the whole scene, a cursor that picks a chip, and the chip becoming a tag inside the box. **The one scene with a pointer in it** | 8.0s, designed | 1920x1080 |
| `motion/ship-it-toolbar.html` | The finished thing under its toolbar, a cursor pressing Install, warm light flooding the frame, then a hard cut to a mark alone on black. Pairs with `aurora-prompt-box` as the closing half | 6.0s, designed | 1920x1080 |
| `motion/spring-stat-tiles.html` | Four figures on tiles that overshoot and settle on real simulated springs, counting up in pure CSS. **The one scene whose motion is physics, not a drawn curve** | 5.0s, designed | 1920x1080 |
| `motion/shared-axis-steps.html` | Four steps handing over along one axis, old leaving left as new arrives from right. The M3 pattern nothing else here could do | 6.4s, designed | 1920x1080 |
| `motion/device-reveal.html` | A portrait screen rising into a near-black frame while the whole stage brightens with it, then stepping aside for a panel under warm light. **The one scene that holds a device** | 7.0s, designed | 1920x1080 |

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

The three-voice subjects share one file name with `-editorial`, `-minimal` or
`-organic` on the end; the explainer diagram is one design, not three, and its
empty middle is on purpose — the diagram is the one thing this library does not
draw for you (GUIDE.md, "The explainer diagram's empty middle").
