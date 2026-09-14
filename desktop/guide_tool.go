package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// guideAsks manages pending questions waiting for AnswerGuide from the window.
type guideAsks struct {
	mu   sync.Mutex
	seq  atomic.Int64
	open map[string]chan guideAnswer
}

type guideAnswer struct {
	resultJSON string
	err        error
}

type guideAskEvent struct {
	ID     string         `json:"id"`
	Action string         `json:"action"`
	Args   map[string]any `json:"args,omitempty"`
}

func (g *guideAsks) add() (string, chan guideAnswer) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open == nil {
		g.open = map[string]chan guideAnswer{}
	}
	id := strconv.FormatInt(g.seq.Add(1), 10)
	ch := make(chan guideAnswer, 1)
	g.open[id] = ch
	return id, ch
}

func (g *guideAsks) take(id string) (chan guideAnswer, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	ch, ok := g.open[id]
	delete(g.open, id)
	return ch, ok
}

// AnswerGuide is called from the frontend when answering a screen:guide event.
func (a *App) AnswerGuide(id, resultJSON string) {
	if ch, ok := a.guideAsks.take(id); ok {
		ch <- guideAnswer{resultJSON: resultJSON}
	}
}

// NewGuideSession opens the guide's conversation in the engine — held beside
// the chat on screen, never shown (engine.OpenGuideSession) — and keeps the
// window's index of the map against its id, which is what the tool's
// description lists. The index comes from the window because the names are
// in the UI's language and the window is where that is known.
func (a *App) NewGuideSession(indexJSON string) (string, error) {
	id, err := a.api.OpenGuideSession()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(indexJSON) != "" {
		a.guideIndices.Store(id, indexJSON)
	}
	return id, nil
}

// AskGuide runs one turn in the guide's conversation and answers with the
// text. The chat on screen is not involved and not moved.
func (a *App) AskGuide(sessionID, text string) (string, error) {
	reply, err := a.api.SendToGuide(sessionID, text)
	if err != nil {
		return "", err
	}
	return reply.Text, nil
}

// CloseGuideSession drops the window's caches for the guide and lets the
// engine delete the conversation. Named for the engine's door on purpose: the
// generated forwarder yields to this one, so the window has one close.
func (a *App) CloseGuideSession(id string) error {
	a.guideIndices.Delete(id)
	a.guideSnapshots.Delete(id)
	return a.api.CloseGuideSession(id)
}

// askGuide sends an event to the frontend window and waits up to 30s for AnswerGuide.
func (a *App) askGuide(ctx context.Context, action string, args map[string]any) (string, error) {
	id, ch := a.guideAsks.add()
	a.emitEvent("screen:guide", guideAskEvent{
		ID:     id,
		Action: action,
		Args:   args,
	})
	select {
	case <-ctx.Done():
		a.guideAsks.take(id)
		return "", ctx.Err()
	case ans := <-ch:
		if ans.err != nil {
			return "", ans.err
		}
		return ans.resultJSON, nil
	case <-time.After(30 * time.Second):
		a.guideAsks.take(id)
		return "", fmt.Errorf("window did not answer in 30s")
	}
}

type guideSkill struct {
	app     *App
	session engine.Session
}

func newGuideSkill(app *App, session engine.Session) skill.Skill {
	return &guideSkill{app: app, session: session}
}

const guideToolName = "guide"

func (*guideSkill) Name() string { return guideToolName }

func (*guideSkill) Description() string {
	return "Guide the user across the Aetox UI: where, describe, point, goto, press"
}

