import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { toolWindow } from '../lib/toolWindow'

describe('toolWindow', () => {
  let node: HTMLElement

  beforeEach(() => {
    node = document.createElement('div')
    document.body.appendChild(node)
  })

  afterEach(() => {
    node.remove()
  })

  function mockDimensions(el: HTMLElement, { clientHeight = 200, scrollHeight = 800 } = {}) {
    Object.defineProperty(el, 'clientHeight', { configurable: true, value: clientHeight })
    Object.defineProperty(el, 'scrollHeight', { configurable: true, value: scrollHeight })
  }

  it('snaps to the bottom immediately when opened with follow: true', () => {
    mockDimensions(node, { clientHeight: 200, scrollHeight: 800 })
    node.scrollTop = 0

    toolWindow(node, { on: true, follow: true })

    // Should immediately snap to floor (800 - 200 = 600) rather than staying at 0
    expect(node.scrollTop).toBe(600)
  })

  it('does not scroll to the bottom when follow: false', () => {
    mockDimensions(node, { clientHeight: 200, scrollHeight: 800 })
    node.scrollTop = 0

    toolWindow(node, { on: true, follow: false })

    expect(node.scrollTop).toBe(0)
  })

  it('resnaps to the bottom when reopened with update(true)', () => {
    mockDimensions(node, { clientHeight: 200, scrollHeight: 800 })
    node.scrollTop = 0

    const win = toolWindow(node, false)
    expect(node.scrollTop).toBe(0)

    win.update(true)
    expect(node.scrollTop).toBe(600)
  })

  it('unpins when scrolling up via wheel event so user is not dragged down', () => {
    mockDimensions(node, { clientHeight: 200, scrollHeight: 800 })
    node.scrollTop = 0

    const win = toolWindow(node, { on: true, follow: true })
    expect(node.scrollTop).toBe(600)

    // User scrolls up with wheel
    node.dispatchEvent(new WheelEvent('wheel', { deltaY: -100 }))

    // Simulate user scrolled up to 300
    node.scrollTop = 300
    node.dispatchEvent(new Event('scroll'))

    // Trigger update/mutation
    win.update(true)

    // Should stay at 300 because it is unpinned!
    expect(node.scrollTop).toBe(300)
  })
})
