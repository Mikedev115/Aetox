# The Google register

Nine scenes on this shelf are written in one house style, and this file is that
style stated once instead of nine times: `search-query-reveal`,
`dot-loader-morph`, `card-expand-hero`, `caption-montage`, `aurora-prompt-box`,
`ship-it-toolbar`, `spring-stat-tiles`, `shared-axis-steps`, `device-reveal`.

Open it when you are **writing a new scene in that register**, or when you are
about to change one of those nine and want to know which numbers are load-bearing.
If you are cutting several of them into one piece, read "The edit, measured" at the
bottom first — it is the only part of this file about assembly rather than about a
single scene.
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
These scenes spend six of them and name each in `:root` rather than typing a
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

Paste that verbatim; it is in six of the eight. `linear()` needs Chrome 113+,
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
`search-query-reveal`, `dot-loader-morph`, `spring-stat-tiles` and
`shared-axis-steps` are here.

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
still holding a dark frame. `card-expand-hero`, `aurora-prompt-box`,
`ship-it-toolbar` and `device-reveal` are in this register.

`caption-montage` is in neither — it is six colour fields, and it cuts against
both.

## Type, and the Thai problem these nine finally answer

**Google Sans went open (SIL OFL) on 10 ธ.ค. 2025**, so it can be used here
outright rather than approximated. Checked 4 ก.ย. 2569: `fonts.googleapis.com`
serves `Google Sans`, `Google Sans Flex` and `Google Sans Code` to a plain
`css2?family=` request.

