package main

// One tool for one thing: the page open in the workbench browser.
//
// It was four — `browser_open`, `browser_read`, `browser_click`, `browser_type`
// — and they were four tools describing one object, which is the shape the
// owner rejected on 2026-08-10: "browser จริง ๆ มันควรจะแพ็ครวมกัน... หากเพิ่ม
// จะได้ไม่ต้องเสียเวลามาไล่เปิดทีละตัว ๆ". Every capability added to the browser
// used to cost another entry in the tool block of every request that carries
// it; now it costs an action.
//
// The objection worth recording, because it is what the design answers: four
// names are four *rights*, and `tools:` in a profile grants by name — collapsing
// them would make "may look at a page" and "may act on one" the same permission,
// which is not a distinction to lose in a product whose rule is that rights come
// only from a list the user can see. The owner's answer was "รวมแล้วแยกย่อย
// ข้างใน", and that is exactly what this file does: one tool on the outside, the
// four original names still the vocabulary of permission on the inside.
//
// So the gate has two levels and they answer different questions:
//
//   - **Does this caller get the browser at all?** The registry filter, on the
//     name `browser` — desk categories, and a profile's `tools:` list.
//   - **Which actions?** The old `browser_<action>` names. A profile that
//     names none of them gets every action; one that names some gets exactly
//     those. Nothing in the vocabulary changed, so a manifest written before
//     this file still means what it said.
//
// The description the model reads lists only the actions that caller may use,
// because a tool that advertises what it will refuse is a wasted turn — the
// same reasoning that keeps a connection's tools out of the block entirely
// until an account exists.
//
// For its first day this file answered the second question itself, by reading
// the open session's chair profile — it could, being the one packed tool that
// lives where chairProfile does. Then shell and github packed too, in a package
// that cannot see a profile, and skill/packed.go generalized the idea: the
// vocabulary moved to the one table every pack declares itself in, and the
// narrowing arrives from subagent.FilterRegistry like everyone else's. The
// private gate was retired the same day it would have become the second
// mechanism answering the same question.

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

type browserSkill struct {
	app *App
	// Whose conversation this pack speaks for. Every desk event the pack
	// raises is stamped with it, so a background chat's page lands on that
	// chat's own desk instead of whichever one is on screen (§187, closed for
	// the browser 1 ก.ย.). nil only where tests build the pack bare.
	conv *conversation
	// actions this caller may use, nil for all of them. Set only by Narrow.
	actions []string
}

// owner is the session id the pack's desk events carry — "" when no
// conversation is attached, which the window draws live (§187.2).
func (s *browserSkill) owner() string {
	if s.conv == nil {
		return ""
	}
	return s.conv.id
}

func (s *browserSkill) allowedActions() []string {
	if s == nil || len(s.actions) == 0 {
		out := make([]string, 0, len(skill.PackedCalls("browser")))
		for _, call := range skill.PackedCalls("browser") {
			out = append(out, call.Action)
		}
		return out
	}
	return s.actions
}

func (s *browserSkill) Actions() []string { return skill.PackedActions("browser") }

// Narrow hands back a browser offering only the named actions — a copy, for
// the same shared-registry reason as shell's.
func (s *browserSkill) Narrow(named []string) skill.Skill {
	narrowed := *s
	var actions []string
	want := map[string]bool{}
	for _, n := range named {
		want[strings.ToLower(strings.TrimSpace(n))] = true
	}
	for _, call := range skill.PackedCalls("browser") {
		if want[call.Permission] {
			actions = append(actions, call.Action)
		}
	}
	// Silence is the whole tool, not an empty one — the rule this file set.
	if len(actions) == 0 {
		return s
	}
	narrowed.actions = actions
	return &narrowed
}

// browserToolName is the one name this pack answers to, spelled once because
// the busy signal matches tool events against it (desktop/app.go
// recordToolAction) and a UI comparing against a literal is a UI that keeps
// working after the tool is renamed, while showing nothing.
const browserToolName = "browser"

