---
name: aetox-slide-templates
description: เทมเพลตจริงที่ก๊อปไปใช้ได้เลย ไม่ใช่คำอธิบายว่าควรหน้าตาแบบไหน เลย์เอาต์สไลด์ 41 แบบ รวมไดอะแกรม 16 แบบที่วาดด้วย CSS กับ SVG ในไฟล์เอง ชุดสี 6 โทนใน themes/ และม่านบนภาพ 5 แบบใน overlays/ ที่สลับได้ด้วยการเปลี่ยนบล็อก :root บล็อกเดียว กล่อง 1280x720 ไฟล์เดียวจบ ไม่มี CSS ภายนอก ไม่มีสคริปต์ ไม่มีกราฟจาก CDN ใช้ตอนกำลังจะเขียนสไลด์หรือหน้าจอแล้วไม่อยากประดิษฐ์องค์ประกอบขึ้นใหม่ทุกครั้ง
---

# Slide templates

Markup that gets copied, not advice about markup. **Slides only.**

The name says the medium on purpose. Aetox produces five kinds of thing and they
do not share a contract: a deck is one self-contained HTML file at a fixed
1280x720 that an off-screen renderer prints, a web page is responsive and lives
in a browser somebody resizes, a document goes out through `doc_write` as OOXML,
a sheet through `sheet_write`, and a picture is SVG or a real photograph found
on the web. A layout that is correct for one is wrong for the next.

So this skill holds slides and nothing else. If web pages or documents ever earn
templates, they earn a skill of their own rather than a folder in here, because
a template nobody can tell the medium of is a template that gets pasted into the
wrong file. (Video already went that way: its scenes are HTML as well, and they
live in the `video` agent's own `video-templates` rather than in here, because a
scene is rendered frame by frame and a slide is printed once.)

## Why this is its own skill

The design skills already describe every layout a deck could want.
`aetox-design-system` carries `data/slide-layouts.csv`, twenty-five rows, each
naming a layout and its content zones and its visual weight. What it gives for
the composition is a `css_structure` column holding one line:

```
display:flex; flex-direction:column; justify-content:center; align-items:center; text-align:center
```

That is a hint. Everything else had to be invented at write time, every time,
and the result was measurable: decks built from the same table came out
differently on every run, and mostly came out as the one skeleton in
`aetox-slides` with different words in it. A run on 29 ส.ค. read
`slide-strategies.csv` and `slide-layouts.csv` before writing a line and still
produced seven ad-hoc slides that ended on a step.

Descriptions were never the missing part. This is the missing part.

It is separate from the skills that describe, rather than folded into them, for
two reasons. A skill's file listing is capped at 80 entries, so a folder of
templates inside `aetox-design-system` would push its own tables past the edge.
The cut now says how many it dropped rather than happening silently, but a file
the listing never names is still a file nothing goes looking for. And templates are one question, "what does this look
like in markup", where the tables are another, "which one should I reach for".
One home each.

## Slide layouts

Read the row from `aetox-design-system` `data/slide-layout-logic.csv` first, so
you know which layout the slide wants. Then open only that file here. Each is a
`<section>` plus the CSS it adds, ready to paste.

| Layout | File | Reach for it when |
|--------|------|-------------------|
| Title | `slides/title.html` | Opening. One idea, enormous, room around it. |
| Agenda | `slides/agenda.html` | What is coming. Five rows at most; six is two slides. |
| Section divider | `slides/section-divider.html` | A breath between acts. Allowed to be nearly empty. |
| Two column split | `slides/two-column.html` | An idea and its evidence, side by side, unequal. |
| Big number | `slides/big-number.html` | One number that is the whole slide. |
| Metrics | `slides/metrics.html` | Three or four numbers that belong together. |
| Timeline | `slides/timeline.html` | Steps where the order is the message. |
| Before after | `slides/before-after.html` | Two states of one thing. Four lines a side at most. |
| Comparison table | `slides/comparison.html` | Options against criteria. Three columns, five rows at most. |
| Chart | `slides/chart-bars.html` | A quantity comparison, drawn in CSS. |
| Quote | `slides/quote.html` | Somebody's words, attributed. |
| Full bleed image | `slides/full-bleed.html` | The picture is the slide. |
| Visual split | `slides/visual-split.html` | The picture beside the words. A product, a screen, the thing itself. |
| Image grid | `slides/image-grid.html` | Three pictures of one idea, a line each. |
| Feature grid | `slides/feature-grid.html` | Three or six parts of one thing. |
| Code | `slides/code.html` | A command or an excerpt, one screen, never scrolled. |
| Terminal | `slides/terminal.html` | A session: commands and the real output they produced. |
| Diff | `slides/diff.html` | Lines that changed. What was removed, what replaced it. |
| Process steps | `slides/process-steps.html` | More than four steps, or a step that needs more than a line. |
| Team grid | `slides/team-grid.html` | The people. A photo, a name, one line each. |
| Roadmap | `slides/roadmap.html` | A plan with dates. Where things stand today, what comes after. |
| Architecture | `slides/architecture.html` | How the parts connect. Boxes and the lines between them. |
| Synthesis | `slides/synthesis.html` | Second to last. What the middle added up to. |
| Close | `slides/close.html` | Last. The one thing to do, remember or decide. |
| Q&A | `slides/qa.html` | Inviting questions. Sometimes last, sometimes not — check the strategy. |

