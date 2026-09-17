import { describe, expect, it } from 'vitest'
import { chooseGuideSettingsLayout } from '../lib/guide/settingsLayout'

describe('guide settings attached-window placement', () => {
  it('uses the open outside edge instead of the mascot side', () => {
    expect(chooseGuideSettingsLayout(
      { left: 80, right: 430, top: 200, bottom: 500 },
      { width: 1200, height: 800 },
    ).placement).toBe('right')

    expect(chooseGuideSettingsLayout(
      { left: 770, right: 1120, top: 200, bottom: 500 },
      { width: 1200, height: 800 },
    ).placement).toBe('left')
  })

  it('opens above or below a full-width docked conversation', () => {
    const bottomDock = chooseGuideSettingsLayout(
      { left: 12, right: 788, top: 520, bottom: 688 },
      { width: 800, height: 700 },
    )
    expect(bottomDock.placement).toBe('above')
    expect(bottomDock.tabEdge).toBe('top')

    const topDock = chooseGuideSettingsLayout(
      { left: 12, right: 788, top: 56, bottom: 224 },
      { width: 800, height: 700 },
    )
    expect(topDock.placement).toBe('below')
    expect(topDock.tabEdge).toBe('bottom')
  })

  it('keeps the menu usable inside when no outside edge can fit it', () => {
    const layout = chooseGuideSettingsLayout(
      { left: 12, right: 308, top: 12, bottom: 628 },
      { width: 320, height: 640 },
    )
    expect(layout.placement).toBe('inside')
    expect(layout.maxHeight).toBeLessThanOrEqual(616)
  })
})
