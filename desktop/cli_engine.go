package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/cliagent"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// The hand-off to an external engine (internal/cliagent): a chat whose
// provider has an engine registered is answered by that program, and this file
// is the whole of what the desktop does about it — decide before the turn,
// translate during it, hand back the two messages after.
//
// It runs from runTurn before the engine's own executor, because two loops
// cannot share one turn, and after the snapshot, so undo covers whatever files
// the program edits the same way it covers Aetox's own. Everything after
// runTurn — the transcript, the store, the learning pass — sees the same two
// messages every other branch returns and never learns who wrote them.
//
// With nothing registered (the state this ships in) cliEngineFor misses on
// every provider and none of this runs.

// cliEngineFor answers whether a provider is served by an external engine.
// The registry is keyed by canonical id, so the alias is resolved here.
func cliEngineFor(providerName string) (cliagent.Engine, bool) {
	return cliagent.For(model.NormalizeProvider(providerName))
}

// ExternalEngineStatus is the settings page's view of one engine — where the
// program is, which version, whether a turn could run, and if not, what to do.
// Ready false with an empty Detail means no engine is registered for the name.
func (a *App) ExternalEngineStatus(providerName string) cliagent.Status {
	engine, ok := cliEngineFor(providerName)
	if !ok {
		return cliagent.Status{}
	}
	return engine.Probe(a.engineCtx())
}

// runCLIEngineTurn answers one message through engine and hands back the
// turn's two messages in the shape runTurn's other branches do.
func (a *App) runCLIEngineTurn(conv *conversation, ctx context.Context, engine cliagent.Engine, text string) (SessionMessage, SessionMessage, error) {
	now := time.Now().Format("15:04")
	user := SessionMessage{Role: "user", Text: text, Time: now}
	agent := SessionMessage{Role: "agent", Time: now}

	if st := engine.Probe(ctx); !st.Ready {
		if st.Detail != "" {
			return user, agent, errors.New(st.Detail)
		}
		return user, agent, fmt.Errorf("%s ยังไม่พร้อมใช้งาน", engine.Label())
	}
	// The engine keeps the conversation under an id this side hands it:
	// minted on the first turn, resumed on every later one.
	resume := conv.cliSession != ""
	if !resume {
		conv.cliSession = cliagent.NewSessionID()
	}
	t := cliagent.Turn{
		Text:       text,
		Dir:        conv.cfg.SandboxRoot,
		SessionID:  conv.cliSession,
		Resume:     resume,
		Model:      strings.TrimSpace(conv.cfg.ModelName),
		Effort:     strings.TrimSpace(conv.cfg.ThinkLevel),
		SystemNote: cliSystemNote(conv),
	}

	col := &cliCollector{app: a, conv: conv, names: map[string]string{}}
	a.emitAgentStatus(conv, "กำลังคิดคำตอบ...")
	res, err := engine.Run(ctx, t, col.onEvent)
	a.emitAgentStatus(conv, "")

	if res.SessionID != "" {
		conv.cliSession = res.SessionID
	}
	agent.Text = col.answer(res)
	agent.Reasoning = strings.TrimSpace(col.reasoning.String())
	agent.ThinkSecs = col.thinkSecs()
	agent.Parts = col.parts
	if res.Usage.InputTokens > 0 || res.Usage.OutputTokens > 0 {
		a.recordTokenUsage(conv, model.Usage{
			PromptTokens:       res.Usage.InputTokens,
			CachedPromptTokens: res.Usage.CachedInputTokens,
			CompletionTokens:   res.Usage.OutputTokens,
		})
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return user, agent, err
		}
		debuglog.Msg("%s: %v", engine.Provider(), err)
		return user, agent, fmt.Errorf("%s: %w", engine.Label(), err)
	}
	if res.IsError {
		// The engine's own failure sentence — a refused sign-in, a quota — is
		// the one the user can act on, so it is shown as it came.
		if res.ErrorText == "" {
			return user, agent, fmt.Errorf("%s ตอบไม่สำเร็จ", engine.Label())
		}
		return user, agent, errors.New(res.ErrorText)
	}
	// The authoritative delivery: the bubble showed fragments as they
	// streamed, and now holds exactly the answer.
	a.emitChatChunk(conv, agent.Text, true)
	return user, agent, nil
}

