// What the guide says the moment it appears.
//
// Not a fixed hello. A person who walks over to help does not open with "ask
// me anything" — they look at what is on your screen and ask whether you are
// stuck on *this*. So the greeting names the room the user is standing in and
// then asks, which is the difference between a search box with a face and
// somebody coming to look (owner, 15 ก.ย. 2026: "กดไกด์มาหน้าไหนมันก็ควรถามนะ
// ว่าเจอปัญหาอะไรตรงไหนไหม เหมือนคนมาดูอ่ะครับว่าไม่เข้าใจตรงไหนหรือเปล่า").
//
// Its own file for the same reason routes.ts and walk.ts are: the words a
// guide opens with are tuned far more often than the machinery that shows
// them, and tuning them should not mean opening the component.

import { t } from '../i18n.svelte'
import { roomOf, type PageId } from '../rooms'
import { GUIDE_MAP, guideText } from './map'
import { GUIDE_ROUTES, routeAllowedForDesk, type GuideDesk, type GuideRouteId } from './routes'
import { cockpit } from '../stores/cockpit.svelte'

/** Room → the locale key naming it. A room with no row here is not an error:
 *  the guide falls back to the room-less greeting, which is still a question. */
const ROOM_KEY: Record<string, string> = {
  chat: 'guide.room.chat',
  settings: 'guide.room.settings',
  capability: 'guide.room.capability',
  office: 'guide.room.office',
  artifacts: 'guide.room.artifacts',
  projects: 'guide.room.projects',
  videowork: 'guide.room.videowork',
  lines: 'guide.room.lines',
}

/** The visible page name, not merely its parent room. Settings and Capability
 * both have pages whose ids deliberately differ from their labels (`models`
 * is ต่อสมอง, `team` is พนักงาน), so those aliases live here once. */
const PAGE_NAME_KEY: Partial<Record<PageId, string>> = {
  'settings.general': 'guide.settings.rail.general.name',
  'settings.appearance': 'guide.settings.rail.appearance.name',
  'settings.avatar': 'guide.settings.rail.avatar.name',
  'settings.you': 'guide.settings.rail.you.name',
  'settings.issues': 'guide.settings.rail.issues.name',
  'settings.models': 'guide.settings.rail.brain.name',
  'settings.main': 'guide.settings.rail.heads.name',
  'settings.team': 'guide.settings.rail.agents.name',
  'settings.teams': 'guide.settings.rail.teams.name',
  'settings.agents': 'guide.settings.rail.hands.name',
  'settings.hands': 'guide.settings.rail.hands.name',
  'settings.voice': 'guide.settings.rail.voice.name',
  'settings.image': 'guide.settings.rail.image.name',
  'settings.studio': 'guide.settings.rail.studio.name',
  'settings.remote': 'guide.settings.rail.remote.name',
  'settings.account': 'guide.settings.rail.account.name',
  'settings.usage': 'guide.settings.rail.usage.name',
  'settings.about': 'guide.settings.rail.about.name',
  'settings.sponsor': 'guide.settings.rail.sponsor.name',
  'capability.mine': 'capability.navMine',
  'capability.desks': 'capability.navDesks',
  'capability.agents': 'capability.navAgents',
  'capability.shelf': 'capability.navShelf',
  'capability.skills': 'capability.navSkills',
  'capability.skagents': 'capability.navSkillAgents',
  'capability.skshelf': 'capability.navSkillShelf',
  'capability.sktune': 'settings.skillTune',
  'capability.tools': 'capability.navTools',
  'capability.prompts': 'capability.navPrompts',
  'capability.habits': 'capability.navHabits',
  'capability.computer': 'capability.navComputer',
  'capability.connections': 'capability.navConnections',
}

/** A page-specific answer to “what do people normally come here to do?”. */
function pageUse(page: PageId): string {
  const key = `guide.page.${page}.use`
  const use = t(key as never)
  return use === key ? '' : use
}

/** The one decision that helps a newcomer turn the page into a workflow. */
function pageAsk(page: PageId): string {
  const key = `guide.page.${page}.ask`
  const ask = t(key as never)
  return ask === key ? '' : ask
}

/** How many mapped things are on screen right now — what the guide could
 *  explain if asked. Counted rather than claimed: a room the map barely
 *  covers should not promise it knows the place. */
