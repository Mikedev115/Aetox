// Picking a mic is easy; picking it SAFELY is the part worth a test.
//
// The bug this store exists for is a recording that came back empty because
// Windows' default capture device was a headset jack with nothing in it. The
// fix — remember a device — has an obvious way to reintroduce exactly the same
// failure: remember a mic, unplug it, and quietly record through whatever the
// browser offers instead. So the constraint is `exact`, and the fallback is a
// deliberate second attempt that leaves a mark.
//
// Three of these four tests are about that distinction: gone falls back, denied
// does not, and a fallback is never silent.
import { describe, it, expect, beforeEach, vi } from 'vitest'

const MIC_KEY = 'aetox-audio-input'
const SPEAKER_KEY = 'aetox-audio-output'

/** The store reads localStorage once at module load, so every test that cares
 *  about a remembered device has to seed it and re-import. */
async function loadStore(saved?: { mic?: string; speaker?: string }) {
  localStorage.clear()
  if (saved?.mic) localStorage.setItem(MIC_KEY, saved.mic)
  if (saved?.speaker) localStorage.setItem(SPEAKER_KEY, saved.speaker)
  vi.resetModules()
  return await import('../lib/audioDevices.svelte')
}

function mockMediaDevices(getUserMedia: (c: MediaStreamConstraints) => Promise<MediaStream>) {
  vi.stubGlobal('navigator', {
    ...navigator,
    mediaDevices: { getUserMedia, enumerateDevices: async () => [] },
  })
}

beforeEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
})

describe('the mic the composer records from', () => {
  it('asks for the remembered device by exact id', async () => {
    const seen: MediaStreamConstraints[] = []
    mockMediaDevices(async (c) => { seen.push(c); return {} as MediaStream })
    const store = await loadStore({ mic: 'usb-mic' })

    await store.openMicStream()

    expect(seen).toHaveLength(1)
    // ideal would silently accept a different mic, which is the whole bug.
    expect(seen[0].audio).toEqual({ deviceId: { exact: 'usb-mic' } })
    expect(store.audioDevices.fellBack).toBe(false)
  })

  it('falls back to the default when the remembered device is gone, and says so', async () => {
    const seen: MediaStreamConstraints[] = []
    mockMediaDevices(async (c) => {
      seen.push(c)
      if (seen.length === 1) throw new DOMException('gone', 'OverconstrainedError')
      return {} as MediaStream
    })
    const store = await loadStore({ mic: 'unplugged' })

    await store.openMicStream()

    expect(seen).toHaveLength(2)
    expect(seen[1].audio).toBe(true)
    // The mark is what turns a silent swap into a sentence on screen.
    expect(store.audioDevices.fellBack).toBe(true)
  })

  it('does not retry when the user refused the mic', async () => {
    let calls = 0
    mockMediaDevices(async () => {
      calls += 1
      throw new DOMException('no', 'NotAllowedError')
    })
    const store = await loadStore({ mic: 'usb-mic' })

    await expect(store.openMicStream()).rejects.toThrow()
    // A second prompt for a permission just refused is nagging, and the
    // fallback would report a device fault for a permission one.
    expect(calls).toBe(1)
    expect(store.audioDevices.fellBack).toBe(false)
  })

  it('asks for whatever Windows has set when nothing is remembered', async () => {
    const seen: MediaStreamConstraints[] = []
    mockMediaDevices(async (c) => { seen.push(c); return {} as MediaStream })
    const store = await loadStore()

    await store.openMicStream()

    expect(seen[0].audio).toBe(true)
  })
})

describe('the speaker replies are read out of', () => {
  it('routes the element at the chosen device', async () => {
    const store = await loadStore({ speaker: 'headphones' })
    const setSinkId = vi.fn(async () => {})
    await store.applySpeaker({ setSinkId } as unknown as HTMLAudioElement)
    expect(setSinkId).toHaveBeenCalledWith('headphones')
  })

  it('leaves the element alone when nothing is chosen', async () => {
    const store = await loadStore()
    const setSinkId = vi.fn(async () => {})
    await store.applySpeaker({ setSinkId } as unknown as HTMLAudioElement)
    expect(setSinkId).not.toHaveBeenCalled()
  })

  it('still plays when the chosen speaker refuses the routing', async () => {
    const store = await loadStore({ speaker: 'gone' })
    const setSinkId = vi.fn(async () => { throw new Error('no such device') })
    // Losing the routing is worth less than losing the audio, so this resolves.
    await expect(
      store.applySpeaker({ setSinkId } as unknown as HTMLAudioElement),
    ).resolves.toBeUndefined()
  })
})
