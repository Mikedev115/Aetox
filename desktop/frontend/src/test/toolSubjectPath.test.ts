import { describe, expect, it } from 'vitest'
import { isDirectoryPathSubject, isFilePathSubject } from '../lib/toolSubjectPath'

describe('tool subject path classification', () => {
  it.each([
    'desktop/frontend/src/lib/Chat.svelte',
    'backend\\app\\main.py',
    'Dockerfile',
    'backend/Makefile',
    'LICENSE',
    '.gitignore',
  ])('recognises a file: %s', (subject) => {
    expect(isFilePathSubject(subject)).toBe(true)
    expect(isDirectoryPathSubject(subject)).toBe(false)
  })

  it.each([
    'backend/tests',
    'backend/app',
    'backend\\fixtures',
    'desktop/frontend/src/lib/',
    '.github/workflows',
  ])('recognises a directory path: %s', (subject) => {
    expect(isFilePathSubject(subject)).toBe(false)
    expect(isDirectoryPathSubject(subject)).toBe(true)
  })

  it('does not turn a web address into a local file or folder', () => {
    expect(isFilePathSubject('https://example.com/report.json')).toBe(false)
    expect(isDirectoryPathSubject('https://example.com/docs')).toBe(false)
  })
})