export function explainableOnScreen(): number {
  if (typeof document === 'undefined') return 0
  const ids = new Set<string>()
  for (const el of document.querySelectorAll<HTMLElement>('[data-guide]')) {
    const id = el.getAttribute('data-guide')
    if (!id || ids.has(id)) continue
    const r = el.getBoundingClientRect()
    if (r.width > 0 && r.height > 0 && GUIDE_MAP.some((e) => e.id === id)) ids.add(id)
  }
  return ids.size
}

/** Room / Section detail: localized name, purpose, common use, and one useful
 * question. The question is what turns a page description into guidance. */
export function pageDetail(page: PageId | null): { name: string; what: string; use: string; ask: string } {
  if (!page) return { name: '', what: '', use: '', ask: '' }

  // Chat is one room id but two deliberately different workbenches. Read the
  // live desk so the guide describes the screen the person is actually using:
  // local-first computer assistance on one side, repo-scoped engineering and
  // the structural project map on the other.
  if (page === 'chat') {
    const side = cockpit.desk === 'coding' ? 'coding' : 'assistant'
    return {
      name: t(`guide.room.chat.${side}.name` as never),
      what: t(`guide.room.chat.${side}.what` as never),
      use: t(`guide.room.chat.${side}.use` as never),
      ask: t(`guide.room.chat.${side}.ask` as never),
    }
  }

  const exactNameKey = PAGE_NAME_KEY[page]
  const exactName = exactNameKey ? t(exactNameKey as never) : ''
  const use = pageUse(page)
  const ask = pageAsk(page)

  if (page.startsWith('settings.')) {
    const rawSec = page.slice('settings.'.length)
    const aliases: Record<string, string> = {
      models: 'brain',
      main: 'heads',
      team: 'agents',
      agents: 'hands',
      hands: 'hands',
    }
    const sec = aliases[rawSec] || rawSec
    const nameKey = `guide.settings.rail.${sec}.name`
    const whatKey = `guide.settings.rail.${sec}.what`
    const secName = t(nameKey as never)
    const what = t(whatKey as never)
    const roomName = t('guide.room.settings' as never)
    const name = exactName || (secName !== nameKey ? secName : '')
    if (name) {
      return { name: `${roomName} (${name})`, what: what !== whatKey ? what : '', use, ask }
    }
    return { name: roomName, what: '', use, ask }
  }

  if (page.startsWith('capability.')) {
    const rawCap = page.slice('capability.'.length)
    const capMap: Record<string, string> = {
      tools: 'builtins',
      mine: 'mcp',
      shelf: 'mcp',
      skills: 'skills',
      prompts: 'prompts',
      computer: 'computer',
      connections: 'connections',
    }
    const cap = capMap[rawCap] || rawCap
    const nameKey = `guide.capability.rail.${cap}.name`
    const whatKey = `guide.capability.rail.${cap}.what`
    const secName = t(nameKey as never)
    const what = t(whatKey as never)
    const roomName = t('guide.room.capability' as never)
    const name = exactName || (secName !== nameKey ? secName : '')
    if (name) {
      return { name: `${roomName} (${name})`, what: what !== whatKey ? what : '', use, ask }
    }
    return { name: roomName, what: '', use, ask }
  }

  const room = roomOf(page)
  const roomNameKey = ROOM_KEY[room]
  const roomName = roomNameKey ? t(roomNameKey as never) : room
  const roomWhatKey = `guide.room.${room}.what`
  const roomWhat = t(roomWhatKey as never)
  return { name: roomName, what: roomWhat !== roomWhatKey ? roomWhat : '', use, ask }
}

/**
 * The opening line for the room the user is in.
 *
 * Tells what this page is, what it has, and invites questions.
 */
export function greetingFor(page: PageId | null, count = explainableOnScreen()): string {
  if (!page) return t('guide.greetPlain' as never)
  const { name, what, use, ask } = pageDetail(page)
  if (!name) return t('guide.greetPlain' as never)
  const whatStr = what ? `\n\n${what}` : ''
  const useStr = use ? `\n\n${use}` : ''
  const askStr = ask ? `\n\n${ask}` : ''
  if (count > 0) return t('guide.greetRoomCount' as never, { room: name, what: whatStr, use: useStr, ask: askStr, n: String(count) })
  return t('guide.greetRoom' as never, { room: name, what: whatStr, use: useStr, ask: askStr })
}

/** A compact page explanation used between two presses in an active walk. */
export function arrivalFor(page: PageId | null): string {
  if (!page) return ''
  const { name, what, use, ask } = pageDetail(page)
  if (!name) return ''
  return t('guide.arrivedPage' as never, {
    room: name,
    what: what ? `\n\n${what}` : '',
    use: use ? `\n\n${use}` : '',
    ask: ask ? `\n\n${ask}` : '',
  })
}


