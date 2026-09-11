package engine

// A packed tool the shape of the window's (§248 B1).
//
// The browser and the machine live in desktop/ now and an engine test has no
// window, but what the engine does to a pack the window lends — narrow it by
// stance, hand it the session's dispatcher, refuse through it — still has to
// be proved on a pack of that shape: one name, an `action` enum from
// skill.PackedCalls, Narrow answering with fewer actions, and a refusal that
// names what this session may use. This is that shape and nothing else; the
// real one is desktop/browser_tool.go, held to the same rules by its own
// tests.

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

type packStub struct {
	name string
	// actions this caller may use, nil for all of them. Set only by Narrow.
	actions []string
}

func newPackStub(name string) *packStub { return &packStub{name: name} }

func (p *packStub) Name() string        { return p.name }
func (p *packStub) Description() string { return "a stand-in for the window's " + p.name }
func (p *packStub) Actions() []string   { return skill.PackedActions(p.name) }

func (p *packStub) allowed() []string {
	if len(p.actions) > 0 {
		return p.actions
	}
	var out []string
	for _, call := range skill.PackedCalls(p.name) {
		out = append(out, call.Action)
	}
	return out
}

func (p *packStub) ToolDefinition() model.ToolDefinition {
	params, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{"type": "string", "enum": p.allowed()},
		},
		"required": []string{"action"},
	})
	return model.ToolDefinition{Type: "function", Function: model.ToolFunction{
		Name: p.name, Description: p.Description(), Parameters: params,
	}}
}

// Narrow hands back the pack offering only the named actions — a copy, for
// the same shared-registry reason as shell's; silence is the whole tool.
func (p *packStub) Narrow(named []string) skill.Skill {
	want := map[string]bool{}
	for _, n := range named {
		want[strings.ToLower(strings.TrimSpace(n))] = true
	}
	var actions []string
	for _, call := range skill.PackedCalls(p.name) {
		if want[call.Permission] {
			actions = append(actions, call.Action)
		}
	}
	if len(actions) == 0 {
		return p
	}
	return &packStub{name: p.name, actions: actions}
}

func (p *packStub) Execute(ctx context.Context, in skill.Input) (skill.Output, error) {
	return p.ExecuteTool(ctx, map[string]any(in))
}

func (p *packStub) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))
	if !slices.Contains(p.allowed(), action) {
		return skill.Output{Name: p.name}, fmt.Errorf("%s %s is not available here — this session may use: %s",
			p.name, action, strings.Join(p.allowed(), ", "))
	}
	return skill.Output{Name: p.name, Success: true, Content: "ok"}, nil
}
