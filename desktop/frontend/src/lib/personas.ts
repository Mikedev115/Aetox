// Persona Presets architecture for Aetox Heads (assistant & coding).
// Allows selecting from curated behavior presets or choosing 'custom' for full markdown editing.

import type { TKey } from './i18n.svelte'
import type { IconName } from './icons'
import type { IdentityHead } from './identity.svelte'

export interface PersonaPreset {
  id: string
  icon: IconName
  titleKey: TKey
  descKey: TKey
  summaryKey: TKey
  getTemplates: (head: IdentityHead) => {
    identityKey: TKey
    thinkingKey: TKey
  }
}

export const PERSONA_PRESETS: PersonaPreset[] = [
  {
    id: 'default',
    icon: 'sparkles',
    titleKey: 'persona.defaultTitle',
    descKey: 'persona.defaultDesc',
    summaryKey: 'persona.defaultSummary',
    getTemplates: (head: IdentityHead) => ({
      identityKey: head === 'coding' ? 'identity.tplIdentityCoding' : 'identity.tplIdentity',
      thinkingKey: head === 'coding' ? 'identity.tplThinkingCoding' : 'identity.tplThinking',
    }),
  },
  // Future presets (e.g. friend, mentor, analyst) can be added here seamlessly.
  {
    id: 'custom',
    icon: 'pencil',
    titleKey: 'persona.customTitle',
    descKey: 'persona.customDesc',
    summaryKey: 'persona.customSummary',
    getTemplates: (head: IdentityHead) => ({
      identityKey: head === 'coding' ? 'identity.tplIdentityCoding' : 'identity.tplIdentity',
      thinkingKey: head === 'coding' ? 'identity.tplThinkingCoding' : 'identity.tplThinking',
    }),
  },
]

export function getPersona(id: string): PersonaPreset {
  return PERSONA_PRESETS.find((p) => p.id === id) ?? PERSONA_PRESETS[0]
}
