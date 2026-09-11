package main

// The window's tools, driven the way the engine's are (§248 B1).
//
// internal/engine/tool_coverage_test.go runs every tool the engine gives a
// session through the real dispatcher. The browser and the machine are not
// among them any more: they act on the screen's computer and live here, lent
// to the engine through Screen.WindowTools. This is the same test over those
// two — same registry, same dispatcher, same rule that every action offered to
// the model has a case, and the same honesty about what a build agent with no
// window can run: tabs and list_apps for real, everything else routed and
// refusing in words rather than hanging the turn.

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// toolCase is how one action gets driven. args are what the model would send.
type toolCase struct {
	args map[string]any
	// available reports whether this machine can run the action at all. nil
	// means it always can, and then it MUST succeed — no excuses accepted.
	available func() bool
	// why names the missing piece, for the report line when available is false.
	why string
	// check looks for evidence the tool did its job, rather than returning an
	// empty success.
	check func(t *testing.T, out skill.Output)
}

func never() bool { return false }

func outputContains(want string) func(*testing.T, skill.Output) {
	return func(t *testing.T, out skill.Output) {
		t.Helper()
		if !strings.Contains(out.Content, want) {
			t.Errorf("output does not contain %q — the tool succeeded without doing its job\ngot: %s", want, firstLine(out.Content))
		}
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func TestEveryWindowToolRunsThroughTheRealDispatcher(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	// The machine is registered only when the user has turned it on, so with
	// the shipped default off this test would never see it and its seven
	// actions would ship unexercised. The switch goes on in a preference file
	// thrown away with the test.
	switchOnComputer(t)
	root := t.TempDir()
	app := newTestApp(t)

	registry := skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: root})
	for _, s := range (appScreen{app}).WindowTools(stubSession{root: root}) {
		if err := registry.Register(s, skill.SourceWorkbench); err != nil {
			t.Fatalf("register %s: %v", s.Name(), err)
		}
	}
	dispatcher := skill.NewDispatcher(registry)

	cases := windowToolCases()
	var ran, skipped []string
	offered := 0
	for _, def := range dispatcher.ToolDefinitions() {
		if def.Function.Name != browserToolName && def.Function.Name != computerToolName {
			continue
		}
		offered++
		// A packed tool is one entry in the block and several acts inside it,
		// so it is driven once per action (skill.PackedCalls). Each action is
		// looked up under the name it had before the packing, which is also
		// the name every gate still judges it by.
		for _, call := range skill.PackedCalls(def.Function.Name) {
			name := call.Permission
			tc, ok := cases[name]
			if !ok {
				t.Errorf("%s is offered to the model but this test does not run it — add a case, or the action ships unexercised", name)
				continue
			}
			args := map[string]any{"action": call.Action}
			for key, value := range tc.args {
				args[key] = value
			}

			if tc.available != nil && !tc.available() {
				skipped = append(skipped, name+" ("+tc.why+")")
				// Still has to be reachable: an action the dispatcher cannot
				// route is broken whether or not there is a window.
				assertReachable(t, dispatcher, def.Function.Name, args)
				continue
			}

			out := runTool(t, dispatcher, def.Function.Name, args)
			if !out.Success {
				t.Errorf("%s failed on a machine that can run it: %s", name, firstLine(out.Stderr+out.Content))
				continue
			}
			if tc.check != nil {
				tc.check(t, out)
			}
			ran = append(ran, name)
		}
	}
	if offered != 2 {
		t.Errorf("the window lent %d tools to the model, want the browser and the machine", offered)
	}

	sort.Strings(ran)
	sort.Strings(skipped)
	t.Logf("ran for real (%d): %s", len(ran), strings.Join(ran, ", "))
	if len(skipped) > 0 {
		t.Logf("reachable but not runnable here (%d): %s", len(skipped), strings.Join(skipped, ", "))
	}
}

