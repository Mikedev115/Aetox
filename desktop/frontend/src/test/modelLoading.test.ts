// A local runtime reading the weights off disk is a wait with nothing else on
// screen to explain it — no token, no tool, no thinking — and it is the wait
// people read as a hung app.
//
// Owner, 6 ก.ย.: "อยากให้มีแบบนี้ด้วย ตอนโหลดโมเดลผ่าน Ollama หรือ LM studio".
// The row says which model and for how long. It deliberately says no
// percentage: neither runtime reports one, and the number would be invented
// (desktop/model_load.go).
import { describe, it, expect, beforeEach, vi } from "vitest"
import { render } from "@testing-library/svelte"
import { tick } from "svelte"
import Chat from "../lib/Chat.svelte"
import { cockpit, applyModelLoading } from "../lib/stores/cockpit.svelte"
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
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: { provider: "ollama", modelName: "qwen3", thinkLevel: "high", approval: "ask", wireFormat: "" } as any,
  messages: [{ role: "user", text: "ไปสิ", time: "22:19" }] as any,
}

beforeEach(() => {
  setLocale("en")
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  cockpit.modelLoading = null
  cockpit.awaitingReply = false
  cockpit.turnSession = ""
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

const row = (c: HTMLElement) => c.querySelector(".model-load")

describe("local model loading row", () => {
  it("names the model and how long, with no percentage", async () => {
    const { container } = render(Chat, {
      ...baseProps,
      modelLoading: { loading: true, provider: "ollama", model: "qwen3:8b", secs: 12 },
    })
    await tick()
    const el = row(container) as HTMLElement
    expect(el).not.toBeNull()
    expect(el.textContent).toContain("qwen3:8b")
    expect(el.textContent).toContain("12")
    expect(el.textContent).not.toContain("%")
  })

  it("still draws the wait when no model name is pinned", async () => {
    const { container } = render(Chat, {
      ...baseProps,
      modelLoading: { loading: true, provider: "lmstudio", model: "", secs: 3 },
    })
    await tick()
    expect(row(container)).not.toBeNull()
  })

  it("draws nothing at all when nothing is loading", async () => {
    const { container } = render(Chat, baseProps)
    await tick()
    expect(row(container)).toBeNull()
  })
})

describe("applyModelLoading", () => {
  it("keeps the wait, then lets the engine take it away", () => {
    cockpit.awaitingReply = true
    applyModelLoading({ loading: true, provider: "ollama", model: "qwen3", secs: 2 } as any)
    expect(cockpit.modelLoading?.model).toBe("qwen3")
    // The clearing event the watcher sends once the weights are in.
    applyModelLoading({ loading: false, provider: "ollama", model: "", secs: 0 } as any)
    expect(cockpit.modelLoading).toBeNull()
  })
})
