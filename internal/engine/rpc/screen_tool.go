package rpc

// The window's tools across the wire (§248 phase 2, design doc §6).
//
// The browser and the machine act on the screen's computer, so they run
// there; what the engine needs is their shape — a name, a description, the
// definition the model reads, the actions a stance may narrow, the guidance
// sent once — and a way to run one call. The screen announces the shapes in
// `hello` when it connects, the engine lends a stub per announced tool to
// every session (Screen.WindowTools), and each call the model makes on a
// stub crosses as `screen.tool` and comes back as the skill.Output the real
// tool produced, with its error and the error's mark. The registry, the
// stance filter, the permission gates and the executor see a skill.Tool that
// is also skill.Packed and skill.Guided, exactly as they saw the real one.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/callfault"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

const (
	MethodHello          = "hello"
	MethodScreenTool     = "screen.tool"
	MethodScreenToolCut  = "screen.toolCut"
	Protocol             = 1
	faultCall            = "call"
	faultState           = "state"
	featureDialogs       = "dialogs"
	featureFileManager   = "fileManager"
	featureWindowTools   = "windowTools"
	featureProviderProxy = "providerProxy"
)

// ToolAnnouncement is one window tool as the screen describes it in hello.
type ToolAnnouncement struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Definition  model.ToolDefinition `json:"definition"`
	// Actions are the per-action permission names (skill.Packed), empty
	// for a tool that is not packed.
	Actions []string `json:"actions"`
	// Guidance is what the tool teaches once, by action — and under "steps"
	// for a batched call — so the engine can send it without asking.
	Guidance map[string]string `json:"guidance"`
}

// HelloParams is the screen's opening message.
type HelloParams struct {
	Protocol int                `json:"protocol"`
	Version  string             `json:"version"`
	Tools    []ToolAnnouncement `json:"tools"`
	Features []string           `json:"features"`
}

