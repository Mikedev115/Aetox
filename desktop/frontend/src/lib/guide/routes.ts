// The guide's routes: a walk is a list of map ids, in order. Nothing else.
//
// THIS FILE IS THE TIMELINE, and it is meant to be edited. To change what the
// guide shows and in what order, reorder these strings — no other file needs
// to know. The rules that keep an edit safe:
//
//   1. Every id must exist in map.ts (guideMap.test.ts fails otherwise), and
//      the guide can only stand beside something that is ON SCREEN when it
//      gets there — `page` in the map row is what takes it to the right page,
//      so a stop whose element only exists after a click needs the click's own
//      id before it.
//   2. Stops on one page, then the next page — every page change costs the
//      walk a beat while the page settles (pages.ts waits for the element).
//   3. What the figure SAYS at a stop is not here: it is the map's `what`,
//      from docs/GUIDE-MAP.md through the locales. Change the words there.
//   4. A walk always begins at stop 1. Nothing is remembered between openings
//      — the guide keeps no state at all — so the first stop is the one every
//      person who asks to be shown around will see.

export type GuideRouteId = 'first' | 'brain' | 'team'

export type GuideRoute = {
  id: GuideRouteId
  name: string
  stops: string[]
}

export const GUIDE_ROUTES: Record<GuideRouteId, GuideRoute> = {
  first: {
    id: 'first',
    name: 'รอบแรก',
    // The tour's order (docs/FIRST-RUN-TOUR.md), at the real buttons: the two
    // heads, where you talk, the brake, the menu that opens settings, the
    // model, the heads' page, memory, and the room that grows what it can do.
    // Every stop is on the assistant door, where a new person lands.
    stops: [
      'topbar.door',
      'composer.input',
      'chat.send',
      'sidebar.footer',
      'settings.rail.brain',
      'settings.rail.heads',
      'settings.head.tab.memory',
      'sidebar.desk.capability',
    ],
  },
  brain: {
    id: 'brain',
    name: 'ต่อสมอง',
    stops: [
      'settings.rail.brain',
      'settings.brain.hero',
      'settings.brain.provider.ollama',
      'settings.brain.add_provider',
    ],
  },
  team: {
    id: 'team',
    name: 'ทีมและยศ',
    stops: [
      'settings.rail.heads',
      'settings.head.hero',
      'settings.head.rank',
      'settings.head.tab.memory',
    ],
  },
}
