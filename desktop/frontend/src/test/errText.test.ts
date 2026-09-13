import { describe, it, expect } from 'vitest'
import { errText } from '../lib/errText'
import { t } from '../lib/i18n.svelte'

// A thrown error reaches the person as a sentence, never as the JavaScript
// engine's own "TypeError: …" line. The one JS error with a meaning of its own
// — the window calling a door the running binary was built before — is said
// as what it is (owner, 14 ก.ย. 2026, reading "window.go.main.App.X is not a
// function" off the create-project form).
describe('errText', () => {
  it('names a window/binary mismatch as one', () => {
    const err = new TypeError('window.go.main.App.PickCodeProjectsDir is not a function')
    expect(errText(err)).toBe(t('app.screenOlderThanBinary'))
    expect(errText(err)).not.toMatch(/TypeError|window\.go/)
  })

  it('keeps the engine\'s own sentence, without the class in front', () => {
    expect(errText(new Error('มีโฟลเดอร์ชื่อนี้อยู่แล้วที่ D:\\x'))).toBe('มีโฟลเดอร์ชื่อนี้อยู่แล้วที่ D:\\x')
    expect(errText('Error: ตั้งชื่อโปรเจกต์ก่อนนะครับ')).toBe('ตั้งชื่อโปรเจกต์ก่อนนะครับ')
    expect(errText('ชื่อโปรเจกต์ยาวเกินไป')).toBe('ชื่อโปรเจกต์ยาวเกินไป')
  })

  it('does not mistake an ordinary TypeError for a mismatch', () => {
    expect(errText(new TypeError('Cannot read properties of undefined'))).toBe('Cannot read properties of undefined')
  })
})