// HelloResult is the engine's answer: who it is and where it runs.
type HelloResult struct {
	Protocol int    `json:"protocol"`
	Version  string `json:"version"`
	Root     string `json:"root"`
	DataRoot string `json:"dataRoot"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	PID      int    `json:"pid"`
	// Hostname is the machine the engine runs on — what a screen attached by
	// hand (AETOX_ENGINE_ADDR) compares with its own to learn whether a path
	// here is a path there (desktop/host_files.go). Empty from an older engine.
	Hostname string `json:"hostname,omitempty"`
}

// ToolCallParams is one call on a window tool.
type ToolCallParams struct {
	Session string         `json:"session"`
	Root    string         `json:"root"`
	Name    string         `json:"name"`
	Actions []string       `json:"actions,omitempty"` // narrowed permission names, none for the whole tool
	Args    map[string]any `json:"args"`
}

// ToolCallResult is what came back: the output, and the error with its
// mark, so the executor's reading of a refusal (callfault) or a report
// about the machine (statereport) is the same as in one process.
type ToolCallResult struct {
	Output skill.Output `json:"output"`
	Error  string       `json:"error,omitempty"`
	Fault  string       `json:"fault,omitempty"`
}

// ToolCutParams asks for a tool narrowed to some of its actions, as the
// screen's own Narrow would shape it: the description and definition the
// model reads then name only those.
type ToolCutParams struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

type ToolCutResult struct {
	Description string               `json:"description"`
	Definition  model.ToolDefinition `json:"definition"`
}

// announced is what the current screen said in hello, on the peer.
type announced struct {
	mu       sync.RWMutex
	tools    []ToolAnnouncement
	features map[string]bool
	version  string
	// cuts caches narrowed shapes by tool and action set, since the block
	// is rebuilt on every switch and the screen's answer does not change.
	cuts map[string]ToolCutResult
}

func (a *announced) set(h HelloParams) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = h.Tools
	a.version = h.Version
	a.features = map[string]bool{}
	for _, f := range h.Features {
		a.features[f] = true
	}
	a.cuts = map[string]ToolCutResult{}
}

func (a *announced) list() []ToolAnnouncement {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tools
}

// Features reports what the connected screen said it can do.
func (p *ScreenPeer) Features() map[string]bool {
	p.announced.mu.RLock()
	defer p.announced.mu.RUnlock()
	out := map[string]bool{}
	for k := range p.announced.features {
		out[k] = true
	}
	return out
}

// hello is the server's handler for the screen's opening message.
func (s *Server) hello(_ context.Context, _ string, params json.RawMessage) (any, error) {
	var h HelloParams
	if err := decodeParams(params, &h); err != nil {
		return nil, err
	}
	if h.Protocol != Protocol {
		return nil, fmt.Errorf("hello: the screen speaks protocol %d, this engine speaks %d", h.Protocol, Protocol)
	}
	s.peer.announced.set(h)
	root, _ := os.Getwd()
	if st := s.engine.GetProjectStatus(); st.Path != "" {
		root = st.Path
	}
	return HelloResult{
		Protocol: Protocol,
		Version:  s.engine.AppVersion(),
		Root:     root,
		DataRoot: s.dataRoot,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		PID:      os.Getpid(),
		Hostname: hostname(),
	}, nil
}

// WindowTools are the screen's announced tools, as stubs bound to the
// session: one per announcement, each a skill.Tool that runs on the screen.
func (p *ScreenPeer) WindowTools(sess engine.Session) []skill.Skill {
	var out []skill.Skill
	for _, ann := range p.announced.list() {
		out = append(out, &screenTool{peer: p, ann: ann, sess: sess})
	}
	return out
}

// screenTool is one announced tool as the engine holds it.
type screenTool struct {
	peer *ScreenPeer
	ann  ToolAnnouncement
	sess engine.Session
	// actions this caller may use, nil for all of them. Set only by Narrow.
	actions []string
	// cut is the narrowed shape, fetched once.
	cutOnce sync.Once
	cut     ToolCutResult
	cutErr  error
}

func (t *screenTool) Name() string { return t.ann.Name }

func (t *screenTool) Description() string {
	if len(t.actions) > 0 {
		if cut, err := t.narrowed(); err == nil {
			return cut.Description
		}
	}
	return t.ann.Description
}

func (t *screenTool) ToolDefinition() model.ToolDefinition {
	if len(t.actions) > 0 {
		if cut, err := t.narrowed(); err == nil {
			return cut.Definition
		}
	}
	return t.ann.Definition
}

func (t *screenTool) Actions() []string { return append([]string(nil), t.ann.Actions...) }

// Narrow is the stub's half of the stance filter: a copy that names fewer
// actions, whose shape the screen is asked for once. Silence is the whole
// tool, the rule every pack follows.
func (t *screenTool) Narrow(named []string) skill.Skill {
	want := map[string]bool{}
	for _, n := range named {
		want[strings.ToLower(strings.TrimSpace(n))] = true
	}
	var actions []string
	for _, a := range t.ann.Actions {
		if want[a] {
			actions = append(actions, a)
		}
	}
	if len(actions) == 0 || len(actions) == len(t.ann.Actions) {
		return t
	}
	return &screenTool{peer: t.peer, ann: t.ann, sess: t.sess, actions: actions}
}

func (t *screenTool) narrowed() (ToolCutResult, error) {
	t.cutOnce.Do(func() {
		key := t.ann.Name + "\x00" + strings.Join(t.actions, ",")
		t.peer.announced.mu.RLock()
		cut, ok := t.peer.announced.cuts[key]
		t.peer.announced.mu.RUnlock()
		if ok {
			t.cut = cut
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		t.cutErr = t.peer.call(ctx, MethodScreenToolCut, []any{ToolCutParams{Name: t.ann.Name, Actions: t.actions}}, &t.cut)
		if t.cutErr == nil {
			t.peer.announced.mu.Lock()
			t.peer.announced.cuts[key] = t.cut
			t.peer.announced.mu.Unlock()
		}
	})
	return t.cut, t.cutErr
}

func (t *screenTool) Guidance(args map[string]any) string {
	if steps, ok := args["steps"].([]any); ok && len(steps) > 0 {
		return t.ann.Guidance["steps"]
	}
	action, _ := args["action"].(string)
	return t.ann.Guidance[strings.ToLower(strings.TrimSpace(action))]
}

func (t *screenTool) Execute(ctx context.Context, in skill.Input) (skill.Output, error) {
	return t.ExecuteTool(ctx, map[string]any(in))
}

func (t *screenTool) ExecuteTool(ctx context.Context, args map[string]any) (skill.Output, error) {
	params := ToolCallParams{Name: t.ann.Name, Actions: t.actions, Args: args}
	if t.sess != nil {
		params.Session, params.Root = t.sess.ID(), t.sess.Root()
	}
	var res ToolCallResult
	if err := t.peer.call(ctx, MethodScreenTool, []any{params}, &res); err != nil {
		if errors.Is(err, ErrNoScreen) {
			return skill.Output{Name: t.ann.Name, Content: "หน้าต่างไม่ได้เชื่อมต่ออยู่ตอนนี้ จึงใช้ " + t.ann.Name + " ไม่ได้", FromWorld: true},
				statereport.Mark(err)
		}
		return skill.Output{Name: t.ann.Name}, err
	}
	var err error
	if res.Error != "" {
		err = errors.New(res.Error)
		switch res.Fault {
		case faultCall:
			err = callfault.Mark(err)
		case faultState:
			err = statereport.Mark(err)
		}
	}
	return res.Output, err
}

var (
	_ skill.Tool   = (*screenTool)(nil)
	_ skill.Packed = (*screenTool)(nil)
	_ skill.Guided = (*screenTool)(nil)
)

// hostname is this machine's name, "" when it has none to give.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