Synthesis and close are not optional and are the two most often missing: a deck
that stops on its final step has not concluded, only stopped. See
`aetox-design-system`, "How a deck ends".

## Diagrams

The table above answers "what goes where on this slide". This one answers a
different question — "what shape is this idea" — and it is the question a deck
gets wrong more often, because the answer is nearly always a bulleted list and
nearly never should be.

Every one of these is drawn in CSS or SVG, in the file, with no library and
nothing fetched. That is not a limitation working out well: a diagram drawn as
markup keeps its text selectable, reskins with the theme, and prints crisp in
the `.pptx` export, where a picture of a diagram goes in as a flat bitmap that
nobody can fix a typo in.

| Shape | File | Reach for it when |
|-------|------|-------------------|
| Ecosystem | `slides/ecosystem.html` | One thing and everything that touches it. Lines mean "related", no arrows. |
| Layers | `slides/layers.html` | Levels that rest on each other. The bottom one holds the rest up. |
| Blueprint | `slides/blueprint.html` | The thing as a measured drawing. Real numbers or no dimension lines. |
| Customer journey | `slides/journey.html` | Stages plus how each one felt. The dip is the slide. |
| Supply chain | `slides/supply-chain.html` | Where it comes from. The legs between the stops carry the cost. |
| Value chain | `slides/value-chain.html` | The same row of steps, asked what each one added. |
| Funnel | `slides/funnel.html` | Many at the top, few at the bottom. Drop-off percentages or it is a triangle. |
| Matrix | `slides/matrix.html` | Options on two axes. The empty quadrant is usually the argument. |
| Flow | `slides/flow.html` | Steps with a decision in them. One diamond, both branches labelled. |
| Scale | `slides/scale.html` | A size nobody can picture, beside one everybody can. |
| Concept map | `slides/concept-map.html` | An idea opened two levels deep. Left to right, never radial. |
| Hierarchy | `slides/hierarchy.html` | Who reports to whom. Right angles, no arrowheads. |
| Scenario | `slides/scenario.html` | One present, three futures, each with a likelihood on it. |
| Case study | `slides/case-study.html` | Problem, what was done, what came of it. The third panel carries a number. |
| Deconstruct | `slides/deconstruct.html` | A word everybody nods at, opened into the parts it is made of. |
| Visual story | `slides/visual-story.html` | Four pictures where the order is the point. Shuffle them and it breaks. |

Each file's opening comment names the layout it is most often confused with and
says which way to go. Those pairs are the ones worth knowing before choosing:
ecosystem against architecture (related versus directional), concept map against
ecosystem (two levels versus one ring), funnel against process-steps (something
narrows versus nothing does), visual story against image grid (an order versus
no order), and journey against timeline (a felt curve versus a flat one).

### What this folder cannot draw

Some visual jobs are photographic, and markup is the wrong tool for every one of
them: a cutaway, an exploded view, an X-ray, a cross-section, an anatomy, an
isometric city, a plausible future version of a real object. Those need an image
model, and `aetox-design` says plainly that this app has no image model and what
the three roads are instead. Do not approximate one of them here with rectangles
and call it done — a cutaway drawn as boxes is not a cutaway, it is architecture
with a misleading title.

## The contract every template obeys

The deck is **one self-contained HTML file**. There is no stylesheet to link, no
script to load, no CDN. `aetox-slides` carries the skeleton these drop into and
the full reasoning; this is the short form.

