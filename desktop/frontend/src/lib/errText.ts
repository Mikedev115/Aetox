import { t } from './i18n.svelte'

// What to show a person when a call to the app fails.
//
// `String(err)` on a thrown Error gives "TypeError: window.go.main.App.X is
// not a function" — the JavaScript engine's sentence, not ours, and the owner
// met exactly that one on the create-project form (14 ก.ย. 2026): the screen
// had hot-reloaded a page that calls a door the running binary was built
// before, and the page printed the stack's first line at him. That case has a
// meaning a person can act on — the window and the app are different builds,
// restart — so it gets said. Everything else is the message the engine wrote,
// without the class name in front of it: the engine's errors are already
// sentences (มีโฟลเดอร์ชื่อนี้อยู่แล้วที่ …), and "Error:" adds nothing to them.
export function errText(err: unknown): string {
  const msg = err instanceof Error ? err.message : String(err ?? '')
  if (/window\.go\..* is not a function/.test(msg) || /\bgo\.main\.App\b.*not a function/.test(msg)) {
    return t('app.screenOlderThanBinary')
  }
  return msg.replace(/^(TypeError|Error|RangeError|ReferenceError):\s*/, '').trim()
}