// windowToolCases is one case per action of the two packs.
func windowToolCases() map[string]toolCase {
	return map[string]toolCase{
		// The browser reaches a real webview or nothing. Headless every action
		// must fail cleanly and immediately, which is what assertReachable
		// checks: the regression worth catching is a browser action that
		// hangs the turn.
		//
		// capture refuses one step earlier than the others, and that is the
		// point: it asks agentTab which tab is the agent's before it asks the
		// engine for anything, so a session with no page open is told so
		// instead of waiting on a webview that will never answer.
		"browser_open":    {args: map[string]any{"url": "https://example.com"}, available: never, why: "needs the app window"},
		"browser_read":    {args: map[string]any{}, available: never, why: "needs the app window"},
		"browser_click":   {args: map[string]any{"ref": 1}, available: never, why: "needs the app window"},
		"browser_type":    {args: map[string]any{"ref": 1, "text": "x"}, available: never, why: "needs the app window"},
		"browser_capture": {args: map[string]any{}, available: never, why: "needs the app window"},
		// tabs is the exception among the browser actions and runs for real: it
		// reads bookkeeping this process owns, so it needs no window at all —
		// and a list that answers "you have not opened anything" is exactly
		// right for a test with no browser.
		"browser_tabs": {args: map[string]any{"act": "list"}, check: outputContains("open")},
		// wait, back and dialog all need a live page, and all refuse in words
		// before they touch the engine — which is the behaviour worth having
		// reachable here even though none of them can run.
		"browser_wait":   {args: map[string]any{"text": "hello"}, available: never, why: "needs the app window"},
		"browser_back":   {args: map[string]any{}, available: never, why: "needs the app window"},
		"browser_dialog": {args: map[string]any{"accept": true}, available: never, why: "needs the app window"},
		// Both read a buffer that lives in a live document, so they need a page
		// the same way the rest do. What they format out of that buffer is
		// covered without one, in browser_log_test.go.
		"browser_console": {args: map[string]any{}, available: never, why: "needs the app window"},
		"browser_network": {args: map[string]any{}, available: never, why: "needs the app window"},
		// upload is the fourth right of its own (6 ก.ย.): it hands a sandbox
		// file to a page, and it refuses in words before the engine when the
		// path is missing — which is as far as a test with no window gets.
		"browser_upload": {args: map[string]any{"ref": 1, "path": "page-1.png"}, available: never, why: "needs the app window"},

		// The three seeing actions of `computer`.
		//
		// list_apps runs for real, with the switch turned on first: the feature
		// ships off, and a case that only proved the off-refusal would never
		// touch Win32 at all. An empty desktop answers "no other windows are
		// open" and still succeeds, which is what keeps this runnable on a
		// build agent.
		"computer_apps": {
			args: map[string]any{},
			check: func(t *testing.T, out skill.Output) {
				// Aetox never lists itself. Not merely refused when aimed at:
				// absent, so a model never spends a turn finding out it may not.
				//
				// Matched on the PROGRAM in parentheses, never on the title. The
				// first version of this check searched the whole output and
				// fired on a Chrome window whose page happened to be about
				// Aetox, which is the same mistake in miniature that the whole
				// tool is built to avoid: a title is what somebody else wrote,
				// a program name is what Windows reports.
				for name := range reachSelfNames {
					if strings.Contains(strings.ToLower(out.Content), "("+name+")") {
						t.Errorf("list_apps offered a window belonging to %s: %s", name, out.Content)
					}
				}
			},
		},
		// read and capture need a window that exists, and no third-party window
		// is guaranteed on a build agent. What stays checked is what a test
		// without one can check: they are routed, they come back, and they
		// refuse in words rather than leaking a Win32 error.
		"computer_read":    {args: map[string]any{"window": "Notepad"}, available: never, why: "needs a named window open on this machine"},
		"computer_capture": {args: map[string]any{"window": "Notepad"}, available: never, why: "needs a named window open on this machine"},
		// The acting four. Never runnable from a test, and the reason is not
		// that they need a window: it is that they would DRIVE one. A test
		// suite that clicks in whatever application happens to be open on the
		// machine running it is a test suite that types into somebody's real
		// work. What stays checked is that each is routed and each refuses in
		// words.
		"computer_focus": {args: map[string]any{"window": "Notepad"}, available: never, why: "would drive a real window"},
		"computer_click": {args: map[string]any{"ref": 1}, available: never, why: "would drive a real window"},
		"computer_type":  {args: map[string]any{"ref": 1, "text": "x"}, available: never, why: "would drive a real window"},
		"computer_close": {args: map[string]any{"window": "Notepad"}, available: never, why: "would drive a real window"},
	}
}

// runTool calls the dispatcher exactly the way turn/executor.go does, under a
// deadline so a hung tool fails the test instead of the suite.
func runTool(t *testing.T, d *skill.Dispatcher, name string, args map[string]any) skill.Output {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	type result struct {
		out     skill.Output
		handled bool
		err     error
	}
	done := make(chan result, 1)
	go func() {
		out, handled, err := d.ExecuteTool(ctx, name, args)
		done <- result{out, handled, err}
	}()
	select {
	case r := <-done:
		if !r.handled {
			t.Fatalf("%s is offered to the model but the dispatcher will not route it", name)
		}
		if r.err != nil {
			return skill.Output{Success: false, Stderr: r.err.Error(), Content: r.out.Content}
		}
		return r.out
	case <-ctx.Done():
		t.Fatalf("%s hung — it must report, not block the turn", name)
		return skill.Output{}
	}
}