func (*browserSkill) Name() string { return browserToolName }

func (*browserSkill) Description() string {
	return "ใช้งานเบราว์เซอร์ของ workbench — เปิดหน้า อ่านหน้า คลิก และกรอกข้อความ (ผู้ใช้เห็นทุกอย่างที่ทำ)"
}

func (s *browserSkill) ToolDefinition() model.ToolDefinition {
	allowed := s.allowedActions()

	// Built from the permitted set so the description never advertises an
	// action this caller would be refused.
	// SIGNATURES ONLY. Every "when should I reach for this" sentence that used
	// to live here now lives in Guidance below and is sent once, on the first
	// call to that action — internal/skill/guidance.go. This entry went from
	// 766 tokens to a fifth of that without the model being told less; it is
	// told the same things at a different time.
	lines := map[string]string{
		"open":    "`open` (url, newTab?) — go to a page and wait for it to load.",
		"read":    "`read` (filter?) — the page's text, plus its interactive elements each tagged [n]; filter lists only the elements whose text contains it.",
		"click":   "`click` (ref | find | x,y, button?: right, count?: 2|3) — press the element with that ref, the one whose text contains find, or the viewport point x,y in CSS px read off a capture; right for a context menu, count 2/3 for double/triple.",
		"type":    "`type` (ref? | find? | x,y?, text, enter?) — fill an input/textarea/select/contenteditable; with no target, keystrokes to wherever the page's focus is; with x,y, click there first.",
		"wait":    "`wait` (text, seconds?) — wait until that text appears.",
		"back":    "`back` — return to the previous page in this tab.",
		"scroll":  "`scroll` (to: down|up|top|bottom, screens) — move the page N screens; add ref or x,y and it is a real mouse wheel over that point instead, for canvas and virtualised apps.",
		"capture": "`capture` (full?, marks?) — a picture of the page; full=true photographs the whole document instead of the visible part, marks=true draws each element's ref onto it so you can click by number instead of by pixel.",
		"tabs":    "`tabs` (act: list|select|close, id) — your own tabs.",
		"dialog":  "`dialog` (accept, text?) — answer this page's next alert/confirm/prompt.",
		"console": "`console` — what this page logged, threw, or had blocked since it loaded.",
		"network": "`network` — the fetch/XHR calls this page's own code made since it loaded.",
		"hover":   "`hover` (ref | find | x,y) — move the pointer there without pressing.",
		"drag":    "`drag` (ref|find|x,y → toRef|toFind|toX,toY) — press at the first, sweep to the second, release: moves things, selects the text swept over.",
		"key":     "`key` (keys) — press keys at the page's focus: names (Escape, Enter, Backspace, ArrowDown, Home, End, F2) and chords (ctrl+a, shift+End, ctrl+shift+ArrowRight), several separated by spaces.",
		"upload":  "`upload` (ref | find, path) — put a file from the sandbox into that file input.",
	}
	var b strings.Builder
	b.WriteString("Work a web page in the workbench browser, where the user can watch. Actions:\n")
	for _, action := range allowed {
		b.WriteString(lines[action] + "\n")
	}
	// The batch form. One line, because it is the shape of a call and not
	// a judgment — the judgment (when to batch, how to end one) is in
	// Guidance, sent once.
	b.WriteString("`steps` (array of these actions) — run several in order in one call; stops at the first that fails, or when the page moves under a step aimed by ref or x,y, and reports every step.\n")
	// Two sentences survive the migration, for two different reasons.
	//
	// The first is signature rather than judgment: which tab an action lands on
	// is part of what calling it means, and it is one line.
	//
	// The second is a SAFETY rule, and safety does not go in Guidance however
	// much it reads like judgment. Guidance rides in the message stream, where a
	// conversation long enough to be summarised can lose it silently. That is a
	// fine price for "read before you photograph" and not for this one.
	//
	// What did move, and where: how a ref goes stale went to the guidance of the
	// actions that spend refs (read, click, type, tabs), and whose tabs are whose
	// went to `tabs`. A session that only opens a page hears neither and needs
	// neither, which is the whole point of keying per action.
	b.WriteString("\nEvery action works the tab you opened last or selected. " +
		"Never type a password or API key into a page; ask the user to.")

	return toolDef("browser", b.String(), map[string]any{
		"type": "object",
		// Types, not sentences. Every one of these used to describe itself —
		// "action=open: a URL, or a file path relative to the sandbox root" —
		// which is the signature line a few lines above, paid for a second time.
		// A parameter whose name and owning action are both already on screen
		// needs nothing here but its type.
		"properties": map[string]any{
			"action":  map[string]any{"type": "string", "enum": allowed},
			"url":     map[string]any{"type": "string"},
			"ref":     map[string]any{"type": "integer"},
			"find":    map[string]any{"type": "string"},
			"x":       map[string]any{"type": "number"},
			"y":       map[string]any{"type": "number"},
			"toRef":   map[string]any{"type": "integer"},
			"toFind":  map[string]any{"type": "string"},
			"toX":     map[string]any{"type": "number"},
			"toY":     map[string]any{"type": "number"},
			"button":  map[string]any{"type": "string", "enum": []string{"left", "right", "middle"}},
			"count":   map[string]any{"type": "integer"},
			"keys":    map[string]any{"type": "string"},
			"path":    map[string]any{"type": "string"},
			"filter":  map[string]any{"type": "string"},
			"text":    map[string]any{"type": "string"},
			"enter":   map[string]any{"type": "boolean"},
			"newTab":  map[string]any{"type": "boolean"},
			"to":      map[string]any{"type": "string"},
			"act":     map[string]any{"type": "string", "enum": []string{"list", "select", "close"}},
			"id":      map[string]any{"type": "string"},
			"seconds": map[string]any{"type": "integer"},
			"screens": map[string]any{"type": "integer"},
			"accept":  map[string]any{"type": "boolean"},
			"full":    map[string]any{"type": "boolean"},
			"marks":   map[string]any{"type": "boolean"},
			"steps": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object"},
			},
		},
	})
}

