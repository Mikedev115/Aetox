package rpc

// A whole turn across the wire, driven with nothing but the bindings — the
// twin of internal/engine/stream_delivery_test.go, which reaches into the
// engine to seed it. This one cannot, and that is the point: a screen has
// only engine.API and the events, and if a turn can be run with those alone
// then the socket is all the screen ever needed.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// chunk is agent:chunk's payload as the wire carries it.
type chunk struct {
	SessionID string `json:"sessionId"`
	Data      struct {
		Text    string `json:"text"`
		Replace bool   `json:"replace"`
	} `json:"data"`
}

// wired is an engine behind a listener and a client on it, with every event
// kept. The engine's data root is the test's own.
func wired(t *testing.T) (*Server, *Client, *eventLog) {
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
	log := &eventLog{}
	c := NewClient(ClientOptions{OnEvent: log.add, OnFailure: func(method string, err error) { t.Errorf("%s: %v", method, err) }})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Connect(ctx, "tcp", strings.TrimPrefix(hs.URL, "http://"), testToken); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return srv, c, log
}

type eventLog struct {
	mu     sync.Mutex
	events []eventParams
}

func (l *eventLog) add(name string, data json.RawMessage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, eventParams{Name: name, Data: append(json.RawMessage(nil), data...)})
}

func (l *eventLog) named(name string) []json.RawMessage {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []json.RawMessage
	for _, e := range l.events {
		if e.Name == name {
			out = append(out, e.Data)
		}
	}
	return out
}

func TestATurnOverTheWireDeliversTheWholeAnswerExactlyOnce(t *testing.T) {
	_, c, log := wired(t)

	if _, err := c.OpenProjectPath(t.TempDir()); err != nil {
		t.Fatalf("OpenProjectPath: %v", err)
	}
	if _, err := c.SwitchProvider("aetox"); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	if _, err := c.SwitchModel("aetox-render:test"); err != nil {
		t.Fatalf("SwitchModel: %v", err)
	}
	if _, err := c.SwitchApprovalMode("full-access"); err != nil {
		t.Fatalf("SwitchApprovalMode: %v", err)
	}
	if _, err := c.NewSession(); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	info := c.GetModelInfo()
	if info.Provider != "aetox" || info.ModelName != "aetox-render:test" {
		t.Fatalf("GetModelInfo() = %+v, the switches did not land", info)
	}

	reply, err := c.SendMessage("เทสๆ", "")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	var deliveries []chunk
	for _, raw := range log.named("agent:chunk") {
		var ch chunk
		if err := json.Unmarshal(raw, &ch); err != nil {
			t.Fatalf("agent:chunk payload is unreadable: %v", err)
		}
		if ch.SessionID == "" {
			t.Error("agent:chunk arrived with no session on it")
		}
		if ch.Data.Replace && strings.TrimSpace(ch.Data.Text) != "" {
			deliveries = append(deliveries, ch)
		}
	}
	if len(deliveries) != 1 {
		t.Fatalf("want exactly one delivery, got %d — every one of them claims to be the whole answer", len(deliveries))
	}
	for _, want := range []string{"## ทดสอบ Markdown", "| คอลัมน์ |", "picsum.photos"} {
		if !strings.Contains(deliveries[0].Data.Text, want) {
			t.Errorf("the delivered answer is missing %q:\n%s", want, deliveries[0].Data.Text)
		}
	}
	if deliveries[0].Data.Text != reply.Text {
		t.Error("the bubble was drawn from something other than what the turn returned")
	}
	if strings.Count(deliveries[0].Data.Text, "\n") < 5 {
		t.Errorf("the delivery lost its line structure:\n%q", deliveries[0].Data.Text)
	}
}

// The binding shapes the wire has to carry beyond (T, error): several values
// (callN), a bare value with the failure hook, an error alone, nothing.
func TestEveryBindingShapeCrossesTheWire(t *testing.T) {
	_, c, _ := wired(t)

	if v := c.AppVersion(); v == "" {
		t.Error("AppVersion() came back empty")
	}
	if quotas, known := c.ProviderQuotas("openrouter"); known || len(quotas) != 0 {
		t.Errorf("ProviderQuotas on a provider that never reported = %v, %v", quotas, known)
	}
	// Two values back: the model the switch landed on, and the wire format,
	// which for this provider is the default and comes back empty.
	if _, err := c.SwitchProvider("aetox"); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	if name, wire := c.ActiveModelFor("aetox"); name != "aetox-grid" || wire != "" {
		t.Errorf("ActiveModelFor(aetox) = %q, %q — callN mixed the values up", name, wire)
	}
	if name, wire := c.ActiveModelFor("openrouter"); name != "" || wire != "" {
		t.Errorf("ActiveModelFor(openrouter) = %q, %q, want both empty for a provider not on screen", name, wire)
	}
	if err := c.AddMCPServer("", nil); err == nil {
		t.Error("AddMCPServer with no name was accepted")
	}
	c.CancelTurn() // nothing to cancel, and nothing to answer: must not hang
	if list := c.ListSkills(); list == nil {
		t.Error("a nil slice crossed the wire as null — the frontend does .length on it")
	}
}

// A method the engine does not have — a screen newer than its engine —
// answers ErrMethodNotFound rather than hanging or crashing either side.
func TestAnUnknownBindingIsRefusedByName(t *testing.T) {
	_, c, _ := wired(t)
	err := c.Conn().Call(context.Background(), "NoSuchBinding", []any{1}, nil)
	if err == nil || !strings.Contains(err.Error(), "NoSuchBinding") {
		t.Fatalf("err = %v", err)
	}
	var _ http.Handler = nil
}
