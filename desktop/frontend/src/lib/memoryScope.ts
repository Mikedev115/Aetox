// Whose memory a proposal is for, in words — and who will read it.
//
// A scope is a wire value with four shapes — 'user:profile' for the person,
// '' for the assistant desk, `mode:<desk>` for a desk, `project:<name>-<hash>`
// for one folder on this machine, and a bare name for a delegate
// (internal/learned). Two surfaces show it, the review list in Settings and
// the card under the answer that proposed it, and before this they both
// printed the raw string: a user deciding whether to keep a line was shown
// "mode:coding" and, once projects gained memory, "project:Aetox-1a2b3c4d".
//
// Since 11 ก.ย. the name is only half of it. The owner's finding, looking at
// the page: every label said WHICH FILE a line was in, and none said WHO READS
// IT — which is the question actually being decided when a line is approved.
// "เกี่ยวกับคุณ" costs every request the app makes; "โปรเจกต์ Aetox" costs
// nothing outside that folder. So every surface draws the audience beside the
// name, in the same tone, from one function — a person who approved a line
// under "ทุกโต๊ะจะเห็น" in the chat has to find that same sentence in Settings.

import { t } from './i18n.svelte'
import { deskLabelKey, NAV } from './desks'
import type { IconName } from './icons'

/** The user's own profile (learned.UserScope). Spelled with a colon in Go so a
 *  delegate's name can never collide with it; spelled out here rather than
 *  matched by prefix because it is the only member of its namespace, and a
 *  prefix test would quietly claim any future one. */
export const USER_SCOPE = 'user:profile'

/** The assistant desk's own file (learned.MainScope). */
export const MAIN_SCOPE = ''

/** The 8 hex characters config.ProjectKey appends so two folders named "app"
 *  cannot collide. It is identity, not information: the folder's own name is
 *  what a person recognises, so the suffix is dropped for display only. */
const KEY_HASH = /-[0-9a-f]{8}$/

/** Which kind of reader a scope has. It picks the colour every surface uses
 *  for that scope, so the same tone means the same audience everywhere. */
export type ScopeTone = 'user' | 'assistant' | 'desk' | 'project' | 'agent'

export interface ScopeMeta {
  scope: string
  /** Whose memory: "เกี่ยวกับคุณ", "โต๊ะโค้ด", "โปรเจกต์ Aetox", a delegate's name. */
  label: string
  /** Who reads it, as a sentence: the decision an approval actually makes. */
  audience: string
  tone: ScopeTone
  icon: IconName
  /** The file under memory/, for the badge — plain markdown the user can open. */
  file: string
}

export function scopeMeta(scope: string): ScopeMeta {
  const s = (scope ?? '').trim()
  if (s === USER_SCOPE) {
    return {
      scope: s, tone: 'user', icon: 'circleUser', file: 'USER.md',
      label: t('settings.learningScopeUser'),
      audience: t('settings.memoryAudienceUser'),
    }
  }
  if (!s) {
    return {
      scope: s, tone: 'assistant', icon: deskIcon('assistant'), file: 'MEMORY.md',
      label: t('settings.learningScopeMain'),
      audience: t('settings.memoryAudienceAssistant'),
    }
  }

  const desk = s.startsWith('mode:') ? s.slice(5) : ''
  if (desk) {
    // A desk the product actually draws gets its own name in the user's
    // language; anything else is a desk manifest somebody added, and its own
    // name is the only name it has.
    const key = deskLabelKey(desk)
    const name = key ? t(key) : desk
    return {
      scope: s, tone: 'desk', icon: deskIcon(desk), file: `modes/${desk}.md`,
      label: t('settings.learningScopeDesk', { name }),
      audience: t('settings.memoryAudienceDesk', { name }),
    }
  }

  const project = s.startsWith('project:') ? s.slice(8) : ''
  if (project) {
    const name = project.replace(KEY_HASH, '')
    return {
      scope: s, tone: 'project', icon: 'folder', file: `projects/${project}.md`,
      label: t('settings.learningScopeProject', { name }),
      audience: t('settings.memoryAudienceProject', { name }),
    }
  }

  // A delegate. Its name is what the user called it, so it is already the label.
  return {
    scope: s, tone: 'agent', icon: 'bot', file: `agents/${s}/MEMORY.md`,
    label: s,
    audience: t('settings.memoryAudienceAgent', { name: s }),
  }
}

/** The name alone, for a surface with room for one word. */
export function scopeLabel(scope: string): string {
  return scopeMeta(scope).label
}

/** The desk's own icon from the nav, so the memory page and the sidebar agree
 *  on what โค้ด looks like; a desk the nav does not draw gets a neutral one. */
function deskIcon(desk: string): IconName {
  return NAV.find((n) => n.kind === 'desk' && n.id === desk)?.icon ?? 'layoutList'
}