func (s *browserSkill) ExecuteTool(ctx context.Context, args map[string]any) (skill.Output, error) {
	return s.run(ctx, args)
}

func (s *browserSkill) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	return s.run(ctx, map[string]any(input))
}

// actingActions are the actions whose answer carries a change note: they
// touch the page, and what the page did about it is the observation the
// model would otherwise spend a call on. read, capture, wait, tabs, console
// and network describe the page and change nothing; open's answer is a page.
var actingActions = map[string]bool{
	"click": true, "type": true, "key": true, "hover": true, "drag": true,
	"scroll": true, "upload": true, "back": true, "dialog": true,
}

func (s *browserSkill) run(ctx context.Context, args map[string]any) (skill.Output, error) {
	if steps, ok := args["steps"].([]any); ok && len(steps) > 0 {
		return s.runSteps(ctx, steps)
	}
	action, err := s.gate(args)
	if err != nil {
		return skill.Output{Name: "browser"}, err
	}

	// The page before, for the note after. Asked only when the action acts,
	// and never a reason to refuse: a page that will not say what it is
	// still gets its click.
	var before browserActResult
	noting := false
	if actingActions[action] {
		if id := AgentTabID(s.app.agentTabPeek()); id != "" {
			before, noting = s.app.pageState(id, stateWait)
		}
	}

	out, err := s.dispatch(ctx, action, args)
	if err == nil && noting && out.Content != "" {
		if id := AgentTabID(s.app.agentTabPeek()); id != "" {
			out.Content += s.app.changeNote(id, before)
			if out.RawOutput != "" {
				out.RawOutput = out.Content
			}
		}
	}

	// Which tab, stamped once, here, for every action that does not already say
	// it mid-sentence. This is the one door every browser action comes through,
	// which is the only reason the fact can be added in one place rather than
	// in eleven — and eleven places saying the same thing is how the four
	// spellings of "which page" happened. See browserTabRef.
	//
	// Read AFTER the action rather than before: `open` and `tabs select` both
	// move which tab is current, and the tab an answer belongs to is the one it
	// left the agent on.
	if out.Content != "" && !browserNamesItsOwnTab[action] {
		if id := AgentTabID(s.app.agentTabPeek()); id != "" {
			out.Content += browserTabRef(id, "")
			// RawOutput is the same sentence when it is set at all, and an
			// empty one is a deliberate state on the error paths.
			if out.RawOutput != "" {
				out.RawOutput = out.Content
			}
		}
	}
	return out, err
}

