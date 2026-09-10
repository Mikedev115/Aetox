import type { ToolStep } from './types'

export interface ChangedFile {
  path: string
  name: string
  dir: string
  added: number
  removed: number
  diff?: string
}

export function splitPath(full: string): { dir: string; name: string } {
  const norm = full.replace(/\\/g, '/')
  const idx = norm.lastIndexOf('/')
  if (idx < 0) return { dir: '', name: full }
  return { dir: norm.slice(0, idx), name: norm.slice(idx + 1) }
}

export function extractChangedFiles(toolSteps: ToolStep[]): ChangedFile[] {
  const map = new Map<string, ChangedFile>()

  for (const s of toolSteps) {
    const hasDiff = !!s.diff && s.diff.trim().length > 0
    const hasLines = (s.added !== undefined && s.added > 0) || (s.removed !== undefined && s.removed > 0)
    const isEditName =
      s.name === 'edit' ||
      s.name === 'write' ||
      s.name === 'edits' ||
      s.name === 'replace_file_content' ||
      s.name === 'write_to_file' ||
      s.name === 'multi_replace_file_content' ||
      s.name === 'apply_patch'

    if (!hasDiff && !hasLines && !isEditName) continue

    const path = s.subject || s.label || ''
    if (!path) continue

    const { dir, name } = splitPath(path)
    const existing = map.get(path)

    if (existing) {
      existing.added += s.added ?? 0
      existing.removed += s.removed ?? 0
      if (s.diff) {
        existing.diff = existing.diff ? `${existing.diff}\n${s.diff}` : s.diff
      }
    } else {
      map.set(path, {
        path,
        name,
        dir,
        added: s.added ?? 0,
        removed: s.removed ?? 0,
        diff: s.diff,
      })
    }
  }

  return Array.from(map.values())
}