- **The slide box is fixed 1280x720 px** with `overflow:hidden`, declared once
  by the skeleton. No template here redefines `.slide`. A deck sized in
  `vw`/`vh` is laid out twice, differently, once in the room and once in the
  off-screen exporter, and you only ever see one of those.
- **Slides sit one after another in normal flow.** Never `position:absolute`
  with `opacity:0` and a script switching between them. The exporter prints what
  the document says, so a deck built that way exports as one slide followed by
  blank rectangles.
- **Entrances go through `.rise`**, which rests visible and animates only on the
  slide the room marks `.onstage`. A slide that starts hidden waiting for script
  exports empty.
- **Charts are CSS or SVG.** A chart from a CDN-fetched library prints blank.
- **Each template adds its own classes only.** Its CSS goes in the deck's one
  `<style>` block, its markup in the body.

Sizes assume the skeleton's 120px side padding. A template that wants the edge
says so and sets its own.

## Using one

Copy the block, change the words, delete what the slide does not need. A
template is a starting composition, not a form to fill in: two slides from the
same template with the same number of cells and the same sentence lengths is the
uniformity these exist to break. Vary the cell count, let one slide be a single
sentence, let another be a number.

## Themes

A layout says where things sit; a theme says what colour they are. They are
separate files because they change for different reasons — the same `metrics`
slide is right on all six of these, and a deck that has to be printed on paper
needs a different stage without needing a different composition.

Six ship in `themes/`. Each is a `:root` block and nothing else: no rule, no
selector, no `.slide`. Drop one into the deck's single `<style>`, replacing the
palette the skeleton came with, and the whole deck reskins. Order does not
matter, because every value travels as a custom property rather than as a rule
that has to win.

| Theme | File | `slide-color-logic.csv` calls it | Reach for it when |
|-------|------|--------------------------------|-------------------|
| House | `themes/house.css` | `dark-surface` | The default. Dark stage, one red doing every job. |
| Ink | `themes/ink.css` | `dark-background` | Flat dark, no gradient. The stage should have no voice of its own. |
| Glow | `themes/glow.css` | `dark-glow` | Dark with the accent blooming off the top edge. Curiosity, a demo. |
| Paper | `themes/paper.css` | `surface` | Light. A deck that gets printed, or a room with the lights on. |
| Elevated | `themes/elevated.css` | `surface-elevated` | Light, cool, cards lifted. Things being compared side by side. |
| Bleed | `themes/bleed.css` | `accent-bleed`, `gradient` | The accent becomes the stage. A title, a divider, a close. |

`aetox-design-system` `data/slide-color-logic.csv` maps an emotion to one of
these in its `theme_file` column, so the choice comes from the same table the
layout does rather than from taste at write time.

### The tokens that look like duplicates and are not

`--stage` is the stage as one flat colour; `--stage-bg` is what it is actually
painted with and may be a gradient. Both exist because a gradient cannot be
punched into a dot on a rail (`timeline`, `roadmap`) or mixed into the scrim
over a photograph (`full-bleed`), and those two jobs need a colour.

The accent has three names because it does three jobs at three sizes.
`--accent` is the graphic — a bar, a rule, a dot, a 280px number — which owes
3:1 against its ground. `--accent-text` is the same accent set as small text — a
16px kicker, a 14px tag, a marked column head — which owes 4.5:1; on the house
stage `#ff3b30` measures 3.70 against the brightest panel, correct as a bar and
short as a word. Templates read that one as `var(--accent-text,var(--accent))`,
so a deck defining only the first name still renders.

`--accent-ink` faces the other way: it is the colour of words sitting **on** the
accent rather than against the stage, which is where a CTA and the
`gradient-accent` overlay both put them. It is not a nicety — measured on the
accent field, `--accent-text` reads between 1.00:1 and 1.35:1 in all six themes.
Invisible in every one. `aetox-web-templates` `sections/page-shell.html` has
carried a token of the same name and meaning since before this folder existed;
this is the deck side of it.

### Contrast is checked, not asserted

Every theme carries its measured worst case in a comment, and
`TestSlideThemesAreLegible` recomputes those numbers from the shipped files on
every run: it composites each translucent token over each colour its own
`--stage-bg` can produce, and holds text to 3:1 and body, caption and
accent-as-text to 4.5:1. A theme edited into something unreadable fails the
build rather than shipping and being noticed on a projector.

