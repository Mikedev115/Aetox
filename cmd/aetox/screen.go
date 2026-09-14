package main

// The console as the engine sees it: engine.Screen for a terminal (§248's
// third screen, after the window and the remote window). What a screen owes
// the engine is the credential signer, the desk questions a key is needed
// for, and somewhere for events to go; what this one has that the window
// has not is nothing — no browser, no computer reach, no deck renderer —
// and it says so rather than pretending.
//
// Events arrive on the engine's own goroutine, so Emit must never block on
// the user: a question is handed to the reader loop and answered from there
// (AnswerUserQuestion takes the same lock beginUserQuestion emits under).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/credentials"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
	"github.com/Mikedev115/Aetox/internal/signer"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// cliScreen is the terminal's Screen. out is where the answer goes (stdout,
// so a script reads the answer and nothing else); log is where the work is
// narrated (stderr — tool calls, questions, status).
type cliScreen struct {
	out io.Writer
	log io.Writer

	mu sync.Mutex
	// preview is what of the live answer has been printed since the last
	// delivery, rounds joined by line breaks, so the authoritative delivery
	// prints only what streaming missed.
	preview strings.Builder
	// delivered says the authoritative answer of this turn has been printed,
	// so finishTurn does not print it a second time from the reply.
	delivered bool
	// pendingCalls are tool calls announced before their arguments were known
	// (a streamed call arrives once by name, once with its subject); printed
	// when the subject arrives, or at the result if it never does.
	pendingCalls map[string]string
	// asks carries a question the engine is waiting on to whoever reads the
	// terminal (console.turn); buffered so Emit never waits on the reader.
	asks chan askEvent
}

type askEvent struct {
	SessionID string
	Question  string
	Options   []string
}

func newCLIScreen() *cliScreen {
	return &cliScreen{out: os.Stdout, log: os.Stderr, asks: make(chan askEvent, 4), pendingCalls: map[string]string{}}
}

