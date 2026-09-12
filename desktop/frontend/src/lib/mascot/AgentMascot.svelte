<script lang="ts">
  // An agent, drawn as the mascot — the robot with the agent's own `icon:` on
  // its ears and the hue its name gives it (or the one its file names).
  //
  // This is the frame the cartoon person (AgentFace.svelte, until 12 ก.ย. 2026)
  // stood in, kept to its contract on purpose: a name in, a face out, `state`
  // in the card's five words, `off` for one the assistant may not hand work
  // to, `size` in pixels. Seven surfaces drew that person — the office, the
  // delegate card in the chat, the background panel, ห้องความสามารถ, งานวิดีโอ,
  // the settings roster and its editor — and every one of them replaced it by
  // changing one import line. Anything the cartoon had that the mascot does
  // not (hair, glasses) is simply not a prop here; anything the mascot has
  // that the cartoon did not (shell, top, face) is one, off the same profile
  // row (agentLook.ts).
  //
  // Role is always `assistant` — COMPANY.md §6, one face: an agent is the
  // assistant's template wearing its own badge, not a second character.
  //
  // Motion follows the card, not the tile. A roster of twenty breathing
  // together is a page that stutters (MASCOT.md §3.7), so by default the
  // mascot is still — except while its state says the job is alive (`think`,
  // `work`), which is when the cartoon person moved too and when the eye is
  // on it. A caller may say `still` either way.
  import Mascot from './Mascot.svelte'
  import { hueOf } from './agentLook'
  import { poseOfFaceState, type FaceState } from './presence'

  let {
    name,
    icon = undefined,
    size = 38,
    off = false,
    hue = undefined,
    accent = undefined,
    state = '',
    shell = undefined,
    top = undefined,
    face = undefined,
    still = undefined,
  }: {
    name: string
    /** The agent's `icon:` — worn on both ears. Blank or unknown = the logo. */
    icon?: string
    size?: number
    off?: boolean
    /** Degrees, or the string the file carries; anything else = from the name
     *  (or from `accent`). A degree is full colour and wins over `accent`. */
    hue?: number | string
    /** An ACCENT row id — the colour the file names, or a persona's. */
    accent?: string
    /** What the card knows: '' | think | work | done | err. */
    state?: FaceState
    shell?: string
    top?: string
    face?: string
    /** Force no motion (a roster tile) or allow it (the one being faced).
     *  Unset = still unless the state is alive. */
    still?: boolean
  } = $props()

  const pose = $derived(poseOfFaceState(state))
  const frozen = $derived(still ?? !(state === 'think' || state === 'work'))
</script>

<Mascot {name} role="assistant" {icon} hue={hueOf(hue)} {accent} {shell} {top} {face} {pose} {size} {off} still={frozen} />