// gate reads the action out of a call and refuses one this caller may not
// use. Refused here as well as hidden from the description, because a
// description is guidance and a gate is a gate. Shared by the single call and
// by every step of a batch, so a batch cannot smuggle an action past it.
func (s *browserSkill) gate(args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(str(args["action"])))
	if action == "" {
		return "", fmt.Errorf("action is required — one of %s", strings.Join(s.allowedActions(), ", "))
	}
	if !slices.Contains(s.allowedActions(), action) {
		return "", fmt.Errorf("browser %s is not available here — this session may use: %s",
			action, strings.Join(s.allowedActions(), ", "))
	}
	return action, nil
}

// runSteps is several actions in one call.
//
// Owner, 6 ก.ย.: *"ทำยังไงให้มันไว เพราะตอนนี้ 1 call ได้แค่ 1 แอ็คชั่น … อยากให้มันเหมือน
// คนควบคุมจริง ๆ"*. A person who knows the next four moves makes them; the
// model was paying a full round trip for each. This is the shape OpenAI's
// computer_call takes (`actions[]`, one screenshot back) and Claude's
// computer use (several tool_use blocks, one screenshot to close), and it is
// the shape of this very session's browser_batch.
//
// Three rules. Every step passes the same gate a single call does, so a
// profile without browser_type cannot type through a batch. It stops at the
// first step that fails and says which, what came before it, and what did
// not run — the model knows exactly where it stands. And there is one change
// note at the end, not one per step: the observation is of where the batch
// left the page.
//
// Between an acting step and the next, the page is given the same settle a
// single call gets from its change note — so open → click → read inside one
// batch does not read the page the click just left.
//
// And it stops when the page moves out from under a step that cannot see
// that it moved. browser-use puts the same rule this way — *"we execute the
// actions until the page changes"* — and the reason is that a stale aim does
// not fail: ref 3 on the page that arrived is a real element, and the point
// x,y read off the old capture is a real place on the screen, so the click
// lands, reports success, and the batch goes on doing something nobody asked
// for. `find` is the exception and goes on, because it resolves against the
// page it lands on and refuses rather than guess. What the model gets back is
// the steps that ran, the ones that did not, and the new page's refs —
// enough to carry on from where the page actually is.
func (s *browserSkill) runSteps(ctx context.Context, steps []any) (skill.Output, error) {
	out := skill.Output{Name: "browser", Command: fmt.Sprintf("browser steps (%d)", len(steps))}
	var b strings.Builder
	var before browserActResult
	noting := false
	if id := AgentTabID(s.app.agentTabPeek()); id != "" {
		before, noting = s.app.pageState(id, stateWait)
	}
	var failed error
	// Where the tab was when the step began, which is the page a ref or an
	// x,y in it was aimed at. `before` cannot serve: it is the start of the
	// batch, and by step three the page may have moved twice.
	current := before
	// The step the batch stopped BEFORE, 1-based, or 0 for a batch that ran
	// to the end. Not a failure — the steps that ran, ran.
	stopped := 0
	for i, raw := range steps {
		if failed != nil || stopped != 0 {
			args, _ := raw.(map[string]any)
			fmt.Fprintf(&b, "\n%d. %s: ยังไม่ได้ทำ", i+1, str(args["action"]))
			continue
		}
		args, ok := raw.(map[string]any)
		if !ok {
			failed = fmt.Errorf("step %d is not an object", i+1)
			fmt.Fprintf(&b, "\n%d. ✗ %v", i+1, failed)
			continue
		}
		action, err := s.gate(args)
		if err != nil {
			failed = err
			fmt.Fprintf(&b, "\n%d. ✗ %v", i+1, err)
			continue
		}
		res, err := s.dispatch(ctx, action, args)
		if res.Content != "" && !browserNamesItsOwnTab[action] {
			if id := AgentTabID(s.app.agentTabPeek()); id != "" {
				res.Content += browserTabRef(id, "")
			}
		}
		out.Images = append(out.Images, res.Images...)
		out.Artifacts = append(out.Artifacts, res.Artifacts...)
		if err != nil {
			failed = fmt.Errorf("step %d (%s): %w", i+1, action, err)
			fmt.Fprintf(&b, "\n%d. %s ✗ %s", i+1, action, strings.TrimSpace(res.Content))
			continue
		}
		fmt.Fprintf(&b, "\n%d. %s: %s", i+1, action, strings.TrimSpace(res.Content))
		if i+1 >= len(steps) {
			continue
		}
		// The page settles before the next step, so a batch that opens a
		// page, acts on it and reads it reads the page it acted on — and the
		// state that comes back is how an unplanned navigation is told from
		// no navigation at all.
		moved := refsDieAfter(action, args)
		if actingActions[action] && noting {
			if id := AgentTabID(s.app.agentTabPeek()); id != "" {
				if after, ok := s.app.settleAfterAct(id, current); ok {
					moved = moved || after.URL != current.URL
					current = after
				}
			}
		}
		if moved && stepAimsBlind(steps[i+1]) {
			stopped = i + 2
		}
	}
	if stopped != 0 {
		fmt.Fprintf(&b, "\n→ หยุดก่อนขั้นที่ %d: หน้าเปลี่ยนระหว่างทาง ref และ x,y ของหน้าก่อนใช้กับหน้านี้ไม่ได้ — เล็งด้วย find หรือ read หน้าใหม่ก่อนค่อยทำต่อ", stopped)
	}
	if noting {
		if id := AgentTabID(s.app.agentTabPeek()); id != "" {
			b.WriteString(s.app.changeNote(id, before))
		}
	}
	out.Content = strings.TrimPrefix(b.String(), "\n")
	out.RawOutput = out.Content
	out.Success = failed == nil
	if failed != nil {
		out.Stderr = failed.Error()
	}
	return out, failed
}

