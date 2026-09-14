package engine

// What the model is TOLD about its team, read off the real engine — the
// prompt and the `task` schema a desktop chat would send — and, live, whether
// a real model then hires the team when asked to.
//
// Twice on 14 ก.ย. 2026 the owner asked for the team and got a helper:
// "ทดสอบ การทำงานทีมเอเจนหน่อยครับ" hired `general`, and "ส่งงานให้
// ผู้ช่วยในคอมพิวเตอร์หน่อยครับ ทดสอบทั้งหมดเลย" — the seed team's own name —
// hired `tester`. The machinery was fine (doc and deepresearch ran when he
// corrected it); what the model had been told was not: the schema still said
// ซับเอเจน, the prompt never said ผู้ช่วย is its own rank, and a chat on no
// team was told nothing at all.

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/safety"
	"github.com/Mikedev115/Aetox/internal/signer"
	"github.com/Mikedev115/Aetox/internal/subagent"
	"github.com/Mikedev115/Aetox/internal/turn"
)

func systemPrompt(t *testing.T, a *Engine) string {
	t.Helper()
	messages := a.cur().agent.ContextMessages()
	if len(messages) == 0 {
		t.Fatal("no system prompt")
	}
	return messages[0].Content
}

// taskAgentDescription is the `agent` parameter's own text — the roster as the
// model reads it when choosing.
func taskAgentDescription(t *testing.T, a *Engine) string {
	t.Helper()
	for _, d := range a.deskTools().ToolDefinitions() {
		if d.Function.Name != "task" {
			continue
		}
		var schema struct {
			Properties struct {
				Agent struct {
					Description string `json:"description"`
				} `json:"agent"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(d.Function.Parameters, &schema); err != nil {
			t.Fatalf("task schema: %v", err)
		}
		return schema.Properties.Agent.Description
	}
	return ""
}

// On the seed team at the assistant desk — the station every fresh install
// opens at — the model is told the team's name, that ผู้ช่วย in it is not
// itself, and is offered all four members under the rank the screen uses.
// Switched to ไม่ใช้ทีมช่วย, the members leave the schema AND the prompt says
// so, in the same words.
func TestTheModelIsToldWhichTeamItIsOnAndWhenItIsOnNone(t *testing.T) {
	a := bootDeskApp(t, mode.Default)
	if a.cur().team != subagent.SeedTeamName {
		t.Fatalf("a fresh assistant chat is on %q, want the seed team", a.cur().team)
	}

	sys := systemPrompt(t, a)
	for _, want := range []string{
		"on the team «" + subagent.SeedTeamName + "»",
		"you are the ผู้ช่วย",
		"is that team, not you",
	} {
		if !strings.Contains(sys, want) {
			t.Errorf("a chat on the seed team is never told %q", want)
		}
	}
	if strings.Contains(sys, "on no team") {
		t.Error("a chat on a team is told it is on none")
	}
	agents := taskAgentEnum(t, a)
	for _, member := range []string{"deepresearch", "doc", "sheet", "video"} {
		if !slices.Contains(agents, member) {
			t.Errorf("%s is on the seed team and not offered: %v", member, agents)
		}
	}
	roster := taskAgentDescription(t, a)
	agentsHalf, helpersHalf, split := strings.Cut(roster, "HELPERS (ลูกมือ)")
	if !split || !strings.Contains(agentsHalf, "AGENTS (พนักงาน / เอเจน)") {
		t.Fatalf("the roster does not use the screen's words:\n%s", roster)
	}
	if !strings.Contains(agentsHalf, "doc") || strings.Contains(helpersHalf, "doc") {
		t.Errorf("doc is filed among the helpers:\n%s", roster)
	}
	if !strings.Contains(helpersHalf, "tester") {
		t.Errorf("tester is not filed as a helper:\n%s", roster)
	}

	if _, err := a.NewTeamSession(mode.Default, subagent.NoTeam); err != nil {
		t.Fatalf("NewTeamSession(no team): %v", err)
	}
	sys = systemPrompt(t, a)
	if !strings.Contains(sys, "This chat is on no team") || !strings.Contains(sys, "you are the ผู้ช่วย") {
		t.Errorf("a chat on no team is not told so:\n%s", sys)
	}
	if strings.Contains(sys, "on the team «") {
		t.Error("a chat on no team is still told a team's name")
	}
	for _, n := range taskAgentEnum(t, a) {
		if p, ok := subagent.Load(n); ok && p.Desk != "" {
			t.Errorf("no team, yet %s is offered", n)
		}
	}
}

// signingScreen is fakeScreen with a real credential: the test's own key,
// signed onto the wire the way desktop/provider_forward.go does it.
type signingScreen struct {
	fakeScreen
	key string
}

func (s *signingScreen) ProviderTransport(provider, wireFormat string) model.Transport {
	return signer.Transport(signer.Options{
		Provider:   provider,
		WireFormat: wireFormat,
		Key:        func(string) string { return s.key },
	})
}

// The owner's own sentence, against a real model, through the real engine:
//
//	AETOX_LIVE=1 go test ./internal/engine/ -run TestLiveTheTeamsNameHiresTheTeam -v -count=1
//
// The turn is cancelled at the first `task` call — which worker it reached
// for is the whole question, and the four agents doing "ทดสอบทั้งหมด" for
// real would spend the key on nothing this asserts.
func TestLiveTheTeamsNameHiresTheTeam(t *testing.T) {
	key := liveDeepSeekKey(t)
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	screen := &signingScreen{key: key}
	rec := &recorder{}
	a := seed(&Engine{
		ctx:    context.Background(),
		emit:   func(name string, data ...any) { rec.add(emitted{name, data}) },
		dbDir:  t.TempDir(),
		screen: screen,
	}, &conversation{id: newSessionID()})
	bootRecorders.Store(a, rec)
	t.Cleanup(func() {
		bootRecorders.Delete(a)
		if a.db != nil {
			_ = a.db.Close()
		}
	})
	m, _ := mode.Load(mode.Default)
	a.cur().desk = m
	a.cur().team = subagent.PreferredTeam(mode.Default)
	a.applyConfig(a.cur(), config.Config{
		SandboxRoot:     t.TempDir(),
		ModelProvider:   "deepseek",
		ModelName:       "deepseek-v4-flash",
		ModelTimeoutSec: 300,
		ApprovalMode:    string(safety.ApprovalFullAccess),
		DelegateAgents:  true,
	})
	if a.cur().team != subagent.SeedTeamName {
		t.Fatalf("not on the seed team: %q", a.cur().team)
	}

	// The first `task` call the head makes ends the turn: which worker it
	// reached for is the whole question.
	firstTask := make(chan struct{}, 1)
	a.emit = func(name string, data ...any) {
		rec.add(emitted{name, data})
		if name != "agent:tool" || len(data) == 0 {
			return
		}
		if se, ok := data[0].(SessionEvent[turn.ToolEvent]); ok && se.Data.Action == "call" && se.Data.Name == "task" && se.Data.Parent == "" {
			select {
			case firstTask <- struct{}{}:
			default:
			}
		}
	}

	ask := "ส่งงานให้ผู้ช่วยในคอมพิวเตอร์หน่อยครับ ทดสอบทั้งหมดเลย"
	done := make(chan struct{})
	var reply TurnReply
	go func() {
		defer close(done)
		reply, _ = a.SendMessage(ask, "")
	}()
	select {
	case <-firstTask:
		// Let the `start` return and be recorded, then stop the delegate.
		time.Sleep(2 * time.Second)
		a.CancelTurn()
		<-done
	case <-done:
	case <-time.After(4 * time.Minute):
		a.CancelTurn()
		<-done
	}

	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	rows, err := db.Query(`SELECT args FROM tool_runs WHERE tool = 'task' AND parent_ref = '' ORDER BY id`)
	if err != nil {
		t.Fatalf("tool_runs: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var hired []string
	for rows.Next() {
		var args string
		if err := rows.Scan(&args); err != nil {
			t.Fatal(err)
		}
		var call struct {
			Action string `json:"action"`
			Agent  string `json:"agent"`
		}
		_ = json.Unmarshal([]byte(args), &call)
		if call.Agent != "" {
			hired = append(hired, call.Agent)
		}
	}
	t.Logf("hired: %v — reply: %s", hired, reply.Text)
	if len(hired) == 0 {
		t.Fatalf("asked for the team by name, the model hired nobody: %s", reply.Text)
	}
	team, _ := subagent.LoadTeam(subagent.SeedTeamName)
	if !team.Has(hired[0]) {
		t.Errorf("asked for «%s», the model hired %s, who is not on it (%v)", subagent.SeedTeamName, hired[0], team.Members)
	}
}
