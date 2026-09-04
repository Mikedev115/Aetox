---
name: aetox-web-templates
description: เทมเพลตหน้าเว็บจริงที่ก๊อปไปใช้ได้เลย 46 ส่วน ตั้งแต่ nav hero ราคา FAQ ไปจนถึงแดชบอร์ด แดชบอร์ดวิเคราะห์ กราฟ 6 ชนิดที่วาดด้วย SVG ล้วน ตาราง ฟอร์ม แจ้งเตือน สลับธีม กริดสื่อ โปรไฟล์ หน้า 404 บทความ และหน้าเอกสาร ไฟล์เดียวจบ responsive ไม่มี framework ไม่มี CDN ไม่ต้อง build ใช้ตอนกำลังจะสร้างหน้าเว็บหรือหน้าจอแล้วไม่อยากประดิษฐ์โครงขึ้นใหม่ทุกครั้ง
---

# Web templates

Markup that gets copied, not advice about markup. **Web pages only.**

The name says the medium on purpose, the same way `aetox-slide-templates` does.
A deck is one file at a fixed 1280x720 that an off-screen renderer prints; a web
page is responsive and lives in a browser somebody resizes, zooms and reads with
a screen reader. A layout correct for one is wrong for the next, so they are two
skills and neither borrows from the other.

## Why this exists

Six skills already describe web and UI work: `aetox-frontend-design` decides the
look, `aetox-ui-design` carries nine implementation guides, `aetox-shadcn` and
`aetox-radix-to-base` cover component libraries, `aetox-design-system` the
tokens, `aetox-design` the pictures. Sixty-six files between them, and on
29 ส.ค. not one of them was markup: every file was prose, a CSV or a licence.

That is the same gap the slide tables had, measured the same way. Descriptions
tell a model what a hero *should be*. They do not stop it building a different
hero every time, which is what a page assembled from scratch on every run
actually looks like.

## The sections

Paste `page-shell.html` first; it carries the tokens, the reset, the landmarks
and dark mode that every section below assumes. Then paste sections into its
`<main>` in the order the page needs.