// refsDieAfter reports whether a step, having run, leaves every ref and every
// x,y written before it pointing at a page that is no longer there.
//
// Two actions do it by definition rather than by accident, which is why they
// are known without asking the page: `open` is a navigation — that is what it
// is for — and `tabs select` moves to a different document entirely. Everything
// else that can navigate does so unpredictably, and is caught by the URL
// comparison in runSteps instead.
func refsDieAfter(action string, args map[string]any) bool {
	switch action {
	case "open":
		return true
	case "tabs":
		return strings.EqualFold(strings.TrimSpace(str(args["act"])), "select")
	}
	return false
}

// stepAimsBlind reports whether a step points at something only the page it
// was written for can resolve: a [n] from a read, or a point off a capture.
//
// A step has up to two aims — drag has a far end — and either one being blind
// is enough. A drag that starts at `find` and ends at toX,toY is half written
// for a page that may be gone, and half is all it takes to sweep across the
// wrong thing.
//
// A step with no aim at all (read, capture, wait, key, tabs) is not blind: it
// does not care which page it lands on.
func stepAimsBlind(raw any) bool {
	args, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	return aimIsBlind(args, "find", "ref", "x", "y") ||
		aimIsBlind(args, "toFind", "toRef", "toX", "toY")
}

// aimIsBlind answers for one end of a step, and answers in targetFrom's terms
// on purpose: text wins over a ref there, and a lone x without a y is not a
// point there, so neither is an aim here.
func aimIsBlind(args map[string]any, findKey, refKey, xKey, yKey string) bool {
	if strings.TrimSpace(str(args[findKey])) != "" {
		return false
	}
	if intArg(args[refKey]) != 0 {
		return true
	}
	_, okX := skill.FloatArg(args[xKey])
	_, okY := skill.FloatArg(args[yKey])
	return okX && okY
}

