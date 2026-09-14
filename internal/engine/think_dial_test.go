package engine

import (
	"testing"

	aetoxapp "github.com/Mikedev115/Aetox/internal/app"
	"github.com/Mikedev115/Aetox/internal/cognitive"
	"github.com/Mikedev115/Aetox/internal/safety"
)

// chatBehind puts the least a chat needs behind a conversation for the dials
// to find one: SwitchThinkLevel goes live only when conv.chat exists, because
// the live path has an executor to hand the level to and the parked path does
// not. No provider — nothing here sends a request.
func chatBehind(t *testing.T, conv *conversation) {
	t.Helper()
	// codex rather than dialledChat's deepseek: its level table answers with
	// no models.dev catalog installed, and a test process has none — on
	// deepseek every level normalizes to "" here and the dial has nothing to
	// move. The screenshot the change came from was this very pair.
	conv.cfg.ModelProvider, conv.cfg.ModelName = "codex", "gpt-5.6-luna"
	agent := cognitive.NewAgent(cognitive.AgentConfig{SystemPrompt: "a test system prompt"})
	chat, err := aetoxapp.NewApp(aetoxapp.Options{Agent: agent, Console: aetoxapp.NewStdIO()})
	if err != nil {
		t.Fatalf("NewApp() = %v", err)
	}
	conv.chat = chat
}

// The depth dial is the one model-menu dial a running turn takes at once: it
// is a field on the request the loop rebuilds every round, so a press
// mid-turn changes the engine's answer now, draws no "รอบถัดไปจะใช้" row, and
// is still there after the turn ends.
func TestThinkLevelMovesUnderItsOwnTurnWithoutQueueing(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	conv := dialledChat(t, a)
	conv.cfg.ThinkLevel = "low"
	chatBehind(t, conv)

	if err := a.beginTurn(conv.id); err != nil {
		t.Fatalf("beginTurn() = %v", err)
	}
	info, err := a.SwitchThinkLevel("high")
	if err != nil {
		t.Fatalf("SwitchThinkLevel mid-turn = %v", err)
	}
	if conv.cfg.ThinkLevel != "high" {
		t.Errorf("conv.cfg.ThinkLevel = %q under the running turn, want high — the press is for this turn", conv.cfg.ThinkLevel)
	}
	if info.ThinkLevel != "high" {
		t.Errorf("ModelInfo.ThinkLevel = %q, want the chip to say what the next round asks for", info.ThinkLevel)
	}
	if info.Pending != nil {
		t.Errorf("Pending = %+v, want nothing queued: no model moved", info.Pending)
	}
	if conv.cfg.ModelName != "gpt-5.6-luna" {
		t.Errorf("ModelName = %q, want the engine left where it was", conv.cfg.ModelName)
	}

	a.endTurn(conv.id)
	if conv.cfg.ThinkLevel != "high" {
		t.Errorf("conv.cfg.ThinkLevel = %q after endTurn, want the press kept", conv.cfg.ThinkLevel)
	}
}

// A level asked of a model that is itself queued belongs to that model — it
// was normalized for it — and lands with it, exactly as §232 had it.
func TestThinkLevelOnAQueuedModelStaysWithTheQueue(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	conv := dialledChat(t, a)
	chatBehind(t, conv)

	if err := a.beginTurn(conv.id); err != nil {
		t.Fatalf("beginTurn() = %v", err)
	}
	if _, err := a.SwitchModel("gpt-5.6-sol"); err != nil {
		t.Fatalf("SwitchModel mid-turn = %v", err)
	}
	before := conv.cfg.ThinkLevel
	info, err := a.SwitchThinkLevel("high")
	if err != nil {
		t.Fatalf("SwitchThinkLevel mid-turn = %v", err)
	}
	if conv.cfg.ThinkLevel != before {
		t.Errorf("conv.cfg.ThinkLevel moved to %q under a queued model, want it left for the queue", conv.cfg.ThinkLevel)
	}
	if info.Pending == nil || info.Pending.ModelName != "gpt-5.6-sol" || info.Pending.ThinkLevel != "high" {
		t.Fatalf("Pending = %+v, want the queued model carrying the level", info.Pending)
	}
}

// Full access pressed while a model switch was already queued used to land at
// the boundary with the gate back on "ask": the park was a copy taken before
// the press, and endTurn rebuilt from it whole.
func TestApprovalPressedBehindAQueuedSwitchSurvivesTheBoundary(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	conv := dialledChat(t, a)
	conv.cfg.ApprovalMode = string(safety.ApprovalAsk)

	if err := a.beginTurn(conv.id); err != nil {
		t.Fatalf("beginTurn() = %v", err)
	}
	if _, err := a.SwitchModel("deepseek-reasoner"); err != nil {
		t.Fatalf("SwitchModel mid-turn = %v", err)
	}
	if _, err := a.SwitchApprovalMode(string(safety.ApprovalFullAccess)); err == nil && conv.cfg.ApprovalMode != string(safety.ApprovalFullAccess) {
		t.Fatalf("conv.cfg.ApprovalMode = %q right after the press, want full access now", conv.cfg.ApprovalMode)
	}

	a.endTurn(conv.id)

	if conv.cfg.ModelName != "deepseek-reasoner" {
		t.Errorf("ModelName = %q after endTurn, want the queued switch landed", conv.cfg.ModelName)
	}
	if conv.cfg.ApprovalMode != string(safety.ApprovalFullAccess) {
		t.Errorf("ApprovalMode = %q after endTurn, want the press kept through the queued rebuild", conv.cfg.ApprovalMode)
	}
}