| Section | File | Reach for it when |
|---------|------|-------------------|
| Page shell | `sections/page-shell.html` | Always first. Tokens, reset, skip link, light and dark. |
| Backgrounds | `sections/backgrounds.html` | Eight paste-on background layers — dots, grid, vignette, glow, fade, noise, aurora, diagonal. One class on a section you already have. |
| Nav | `sections/nav.html` | Site header. Opens on mobile with no script. |
| Theme toggle | `sections/theme-toggle.html` | A manual light/dark switch that remembers the choice, instead of only following the OS. |
| Hero | `sections/hero.html` | The thesis of the page, above the fold. |
| Logo bar | `sections/logo-bar.html` | Named social proof under the hero. |
| Features | `sections/features.html` | What it does. Scales 3 to 12 with no new layout. |
| How it works | `sections/how-it-works.html` | Ordered steps, where the order is the meaning. |
| Stats | `sections/stats.html` | Numbers, each with its source. |
| Testimonial | `sections/testimonial.html` | One quote, attributed to a real person. |
| Team | `sections/team.html` | Named people: photo, role, and a way to reach them. |
| Profile spotlight | `sections/profile-spotlight.html` | One person in full: bio, real contact, their featured work. |
| Pricing | `sections/pricing.html` | Up to three tiers, one marked. |
| FAQ | `sections/faq.html` | Objections. Accordion with no script. |
| CTA band | `sections/cta-band.html` | Mid-page and end-of-page conversion block. |
| Newsletter | `sections/newsletter.html` | Email capture or short contact form. |
| Footer | `sections/footer.html` | Closure and the alternative paths. |
| Dashboard shell | `sections/dashboard-shell.html` | Sidebar, topbar, content well for an app. |
| Analytics overview | `sections/analytics-overview.html` | A finished analytics home rather than an empty frame: funnel, totals, retention, distributions, one insight. Frosted chrome over an ambient ground. |
| Stat cards | `sections/stat-cards.html` | The number row at the top of a dashboard. |
| Stat cards + sparkline | `sections/stat-sparkline-row.html` | The same number row when the shape of the last eight weeks matters as much as the number. |
| Area price chart | `sections/chart-area-price.html` | One series over time, for the question "up or down". Latest value labelled at the end of the line. |
| Multi-series line | `sections/chart-multi-line.html` | Three series that share one unit, compared on one axis. Never two axes. |
| Stacked bars | `sections/chart-stacked-bars.html` | Totals over time, and what the totals are made of. Three series is the ceiling. |
| Ranked bars | `sections/chart-ranked-bars.html` | A ranking read in two seconds. Use it wherever a donut would have more than five slices. |
| Donut | `sections/chart-donut.html` | Share of a whole that genuinely sums to 100%, in at most five slices. |
| Data table | `sections/data-table.html` | Tabular data that scrolls on a narrow screen. |
| Form card | `sections/form-card.html` | Sign in, sign up, any short form. |
| Form controls | `sections/form-controls.html` | Every native control styled once — text, select, checkbox, radio, switch, range, date, file — with error, disabled and read-only states. |
| Alert | `sections/alert.html` | An inline status message, or a grouped error summary for a multi-field form. |
| Dialog | `sections/dialog.html` | A question that has to be answered before anything else — confirm, filter, quick edit. Native `<dialog>`, so Esc, focus trap and top layer come free. |
| Toast | `sections/toast.html` | The result of an action nobody needs to acknowledge. Floats, stacks, and leaves on its own. |
| Tooltip | `sections/tooltips.html` | A short explanation on hover or focus, in any of four directions, keeping itself inside the viewport. |
| Switch | `sections/switch.html` | A switch you can actually drag, with a spring on release. Heavier than the one in form-controls, for settings people flip often. |
| Progress | `sections/progress.html` | Bars and rings, for work whose end is known and for work whose end is not. |
| Breadcrumbs (collapsing) | `sections/breadcrumbs.html` | The path back up when it is too long for the space — the middle folds into an ellipsis. |
| Split button | `sections/split-buttons.html` | One obvious action with the rest behind an arrow beside it. |
| Card stack | `sections/card-stack.html` | Several things in a small space, fanning open on hover. |
| Empty state | `sections/empty-state.html` | A list with nothing in it yet. |
| Error page | `sections/error-page.html` | 404, 500, anything that went wrong. |
| Article index | `sections/article-index.html` | The list of posts that links into Article. |
| Article | `sections/article.html` | A post or case study, read rather than scanned. |
| Media grid | `sections/media-grid.html` | Thumbnail-first content — clips, recordings, uploads — with a type badge and reactions. |
| Search box | `sections/search-box.html` | A real search input, wired to whatever indexes the pages. |
| Docs page | `sections/docs-page.html` | Sidebar, prose, on-this-page rail. |
| Breadcrumb | `sections/breadcrumb.html` | The path back up, for a page nested more than one level deep. |

**Page order for a marketing page**, which is the visitor's own decision
journey: nav → hero → logo bar → features → how it works → testimonial →
pricing → FAQ → CTA band → footer. Cut freely; do not reorder without a reason,
because each section answers the question the one before it raises.

## The standard

`STANDARD.md` in this folder is the bar every file here answers to, and it is worth
reading before adding one. The short version is one sentence: **a template that
encodes the average does not help a capable model, it holds it down.** So each file
must carry something that cannot be generated on the spot — a measured value, a
technique that took ten tries, or an art direction that is a real choice rather than
the default. It also holds the motion vocabulary, the four art directions, and the
trap book of things that actually broke in this browser.

## The contract every template obeys

- **One self-contained file.** No framework, no CDN, no build step, no
  `node_modules`. The page opens from disk, from a USB stick, from an email
  attachment. Same reason a deck is one file.
- **Responsive without breakpoint bookkeeping.** `clamp()` for type and space,
  `repeat(auto-fit,minmax(min(280px,100%),1fr))` for grids, so the column count
  follows the content and the viewport rather than a list of widths somebody has
  to maintain. Media queries appear only where the layout genuinely changes
  shape, such as the dashboard sidebar becoming a top scroller.
