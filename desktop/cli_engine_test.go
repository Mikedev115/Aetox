package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/cliagent"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// fakeEngine is the only engine in this repository: enough of one to prove
// the seam carries a turn from the registry to the transcript and back. A real
// engine would replace its scripted events with a program's output.
type fakeEngine struct {
	provider string
	status   cliagent.Status
	events   []cliagent.Event
	result   cliagent.Result
	err      error
	seen     []cliagent.Turn
}

func (f *fakeEngine) Provider() string                      { return f.provider }
func (f *fakeEngine) Label() string                         { return "Fake" }
func (f *fakeEngine) Probe(context.Context) cliagent.Status { return f.status }
func (f *fakeEngine) Run(_ context.Context, t cliagent.Turn, on func(cliagent.Event)) (cliagent.Result, error) {
	f.seen = append(f.seen, t)
	for _, ev := range f.events {
		on(ev)
	}
	return f.result, f.err
}

// registerFake puts an engine in the registry under a name no catalog row has,
// and takes it out again after the test.
func registerFake(t *testing.T, f *fakeEngine) {
	t.Helper()
	cliagent.Register(f)
	t.Cleanup(func() { cliagentUnregister(f.provider) })
}

func TestNothingIsRegisteredInThisBuild(t *testing.T) {
	// The state the turn path relies on: every lookup misses, so runTurn
	// takes the road it always took.
	if names := cliagent.Providers(); len(names) != 0 {
		t.Fatalf("engines registered at rest: %v", names)
	}
	if _, ok := cliEngineFor("anthropic"); ok {
		t.Error("cliEngineFor hit with nothing registered")
	}
	if st := (&App{}).ExternalEngineStatus("anthropic"); st.Ready || st.Detail != "" {
		t.Errorf("status for an unserved provider = %+v", st)
	}
}

func TestCLIEngineTurnDrawsAndRecordsTheWork(t *testing.T) {
	f := &fakeEngine{
		provider: "fake-engine",
		status:   cliagent.Status{Path: "fake", Ready: true},
		events: []cliagent.Event{
			{Kind: cliagent.EventStatus, Text: "กำลังอ่านไฟล์..."},
			{Kind: cliagent.EventThinking, Text: "hmm"},
			{Kind: cliagent.EventText, Text: "สวัส"},
			{Kind: cliagent.EventText, Text: "ดี"},
			{Kind: cliagent.EventAnswer, Text: "สวัสดี"},
			{Kind: cliagent.EventToolCall, Tool: cliagent.Tool{Ref: "t1", Name: "Read", Subject: "a.go"}},
			{Kind: cliagent.EventToolResult, Tool: cliagent.Tool{Ref: "t1", OK: true}},
			// A delegate's work: under its row, never in the bubble.
			{Kind: cliagent.EventText, Text: "ignored", Parent: "t1"},
			{Kind: cliagent.EventToolCall, Tool: cliagent.Tool{Ref: "t2", Name: "Grep", Subject: "main"}, Parent: "t1"},
			{Kind: cliagent.EventAnswer, Text: "เสร็จแล้ว"},
		},
		result: cliagent.Result{Text: "เสร็จแล้ว", SessionID: "engine-said-this", Usage: cliagent.Usage{InputTokens: 40, CachedInputTokens: 30, OutputTokens: 5}},
	}
	registerFake(t, f)

	a := newTestApp(t, t.TempDir())
	conv := a.cur()
	conv.cfg.ModelProvider = "fake-engine"
	conv.cfg.ModelName = "fast"
	conv.cfg.ThinkLevel = "high"

	var chunks, tools, statuses []string
	a.emit = func(event string, data ...any) {
		if len(data) == 0 {
			return
		}
		switch event {
		case "agent:chunk":
			c := data[0].(sessionEvent[chatChunk]).Data
			if c.Replace {
				chunks = append(chunks, "="+c.Text)
			} else {
				chunks = append(chunks, "+"+c.Text)
			}
		case "agent:tool":
			ev := data[0].(sessionEvent[turn.ToolEvent]).Data
			tools = append(tools, ev.Action+":"+ev.Name+":"+ev.Ref+":"+ev.Parent)
		case "agent:status":
			statuses = append(statuses, data[0].(sessionEvent[string]).Data)
		}
	}

	engine, ok := cliEngineFor("fake-engine")
	if !ok {
		t.Fatal("engine not found after Register")
	}
	user, agent, err := a.runCLIEngineTurn(conv, context.Background(), engine, "ทักหน่อย")
	if err != nil {
		t.Fatal(err)
	}
	if user.Text != "ทักหน่อย" || user.Role != "user" {
		t.Errorf("user = %+v", user)
	}
	if agent.Text != "สวัสดี\n\nเสร็จแล้ว" {
		t.Errorf("answer = %q", agent.Text)
	}
	if agent.Reasoning != "hmm" || agent.ThinkSecs < 1 {
		t.Errorf("reasoning = %q secs=%d", agent.Reasoning, agent.ThinkSecs)
	}
	// Parts: text, tool, text — the delegate's Grep is not in the main sequence.
	kinds := make([]string, 0, len(agent.Parts))
	for _, p := range agent.Parts {
		kinds = append(kinds, string(p.Kind))
	}
	if got := strings.Join(kinds, ","); got != "text,tool,text" {
		t.Errorf("parts = %s", got)
	}
	if agent.Parts[1].Tool == nil || agent.Parts[1].Tool.Name != "Read" || agent.Parts[1].Tool.Subject != "a.go" {
		t.Errorf("tool part = %+v", agent.Parts[1].Tool)
	}
	// The bubble: fragments, then the whole answer as the authoritative write.
	if got := strings.Join(chunks, " "); got != "+สวัส +ดี =สวัสดี\n\nเสร็จแล้ว" {
		t.Errorf("chunks = %q", got)
	}
	// The timeline: call and result paired by ref, the delegate stamped.
	if got := strings.Join(tools, " "); got != "call:Read:t1: result:Read:t1: call:Grep:t2:t1" {
		t.Errorf("tools = %q", got)
	}
	if got := strings.Join(statuses, "|"); got != "กำลังคิดคำตอบ...|กำลังอ่านไฟล์...|" {
		t.Errorf("statuses = %q", got)
	}
	// What the engine was told, and what it was told next time.
	first := f.seen[0]
	if first.Resume || first.SessionID == "" || first.Model != "fast" || first.Effort != "high" || first.Text != "ทักหน่อย" {
		t.Errorf("first turn = %+v", first)
	}
	if !strings.Contains(first.SystemNote, "Aetox") {
		t.Errorf("system note = %q", first.SystemNote)
	}
	if conv.cliSession != "engine-said-this" {
		t.Errorf("session after turn = %q, engine's own id not kept", conv.cliSession)
	}
	if _, _, err := a.runCLIEngineTurn(conv, context.Background(), engine, "ต่อ"); err != nil {
		t.Fatal(err)
	}
	if second := f.seen[1]; !second.Resume || second.SessionID != "engine-said-this" {
		t.Errorf("second turn = %+v", second)
	}
}

