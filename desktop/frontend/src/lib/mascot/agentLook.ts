// What a profile says about its own mascot, and what the mascot takes.
//
// The fields arrive exactly as a `key: value` line in an AGENT.md carries
// them — strings, `hue` included, because that is what a .md is — on every
// row the Go side hands the pages (Chair, Profile). One function turns a row
// into props, because seven surfaces draw an agent and a page converting the
// row its own way is exactly the drift the cartoon faces spent a commit
// removing (faceOf, agentFace.ts, now gone).
//
// A value that is blank, mistyped or names a row this build does not have
// comes back undefined rather than as a colour or a part: these are files
// written by hand by somebody who cannot see the catalogue, and a wrong
// answer must land on the derived look — the hue off the name, the
// assistant's own template — never on grey or on an error. The catalogue
// lookups (shellOf, row) already do that for the part ids; the hue is the one
// field with arithmetic in it, so it is parsed here.
import type { MascotOptions } from './rig'

/** The look fields exactly as a profile writes them. `hair` and `accessory`
 *  were the cartoon face's and are read no more — see profile.go. */
export type LookFields = {
  icon?: string
  hue?: string | number
  shell?: string
  top?: string
  face?: string
}

/** What AgentMascot takes: the `icon:` on the ears, and the four identity
 *  dials the assistant's avatar page offers, parsed. */
export type AgentLook = Pick<MascotOptions, 'shell' | 'top' | 'face'> & {
  icon?: string
  hue?: number
}

/** A hue as a file wrote it → degrees, or undefined for anything that is not
 *  a whole number 0..360. "360" is red (0), as it is on the wheel. */
export function hueOf(raw: string | number | undefined): number | undefined {
  if (typeof raw === 'number') return Number.isFinite(raw) ? ((raw % 360) + 360) % 360 : undefined
  const s = (raw ?? '').trim()
  const n = /^\d{1,3}$/.test(s) ? Number(s) : NaN
  return Number.isFinite(n) && n <= 360 ? n % 360 : undefined
}

/** One row → AgentMascot's props. Blank fields are left out, not passed as
 *  '', so the mascot's own defaults (and Mascot.svelte's `??`) apply. */
export function lookOf(f: LookFields | undefined): AgentLook {
  const out: AgentLook = {}
  const icon = (f?.icon ?? '').trim()
  if (icon) out.icon = icon
  const hue = hueOf(f?.hue)
  if (hue !== undefined) out.hue = hue
  for (const k of ['shell', 'top', 'face'] as const) {
    const v = (f?.[k] ?? '').trim()
    if (v) out[k] = v
  }
  return out
}
