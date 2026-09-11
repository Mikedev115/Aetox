// Whose memory a proposal is for, in words. Two surfaces read this — the review
// list in Settings and the card under the answer that proposed the line — and
// they must agree: a person who approved "เกี่ยวกับคุณ" in the chat has to find
// that same name in Settings.
//
// The profile arrived on 6 ก.ย. and is the case worth pinning, because its wire
// value is the one that looks like something else. `user:profile` carries a
// colon, which is how a desk (`mode:`) and a project (`project:`) are spelled
// too — so a label built by splitting on ':' would render it as a desk named
// "profile", which is a thing that does not exist and reads like one that does.
import { describe, it, expect, beforeEach } from 'vitest'
import { setLocale } from '../lib/i18n.svelte'
import { scopeLabel, scopeMeta } from '../lib/memoryScope'

beforeEach(() => setLocale('th'))

describe('scopeLabel', () => {
  it('names the user profile as being about the user', () => {
    expect(scopeLabel('user:profile')).toBe('เกี่ยวกับคุณ')
    expect(scopeLabel('  user:profile  ')).toBe('เกี่ยวกับคุณ')
  })

  it('still names the other three scopes', () => {
    expect(scopeLabel('')).toBe('ผู้ช่วย')
    // A desk and a project keep their own shapes; the profile did not become a
    // fourth prefix, so nothing here changed when it was added.
    expect(scopeLabel('mode:coding')).toBe('โค้ด')
    expect(scopeLabel('project:Aetox-1a2b3c4d')).toContain('Aetox')
    expect(scopeLabel('project:Aetox-1a2b3c4d')).not.toContain('1a2b3c4d')
    // A delegate's name is already the label the user gave it.
    expect(scopeLabel('explore')).toBe('explore')
  })

  it('does not read the profile as a desk in English either', () => {
    setLocale('en')
    expect(scopeLabel('user:profile')).toBe('About you')
  })

  // Since 11 ก.ย. the name is half of it: who reads the file is the decision
  // an approval makes, and every surface draws it from here in one tone.
  it('says who reads each file, in the tone every surface shares', () => {
    expect(scopeMeta('user:profile')).toMatchObject({ tone: 'user', file: 'USER.md', audience: 'ทั้งผู้ช่วย โค้ด และทุกซับเอเจนจะเห็น' })
    expect(scopeMeta('')).toMatchObject({ tone: 'assistant', file: 'MEMORY.md', audience: 'เฉพาะแชทกับผู้ช่วย' })
    expect(scopeMeta('mode:coding')).toMatchObject({ tone: 'desk', file: 'modes/coding.md', icon: 'fileCode', audience: 'เฉพาะโค้ด ทุกโปรเจกต์' })
    expect(scopeMeta('project:Aetox-1a2b3c4d')).toMatchObject({ tone: 'project', file: 'projects/Aetox-1a2b3c4d.md', audience: 'เฉพาะตอนเปิดโฟลเดอร์ Aetox' })
    expect(scopeMeta('explore')).toMatchObject({ tone: 'agent', label: 'explore', audience: 'เฉพาะซับเอเจน explore' })
  })
})
