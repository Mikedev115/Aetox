import { describe, it, expect, beforeEach } from 'vitest'
import { nextStepTo, stepsTo, checkPrecondition, resolveNavigation, resolvePageNavigation } from '../lib/guide/path'

describe('Conditional Navigation Graph', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('points to in-page precondition (via) when target is not visible but precondition is', () => {
    // sidebar.create_project is visible, but sidebar.create_project_name is not yet
    const createBtn = document.createElement('button')
    createBtn.setAttribute('data-guide', 'sidebar.create_project')
    createBtn.getBoundingClientRect = () => ({ width: 100, height: 30 } as DOMRect)
    document.body.appendChild(createBtn)

    const next = nextStepTo('sidebar.create_project_name')
    expect(next).toBe('sidebar.create_project')

    const steps = stepsTo('sidebar.create_project_name')
    expect(steps).toContain('sidebar.create_project')
  })

  it('uses the deepest visible in-page door after an outer panel is already open', () => {
    const inspector = document.createElement('button')
    inspector.setAttribute('data-guide', 'topbar.inspector_btn')
    inspector.getBoundingClientRect = () => ({ width: 36, height: 36 } as DOMRect)
    document.body.appendChild(inspector)

    const addTab = document.createElement('button')
    addTab.setAttribute('data-guide', 'workbench.add_tab')
    addTab.getBoundingClientRect = () => ({ width: 36, height: 36 } as DOMRect)
    document.body.appendChild(addTab)

    expect(nextStepTo('topbar.tab.browser')).toBe('workbench.add_tab')
    expect(resolveNavigation('topbar.tab.browser').nextStepId).toBe('workbench.add_tab')
  })

  it('returns null when in-page target is already on screen', () => {
    const input = document.createElement('input')
    input.setAttribute('data-guide', 'sidebar.create_project_name')
    input.getBoundingClientRect = () => ({ width: 150, height: 32 } as DOMRect)
    document.body.appendChild(input)

    const next = nextStepTo('sidebar.create_project_name')
    expect(next).toBeNull()
  })

  it('checks semantic precondition without reading internal stores', () => {
    // Condition checks target readiness
    const resWithoutModal = checkPrecondition('sidebar.create_project_name')
    expect(resWithoutModal.ok).toBe(false)
    expect(resWithoutModal.reason).toBeTruthy()

    // Add modal container with target
    const modalInput = document.createElement('input')
    modalInput.setAttribute('data-guide', 'sidebar.create_project_name')
    document.body.appendChild(modalInput)

    const resWithModal = checkPrecondition('sidebar.create_project_name')
    expect(resWithModal.ok).toBe(true)
  })

  it('resolveNavigation returns full resolution with availability, preconditions, and blockedReason', () => {
    // 1. When precondition button is visible but target is not
    const createBtn = document.createElement('button')
    createBtn.setAttribute('data-guide', 'sidebar.create_project')
    createBtn.getBoundingClientRect = () => ({ width: 100, height: 30 } as DOMRect)
    document.body.appendChild(createBtn)

    const resUnmet = resolveNavigation('sidebar.create_project_name')
    expect(resUnmet.targetId).toBe('sidebar.create_project_name')
    expect(resUnmet.available).toBe(false)
    expect(resUnmet.reachable).toBe(true)
    expect(resUnmet.nextStepId).toBe('sidebar.create_project')
    expect(resUnmet.preconditionMet).toBe(false)
    expect(resUnmet.blockedReason).toBeTruthy()

    // 2. When target is visible on screen and condition met
    const modalInput = document.createElement('input')
    modalInput.setAttribute('data-guide', 'sidebar.create_project_name')
    modalInput.getBoundingClientRect = () => ({ width: 150, height: 32 } as DOMRect)
    document.body.appendChild(modalInput)

    const resMet = resolveNavigation('sidebar.create_project_name')
    expect(resMet.targetId).toBe('sidebar.create_project_name')
    expect(resMet.available).toBe(true)
    expect(resMet.reachable).toBe(true)
    expect(resMet.nextStepId).toBeNull()
    expect(resMet.preconditionMet).toBe(true)

    // 3. For unknown target
    const resUnknown = resolveNavigation('nonexistent.target')
    expect(resUnknown.reachable).toBe(false)
    expect(resUnknown.available).toBe(false)
    expect(resUnknown.nextStepId).toBeNull()
  })

  it('treats a section rail as a door for a page destination, not as arrival', () => {
    const rail = document.createElement('button')
    rail.setAttribute('data-guide', 'settings.rail.brain')
    rail.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(rail)

    const page = resolvePageNavigation('settings.models')
    expect(page.available).toBe(false)
    expect(page.nextStepId).toBe('settings.rail.brain')
    expect(page.stepsRemaining).toEqual(['settings.rail.brain'])
  })

  it('opens a folded sidebar before pointing at a sidebar-only door', () => {
    const sidebarToggle = document.createElement('button')
    sidebarToggle.setAttribute('data-guide', 'topbar.sidebar_btn')
    sidebarToggle.setAttribute('aria-expanded', 'false')
    sidebarToggle.getBoundingClientRect = () => ({ width: 36, height: 36 }) as DOMRect
    document.body.appendChild(sidebarToggle)

    // The real compact rail keeps this footer icon measurable. That must not
    // be mistaken for the full sidebar being open.
    const footer = document.createElement('button')
    footer.setAttribute('data-guide', 'sidebar.footer')
    footer.getBoundingClientRect = () => ({ width: 57, height: 36 }) as DOMRect
    document.body.appendChild(footer)

    const folded = resolvePageNavigation('settings.models')
    expect(folded.reachable).toBe(true)
    expect(folded.nextStepId).toBe('topbar.sidebar_btn')
    expect(folded.stepsRemaining[0]).toBe('topbar.sidebar_btn')
    expect(folded.stepsRemaining).toContain('sidebar.footer')

    // Once the sidebar is open, the deeper visible door wins. The toggle is
    // still on screen but must not be pointed at again (which would fold it).
    sidebarToggle.setAttribute('aria-expanded', 'true')
    footer.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect

    const open = resolvePageNavigation('settings.models')
    expect(open.nextStepId).toBe('sidebar.footer')
    expect(open.stepsRemaining[0]).toBe('sidebar.footer')
    expect(open.stepsRemaining).not.toContain('topbar.sidebar_btn')
  })

  it('treats a visible rail as the next press when its destination page is not open yet', () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'settings.general')
    sign.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(sign)

    const rail = document.createElement('button')
    rail.setAttribute('data-guide', 'settings.rail.teams')
    rail.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(rail)

    const nav = resolveNavigation('settings.rail.teams')
    expect(nav.available).toBe(false)
    expect(nav.reachable).toBe(true)
    expect(nav.nextStepId).toBe('settings.rail.teams')
    expect(nav.stepsRemaining).toEqual(['settings.rail.teams'])
  })

  it('never points back at the active rail when a landmark on the open page is temporarily absent', () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'capability.mine')
    sign.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(sign)

    const activeRail = document.createElement('button')
    activeRail.setAttribute('data-guide', 'capability.rail.mcp')
    activeRail.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(activeRail)

    const nav = resolveNavigation('capability.mcp.header')
    expect(nav.available).toBe(false)
    expect(nav.nextStepId).toBeNull()
    expect(nav.stepsRemaining).not.toContain('capability.rail.mcp')
  })

  it('still treats a visible room control as a door while the page sign is between frames', () => {
    const capability = document.createElement('button')
    capability.setAttribute('data-guide', 'sidebar.desk.capability')
    capability.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(capability)

    const nav = resolveNavigation('sidebar.desk.capability')
    expect(nav.available).toBe(false)
    expect(nav.nextStepId).toBe('sidebar.desk.capability')
  })

  it('leaves a full-screen settings room before routing to another room', () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'settings.main')
    sign.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(sign)

    const back = document.createElement('button')
    back.setAttribute('data-guide', 'settings.back')
    back.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(back)

    // The chat shell is still mounted behind Settings and therefore still
    // measurable. Room identity must beat this stale rectangle.
    const hiddenCapability = document.createElement('button')
    hiddenCapability.setAttribute('data-guide', 'sidebar.desk.capability')
    hiddenCapability.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(hiddenCapability)

    const nav = resolveNavigation('sidebar.desk.capability')
    expect(nav.reachable).toBe(true)
    expect(nav.nextStepId).toBe('settings.back')
    expect(nav.stepsRemaining[0]).toBe('settings.back')
  })

  it('leaves the capability room before pointing at a stale sidebar behind it', () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'capability.mine')
    sign.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(sign)

    const back = document.createElement('button')
    back.setAttribute('data-guide', 'room.back')
    back.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(back)

    // The ordinary app sidebar remains measurable under the full-screen room.
    // It must not win until the user has visibly left this room.
    const staleArtifacts = document.createElement('button')
    staleArtifacts.setAttribute('data-guide', 'sidebar.desk.artifacts')
    staleArtifacts.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(staleArtifacts)

    const nav = resolveNavigation('artifacts.header')
    expect(nav.reachable).toBe(true)
    expect(nav.nextStepId).toBe('room.back')
    expect(nav.stepsRemaining[0]).toBe('room.back')
  })

  it('opens the new-team form before pointing at its name field', () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'settings.teams')
    sign.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(sign)

    const rail = document.createElement('button')
    rail.setAttribute('data-guide', 'settings.rail.teams')
    rail.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(rail)
    const create = document.createElement('button')
    create.setAttribute('data-guide', 'team.new_btn')
    create.getBoundingClientRect = () => ({ width: 120, height: 36 }) as DOMRect
    document.body.appendChild(create)

    const nav = resolveNavigation('team.name_input')
    expect(nav.reachable).toBe(true)
    expect(nav.available).toBe(false)
    expect(nav.nextStepId).toBe('team.new_btn')
    expect(nav.stepsRemaining).toEqual(['team.new_btn'])
  })
})