// Emit is every engine event. The payload is the engine's own Go value,
// several of them unexported types, so it is read through JSON the way the
// wire does (or by field name where JSON would damage it) — not a type
// switch on names this file cannot see.
func (s *cliScreen) Emit(event string, data any) {
	switch event {
	case "agent:chunk":
		// Read by reflection, not JSON: a streamed chunk can end in the
		// middle of a rune, and json.Marshal would replace the torn bytes
		// with U+FFFD — printing a � where the next chunk completes the
		// letter, and leaving the preview unequal to the answer it was.
		chunk, ok := field(data, "Data")
		if !ok {
			return
		}
		text, _ := field(chunk, "Text")
		replace, _ := field(chunk, "Replace")
		s.mu.Lock()
		defer s.mu.Unlock()
		if r, _ := replace.(bool); !r {
			s.preview.WriteString(text.(string))
			fmt.Fprint(s.out, text.(string))
			return
		}
		if text.(string) == "" {
			// A discarded preview — the engine clears the bubble before the
			// authoritative text lands. What streamed stays on screen (it
			// was said), and stays in preview too: the delivery that follows
			// is compared against it, or the answer would print twice.
			s.endLine()
			return
		}
		s.deliver(text.(string))
	case "agent:tool":
		var ev struct {
			Data struct {
				Action  string `json:"action"`
				Name    string `json:"name"`
				Ref     string `json:"ref"`
				Subject string `json:"subject"`
				Act     string `json:"act"`
				Parent  string `json:"parent"`
			} `json:"data"`
		}
		if decode(data, &ev) != nil {
			return
		}
		indent := ""
		if ev.Data.Parent != "" {
			indent = "    "
		}
		label := ev.Data.Name
		if ev.Data.Act != "" {
			label += " " + ev.Data.Act
		}
		if ev.Data.Subject != "" {
			label += " " + ev.Data.Subject
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		switch ev.Data.Action {
		case "call":
			// A streamed call is announced twice — once by name when the
			// provider names it, once more when its arguments have parsed
			// and the subject is known. The window updates one row; a
			// terminal cannot, so the bare announcement waits for the fuller
			// one, and is printed as it is only if none comes by the result.
			if ev.Data.Ref != "" && ev.Data.Subject == "" && ev.Data.Act == "" {
				s.pendingCalls[ev.Data.Ref] = indent + "  ⚙ " + label
				return
			}
			delete(s.pendingCalls, ev.Data.Ref)
			s.endLine()
			fmt.Fprintf(s.log, "%s  ⚙ %s\n", indent, label)
		case "result":
			if line, ok := s.pendingCalls[ev.Data.Ref]; ok {
				delete(s.pendingCalls, ev.Data.Ref)
				s.endLine()
				fmt.Fprintln(s.log, line)
			}
		}
	case "agent:status":
		var ev struct {
			Data string `json:"data"`
		}
		if decode(data, &ev) != nil || strings.TrimSpace(ev.Data) == "" {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.endLine()
		fmt.Fprintf(s.log, "  … %s\n", ev.Data)
	case "ask:user":
		var ev struct {
			SessionID string `json:"sessionId"`
			Data      struct {
				Question string   `json:"question"`
				Options  []string `json:"options"`
			} `json:"data"`
		}
		if decode(data, &ev) != nil {
			return
		}
		select {
		case s.asks <- askEvent{SessionID: ev.SessionID, Question: ev.Data.Question, Options: ev.Data.Options}:
		default:
			fmt.Fprintln(s.log, "  ! a question arrived while another was still unanswered — dropped")
		}
	}
}

// breakPreview ends the answer line before a log line, so narration never
// lands in the middle of the model's sentence.
func (s *cliScreen) breakPreview() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endLine()
}

// endLine is breakPreview under the lock: only when something has streamed
// since the last break.
func (s *cliScreen) endLine() {
	if s.preview.Len() > 0 && !strings.HasSuffix(s.preview.String(), "\n") {
		fmt.Fprintln(s.out)
		s.preview.WriteString("\n")
	}
}

// deliver prints what of the authoritative answer streaming did not: nothing
// when the preview was the answer, the tail when the answer extends it, and
// the whole answer on its own line otherwise. Under the lock.
func (s *cliScreen) deliver(final string) {
	shown := s.preview.String()
	printed := strings.TrimSuffix(shown, "\n")
	s.preview.Reset()
	s.delivered = true
	switch {
	case strings.TrimSpace(final) == "":
		return
	// The last round's preview is the answer; earlier rounds' text (the
	// model talking before a tool call) stays above it, as it was said.
	case final == printed, strings.HasSuffix(printed, strings.TrimSpace(final)):
		if !strings.HasSuffix(shown, "\n") {
			fmt.Fprintln(s.out)
		}
		return
	case printed != "" && strings.HasPrefix(final, printed):
		fmt.Fprint(s.out, strings.TrimPrefix(final, printed))
	default:
		if printed != "" && !strings.HasSuffix(shown, "\n") {
			fmt.Fprintln(s.out)
		}
		fmt.Fprint(s.out, final)
	}
	if !strings.HasSuffix(final, "\n") {
		fmt.Fprintln(s.out)
	}
}

// finishTurn is the end of a turn as SendMessage reported it: the reply is
// printed only when no chunk event delivered it already, and the turn's
// bookkeeping is cleared for the next one.
func (s *cliScreen) finishTurn(final string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.delivered {
		s.deliver(final)
	}
	s.delivered = false
	s.preview.Reset()
	for k := range s.pendingCalls {
		delete(s.pendingCalls, k)
	}
}

func decode(data any, into any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

// field reads one exported field of a struct the engine emitted, whatever
// the struct's own name — the event types are the engine's and several are
// unexported, but their fields are not.
func field(data any, name string) (any, bool) {
	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, false
	}
	f := v.FieldByName(name)
	if !f.IsValid() || !f.CanInterface() {
		return nil, false
	}
	return f.Interface(), true
}

// The credential: this screen's stores are the pasted keys
// (internal/credentials) and the sign-ins (internal/oauth), read per
// request by the shared signer. A key typed on the command line belongs to
// its provider alone (keyFor). Nothing rides anywhere in particular — the
// engine is in this process, on this machine, so every URL it built came
// from settings on this disk (the desktop's credentialMayRide says the same
// for its in-process case).
func (s *cliScreen) ProviderTransport(provider, wireFormat string) model.Transport {
	return signer.Transport(signer.Options{Provider: provider, WireFormat: wireFormat, Key: keyFor})
}

func (s *cliScreen) ProviderEndpoint(provider string) string { return oauth.Endpoint(provider) }

// WindowTools: none. A terminal has no browser tab and no window to drive;
// the engine's own tools are the whole reach, which is also the reach a
// benchmark wants (harness-bench-2026-09-14.md H4).
func (s *cliScreen) WindowTools(engine.Session) []skill.Skill { return nil }
func (s *cliScreen) AgentTab() string                         { return "" }

func (s *cliScreen) DefaultModel(provider, baseURL string) string {
	return model.ResolveDefaultModel(provider, baseURL, keyFor(provider))
}

func (s *cliScreen) Probe(provider, modelName, baseURL, wireFormat string) (string, error) {
	return signer.Probe(provider, modelName, baseURL, keyFor(provider), wireFormat)
}

func (s *cliScreen) ModelResident(provider, baseURL, modelName string) bool {
	return model.LocalModelResident(provider, baseURL, keyFor(provider), modelName)
}

var errNoWebview = errors.New("the console has no webview to render a deck with — open the desktop app for that")

func (s *cliScreen) RenderDeck(context.Context, string, engine.DeckRender) (engine.DeckRendered, error) {
	return engine.DeckRendered{}, errNoWebview
}

var _ engine.Screen = (*cliScreen)(nil)

// flagAPIKey is --model-api-key, and flagAPIKeyProvider the provider it was
// given for: a key typed on the command line belongs to that provider alone.
var flagAPIKey, flagAPIKeyProvider string

// keyFor is the key this screen reaches a provider with: the command-line key
// when it was given for this provider, else what the store or the provider's
// environment variable holds (credentials.KeyFor).
func keyFor(providerName string) string {
	canonical := model.NormalizeProvider(providerName)
	if flagAPIKey != "" && canonical == flagAPIKeyProvider {
		return flagAPIKey
	}
	return credentials.KeyFor(canonical)
}

// rememberKey stores a key the user just typed for a provider, so the next
// launch does not ask again. Empty is nothing to remember, not a deletion.
func rememberKey(providerName, apiKey string) {
	if strings.TrimSpace(apiKey) == "" {
		return
	}
	if err := credentials.Set(providerName, apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot save API key: %v\n", err)
	}
}
