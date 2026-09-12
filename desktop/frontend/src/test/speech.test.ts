import { describe, it, expect, beforeEach, vi } from 'vitest'
import { speech, speak, stopSpeech, stopSpeechIf, speechText } from '../lib/speech.svelte'
import { voice } from '../lib/mascot/voice.svelte'
import { StartSpeech, StopSpeech, SpeechPlaying } from './mocks/wailsApp'
import { EventsOn } from './mocks/wailsRuntime'

// The window's one player. What is guarded is the one-voice rule the owner
// asked for the day the companion learned to talk ("เสียงชนกัน"): a new read
// replaces the old one at both ends, pieces play one after another, and a
// piece that is late is a pause rather than a second voice.

type Chunk = { job: string; seq: number; url: string; mime: string; last: boolean; error?: string }

/** jsdom's <audio> plays nothing; this one remembers what it was asked. */
class FakeAudio {
  static made: FakeAudio[] = []
  src: string
  preload = ''
  dataset: Record<string, string> = {}
  onended: (() => void) | null = null
  onerror: (() => void) | null = null
  paused = true
  constructor(src: string) {
    this.src = src
    FakeAudio.made.push(this)
  }
  load(): void {}
  async play(): Promise<void> { this.paused = false }
  pause(): void { this.paused = true }
}

function feed(): (c: Chunk) => void {
  const call = vi.mocked(EventsOn).mock.calls.find((c) => c[0] === 'speech:chunk')
  return call![1] as (c: Chunk) => void
}

beforeEach(() => {
  vi.stubGlobal('Audio', FakeAudio)
  FakeAudio.made = []
  vi.mocked(StartSpeech).mockClear()
  vi.mocked(StopSpeech).mockClear()
  vi.mocked(SpeechPlaying).mockClear()
  stopSpeech()
  vi.mocked(StopSpeech).mockClear()
})

describe('the one player', () => {
  it('plays pieces in turn, reports each, and closes the job after the last', async () => {
    vi.mocked(StartSpeech).mockResolvedValueOnce('j1')
    await speak('m1', 'hello **world**')
    expect(vi.mocked(StartSpeech).mock.calls[0][0]).toBe('hello world')
    expect(speech.key).toBe('m1')
    expect(speech.busyKey).toBe('m1')
    expect(voice.speaking).toBe(true)

    feed()({ job: 'j1', seq: 0, url: 'u0', mime: 'audio/wav', last: false })
    await Promise.resolve(); await Promise.resolve()
    expect(FakeAudio.made.map((a) => a.src)).toEqual(['u0'])
    expect(FakeAudio.made[0].paused).toBe(false)
    expect(speech.busyKey).toBe('')
    expect(vi.mocked(SpeechPlaying).mock.calls[0]).toEqual(['j1', 0])

    // the second piece arrives while the first talks: fetched, not played
    feed()({ job: 'j1', seq: 1, url: 'u1', mime: 'audio/wav', last: true })
    expect(FakeAudio.made[1].paused).toBe(true)
    // the first ends: the second plays; the second ends: the read is over
    FakeAudio.made[0].onended?.()
    await Promise.resolve(); await Promise.resolve()
    expect(FakeAudio.made[1].paused).toBe(false)
    FakeAudio.made[1].onended?.()
    await Promise.resolve(); await Promise.resolve()
    expect(speech.key).toBe('')
    expect(voice.speaking).toBe(false)
    expect(vi.mocked(StopSpeech)).toHaveBeenCalledWith('j1')
  })

  // A piece not yet made is silence, not a second voice: with nothing
  // queued the player waits for the chunk to call back in.
  it('waits for a late piece instead of ending or overlapping', async () => {
    vi.mocked(StartSpeech).mockResolvedValueOnce('j2')
    await speak('m2', 'one')
    feed()({ job: 'j2', seq: 0, url: 'u0', mime: 'audio/wav', last: false })
    await Promise.resolve(); await Promise.resolve()
    FakeAudio.made[0].onended?.()
    await Promise.resolve()
    expect(speech.key).toBe('m2')
    expect(vi.mocked(StopSpeech)).not.toHaveBeenCalled()
    feed()({ job: 'j2', seq: 1, url: 'u1', mime: 'audio/wav', last: true })
    await Promise.resolve(); await Promise.resolve()
    expect(FakeAudio.made[1].paused).toBe(false)
  })

  it('lets the newest read replace the one playing, at both ends', async () => {
    vi.mocked(StartSpeech).mockResolvedValueOnce('a').mockResolvedValueOnce('b')
    await speak('first', 'first')
    feed()({ job: 'a', seq: 0, url: 'ua', mime: 'audio/wav', last: false })
    await Promise.resolve(); await Promise.resolve()
    await speak('second', 'second')
    expect(FakeAudio.made[0].paused).toBe(true)
    expect(vi.mocked(StopSpeech)).toHaveBeenCalledWith('a')
    expect(speech.key).toBe('second')
    // a straggler from the old job is ignored
    feed()({ job: 'a', seq: 1, url: 'ua1', mime: 'audio/wav', last: true })
    expect(FakeAudio.made.length).toBe(1)
  })

  it('stopSpeechIf ends only its own read', async () => {
    vi.mocked(StartSpeech).mockResolvedValueOnce('c')
    await speak('mine', 'x')
    stopSpeechIf('theirs')
    expect(speech.key).toBe('mine')
    stopSpeechIf('mine')
    expect(speech.key).toBe('')
  })

  // The ฟัง button hears why; the companion passes no reporter and gets
  // silence — either way the read is closed.
  it('hands errors to the caller that asked, and is silent for one that did not', async () => {
    vi.mocked(StartSpeech).mockRejectedValueOnce(new Error('ไม่มีเสียง'))
    const heard: string[] = []
    await speak('k', 'x', (e) => heard.push(e))
    expect(heard[0]).toContain('ไม่มีเสียง')
    expect(speech.key).toBe('')
    vi.mocked(StartSpeech).mockRejectedValueOnce(new Error('ไม่มีเสียง'))
    await expect(speak('k2', 'x')).resolves.toBeUndefined()
    expect(speech.key).toBe('')
    // a piece that fails mid-read
    vi.mocked(StartSpeech).mockResolvedValueOnce('d')
    await speak('k3', 'x', (e) => heard.push(e))
    feed()({ job: 'd', seq: 0, url: '', mime: '', last: false, error: 'พัง' })
    expect(heard[1]).toBe('พัง')
    expect(speech.key).toBe('')
  })

  it('reads the answer, not its markdown', () => {
    expect(speechText('# หัว\n- ข้อ `a`\n```js\ncode\n```\n[ลิงก์](http://x) **หนา**')).toBe('หัว ข้อ a ลิงก์ หนา')
  })
})
