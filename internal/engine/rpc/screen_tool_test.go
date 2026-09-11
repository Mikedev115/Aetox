package rpc

// The window's tools across the wire: announced in hello, lent as stubs,
// run on the screen, answered with the real output and the error's mark.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/callfault"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

// fakePack is a window tool the shape of the browser: packed, guided, and
// keeping a record of every call the engine sent it.
type fakePack struct {
	name    string
	actions []string // narrowed, nil for all
	mu      *sync.Mutex
	calls   *[]ToolCallParams
	sess    engine.Session
}

func newFakePack(name string) *fakePack {
	return &fakePack{name: name, mu: &sync.Mutex{}, calls: &[]ToolCallParams{}}
}

func (f *fakePack) allowed() []string {
	if len(f.actions) > 0 {
		return f.actions
	}
	var out []string
	for _, c := range skill.PackedCalls(f.name) {
		out = append(out, c.Action)
	}
	return out
}

func (f *fakePack) Name() string { return f.name }
func (f *fakePack) Description() string {
	return "fake " + f.name + ": " + strings.Join(f.allowed(), " ")
}
func (f *fakePack) Actions() []string { return skill.PackedActions(f.name) }
func (f *fakePack) ToolDefinition() model.ToolDefinition {
	params, _ := json.Marshal(map[string]any{"type": "object", "properties": map[string]any{
		"action": map[string]any{"type": "string", "enum": f.allowed()},
	}})
	return model.ToolDefinition{Type: "function", Function: model.ToolFunction{Name: f.name, Description: f.Description(), Parameters: params}}
}
func (f *fakePack) Narrow(named []string) skill.Skill {
	want := map[string]bool{}
	for _, n := range named {
		want[n] = true
	}
	var actions []string
	for _, c := range skill.PackedCalls(f.name) {
		if want[c.Permission] {
			actions = append(actions, c.Action)
		}
	}
	if len(actions) == 0 {
		return f
	}
	cp := *f
	cp.actions = actions
	return &cp
}
func (f *fakePack) Guidance(args map[string]any) string {
	if _, ok := args["steps"]; ok {
		return "batch wisely"
	}
	action, _ := args["action"].(string)
	return "about " + action
}
func (f *fakePack) Execute(ctx context.Context, in skill.Input) (skill.Output, error) {
	return f.ExecuteTool(ctx, map[string]any(in))
}
func (f *fakePack) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	f.mu.Lock()
	rec := ToolCallParams{Name: f.name, Args: args, Actions: f.actions}
	if f.sess != nil {
		rec.Session, rec.Root = f.sess.ID(), f.sess.Root()
	}
	*f.calls = append(*f.calls, rec)
	f.mu.Unlock()
	action, _ := args["action"].(string)
	switch action {
	case "click":
		return skill.Output{Name: f.name}, callfault.New("browser click needs a ref")
	case "read":
		return skill.Output{Name: f.name}, statereport.New("the page is still loading")
	}
	return skill.Output{Name: f.name, Success: true, Content: "did " + action + " for " + rec.Session, Images: []model.Image{{MediaType: "image/png", Data: []byte("png")}}}, nil
}

