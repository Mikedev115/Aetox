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
import { GUIDE_MAP } from './map'
import { GUIDE_ROUTES, type GuideRouteId } from './routes'

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

/**
 * The opening line for the room the user is in.
 *
 * Three shapes, in order of how much the guide can honestly claim:
 *  - it knows the room and has things to point at → names the room, says how
 *    many, asks what is unclear
 *  - it knows the room but nothing on screen is mapped → names the room and
 *    asks anyway, without promising
 *  - it does not know the room → asks plainly
 *
 * Always a question. That is the whole point of it.
 */
export function greetingFor(page: PageId | null, count = explainableOnScreen()): string {
  const roomKey = page ? ROOM_KEY[roomOf(page)] : undefined
  if (!roomKey) return t('guide.greetPlain' as never)
  const room = t(roomKey as never)
  if (count > 0) return t('guide.greetRoomCount' as never, { room, n: String(count) })
  return t('guide.greetRoom' as never, { room })
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
export function offeredWalks(): { id: GuideRouteId; label: string }[] {
  return (Object.keys(GUIDE_ROUTES) as GuideRouteId[]).map((id) => {
    const key = `guide.walk.${id}`
    const label = t(key as never)
    return { id, label: label === key ? GUIDE_ROUTES[id].name : label }
  })
}
