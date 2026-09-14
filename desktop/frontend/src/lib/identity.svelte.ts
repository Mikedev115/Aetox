// The AI's cross-project identity files (config.IdentityDir at DataRoot) —
// every *.md file in a head's folder is folded into that head's system prompt
// regardless of which project is open (internal/prompt's "Personal
// instructions" layer). Multiple files (identity.md, thinking.md, context.md,
// context.md), not one blob — independent of any single project's state.
//
// One folder per head since 14 ก.ย. 2026 (§266): ผู้ช่วย and โค้ด each have
// their own set, edited on that head's own page (ตั้งค่า › ตัวหลัก › ตัวตน).
// The store holds one head at a time — the page shows one head at a time —
// and switching heads drops the open draft, so a save can never land in the
// other head's folder.

import { ListIdentityFiles, ReadIdentityFile, SaveIdentityFile, DeleteIdentityFile } from '../../wailsjs/go/main/App'
import { t } from './i18n.svelte'

export type IdentityHead = 'assistant' | 'coding'

export const identity = $state<{
  head: IdentityHead
  files: { name: string }[]
  activeName: string
  draft: string
  saved: string
  loaded: boolean
  saving: boolean
}>({ head: 'assistant', files: [], activeName: '', draft: '', saved: '', loaded: false, saving: false })

// Reads the head's folder. Nothing is opened for editing here: the editor is
// a row until asked for (owner, 14 ก.ย. 2026: "กด + ก่อนค่อยแสดง ไม่กด ก็ไม่แสดง").
export async function loadIdentityFiles(head: IdentityHead): Promise<void> {
  if (identity.head !== head) {
    identity.head = head
    identity.activeName = ''
    identity.draft = ''
    identity.saved = ''
  }
  identity.files = (await ListIdentityFiles(head)) ?? []
  identity.loaded = true
}

export async function openIdentityFile(name: string): Promise<void> {
  identity.activeName = name
  const text = await ReadIdentityFile(identity.head, name)
  identity.draft = text
  identity.saved = text
}

export function closeIdentityFile(): void {
  identity.activeName = ''
  identity.draft = ''
  identity.saved = ''
}

export async function saveIdentityFile(): Promise<void> {
  if (!identity.activeName) return
  identity.saving = true
  try {
    await SaveIdentityFile(identity.head, identity.activeName, identity.draft)
    identity.saved = identity.draft
  } finally {
    identity.saving = false
  }
}

export async function createIdentityFile(name: string, content = ''): Promise<void> {
  const trimmed = name.trim()
  if (!trimmed) return
  const finalName = trimmed.toLowerCase().endsWith('.md') ? trimmed : trimmed + '.md'
  await SaveIdentityFile(identity.head, finalName, content)
  await loadIdentityFiles(identity.head)
  await openIdentityFile(finalName)
}

// Suggested starting files (ARCHITECTURE.md §11, 2026-07-24) — convention
// only, the engine treats every *.md in the head's folder identically.
// thinking.md is deliberately "discipline, not steps": step-by-step
// instructions can interfere with native-reasoning models, values don't.
//
// A function and not a const, because the bodies live in the locale files now
// and `t` has to be read at call time. They were Thai literals here, which made
// this the one place a user who picked another language still got Thai — and it
// is the least forgivable place for it: this is the file where they write down
// who they are, and it opened in a language they did not choose. The filenames
// stay untranslated on purpose; they are addresses the engine reads.
export function identityTemplates(): { name: string; content: string }[] {
  return [
    { name: 'identity.md', content: t('identity.tplIdentity') },
    { name: 'thinking.md', content: t('identity.tplThinking') },
    // Blank on purpose: the person's own words about themselves, not a
    // scaffold (owner, 14 ก.ย. 2026: "ควรจะโล่งเป็นค่าเริ่มต้น").
    { name: 'context.md', content: '' },
  ]
}

export async function deleteIdentityFile(name: string): Promise<void> {
  await DeleteIdentityFile(identity.head, name)
  if (identity.activeName === name) closeIdentityFile()
  await loadIdentityFiles(identity.head)
}
