// A tool subject is presentation text, not filesystem metadata. Keep this
// heuristic deliberately conservative: it only decides which affordance a
// tool row shows. The workbench still validates the path when it is opened.

const SPECIAL_FILE_NAMES = new Set([
  'dockerfile',
  'makefile',
  'license',
  'readme',
  'changelog',
  'notice',
  'copying',
  'gemfile',
  'rakefile',
  'procfile',
  'vagrantfile',
  'brewfile',
  'justfile',
  '.gitignore',
  '.gitattributes',
  '.dockerignore',
  '.editorconfig',
  '.npmrc',
  '.nvmrc',
  '.env',
])

function isWebAddress(value: string): boolean {
  return /^[a-z][a-z0-9+.-]*:\/\//i.test(value)
}

/** True when the subject ends in a filename users reasonably expect to edit. */
export function isFilePathSubject(subject: string): boolean {
  const value = subject.trim()
  if (!value || isWebAddress(value) || /[\\/]$/.test(value)) return false

  const base = value.split(/[\\/]/).pop()?.trim() ?? ''
  if (!base || base === '.' || base === '..') return false
  if (SPECIAL_FILE_NAMES.has(base.toLowerCase())) return true

  // A leading dot alone is not an extension: `.github` is commonly a folder.
  // Named dotfiles above are the intentionally supported extensionless files.
  const dot = base.lastIndexOf('.')
  if (dot <= 0 || dot === base.length - 1) return false
  return /^[a-z0-9_-]+$/i.test(base.slice(dot + 1))
}

/** A local-looking path with separators and no filename-shaped final segment. */
export function isDirectoryPathSubject(subject: string): boolean {
  const value = subject.trim()
  if (!value || isWebAddress(value) || !/[\\/]/.test(value)) return false
  return !isFilePathSubject(value)
}
