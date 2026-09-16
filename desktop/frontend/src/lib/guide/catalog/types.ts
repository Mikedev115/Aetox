import type { PageId } from '../../rooms'

export type CatalogKind = 'target' | 'concept'

export interface CatalogEvidence {
  ref?: string // e.g. "§293", "§106"
  sourceDoc?: string // e.g. "docs/DECISIONS.md"
}

export interface StateCondition {
  selector?: string // CSS selector e.g. "[data-guide-state='modal-open']"
  attr?: string     // e.g. "aria-expanded", "data-guide-state"
  value?: string    // expected value e.g. "true", "active"
  notOnScreenMessageKey?: string // e.g. "guide.condition.need_project"
}

export interface TargetNavigationReq {
  via?: string[] // IDs of buttons that must be pressed first in order
  condition?: StateCondition
}

export interface ActionPolicy {
  safe: boolean
  actionType: 'click' | 'focus' | 'input' | 'none'
  confirmationRequired?: boolean
}

export interface BaseCatalogEntry {
  id: string
  kind: CatalogKind
  synonyms?: string[]
  evidence?: CatalogEvidence
}

export interface TargetCatalogEntry extends BaseCatalogEntry {
  kind: 'target'
  page: PageId
  head?: 'assistant' | 'coding'
  policy: ActionPolicy
  navigation?: TargetNavigationReq
}

export interface ConceptCatalogEntry extends BaseCatalogEntry {
  kind: 'concept'
  relatedTargets: string[] // linked targets e.g. ['composer.stance']
  preferredPage?: PageId   // recommended page to view when explaining
}

export type CatalogEntry = TargetCatalogEntry | ConceptCatalogEntry