// browserNamesItsOwnTab is the actions whose answers already carry the tab.
//
// Those that act say it mid-sentence through browserWhere, where the fact
// belongs — a click that landed somewhere is about the page it landed on, not
// about a bracket at the end. `tabs` enumerates every tab by id and marks the
// current one, so a stamp would be a fourth mention in three lines.
var browserNamesItsOwnTab = map[string]bool{
	"click": true, "type": true, "back": true, "tabs": true,
	"hover": true, "drag": true, "key": true, "upload": true,
}

func (s *browserSkill) dispatch(ctx context.Context, action string, args map[string]any) (skill.Output, error) {
	target := targetFrom(args, "ref", "find", "x", "y")
	switch action {
	case "open":
		return (&browserOpenSkill{app: s.app, owner: s.owner()}).open(ctx, str(args["url"]), boolArg(args["newTab"]))
	case "read":
		return (&browserReadSkill{app: s.app}).Execute(ctx, skill.Input{"filter": str(args["filter"])})
	case "click":
		return (&browserClickSkill{app: s.app}).click(target, str(args["button"]), intArg(args["count"]))
	case "type":
		return (&browserTypeSkill{app: s.app}).typeText(target, str(args["text"]), boolArg(args["enter"]))
	case "hover":
		return (&browserHoverSkill{app: s.app}).hover(target)
	case "drag":
		return (&browserDragSkill{app: s.app}).drag(target, targetFrom(args, "toRef", "toFind", "toX", "toY"))
	case "key":
		return (&browserKeySkill{app: s.app}).key(str(args["keys"]))
	case "upload":
		return (&browserUploadSkill{app: s.app}).upload(target, str(args["path"]))
	case "capture":
		return (&browserCaptureSkill{app: s.app, owner: s.owner()}).capture(ctx, boolArg(args["full"]), boolArg(args["marks"]))
	case "tabs":
		return (&browserTabsSkill{app: s.app, owner: s.owner()}).run(ctx, str(args["act"]), str(args["id"]))
	case "wait":
		return (&browserWaitSkill{app: s.app}).wait(ctx, str(args["text"]), intArg(args["seconds"]))
	case "back":
		return (&browserBackSkill{app: s.app}).back(ctx)
	case "scroll":
		return (&browserScrollSkill{app: s.app}).scroll(str(args["to"]), intArg(args["screens"]), target)
	case "dialog":
		return (&browserDialogSkill{app: s.app}).dialog(boolArg(args["accept"]), str(args["text"]))
	case "console", "network":
		return (&browserLogSkill{app: s.app, kind: action}).run(ctx)
	}
	return skill.Output{Name: "browser"}, fmt.Errorf("unknown browser action %q", action)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// Both of these used to be written out here, and both were a shape short: an
// int and a float64 but not a quoted number, a bool but not a quoted bool.
// Models send the quoted forms — `{"action":"click","ref":"1"}` is in this
// machine's tool_runs twelve times — and each one arrived as a zero-value that
// no longer resembled what was asked for. internal/skill has had the right rule
// since `read` needed it; this defers to it rather than agreeing with it.
func intArg(v any) int   { return skill.IntArg(v) }
func boolArg(v any) bool { return skill.BoolArg(v) }