func TestCLIEngineTurnReportsWhyItCannotRun(t *testing.T) {
	f := &fakeEngine{provider: "fake-down", status: cliagent.Status{Detail: "ติดตั้งโปรแกรมก่อน"}}
	registerFake(t, f)
	a := newTestApp(t, t.TempDir())
	conv := a.cur()
	conv.cfg.ModelProvider = "fake-down"
	engine, _ := cliEngineFor("fake-down")
	if _, _, err := a.runCLIEngineTurn(conv, context.Background(), engine, "x"); err == nil || err.Error() != "ติดตั้งโปรแกรมก่อน" {
		t.Errorf("err = %v", err)
	}
	if len(f.seen) != 0 {
		t.Error("Run was called on an engine that said it was not ready")
	}
	if a.ProviderReady("fake-down") {
		t.Error("ProviderReady said yes for an engine that is not")
	}
}

func TestCLIEngineTurnKeepsTheEnginesOwnFailure(t *testing.T) {
	f := &fakeEngine{
		provider: "fake-refused",
		status:   cliagent.Status{Path: "fake", Ready: true},
		result:   cliagent.Result{IsError: true, ErrorText: "Not logged in · Please run /login"},
	}
	registerFake(t, f)
	a := newTestApp(t, t.TempDir())
	conv := a.cur()
	conv.cfg.ModelProvider = "fake-refused"
	engine, _ := cliEngineFor("fake-refused")
	_, _, err := a.runCLIEngineTurn(conv, context.Background(), engine, "x")
	if err == nil || err.Error() != "Not logged in · Please run /login" {
		t.Errorf("err = %v", err)
	}

	f.result = cliagent.Result{}
	f.err = errors.New("exec: not found")
	_, _, err = a.runCLIEngineTurn(conv, context.Background(), engine, "x")
	if err == nil || !strings.HasPrefix(err.Error(), "Fake: ") {
		t.Errorf("program failure not labelled: %v", err)
	}
}

// cliagentUnregister reaches into the registry for tests, the same way the
// package's own tests do; production never takes an engine out.
func cliagentUnregister(provider string) { cliagent.UnregisterForTest(provider) }
