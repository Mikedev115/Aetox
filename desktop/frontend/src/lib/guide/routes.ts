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
