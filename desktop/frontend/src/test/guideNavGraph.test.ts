import { describe, it, expect, beforeEach } from 'vitest'
import { nextStepTo, stepsTo, checkPrecondition } from '../lib/guide/path'

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
})
