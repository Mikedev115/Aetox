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
})
