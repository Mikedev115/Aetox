---
name: aetox-ui-design
description: ลงมือ implement UI ระดับ production, design tokens/theming, responsive (container queries, fluid type, CSS grid), interaction/microinteraction/motion, คอมโพเนนต์เว็บ (React/Vue/Svelte), การเข้าถึง WCAG 2.2 และ native (iOS HIG/SwiftUI, Material 3/Compose, React Native) รวมเป็นประตูเดียว ใช้ตอนลงมือเขียนจริง ไม่ใช่แค่ตัดสินทิศทาง (ทิศทางดู aetox-frontend-design)
source: https://github.com/wshobson/agents (plugins/ui-design)
license: MIT
copyright: Copyright (c) 2024 Seth Hobson. Full terms in LICENSE
---

# UI Design (implementation)

Nine implementation guides, one per problem, the how-to-build layer.
`aetox-frontend-design` decides the look and `aetox-design-system` gives the
tokens; these turn that into code. Open the one the task is actually about;
each file carries its source skill's own reference material inline.

**Write the skeleton yourself, then check it.** `aetox-web-templates` carries
the contract a page has to pass — self-contained file, responsive without
breakpoint bookkeeping, real landmarks, tokens for light and dark, accessible by
construction — and no longer carries sections to paste. The library was removed
on 5 ก.ย. 2569: markup that encodes the average holds a capable model at the
average, while a rule it has to satisfy does not.

**On the web**

- `references/design-system-patterns.md`, design tokens, theming and
  theme-switching, component architecture. Read first when starting a system.
- `references/visual-design-foundations.md`, typography, colour theory, spacing
  systems and iconography as concrete tokens.
- `references/responsive-design.md`, container queries, fluid typography, CSS
  Grid, mobile-first breakpoints, component-level responsiveness.
- `references/interaction-design.md`, microinteractions, motion, transitions,
  loading states and feedback patterns.
- `references/web-component-design.md`, React/Vue/Svelte component patterns,
  CSS-in-JS, composition, reusable component APIs.
- `references/accessibility-compliance.md`, WCAG 2.2 in code: ARIA patterns,
  screen-reader support, mobile accessibility, inclusive patterns.

**Native**

- `references/mobile-ios-design.md`, iOS Human Interface Guidelines and SwiftUI.
- `references/mobile-android-design.md`, Material Design 3 and Jetpack Compose.
- `references/react-native-design.md`, React Native styling, navigation and
  Reanimated animations, cross-platform.

## The bar, whichever guide you opened

The look was decided elsewhere; what leaves this skill is code, and code has
numbers. Same bar for every model:

- WCAG 2.2 AA in the file, not in intent: text 4.5:1, large text and UI
  parts 3:1, targets 24 px or more with spacing, focus visible on everything
  focusable, no meaning carried by colour alone, `prefers-reduced-motion`
  respected, and the keyboard path through every flow walked once.
- Both themes from tokens, every colour defined in the bare `:root` before
  any media or `[data-theme]` block redefines it. A colour whose only
  definition sits in one theme block is the classic unreadable page.
- Responsive by the component, not by the page: container queries and
  fluid type where the guide names them; a `@media` only where the layout
  changes shape. Read at 360 px and at 1440 px.
- States are designed, not implied: loading, empty, error, disabled, long
  content, in the same component, before it is called finished.
- Looked at. `browser` capture at both widths and both themes; the console
  clean. A component that was only read in the source was not seen. Once,
  at the end, at reduced scale: a picture stays in the context and is paid
  for on every round after it, so text checks (`read`, the console) carry
  the iterations and the pictures close them.
- The guide is read whole before the first line, not searched for the
  one snippet that looks right; the rule you skipped is the one the page
  breaks on.

Adapted from wshobson/agents' ui-design plugin (MIT), nine skills folded into
one door, each kept whole with its references appended.