// screenWithTools is an engine behind the wire whose screen lends the fake
// browser, hello already said.
func screenWithTools(t *testing.T) (*Server, *Client, *fakePack) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	srv := NewServer(testToken, engine.NewEngine)
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		engine.Shutdown(srv.Engine(), ctx)
	})
	browser := newFakePack("browser")
	c := NewClient(ClientOptions{})
	ServeTools(c, func(sess engine.Session) []skill.Skill {
		cp := *browser
		cp.sess = sess
		return []skill.Skill{&cp}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Connect(ctx, "tcp", strings.TrimPrefix(hs.URL, "http://"), testToken); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	hello, err := c.Hello(ctx, "test", []skill.Skill{browser}, []string{FeatureWindowTools})
	if err != nil {
		t.Fatalf("hello: %v", err)
	}
	if hello.Protocol != Protocol || hello.PID == 0 || hello.OS == "" {
		t.Errorf("hello answered %+v", hello)
	}
	return srv, c, browser
}

func TestAnAnnouncedToolIsLentAndRunsOnTheScreen(t *testing.T) {
	srv, _, browser := screenWithTools(t)

	tools := srv.Screen().WindowTools(session{"s1", `C:\proj`})
	if len(tools) != 1 || tools[0].Name() != "browser" {
		t.Fatalf("the engine lends %d tools, want the announced browser", len(tools))
	}
	stub := tools[0].(skill.Tool)
	if stub.Description() != browser.Description() {
		t.Errorf("description crossed as %q", stub.Description())
	}
	if string(stub.ToolDefinition().Function.Parameters) != string(browser.ToolDefinition().Function.Parameters) {
		t.Error("the definition did not cross byte for byte")
	}
	if g := tools[0].(skill.Guided).Guidance(map[string]any{"action": "open"}); g != "about open" {
		t.Errorf("guidance for open = %q", g)
	}
	if g := tools[0].(skill.Guided).Guidance(map[string]any{"steps": []any{1}}); g != "batch wisely" {
		t.Errorf("guidance for steps = %q", g)
	}

	out, err := stub.ExecuteTool(context.Background(), map[string]any{"action": "open", "url": "https://x"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !out.Success || out.Content != "did open for s1" {
		t.Errorf("output = %+v", out)
	}
	if len(out.Images) != 1 || string(out.Images[0].Data) != "png" {
		t.Errorf("the picture did not cross: %+v", out.Images)
	}
	browser.mu.Lock()
	calls := append([]ToolCallParams(nil), *browser.calls...)
	browser.mu.Unlock()
	if len(calls) != 1 || calls[0].Session != "s1" || calls[0].Root != `C:\proj` || calls[0].Args["url"] != "https://x" {
		t.Errorf("the screen's tool saw %+v", calls)
	}
}

// The error's mark survives: a refusal of the call is still a callfault on
// the engine's side, a report about the machine still a statereport.
func TestAToolErrorKeepsItsMarkAcrossTheWire(t *testing.T) {
	srv, _, _ := screenWithTools(t)
	stub := srv.Screen().WindowTools(session{"s1", ""})[0].(skill.Tool)

	_, err := stub.ExecuteTool(context.Background(), map[string]any{"action": "click"})
	if !callfault.Is(err) || err == nil || !strings.Contains(err.Error(), "needs a ref") {
		t.Errorf("click = %v, want a callfault with the tool's sentence", err)
	}
	_, err = stub.ExecuteTool(context.Background(), map[string]any{"action": "read"})
	if !statereport.Is(err) {
		t.Errorf("read = %v, want a statereport", err)
	}
}

// Narrowing on the engine's side reaches the screen's own Narrow: the shape
// the model reads names only the actions the stance allows, and a call for
// another goes to the narrowed tool.
func TestANarrowedStubTakesItsShapeFromTheScreen(t *testing.T) {
	srv, _, browser := screenWithTools(t)
	stub := srv.Screen().WindowTools(session{"s1", ""})[0]
	cut := stub.(skill.Packed).Narrow([]string{"browser_read", "browser_open"})
	if cut == stub {
		t.Fatal("narrowing to two actions handed back the whole tool")
	}
	var schema struct {
		Properties struct {
			Action struct {
				Enum []string `json:"enum"`
			} `json:"action"`
		} `json:"properties"`
	}
	def := cut.(skill.Tool).ToolDefinition()
	if err := json.Unmarshal(def.Function.Parameters, &schema); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(schema.Properties.Action.Enum, "click") || !slices.Contains(schema.Properties.Action.Enum, "open") {
		t.Errorf("narrowed enum = %v", schema.Properties.Action.Enum)
	}
	if !strings.Contains(cut.Description(), "open") || strings.Contains(cut.Description(), "click") {
		t.Errorf("narrowed description = %q", cut.Description())
	}
	// The call carries the narrowing, so the screen runs the narrowed tool.
	if _, err := cut.(skill.Tool).ExecuteTool(context.Background(), map[string]any{"action": "open"}); err != nil {
		t.Fatal(err)
	}
	browser.mu.Lock()
	last := (*browser.calls)[len(*browser.calls)-1]
	browser.mu.Unlock()
	if len(last.Actions) == 0 || slices.Contains(last.Actions, "click") {
		t.Errorf("the screen ran the tool with actions %v, want the narrowed set", last.Actions)
	}
}

// After hello the engine's own registry carries the browser as a workbench
// tool — the announcement reached workbenchSkills, not only WindowTools.
func TestTheEngineListsTheScreensToolAfterHello(t *testing.T) {
	_, c, _ := screenWithTools(t)
	if _, err := c.OpenProjectPath(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SwitchProvider("aetox"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range c.ListTools() {
		if tool.Name == "browser" {
			found = true
			if tool.Source != "workbench" {
				t.Errorf("browser has source %q, want workbench — it ships with Aetox", tool.Source)
			}
		}
	}
	if !found {
		t.Error("the browser the screen announced is not among the session's tools")
	}
}

// With no screen the stub does not hang the turn: it waits its budget and
// answers as a report about the machine, in words the model can act on.
func TestAStubWithNoScreenReportsRatherThanHangs(t *testing.T) {
	prev := screenWait
	screenWait = 100 * time.Millisecond
	t.Cleanup(func() { screenWait = prev })
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	srv := NewServer(testToken, engine.NewEngine)
	srv.Screen().announced.set(HelloParams{Protocol: Protocol, Tools: Announce([]skill.Skill{newFakePack("browser")})})
	stub := srv.Screen().WindowTools(session{"s1", ""})[0].(skill.Tool)
	out, err := stub.ExecuteTool(context.Background(), map[string]any{"action": "open"})
	if !errors.Is(err, ErrNoScreen) || !statereport.Is(err) {
		t.Errorf("err = %v, want ErrNoScreen marked as a state report", err)
	}
	if out.Success || !out.FromWorld || out.Content == "" {
		t.Errorf("output = %+v", out)
	}
	_ = fmt.Sprint
}