/**
 * What the guide offers to show, as buttons under the greeting.
 *
 * A question with no visible answers is a search box: somebody who does not
 * know the app also does not know what to ask it. Naming the walks turns
 * "มีอะไรให้ช่วยไหม" into something a person can answer by pointing (owner,
 * 15 ก.ย. 2026: *"มีปุ่มให้เลือกไกด์ทีละส่วน"*).
 *
 * Read off GUIDE_ROUTES, so a route added there is offered here without a
 * second list to keep. The label is a locale key per route id; a route with
 * no key falls back to its own name, which is better than an empty button.
 */
export type GuideWalkCategoryId = 'setup' | 'use' | 'understand'

export type GuideContextStart = {
  id: string
  kind: 'route' | 'target' | 'page'
  label: string
}

/** The first choices belong to the screen the person is looking at. A coding
 * desk should not open with the same setup/about menu as every other room;
 * it should offer its own workbench, and hidden tools carry their mapped path
 * through the inspector and + menu. Other pages derive their choices from
 * that page's safe controls, excluding the shared navigation rail. */
export function contextualGuideStarts(page: PageId | null, desk: 'assistant' | 'coding'): GuideContextStart[] {
  if (page === 'chat' && desk === 'coding') {
    const choices: GuideContextStart[] = [
      { id: 'code_desk', kind: 'route', label: GUIDE_ROUTES.code_desk.name },
      { id: 'code_map', kind: 'route', label: GUIDE_ROUTES.code_map.name },
      { id: 'topbar.tab.terminal', kind: 'target', label: t('guide.context.code.terminal' as never) },
      { id: 'topbar.tab.editor', kind: 'target', label: t('guide.context.code.files' as never) },
      { id: 'topbar.tab.diff', kind: 'target', label: t('guide.context.code.git' as never) },
      { id: 'topbar.tab.git_log', kind: 'target', label: t('guide.context.code.timeline' as never) },
      { id: 'topbar.tab.pr', kind: 'target', label: t('guide.context.code.pr' as never) },
      { id: 'topbar.tab.browser', kind: 'target', label: t('guide.context.code.browser' as never) },
    ]
    return choices.filter((choice) => choice.label.trim())
  }

  if (page === 'chat') {
    const choices: GuideContextStart[] = [
      { id: 'assistant_desk', kind: 'route', label: GUIDE_ROUTES.assistant_desk.name },
    ]
    return choices.concat(['composer.who', 'composer.team', 'chat.starter', 'composer.input', 'composer.model'].map((id) => ({
      id,
      kind: 'target' as const,
      label: guideText(id, 'name'),
    }))).filter((choice) => choice.label.trim())
  }

  if (!page) return []
  return GUIDE_MAP
    .filter((entry) => entry.page === page
      && entry.safe
      && entry.actionType !== 'none'
      && entry.id !== 'room.back'
      && !entry.id.startsWith('settings.rail.'))
    .map((entry) => ({ id: entry.id, kind: 'target' as const, label: guideText(entry.id, 'name') }))
    .filter((choice) => choice.label.trim())
}

const WALK_CATEGORIES: { id: GuideWalkCategoryId; routes: GuideRouteId[] }[] = [
  { id: 'setup', routes: ['brain', 'team', 'connections'] },
  { id: 'use', routes: ['first', 'assistant_desk', 'workbench_panel', 'code_desk', 'code_map', 'tools', 'artifacts'] },
  { id: 'understand', routes: ['capabilities', 'memory', 'support'] },
]

export function offeredWalkCategories(): { id: GuideWalkCategoryId; label: string }[] {
  return WALK_CATEGORIES.map(({ id }) => ({
    id,
    label: t(`guide.walkCategory.${id}` as never),
  }))
}

export function offeredWalks(category?: GuideWalkCategoryId, desk?: GuideDesk): { id: GuideRouteId; label: string }[] {
  const ids = category
    ? WALK_CATEGORIES.find((entry) => entry.id === category)?.routes ?? []
    : (Object.keys(GUIDE_ROUTES) as GuideRouteId[])
  return ids.filter((id) => !desk || routeAllowedForDesk(id, desk)).map((id) => {
    const key = `guide.walk.${id}`
    const label = t(key as never)
    return { id, label: label === key ? GUIDE_ROUTES[id].name : label }
  })
}