// cliSystemNote is what Aetox adds to an engine's own system prompt: that the
// reply is drawn in a window with no terminal behind it, and which desk the
// user opened. Short on purpose — the engine's prompt is what makes its tools
// work, and Aetox's own tool prompt would describe tools that loop does not
// have.
func cliSystemNote(conv *conversation) string {
	var b strings.Builder
	b.WriteString("You are answering inside Aetox, a desktop app that shows your reply and your tool calls in its own window; there is no terminal and nobody can answer a prompt from you. ")
	b.WriteString("Reply in the language the user writes in.")
	if conv.desk != nil && conv.desk.Name != "" {
		fmt.Fprintf(&b, " The user opened you at the %q desk.", conv.desk.Name)
	}
	return b.String()
}

// cliCollector folds an engine's events into what the window draws now and
// what the transcript keeps: streamed text into the bubble, tool calls into
// the timeline, thinking into the reasoning panel, and the whole turn into
// Parts.
type cliCollector struct {
	app  *App
	conv *conversation
	// names remembers each tool call's name by ref, so a result that carries
	// only the ref can be drawn under the right row.
	names map[string]string

	texts                 []string
	parts                 []turn.TurnPart
	reasoning             strings.Builder
	firstThink, lastThink time.Time
}

func (c *cliCollector) onEvent(ev cliagent.Event) {
	main := ev.Parent == ""
	switch ev.Kind {
	case cliagent.EventText:
		// A delegate's prose goes under its row, never into the answer.
		if main {
			c.app.previewAnswer(c.conv, ev.Text)
		}
	case cliagent.EventThinking:
		if !main || ev.Text == "" {
			return
		}
		if c.firstThink.IsZero() {
			c.firstThink = time.Now()
		}
		c.lastThink = time.Now()
		c.reasoning.WriteString(ev.Text)
		c.app.emitEvent("agent:reasoning", sessionEvent[string]{SessionID: c.conv.id, Data: ev.Text})
	case cliagent.EventAnswer:
		if !main || strings.TrimSpace(ev.Text) == "" {
			return
		}
		c.texts = append(c.texts, ev.Text)
		c.parts = append(c.parts, turn.TurnPart{Kind: turn.PartText, Text: ev.Text})
	case cliagent.EventToolCall:
		if ev.Tool.Ref != "" {
			c.names[ev.Tool.Ref] = ev.Tool.Name
		}
		c.app.recordToolAction(c.conv, turn.ToolEvent{
			Action: "call", Name: ev.Tool.Name, Ref: ev.Tool.Ref, Subject: ev.Tool.Subject, Parent: ev.Parent,
		})
		if main {
			c.parts = append(c.parts, turn.TurnPart{Kind: turn.PartTool, Tool: &turn.ToolPart{
				Ref: ev.Tool.Ref, Name: ev.Tool.Name, Subject: ev.Tool.Subject,
			}})
		}
	case cliagent.EventToolResult:
		name := ev.Tool.Name
		if name == "" {
			name = c.names[ev.Tool.Ref]
		}
		res := turn.ToolEvent{Action: "result", Name: name, Ref: ev.Tool.Ref, Parent: ev.Parent, OK: ev.Tool.OK}
		if !ev.Tool.OK {
			res.Error = ev.Tool.Error
		}
		c.app.recordToolAction(c.conv, res)
	case cliagent.EventStatus:
		c.app.emitAgentStatus(c.conv, ev.Text)
	}
}

// answer is the turn's text: every finished block the engine reported on its
// own thread, in order, or the engine's final result when it reported none.
func (c *cliCollector) answer(res cliagent.Result) string {
	if len(c.texts) == 0 {
		return res.Text
	}
	return strings.Join(c.texts, "\n\n")
}

func (c *cliCollector) thinkSecs() int {
	if c.firstThink.IsZero() {
		return 0
	}
	secs := int(c.lastThink.Sub(c.firstThink).Round(time.Second) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return secs
}
