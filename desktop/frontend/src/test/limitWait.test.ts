// A Codex plan meters five hours at a time, and when the window is spent the
// turn used to end red with "resets in 2 hours" — and everything the run had
// done was the user's to rebuild. Since §269 the turn holds on, and this row
// is the only thing that makes a two-hour hold different from a hung app.
//
// Owner, 14 ก.ย.: "ตอนมันหมดก็ขึ้นรอ รอมันมาก็ทำงานต่อได้เลย".
// The row names the provider, says it will carry on by itself, and counts the
// seconds down here — the engine states them once, at the start.
import { describe, it, expect, beforeEach, vi } from "vitest"
import { render } from "@testing-library/svelte"
import { tick } from "svelte"
import Chat from "../lib/Chat.svelte"
import { cockpit, applyLimitWait } from "../lib/stores/cockpit.svelte"
import { setLocale } from "../lib/i18n.svelte"
import { GuideTopics } from "./mocks/wailsApp"

const baseProps = {
  task: { title: "", steps: [] } as any,
  awaitingReply: true,
  agentStatus: "",
  toolSteps: [] as any[],
  streamingText: "",
  reasoningText: "",
  modelLoading: null as any,
  limitWait: null as any,
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: { provider: "codex", modelName: "gpt-5.6-luna", thinkLevel: "high", approval: "ask", wireFormat: "" } as any,
  messages: [{ role: "user", text: "ทำต่อให้จบ", time: "22:19" }] as any,
}

beforeEach(() => {
  setLocale("en")
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  cockpit.modelLoading = null
  cockpit.limitWait = null
  cockpit.awaitingReply = false
  cockpit.turnSession = ""
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

const row = (c: HTMLElement) => c.querySelector(".limit-wait")

describe("plan-window wait row", () => {
  it("names the provider and counts hours down as a clock", async () => {
    vi.useFakeTimers()
    try {
      const { container } = render(Chat, {
        ...baseProps,
        limitWait: { waiting: true, provider: "codex", secs: 2 * 3600 + 13 * 60 + 5 },
      })
      await tick()
      const el = row(container) as HTMLElement
      expect(el).not.toBeNull()
      expect(el.textContent).toContain("codex")
      expect(el.textContent).toContain("2:13:05")
      // The clock runs here, not in the engine.
      vi.advanceTimersByTime(5_000)
      await tick()
      expect(el.textContent).toContain("2:13:00")
    } finally {
      vi.useRealTimers()
    }
  })

  it("reads minutes without a zero hour", async () => {
    const { container } = render(Chat, {
      ...baseProps,
      limitWait: { waiting: true, provider: "codex", secs: 9 * 60 + 7 },
    })
    await tick()
    expect((row(container) as HTMLElement).textContent).toContain("9:07")
  })

  it("draws nothing when no window is being waited for", async () => {
    const { container } = render(Chat, baseProps)
    await tick()
    expect(row(container)).toBeNull()
  })
})

describe("applyLimitWait", () => {
  it("puts the wait up, then lets the engine take it down", () => {
    cockpit.awaitingReply = true
    applyLimitWait({ waiting: true, provider: "codex", secs: 7200 } as any)
    expect(cockpit.limitWait?.provider).toBe("codex")
    // The reset arrived and the turn resumed.
    applyLimitWait({ waiting: false, provider: "codex", secs: 0 } as any)
    expect(cockpit.limitWait).toBeNull()
  })
})
