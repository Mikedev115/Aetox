// Roles — the templates a whole mascot is picked from.
//
// The owner's sheets draw two: Aetox Assistant (orb, logo on the ears,
// neutral face, laptop with the mark) and Aetox Code (chevrons, `</>` on
// the ears, focused face, terminal on the laptop). Same body to the last
// part; the difference is four slots. A role is therefore nothing but a
// named set of slot ids — one row here — and an agent that ships with an
// `icon:` in its AGENT.md is a role too: the assistant's template with that
// icon on its ears and the hue its name gives it.
//
// What a role is NOT: who the assistant is. COMPANY.md §6 keeps identity in
// the identity layer, so Chat draws the assistant's role whatever desk it
// sits at; `code` exists for the office roster and for the day the owner
// decides a desk may wear a uniform, and that decision is his, not this
// file's.
import { isBadge } from './parts'
import type { MascotOptions } from './rig'

export type Role = {
  id: string
  label: string
  /** The slots this template fills. Anything a caller passes on top wins. */
  parts: Pick<MascotOptions, 'top' | 'badge' | 'badgeR' | 'face' | 'prop' | 'shell'>
}

// Append only.
export const ROLE: Role[] = [
  { id: 'assistant', label: 'Aetox Assistant', parts: { top: 'orb', badge: 'logo', face: 'neutral', prop: 'laptopA', shell: 'white' } },
  { id: 'code', label: 'Aetox Code', parts: { top: 'chevrons', badge: 'terminal', face: 'focused', prop: 'laptopTerm', shell: 'white' } },
]

export const DEFAULT_ROLE = 'assistant'

export function roleOf(id: string | undefined): Role {
  return ROLE.find((r) => r.id === id) ?? (ROLE.find((r) => r.id === DEFAULT_ROLE) as Role)
}

/** The slots for a mascot: a role's template, an agent's `icon:` on the ears
 *  when it has one, and whatever the caller set explicitly on top. */
export function roleOptions(roleId: string | undefined, icon: string | undefined, over: MascotOptions = {}): MascotOptions {
  const base = roleOf(roleId).parts
  const badge = isBadge(icon) ? { badge: icon, badgeR: icon } : {}
  const explicit = Object.fromEntries(Object.entries(over).filter(([, v]) => v !== undefined))
  return { ...base, ...badge, ...explicit }
}
