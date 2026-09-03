# The Google register

Four scenes on this shelf are written in one house style, and this file is that
style stated once instead of four times: `search-query-reveal`,
`dot-loader-morph`, `card-expand-hero`, `caption-montage`.

Open it when you are **writing a new scene in that register**, or when you are
about to change one of those four and want to know which numbers are load-bearing.
To simply *use* one of them, the table in `SKILL.md` is enough — everything here
is already baked into the files.

## Why this register exists

The rest of the library is drawn from editorial and studio traditions — Swiss
grids, film stock, newspaper charts. None of that is the language most people
actually mean when they say "make it look like a real product ad". That language
is a specific one, and unlike most house styles it is **written down**: Google
publishes its motion system as tokens with exact values, and ships two
implementations of it. So it can be learned properly rather than eyeballed.

Everything in this file is either a published specification or a measurement
taken on a stated date. Nothing here is an impression of a film.

## The motion numbers, and where each came from

Verified 4 ก.ย. 2569 against
[material-components-android `docs/theming/Motion.md`](https://github.com/material-components/material-components-android/blob/master/docs/theming/Motion.md),
cross-checked against [mdui's token table](https://www.mdui.org/en/docs/2/styles/design-tokens),
which agrees value for value.

| Easing | Value |
|:--|:--|
| Standard | `cubic-bezier(0.2, 0, 0, 1)` |
| Standard decelerate | `cubic-bezier(0, 0, 0, 1)` |
| Standard accelerate | `cubic-bezier(0.3, 0, 1, 1)` |
| Emphasized decelerate | `cubic-bezier(0.05, 0.7, 0.1, 1)` |
| Emphasized accelerate | `cubic-bezier(0.3, 0, 0.8, 0.15)` |
| Emphasized | **not a cubic-bezier** — see below |
| Linear | `cubic-bezier(0, 0, 1, 1)` |

Durations are a ladder of sixteen: short1–4 at 50/100/150/200ms, medium1–4 at
250/300/350/400, long1–4 at 450/500/550/600, extra-long1–4 at 700/800/900/1000.
The four scenes spend six of them and name each in `:root` rather than typing a
millisecond number into a rule, which is the same discipline `DESIGN.md` §4
already imposes on Aetox's own UI.

**A caution about the Android doc's prose.** Its transition-pattern paragraphs
quote "motionDurationLong1 (300ms)" while its own token table says Long1 = 450ms.
Those are the Android *platform attribute* defaults, not the M3 spec values, and
they do not agree. The table is what to trust; that is what is used here.

## Emphasized cannot be written as `cubic-bezier`, and this is the fix

The spec does not give Emphasized as a curve. It gives it as a **path with two
cubic segments**:

```
M 0,0  C 0.05,0  0.133333,0.06  0.166666,0.4
       C 0.208333,0.82  0.25,1   1,1
```

Read it: the value reaches **40% of the distance in the first 16.7% of the
time**, then spends the remaining 83% settling. No single `cubic-bezier()` has
that shape, so writing `cubic-bezier(0.2, 0, 0, 1)` and calling it Emphasized —
which is what most ports of these tokens do, mdui included — quietly substitutes
Standard and loses exactly the snap that makes the register recognisable.

CSS `linear()` can express it. Sampled at 25 even steps in x:

```css
--e-emphasized: linear(0, 0.0141, 0.0616, 0.1622, 0.4, 0.6686, 0.7728, 0.8315,
  0.8706, 0.8989, 0.9204, 0.9372, 0.9506, 0.9614, 0.9701, 0.9773, 0.983, 0.9877,
  0.9914, 0.9943, 0.9965, 0.9981, 0.9992, 0.9998, 1);
```

Paste that verbatim; it is in all four scenes. `linear()` needs Chrome 113+,
which the renderer is well past.

## The cross-fade split is not 50/50

Every content swap in these scenes uses the same three numbers, taken from
Google's own implementation rather than from prose —
[`fade_through_transition.dart`](https://github.com/flutter/packages/blob/main/packages/animations/lib/src/fade_through_transition.dart)
in the Flutter `animations` package:

- outgoing content fades out over the **first 6/20** of the run
- incoming content fades in over the **last 14/20**
- incoming content scales **0.92 → 1** as it arrives

On an 800ms container transform that is 240ms out, then 560ms in starting at
240ms. They do not overlap, and that is the point: a symmetrical crossfade
produces a muddy middle where both layers are half-visible, and this produces a
clean handover with one brief empty beat instead.

## Two surfaces, not one

Asking for "the Google look" gets you one of two quite different things, and
mixing them in one piece reads as a mistake. Pick one and stay in it.

**Light — the search register.** Near-white ground, one rounded field or card
holding everything, ink-on-paper contrast, colour used only as accent.
`search-query-reveal` and `dot-loader-morph` are here.

**Dark — the AI Studio register.** Measured off `aistudio.google.com` on
4 ก.ย. 2569, because guessing at a surface you can open is guessing for no
reason:

| | Measured |
|:--|:--|
| Stage | `rgb(18, 19, 23)` — warm near-black, **not** `#000` |
| Card fill | `linear-gradient(135deg, rgb(28,30,35), rgb(42,45,53))` at `20px` radius |
| Radius ladder in use | 8 / 12 / 16 / 20 / 28 / 36 — 12 and 20 carry most of it |
| Primary button | a **white** pill, `#0E0F12` text, `12px` radius |
| Display face | Google Sans Flex, weight **450**, `-1.44px` tracking on 64px = **-0.0225em** |

Weight 450 is worth noticing: it is a variable-font intermediate, not 400 and not
500, and it is why the headlines read as lighter than a "bold ad headline" while
still holding a dark frame. `card-expand-hero` is in this register.

## Type, and the Thai problem these four finally answer

**Google Sans went open (SIL OFL) on 10 ธ.ค. 2025**, so it can be used here
outright rather than approximated. Checked 4 ก.ย. 2569: `fonts.googleapis.com`
serves `Google Sans`, `Google Sans Flex` and `Google Sans Code` to a plain
`css2?family=` request.

**These four scenes are the first on this shelf whose Thai copy lands in a face
somebody chose.** `GUIDE.md` records the finding that not one of the previous 51
covered Thai — Thai text fell through to Leelawadee UI or Tahoma, and every check
in the pipeline passed it. Each of the four names `Anuphan` (Thai + Latin,
geometric, close in feel to Google Sans) after the Latin face in its stack, so
Thai glyphs resolve to Anuphan and Latin glyphs to Google Sans. `IBM Plex Sans
Thai` is the alternative and is in each stack as the second fallback.

The `<link>` elements are **separate per family**, deliberately. The renderer
fetches these at render time, a failed fetch produces a wrong-looking scene
rather than an error, and one request naming three families loses all three when
it fails. Split, a Latin failure still leaves the Thai face standing.

## What was taken, and what was deliberately not

Taken: published motion tokens, a published transition implementation, measured
surface values off a public page, and an openly licensed typeface.

Not taken, and not to be added later:

- **Any Google mark** — logo, wordmark, product name, or the "G".
- **The four brand hues.** The four-hue accent rotation is a widely used
  grammar; `#4285F4 / #EA4335 / #FBBC04 / #34A853` specifically is a trademark.
  These scenes use indigo `#3B4AE0`, coral `#E8543F`, amber `#F2B33D`, teal
  `#1F9D74`.
- **The four-hue *sequence*.** This one is easy to miss and was missed on the
  first pass here. Four dots reading blue → red → yellow → green identify a
  company on sight no matter what the hex values are, because at 30px nobody is
  reading hex values. `dot-loader-morph` runs indigo → teal → amber → coral for
  that reason, and says so in the file. Reorder away from that sequence, never
  back toward it.
- **Any content from an actual advertisement** — no shot, no line of copy, no
  edit rhythm lifted from a specific film. The sample copy in all four is about
  this template library.

The short version: a specification is there to be implemented, a measurement is
a fact about a page, and a brand's identity is neither.
