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
    stops: [
      'sidebar.projects',
      'topbar.door',
      'composer.input',
      'chat.send',
      'settings.rail.brain',
      'settings.rail.heads',
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
