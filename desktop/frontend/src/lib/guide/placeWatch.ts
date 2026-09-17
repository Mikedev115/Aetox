// Noticing that the screen moved.
//
// `where.ts` can say which page is on screen, but only when somebody asks. The
// guide used to ask exactly twice — when it opened, and after a press it had
// asked for — so a person who simply used their app while it was up left it
// standing beside a button that was no longer there, still saying the sentence
// for a page they had left (owner, 15 ก.ย. 2026: *"เวลากดไปหน้าต่างๆขณะไกด์
// มันยังค้างแบบนี้อยู่ คือมันควรรู้ตัวด้วยสิว่าตอนนี้อยู่หน้าไหน"*).
//
// This is the missing sense: it watches the signs and says when the answer
// changed. Its own module, because "how do we find out" and "what do we do
// about it" are tuned at completely different times — the debounce below is a
// performance dial, and what the guide DOES when the ground moves is a
// behaviour decision that lives in guideState.
//
// It watches the document rather than subscribing to the app's router on
// purpose. The guide knows nothing about how this app changes pages, and a
// sign appearing is true however it got there — a room swapping, a section
// mounting, a panel being hidden by a class (which is why `class` and `style`
// are watched too: `currentPage` only counts signs that are laid out).

import { CONTEXT_ATTR, PLACE_ATTR, type PageId } from '../rooms'
import { currentContext, currentPage, onScreen } from './where'

// Long enough that one navigation is one answer, not a dozen — mounting a page
// fires hundreds of mutations — and short enough that the guide reacts while
// the person is still looking at what they pressed.
const SETTLE_MS = 150

/**
 * Call `onChange` whenever the place on screen becomes a different one.
 *
 * Only on a real change: a burst of mutations that leaves the same page (a
 * chat streaming, a list growing) costs one `currentPage()` and says nothing.
 * Returns the way to stop watching, and the caller must — the observer
 * outliving the figure is a leak with opinions.
 */
export function watchPlace(onChange: (page: PageId | null, contextChanged: boolean) => void): () => void {
  if (typeof document === 'undefined' || typeof MutationObserver === 'undefined') return () => {}

  let last = currentPage()
  let lastContext = currentContext()
  let timer: ReturnType<typeof setTimeout> | null = null

  const look = () => {
    timer = null
    const now = currentPage()
    const context = currentContext()
    if (now === last && context === lastContext) return
    const contextChanged = context !== lastContext
    last = now
    lastContext = context
    onChange(now, contextChanged)
  }

  const nudge = () => {
    if (timer === null) timer = setTimeout(look, SETTLE_MS)
  }

  const obs = new MutationObserver(nudge)
  obs.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['class', 'style', 'hidden', PLACE_ATTR, CONTEXT_ATTR],
  })

  return () => {
    obs.disconnect()
    if (timer !== null) clearTimeout(timer)
  }
}

/** The inspector is a sub-room inside the Code page: opening it does not
 * change `data-guide-place`, but it changes what the person can learn next.
 * Observe the same DOM evidence navigation uses so mouse, shortcut, and an
 * automatic reveal all produce the same guide reaction. */
export function watchInspector(onChange: (open: boolean) => void): () => void {
  if (typeof document === 'undefined' || typeof MutationObserver === 'undefined') return () => {}

  let last = onScreen('workbench.add_tab')
  let timer: ReturnType<typeof setTimeout> | null = null
  const look = () => {
    timer = null
    const open = onScreen('workbench.add_tab')
    if (open === last) return
    last = open
    onChange(open)
  }
  const nudge = () => {
    if (timer === null) timer = setTimeout(look, SETTLE_MS)
  }
  const obs = new MutationObserver(nudge)
  obs.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['class', 'style', 'hidden'],
  })
  return () => {
    obs.disconnect()
    if (timer !== null) clearTimeout(timer)
  }
}
