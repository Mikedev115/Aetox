// Ready-made asks — "พาไปหน้าโค้ดหน่อย", "ต่อสมองยังไง" — as buttons.
//
// Each one is a place on the map, not a sentence to send. That matters: a
// canned sentence only works when a model is there to read it, while a map id
// works on the map brain too, which is the guide's floor (§293). So a preset
// is a walk to somewhere, and it behaves identically with or without a model.
//
// Adding one is a single row here plus a label in three locales — nothing in
// the panel, nothing in the state, no code (owner, 15 ก.ย. 2026: *"คำแนะนำ
// สำเร็จ เช่น พาไกด์หน้าโค้ดหน่อย พาไกด์หน้าผู้ช่วยหน่อย บลาๆ ทำให้เพิ่มได้ง่าย
// ด้วยในอนาคต"*). `guidePresets.test.ts` fails if a row names an id the map
// does not have, so a preset can never quietly point nowhere.

export type GuidePreset = {
  id: string
  /** A `data-guide` id from the map — where this preset takes you. */
  to: string
  /** Drawn with a little more weight, for the one or two worth reaching for
   *  first. Purely presentation. */
  lead?: true
}

/** Append only, in the order they are offered. */
export const GUIDE_PRESETS: GuidePreset[] = [
  { id: 'brain', to: 'settings.rail.brain', lead: true },
  { id: 'codeDesk', to: 'sidebar.desk.coding' },
  { id: 'assistantDesk', to: 'sidebar.desk.assistant' },
  { id: 'memory', to: 'settings.head.tab.memory' },
  { id: 'capability', to: 'sidebar.desk.capability' },
  { id: 'team', to: 'sidebar.desk.office' },
  { id: 'voice', to: 'settings.rail.voice' },
  { id: 'avatar', to: 'settings.rail.avatar' },
]

/** The locale key for a preset's label. */
export function presetLabelKey(p: GuidePreset): string {
  return `guide.preset.${p.id}`
}