**These nine scenes are the first on this shelf whose Thai copy lands in a face
somebody chose.** `GUIDE.md` records the finding that not one of the other 51
covers Thai — Thai text falls through to Leelawadee UI or Tahoma, and every check
in the pipeline passes it. Each of the nine names `Anuphan` (Thai + Latin,
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

- **Any Google mark** — logo, wordmark, product name, or the "G". The film
  watched for the last two scenes has a "G" in the corner of nearly every frame
  and a product logo as its final image; neither is anywhere in these files.
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

## Springs: what Material 3 Expressive replaced the curves with

Everything above this section is bezier. In 2025 M3 Expressive stopped describing
its motion as curves and started describing it as **springs**, each stated as a
damping ratio and a stiffness. Read off AOSP (Apache-2.0) on 4 ก.ย. 2569,
`compose/material3/.../tokens/ExpressiveMotionTokens.kt` and
`StandardMotionTokens.kt`, v0_14_0:

| Token | Expressive | Standard |
|:--|:--|:--|
| spatial default | damping 0.8, stiffness 380 | 0.9 / 700 |
| spatial fast | **0.6 / 800** | 0.9 / 1400 |
| spatial slow | 0.8 / 200 | 0.9 / 300 |
| effects default | 1.0 / 1600 | 1.0 / 1600 |
| effects fast | 1.0 / 3800 | 1.0 / 3800 |
| effects slow | 1.0 / 800 | 1.0 / 800 |

**Two things fall out of those numbers that no amount of prose about the schemes
tells you.** Simulated as unit step responses with mass 1 — which is what
androidx's `SpringSimulation` uses — and measured:

- **Spatial springs overshoot; effects springs never do.** Expressive's fast
  spatial spring travels to **1.0939**, 9.4% past its target, before settling at
  330ms. Every effects spring has damping exactly 1.0, is critically damped, and
  peaks at 1.0000. That is the whole "things that move may bounce, things that
  fade may not" rule, in two numbers.
- **The Standard scheme does not bounce at all.** At damping 0.9 its peak is
  1.0000 to four places. The difference between the two schemes is not "how much
  personality" — it is that one overshoots and one does not.

Settle times, which are what the scenes use as durations rather than round
numbers: spatial default 382ms, spatial fast 330ms, spatial slow 527ms, effects
default 193ms, effects fast 125ms, effects slow 272ms.

CSS has no spring, so each was sampled into a `linear()` of 33 stops. They are
pasted at the top of `spring-stat-tiles`, `aurora-prompt-box`, `ship-it-toolbar`
and `shared-axis-steps`; copy from any of those rather than re-deriving them.

**The trap.** Do not put a spatial spring on an `opacity`. The overshoot goes
past 1.0, gets clamped, and the bounce silently becomes a stall — the element
just sits at full opacity for the length of the overshoot. Position and scale get
spatial springs; opacity and colour get effects springs. That split is the reason
those scenes declare two animations on an element where one would look simpler.

## Shared axis: 30 pixels, not a screen width

The fourth M3 pattern, and the one most often built wrong. Google's own
implementation, `shared_axis_transition.dart`:

- outgoing: translate `0 -> -30`, opacity `1 -> 0` over `Interval(0.0, 0.3)`,
  `legacyAccelerate`
- incoming: translate `+30 -> 0`, opacity `0 -> 1` over `Interval(0.3, 1.0)`,
  `legacyDecelerate`
- the scaled variant substitutes scale `0.80 -> 1.00` in and `1.00 -> 1.10` out

Thirty **logical pixels**. On a 360dp phone that is 8.3% of the width — a nudge
that says "same track, next item", not a slide. `shared-axis-steps` converts that
honestly to this frame: 8.3% of 1920 = **160px**, stated in the file because it is
the one number there that is ours rather than Google's.

The easings are the *legacy* set, not the M3 standard curves, and that is
deliberate on Google's part rather than an oversight: `legacy`
`cubic-bezier(0.4,0,0.2,1)`, `legacyDecelerate` `(0,0,0.2,1)`, `legacyAccelerate`
`(0.4,0,1,1)`, from Flutter's `Easing` class.

The related Fade pattern (`fade_scale_transition.dart`) enters over the first 30%
with scale `0.80 -> 1.00` at 150ms and exits in 75ms. Note that Google's Android
docs quote different numbers for the same pattern — short2 100ms in, short1 50ms
out. The two implementations disagree; pick one and say which.

## What frame-by-frame gave that a spec could not

`aurora-prompt-box` and `ship-it-toolbar` came from watching a real film rather
than reading about one: "Build Android Apps From A Text Prompt with Google AI
Studio", Google for Developers, 31.4s, opened in a browser on 4 ก.ย. 2569,
paused, stepped, and its frames drawn to a canvas so the saturated pixels could
be bucketed by hue. Three findings, none of which is in any specification:

1. **The glow around the prompt box rotates hue across the length of the piece.**
   Sampled: green at 3s, green and amber at 12s, blue-teal at 18s, a heavy amber
   wash at 21s, green again at 25s. It looks like a rainbow border in a still and
   it is not one — it is one slow turn. `aurora-prompt-box` makes that a full
   360° over its 8 seconds, which is why its duration is not adjustable without
   leaving the hue stopped somewhere arbitrary.
2. **The payoff beat is lit warmer than the work.** At t=21s, the moment the
   thing gets shipped, ~29,600px sit at hue 40° and ~18,300px at hue 20°. Every
   beat before it is neutral or cool. This is the cheapest idea in the whole film
   and the most transferable: light the payoff warm.
3. **The film ends on a mark alone, small, dead centre, on near-black.** No
   headline, no URL, no tagline, after thirty seconds of dense interface. The
   restraint is the punctuation.

### What only a legible frame could give — and four things it corrected

Everything above came from average colours and cut timings. Those describe a film; they
do not show you a layout. Reading one needed the frame at a size a person can read, and
the browser pane is 470px wide, so: crop half a frame into a full-viewport canvas and it
arrives at **1.6x native**. That was done on 4 ก.ย. 2569 and it corrected four things the
first pass had guessed wrong. All four are now fixed in the scenes.

| Guessed | Actually |
|:--|:--|
| The panel grows to fill the frame | **It floats, inset, with the colour field showing all the way round it.** The colour is *behind* the panel, not inside it — this is the single most identifying thing about the register, and filling the frame destroys it |
| Buttons are filled white pills | **Every control is an outlined pill.** Nothing is a solid block, including on press: the press brightens the outline and adds about 10% white, it does not flip to solid |
| The hero headline is white | **It is blue and light-weight** — the only coloured type in the shot, on an otherwise monochrome dark stage |
| The film closes on a small mark | **It closes on a wordmark at headline size**, centred, alone on black. At small size the same idea reads as a footer instead of an ending |

Two things the first pass had right and are worth keeping: the input box really does carry
a gradient outline that travels (red-orange on one edge, green on the other), and the
suggestion chips really do sit in a rail below the box.

One more that only became visible at this size: the device control strip is **a separate
floating rounded column**, not attached to the panel it belongs to, and the phone sits
left of centre inside the preview area with a lot of empty dark to its right. Emptiness is
used as composition, not left over.

**The method note, because it will be needed again.** A 64x36 average tells you a film is
dark and where it cuts. It cannot tell you a layout, and reading one off it produces
confident, wrong answers — which is exactly what happened here. If a scene is going to be
built from a film, look at a frame you can actually read first.

And one device worth naming on its own: **the cursor is a character.** It crosses
the frame, hovers, presses. Of the 59 scenes on this shelf, only these two have a
pointer in them, and in the film the pointer is what carries the story from beat
to beat. If a scene is about somebody using software, the cursor is the actor — and it
is lit to be followed, not to be realistic: large, bright green, with a dark keyline.

**A note on method, since it will come up again.** Reading a video's title or
description tells you nothing about how it looks; stepping its frames tells you
everything. The frames were viewed, measured and discarded — nothing was saved
and no frame is reproduced in any file here. The sampled hex values are recorded
in `CREDITS.md` as evidence of what was measured, and deliberately not used: one
of them rounds to a brand colour.

## The edit, measured

Everything above is about how one scene behaves. This section is about how
several of them go together, and it is the only part of this file that was
obtained by taking a film apart completely rather than sampling it.

**Method.** The 31.4s Google for Developers film was played at 1.5x with every
frame drawn to a 64x36 canvas and reduced to a mean RGB plus a per-pixel
difference against the previous frame. **1,250 samples, one every 25ms.** A cut
is a difference spike above mean + 4 standard deviations that is also a local
peak. Nothing was saved; the numbers below are all that survives.

### Where the cuts are

```
12.55  15.78  18.12  19.32  20.40  22.16  24.17  25.72  26.76  28.36
```

Ten cuts in 31.4 seconds, and the shape of that list matters more than the count:

- **Nothing is cut for the first 12.55 seconds.** Two fifths of the film is one
  unbroken take. Whatever you assumed a product ad's rhythm is, it is not this.
- Then ten cuts in sixteen seconds, gaps of `3.23, 2.35, 1.20, 1.07, 1.76, 2.01,
  1.55, 1.04, 1.60` — **median 1.6s**, tightening to about 1.05s through the
  middle and loosening again at the end.
- **The last cut is at 28.36s and the film runs to 31.42s: a 3.06s held ending**,
  the second-longest single shot in the piece.

Patience, acceleration, rest. If you are assembling scenes from this shelf into
one piece, that is the shape to reach for — not an even cadence.

### The luminance curve, which is the real surprise

Mean frame luminance out of 255, one reading per second:

```
0-1s    L2  L4        near-black, no saturation
2-8s    L4 -> L14     still almost black; hue drifting 150deg -> 60deg
9-14s   L6 -> L9      DARKER again at 9s, then back to L15
15s     L20           starting to lift
16-18s  L53 L55 L39   the device shot. hue 202-205 (blue)
19-21s  L55 L52 L48   hue swings 96deg -> 37deg -> 29deg. saturation 0.09 -> 0.37
22s     L11           hard drop back to dark
23s     L8            hue 216 (blue), saturation 0.45
24-27s  L23 -> L56    hue 73 -> 132 (green), saturation peaks 0.46
28s     L21
29-31s  L5            saturation 0.00. colourless, and held
```

Three rules fall out of that, and none of them is in any style guide:

1. **Hold the dark.** Fourteen seconds between L2 and L15 — the film is nearly
   black for its whole first half. The brightness later is earned by that.
2. **Spend the brightness on the payoff.** The jump is a factor of three and a
   half, in one cut, on the shot that finally shows the built thing. It happens
   once. `device-reveal` is built to be that cut.
3. **End colourless.** The last three seconds sit at L5 with saturation exactly
   0.00, held without a cut. After thirty seconds of interface and colour, the
   ending is a mark on black and nothing else.

The warm-payoff finding recorded earlier fits inside this: the swing to hue 29deg
at 21s is not just warm, it is warm *at the brightest moment of the film*.

### What this changes about using these nine

- `aurora-prompt-box` (8.0s) and the film's 12.5s opening take are the same beat.
  Do not cut inside it.
- `caption-montage` cuts every 1.2s. That is inside the film's tightened middle
  range (1.04-1.20s) and deliberately faster than its 1.6s median — it is a
  montage, which is the one place a faster cadence belongs.
- `ship-it-toolbar` holds 2.6s after its cut; the film holds 3.06s. Either is
  right. What is wrong is cutting away from an ending.
- If a piece uses the dark scenes and the light ones together, put the dark ones
  first. The film's arc only works in that order, and the light register at L200+
  after a light register reads as no arc at all.
