import { describe, it, expect } from 'vitest'
import { GALLERY_GROUPS, GALLERY_ROLES, galleryBody, galleryIds, galleryMatches } from '../lib/agentGallery'

// The shelf and the catalogue are two lists in two files (agentGallery/*.md
// and GALLERY_ROLES), and the day they disagree a card opens onto nothing or
// a body has no card. Pinned here so that day fails the build instead.
describe('the role shelf', () => {
  it('every card has a body and every body a card', () => {
    const cards = GALLERY_ROLES.map((r) => r.id).sort()
    const files = galleryIds().sort()
    expect(cards).toEqual(files)
    expect(new Set(cards).size).toBe(cards.length)
  })

  it('every group has cards, and every card a group the gallery draws', () => {
    const drawn = new Set(GALLERY_GROUPS.map((g) => g.id))
    for (const r of GALLERY_ROLES) expect(drawn.has(r.group)).toBe(true)
    for (const g of GALLERY_GROUPS) expect(GALLERY_ROLES.some((r) => r.group === g.id)).toBe(true)
  })

  it("a card's word count is the body's, and the body is trimmed the way README says", async () => {
    for (const r of GALLERY_ROLES) {
      const body = await galleryBody(r.id)
      const words = body.split(/\s+/).filter(Boolean).length
      // The count on the card is rounded to the hundred; the catalogue's own
      // number must be within a few percent of the file or someone swapped a
      // body without the script.
      expect(Math.abs(words - r.words) / r.words, r.id).toBeLessThan(0.05)
      expect(body.startsWith('---'), r.id).toBe(false)
      expect(body, r.id).not.toMatch(/^## .*Learning & Memory/m)
      expect(body, r.id).not.toMatch(/^#+ [\u{1F300}-\u{1FAFF}]/mu)
      expect(body.includes('\r'), r.id).toBe(false)
    }
  })

  it('an id off the shelf is refused, not an empty brief', async () => {
    await expect(galleryBody('nobody')).rejects.toThrow('no gallery role')
  })

  it('search reads the title, the line and the id, case-blind', () => {
    const r = GALLERY_ROLES.find((x) => x.id === 'code-reviewer')!
    expect(galleryMatches(r, 'REVIEW')).toBe(true)
    expect(galleryMatches(r, 'correctness')).toBe(true)
    expect(galleryMatches(r, 'code-rev')).toBe(true)
    expect(galleryMatches(r, '')).toBe(true)
    expect(galleryMatches(r, 'tiktok')).toBe(false)
  })
})
