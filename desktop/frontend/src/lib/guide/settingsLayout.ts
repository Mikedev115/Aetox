export type GuideSettingsPlacement = 'left' | 'right' | 'above' | 'below' | 'inside'
export type GuideSettingsEdge = 'left' | 'right' | 'top' | 'bottom'

export interface GuideSettingsLayout {
  placement: GuideSettingsPlacement
  tabEdge: GuideSettingsEdge
  align: 'start' | 'end'
  maxHeight: number
}

interface Box {
  left: number
  right: number
  top: number
  bottom: number
}

interface Viewport {
  width: number
  height: number
}

/** Pick an attached edge that keeps both the chat and settings readable. */
export function chooseGuideSettingsLayout(box: Box, viewport: Viewport): GuideSettingsLayout {
  const margin = 12
  const gap = 10
  const preferredWidth = Math.min(292, Math.max(0, viewport.width - margin * 2))
  const preferredHeight = Math.min(460, Math.max(0, viewport.height - margin * 2))
  const room = {
    left: box.left - gap - margin,
    right: viewport.width - box.right - gap - margin,
    above: box.top - gap - margin,
    below: viewport.height - box.bottom - gap - margin,
  }

  // Prefer a side window. If both sides fit, take the more open side instead
  // of assuming the mascot is always on the same edge of the bubble.
  if (room.left >= preferredWidth || room.right >= preferredWidth) {
    const placement: 'left' | 'right' = room.right >= preferredWidth && room.right >= room.left ? 'right' : 'left'
    const align = viewport.height - box.top - margin >= preferredHeight ? 'start' : 'end'
    return { placement, tabEdge: placement, align, maxHeight: preferredHeight }
  }

  // Full-width/docked conversations have no side room. Use the taller
  // vertical edge and let the panel scroll within that exact space.
  const minimumUsefulHeight = Math.min(180, preferredHeight)
  if (room.above >= minimumUsefulHeight || room.below >= minimumUsefulHeight) {
    const placement: 'above' | 'below' = room.below >= minimumUsefulHeight && room.below >= room.above ? 'below' : 'above'
    return {
      placement,
      tabEdge: placement === 'above' ? 'top' : 'bottom',
      align: 'start',
      maxHeight: Math.max(120, Math.min(preferredHeight, room[placement])),
    }
  }

  // Phone-sized and maximized chat surfaces have no outside edge large
  // enough. Overlay as a last resort rather than clipping the control away.
  return {
    placement: 'inside',
    tabEdge: 'bottom',
    align: 'start',
    maxHeight: Math.max(120, Math.min(preferredHeight, viewport.height - margin * 2)),
  }
}