- **No script unless the semantics cannot be had without it.** The FAQ is
  `<details>`, the mobile menu is a checkbox. Both are keyboard operable and
  survive a JS error, which a scripted version of either does not. The one
  exception is `sections/theme-toggle.html`: a theme choice that resets on
  every reload has no script-free equivalent worth calling a feature, so it
  carries a small inline script rather than fake one. Nothing else here does.
- **Real landmarks and real elements.** `<nav>`, `<main>`, `<footer>`,
  `<blockquote>` with `<cite>`, `<ol>` when the order is the meaning, `<table>`
  with `scope` on every header. This is not decoration; it is what a screen
  reader navigates by.
- **Every class is namespaced `w-`.** `aetox-frontend-design` warns that
  generated CSS most often breaks when a class selector and an element selector
  fight over the same padding. Two templates pasted into one page must not be
  able to reach into each other.
- **Light and dark from tokens**, defined once in the shell. No template hard
  codes a colour.
- **Charts are SVG and CSS in the file.** No charting library, no CDN, for the
  same reason as everything else here: a page whose picture arrives from
  somewhere else is a page with a hole in it when that somewhere else is down.
  Series colours come from `--viz-1` … `--viz-6` in the shell, in that fixed
  order, never cycled; the ramp `--viz-s1` … `--viz-s5` carries magnitude. Those
  six ran through the `dataviz` skill's validator in both modes — lightness
  band, chroma floor, colour-blind separation of adjacent pairs, contrast
  against the surface. Change a value and run it again rather than eyeballing.
  Every mark carries its own `<title>`, so the browser draws the tooltip and a
  screen reader reads the number without a line of script, and every chart ships
  the same numbers as a table inside `<details>`.
- **Accessible by construction.** Visible focus ring, a skip link, labels that
  stay visible when the field is filled, `prefers-reduced-motion` respected, and
  status never carried by colour alone.

## Using one

Copy the section, change the words, delete what the page does not need. A
template is a starting composition, not a form to fill in. Three identical
card rows down one page is the tell `aetox-anti-slop` is written about: vary
the cell count, let one section be a single sentence, let another be a number.

Read `aetox-frontend-design` before choosing the look, and `aetox-design-system`
for the token layers if the page needs a palette of its own. This skill decides
none of that. It only means you do not rebuild the skeleton of a pricing table
from memory again.

**Redesigning something that already exists.** Read the real page or
component before reaching for a template to replace it: open the file that
actually renders on that route, not a sibling that looks similar, and carry
over what it actually says and does. A template pasted in from a guess at the
old page's content is not a redesign, it is a different page that happens to
sit at the same URL.

**Iterating on what you built.** Match the edit to what was actually asked. A
typo or a wrong number is a direct edit to that one line, not a rewrite of the
section around it. A request for a different direction ("try something bolder
for the hero") is worth composing two or three variants side by side rather
than guessing which lands, since comparing is cheaper than regenerating blind.
A request to keep refining the one already there stays a small, targeted edit
to that same markup. Confusing these three wastes more than it saves —
rewriting a whole section to fix a typo throws away everything about it that
was already right.

Whatever the edit, the token system chosen at the start still governs it. A
later revision that reaches for a colour or a font not in that system has
quietly drifted back toward the generic default `aetox-frontend-design` warns
about. If the palette genuinely needs to change, that is itself a plan
revision — named and deliberate, not one stray value slipping in three edits
later.

## When the codebase has a better answer

These are starting points, not the final word. If a real Aetox screen ends up
with a nav, a form, or a table that is genuinely better than the matching
template here — clearer, more accessible, solving something this file does
not — feed it back: update the template file itself, in the same contract as
the rest of this folder, rather than letting the improvement live only on
that one page. A library that never learns from the product it describes
quietly falls behind it.

## Not in here

Video, because Aetox produces none; `video_ocr` reads one. Slides, which are
`aetox-slide-templates` and a different contract entirely. Documents and sheets,
which leave through `doc_write` and `sheet_write` as OOXML and are not HTML at
all. If any of those earn templates, they earn a skill of their own rather than
a folder in here: a template nobody can tell the medium of is a template that
gets pasted into the wrong file.
