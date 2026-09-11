package rpc

// The screen's half of the window tools: the announcement it makes in
// hello, and the two handlers that run a call and shape a narrowed tool.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Mikedev115/Aetox/internal/callfault"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

// Tools is what the screen lends, built per session — the same function
// the in-process Screen.WindowTools was.
type Tools func(sess engine.Session) []skill.Skill

// session is the little the screen's tools need to know about the chat
// they act for, as the engine sent it.
type session struct{ id, root string }

func (s session) ID() string   { return s.id }
func (s session) Root() string { return s.root }

// Announce describes the screen's tools for hello: name, description, the
// definition, the actions, and the guidance per action — read once here so
// the engine never has to ask.
func Announce(tools []skill.Skill) []ToolAnnouncement {
	var out []ToolAnnouncement
	for _, s := range tools {
		tool, ok := s.(skill.Tool)
		if !ok {
			continue
		}
		ann := ToolAnnouncement{Name: s.Name(), Description: s.Description(), Definition: tool.ToolDefinition(), Guidance: map[string]string{}}
		if packed, ok := s.(skill.Packed); ok {
			ann.Actions = packed.Actions()
		}
		if guided, ok := s.(skill.Guided); ok {
			for _, call := range skill.PackedCalls(s.Name()) {
				if g := guided.Guidance(map[string]any{"action": call.Action}); g != "" {
					ann.Guidance[call.Action] = g
				}
			}
			if g := guided.Guidance(map[string]any{"steps": []any{"x"}}); g != "" {
				ann.Guidance["steps"] = g
			}
			if len(ann.Actions) == 0 {
				if g := guided.Guidance(map[string]any{}); g != "" {
					ann.Guidance[""] = g
				}
			}
		}
		out = append(out, ann)
	}
	return out
}

// ServeTools registers the screen's window tools on the client: the
// handlers for a call and a cut. Call before Connect; Hello announces them.
func ServeTools(c *Client, tools Tools) {
	c.Handle(MethodScreenTool, func(ctx context.Context, _ string, params json.RawMessage) (any, error) {
		var p ToolCallParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		tool := findTool(tools(session{p.Session, p.Root}), p.Name, p.Actions)
		if tool == nil {
			return nil, fmt.Errorf("screen.tool: this screen lends no %q", p.Name)
		}
		out, err := tool.ExecuteTool(ctx, p.Args)
		res := ToolCallResult{Output: out}
		if err != nil {
			res.Error = err.Error()
			switch {
			case callfault.Is(err):
				res.Fault = faultCall
			case statereport.Is(err):
				res.Fault = faultState
			}
		}
		return res, nil
	})
	c.Handle(MethodScreenToolCut, func(_ context.Context, _ string, params json.RawMessage) (any, error) {
		var p ToolCutParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		tool := findTool(tools(session{}), p.Name, p.Actions)
		if tool == nil {
			return nil, fmt.Errorf("screen.toolCut: this screen lends no %q", p.Name)
		}
		return ToolCutResult{Description: tool.Description(), Definition: tool.ToolDefinition()}, nil
	})
}

// findTool is the named tool, narrowed to the actions if any were given.
func findTool(tools []skill.Skill, name string, actions []string) skill.Tool {
	for _, s := range tools {
		if s.Name() != name {
			continue
		}
		if len(actions) > 0 {
			if packed, ok := s.(skill.Packed); ok {
				s = packed.Narrow(actions)
			}
		}
		if tool, ok := s.(skill.Tool); ok {
			return tool
		}
	}
	return nil
}

// Hello is the screen's opening message: the protocol, its version, its
// tools and what it can do. Sent right after Connect, before anything else
// — the engine lends no window tools to a session until it has heard this.
func (c *Client) Hello(ctx context.Context, version string, tools []skill.Skill, features []string) (HelloResult, error) {
	conn := c.Conn()
	if conn == nil {
		return HelloResult{}, ErrDisconnected
	}
	var out HelloResult
	err := conn.Call(ctx, MethodHello, []any{HelloParams{Protocol: Protocol, Version: version, Tools: Announce(tools), Features: features}}, &out)
	if err != nil {
		return HelloResult{}, err
	}
	if out.Protocol != Protocol {
		return out, errors.New("hello: the engine speaks another protocol")
	}
	return out, nil
}

// The features a screen may announce.
const (
	FeatureDialogs       = featureDialogs
	FeatureFileManager   = featureFileManager
	FeatureWindowTools   = featureWindowTools
	FeatureProviderProxy = featureProviderProxy
)