func (s *guideSkill) ToolDefinition() model.ToolDefinition {
	desc := `Walk the Aetox window: the app's own buttons, pages and panels, known to you through a map of ids. One tool, five actions.

- where: the page on screen, where you stand (guideAt), and the ids visible right now. After the first call only what CHANGED comes back; pass full=true to see everything again.
- describe {id}: the map's entry for one id — name, what it is, why it was designed that way (with the decision it cites), whether it is safe to press, and the page it lives on. The why is the only source of a why: never invent one.
- point {id}: walk to that element and stand beside it, on whatever page it is on. End a turn with at most ONE point or goto.
- goto {page}: open a page. page is one of: chat · settings.<rail> (the rail ids the index lists under settings.rail.*, e.g. settings.brain) · capability.<page> (mcp, skills, builtins, computer, connections, prompts) · office · artifacts.
- press {id}: press an element the map marks safe (opening, switching, expanding). Anything else is refused by the window — say so and point at it for the user to press.

Answer in the user's language, in plain words, two short sentences. Do not narrate these actions.`

	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID()
	}

	if sessionID != "" {
		if raw, ok := s.app.guideIndices.Load(sessionID); ok {
			if str, ok := raw.(string); ok && str != "" {
				var items []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Safe bool   `json:"safe"`
				}
				if json.Unmarshal([]byte(str), &items) == nil && len(items) > 0 {
					var b strings.Builder
					b.WriteString(desc)
					b.WriteString("\n\nKnown UI Map Index:\n")
					for _, it := range items {
						safeStr := "safe"
						if !it.Safe {
							safeStr = "NOT safe"
						}
						b.WriteString(fmt.Sprintf("- %s (%s) [%s]\n", it.ID, it.Name, safeStr))
					}
					desc = b.String()
				}
			}
		}
	}

	return toolDef("guide", desc, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{"where", "describe", "point", "goto", "press"},
			},
			"id":   map[string]any{"type": "string"},
			"page": map[string]any{"type": "string"},
			"full": map[string]any{"type": "boolean"},
		},
		"required": []string{"action"},
	})
}

func (s *guideSkill) ExecuteTool(ctx context.Context, args map[string]any) (skill.Output, error) {
	return s.run(ctx, args)
}

func (s *guideSkill) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	return s.run(ctx, map[string]any(input))
}

type whereOutput struct {
	Page    string         `json:"page"`
	GuideAt string         `json:"guideAt"`
	Visible []guideElement `json:"visible"`
}

type guideElement struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Safe bool   `json:"safe"`
}

type pressOutput struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (s *guideSkill) run(ctx context.Context, args map[string]any) (skill.Output, error) {
	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))

	switch action {
	case "where":
		full, _ := args["full"].(bool)
		resJSON, err := s.app.askGuide(ctx, "where", args)
		if err != nil {
			return skill.Output{}, err
		}

		var fullRes whereOutput
		if err := json.Unmarshal([]byte(resJSON), &fullRes); err != nil {
			return skill.Output{Content: resJSON, Success: true}, nil
		}

		sessID := ""
		if s.session != nil {
			sessID = s.session.ID()
		}

		// Calculate diff if not full and previous snapshot exists
		if !full && sessID != "" {
			if prevVal, ok := s.app.guideSnapshots.Load(sessID); ok {
				if prev, ok := prevVal.(whereOutput); ok && prev.Page == fullRes.Page {
					prevIDs := map[string]bool{}
					for _, el := range prev.Visible {
						prevIDs[el.ID] = true
					}
					var diffVisible []guideElement
					for _, el := range fullRes.Visible {
						if !prevIDs[el.ID] {
							diffVisible = append(diffVisible, el)
						}
					}
					s.app.guideSnapshots.Store(sessID, fullRes)
					diffOut := whereOutput{
						Page:    fullRes.Page,
						GuideAt: fullRes.GuideAt,
						Visible: diffVisible,
					}
					outBytes, _ := json.Marshal(diffOut)
					return skill.Output{Content: string(outBytes), Success: true}, nil
				}
			}
		}

		if sessID != "" {
			s.app.guideSnapshots.Store(sessID, fullRes)
		}
		return skill.Output{Content: resJSON, Success: true}, nil

	case "press":
		resJSON, err := s.app.askGuide(ctx, "press", args)
		if err != nil {
			return skill.Output{}, err
		}
		var p pressOutput
		if err := json.Unmarshal([]byte(resJSON), &p); err == nil && !p.OK {
			errStr := p.Error
			if errStr == "" {
				errStr = "not safe"
			}
			return skill.Output{}, fmt.Errorf("%s", errStr)
		}
		return skill.Output{Content: resJSON, Success: true}, nil

	default:
		resJSON, err := s.app.askGuide(ctx, action, args)
		if err != nil {
			return skill.Output{}, err
		}
		return skill.Output{Content: resJSON, Success: true}, nil
	}
}