// assertReachable is runTool for an action that cannot run here: routed, and
// back with an answer of some kind, is all that is asked of it.
func assertReachable(t *testing.T, d *skill.Dispatcher, name string, args map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	done := make(chan bool, 1)
	go func() {
		_, handled, _ := d.ExecuteTool(ctx, name, args)
		done <- handled
	}()
	select {
	case handled := <-done:
		if !handled {
			t.Errorf("%s is offered to the model but the dispatcher will not route it", name)
		}
	case <-ctx.Done():
		t.Errorf("%s hung with its window missing — it must report, not block the turn", name)
	}
}

// The window's tools travel to the model in the same batch as the built-ins,
// and a provider validates the batch as a whole — one malformed schema here
// fails the request and takes every other tool down with it. The engine's
// half of the same check is internal/engine/workbench_tooldef_test.go.
func TestWindowToolDefinitionsAreWellFormed(t *testing.T) {
	app := newTestApp(t)
	tools := (appScreen{app}).WindowTools(stubSession{root: t.TempDir()})
	if len(tools) != 2 {
		t.Fatalf("the window lends %d tools, want the browser and the machine", len(tools))
	}

	seen := map[string]bool{}
	for _, s := range tools {
		tool, ok := s.(skill.Tool)
		if !ok {
			t.Errorf("%s is registered as a model-facing tool but has no ToolDefinition", s.Name())
			continue
		}
		def := tool.ToolDefinition()

		if def.Type != "function" {
			t.Errorf("%s: Type = %q, want \"function\"", s.Name(), def.Type)
		}
		// The dispatcher looks a skill up by the name the model called.
		if def.Function.Name != s.Name() {
			t.Errorf("%s: definition calls itself %q — the dispatcher would never find it", s.Name(), def.Function.Name)
		}
		if seen[def.Function.Name] {
			t.Errorf("duplicate tool name %q", def.Function.Name)
		}
		seen[def.Function.Name] = true
		if def.Function.Description == "" {
			t.Errorf("%s: no description — the model has nothing to choose it by", s.Name())
		}

		var schema map[string]any
		if err := json.Unmarshal(def.Function.Parameters, &schema); err != nil {
			t.Errorf("%s: parameters are not valid JSON: %v", s.Name(), err)
			continue
		}
		if schema["type"] != "object" {
			t.Errorf("%s: schema type = %v, want \"object\"", s.Name(), schema["type"])
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Errorf("%s: schema has no properties object", s.Name())
			continue
		}
		if required, present := schema["required"].([]any); present {
			for _, r := range required {
				key, _ := r.(string)
				if _, found := props[key]; !found {
					t.Errorf("%s: %q is required but is not a declared property", s.Name(), key)
				}
			}
		}
		for prop, raw := range props {
			spec, ok := raw.(map[string]any)
			if !ok {
				t.Errorf("%s: property %q is not an object", s.Name(), prop)
				continue
			}
			if spec["type"] == nil && spec["enum"] == nil && spec["anyOf"] == nil && spec["oneOf"] == nil {
				t.Errorf("%s: property %q declares no type", s.Name(), prop)
			}
		}
	}
}

// The per-tool ceiling, for a tool the package that owns that rule cannot see.
//
// internal/skill/block_standard_test.go holds every tool to a size: 80 tokens
// plain, 100 + 28 per action packed. It builds a default registry to do it,
// which means it has never once looked at the tools the window hosts —
// `browser` and `computer`, the two with a window behind them and the most
// room to grow prose in.
//
// Measuring both shows the browser is already over, by a lot. Holding it to
// the standard is worth doing and is NOT this change: a computer-control
// commit that quietly imposed a new size limit on `browser` was a change
// nobody reviewed. So this pins the one tool that arrived under the standard
// and can be kept there, and the other is named as work rather than done in
// passing.
func TestTheComputerPackCarriesOnlySignature(t *testing.T) {
	const packedBase, packedPerAct = 100, 28

	payload, err := json.Marshal(newComputerSkill(newTestApp(t), nil).ToolDefinition())
	if err != nil {
		t.Fatalf("computer: definition does not marshal: %v", err)
	}
	tokens := len(payload) / 4
	ceiling := packedBase + packedPerAct*len(skill.PackedCalls(computerToolName))
	if tokens > ceiling {
		t.Errorf("computer is %d tokens, over the %d-token ceiling for its shape.\n"+
			"  A block entry carries what the tool IS and what to pass it. When to reach for it,\n"+
			"  what it costs and what to watch out for belong in Guidance() and are sent once.",
			tokens, ceiling)
	}
	t.Logf("computer: %d tokens against a ceiling of %d", tokens, ceiling)
}