Two results of that worth knowing before reaching for a theme. **Bleed carries
two weights of light, not three** — on a saturated field the third always falls
under 4.5:1, so `--body` and `--muted` are nearly equal there by design and
hierarchy comes from size instead. And **bleed's red is darker than it wants to
be**: the hotter `#ff4b32` that was drawn first gives white body copy 2.86:1,
which no amount of white can rescue.

`--line` is deliberately outside all of this. It is a hairline between rows and
a rail behind steps — decoration, not a control and not the thing carrying the
meaning, which on those rails is the accent-bordered dot. Held to 3:1 it stops
being a hairline.

## Pictures

`aetox-slides` says it outright: picture-carried is the default and a flat slide
is the exception, because a dark stage with nothing but text on it, slide after
slide, is what reads as unfinished. Three layouts here carry one — `full-bleed`
behind the words, `visual-split` beside them, `image-grid` three at once — and
`team-grid` carries faces. Go and find the pictures before laying anything out;
`aetox-design` has the recipe and the rule that this app finds real photographs
rather than inventing them, and that the licence is checked every time.

### Overlays

A photograph almost never takes text straight on top of it. `aetox-design-system`
`data/slide-backgrounds.csv` has said which veil each kind of slide wants since
before any of this existed — `overlay_style` naming five and `text_placement`
naming four — and none of the nine had markup. They do now, in
`overlays/photo-overlays.css`.

| `overlay_style` | Class | The table asks for it on |
|-----------------|-------|--------------------------|
| `gradient-dark` | `.ov-gradient-dark` | hero, team, hook, demo, social |
| `gradient-brand` | `.ov-gradient-brand` | vision, cta |
| `gradient-accent` | `.ov-gradient-accent` | solution |
| `blur-dark` | `.ov-blur-dark` | testimonial, social |
| `desaturate-dark` | `.ov-desaturate-dark` | problem |

`text_placement` is `.tp-center`, `.tp-left`, `.tp-right`, `.tp-bottom`.

The file is **optional**. Every picture template carries its own default scrim,
so copying one block is still enough; open this only when a slide wants one of
the other four. And note what "dark" means in those names: **toward `--stage`,
never toward black.** On `paper` and `elevated` the stage is light, so a literal
black veil buries the photograph and prints dark text over the top of it —
which is exactly the bug `full-bleed` shipped with until the theme pass, and the
reason every scrim here is mixed with `color-mix` from a token instead of
written out as a colour.

Two mechanics worth not rediscovering. The blur and the desaturation are
`filter` on the `<img>` and never `backdrop-filter` on the layer above it,
because the off-screen export has no reliable backdrop and prints that as a
plain rectangle over the picture. And the selector is
`:is([class^="ov-"],[class*=" ov-"])`, never `[class*="ov-"]`, which matches a
substring anywhere and reaches into class names that have nothing to do with
overlays — the web templates shipped that mistake once already.

## Where the inventory came from

The list of layouts a deck system needs was cross-checked against
`html-ppt-skill` by lewis (MIT), which catalogues 31 and named four that
`slide-layouts.csv` did not: Code, Terminal, Diff and Process Steps.

Its markup was **not** taken, and could not have been. Its slides are `100vw` by
`100vh`, stacked with `position:absolute` and `opacity:0` for a runtime to
switch between, and each of its layout files is a fragment depending on four
external stylesheets and a script. Under the contract above that deck exports as
one slide and a run of blank rectangles. Structure borrowed, ornament not.

Terminal, Diff and Process steps, the three named above and left out at the
time, were built on 3 ก.ย. against the same test the rest of this file exists
to enforce: a deck that could not be told with the other 16, not a layout
added because a catalogue elsewhere had a name for it. Team grid the same
week closed a gap `aetox-design-system`'s own tables already named and never
delivered on — `data/slide-layouts.csv` row 8 and the `avatar-ring` card style
in `data/slide-color-logic.csv` both predate this file by longer than anyone
would guess. Roadmap and Architecture answer two the tables never named at
all: a plan with dates, which Timeline cannot carry because it orders steps
with no calendar attached, and a system's parts and the lines between them,
which no chart in this folder draws because a chart compares quantities and
a relationship is not one.

Q&A followed the same day, closing a gap `slide-layouts.csv` had named at row
25 since before any of this existed and never got a file. `slide-strategies.csv`
names it as the literal last slide in two of its fifteen structures, Conference
Talk and Workshop Training, and neither Synthesis nor Close could carry it:
an invitation to ask is not a recap and not a single action, and it does not
share their fixed place at the end, either — some structures run it before
Close, not after.
