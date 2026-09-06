package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
)

// The wait before the wait: a local runtime reading the weights off disk, which
// looks exactly like a hung app because nothing else on screen has anything to
// say yet — no token, no tool, no reasoning.
//
// What is asserted here is as much what is NOT sent as what is. A model already
// in memory must produce no row at all (the common case: every turn after the
// first), and a row that was drawn must be taken away by exactly one clearing
// event, or a chat that has stopped keeps a spinner claiming otherwise.
func loadWatcher(t *testing.T, resident bool) (*App, *conversation, chan ModelLoading, *httptest.Server) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		models := []map[string]any{}
		if resident {
			models = append(models, map[string]any{"name": "qwen3:8b"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
	}))
	t.Cleanup(runtime.Close)

	conv := &conversation{id: newSessionID()}
	a := seed(&App{ctx: context.Background(), dbDir: t.TempDir()}, conv)
	t.Cleanup(func() {
		if a.db != nil {
			_ = a.db.Close()
		}
	})
	seen := make(chan ModelLoading, 16)
	a.emit = func(event string, data ...any) {
		if event != "model:loading" || len(data) == 0 {
			return
		}
		ev, ok := data[0].(sessionEvent[ModelLoading])
		if !ok {
			t.Errorf("model:loading payload is %T, want a stamped sessionEvent", data[0])
			return
		}
		if ev.SessionID != conv.id {
			t.Errorf("model:loading arrived stamped %q, want the chat that asked (%q)", ev.SessionID, conv.id)
		}
		seen <- ev.Data
	}
	conv.cfg = config.Config{ModelProvider: "ollama", ModelBaseURL: runtime.URL, ModelName: "qwen3"}
	return a, conv, seen, runtime
}

func TestModelLoadWatchReportsTheWaitAndEndsIt(t *testing.T) {
	quickLoadWatch(t)
	a, conv, seen, _ := loadWatcher(t, false)

	stop := a.watchModelLoad(context.Background(), conv)
	var first ModelLoading
	select {
	case first = <-seen:
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was reported — a model that never goes resident is a wait nobody can see")
	}
	if !first.Loading {
		t.Fatalf("first event says loading=%v, want the row to be drawn", first.Loading)
	}
	if first.Model != "qwen3" || first.Provider != "ollama" {
		t.Fatalf("row names %q on %q, want the turn's own model and runtime", first.Model, first.Provider)
	}

	stop()
	// Drain until the clearing event: the ticker may have queued another
	// still-loading update before stop() landed, which is not a failure.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-seen:
			if !ev.Loading {
				return
			}
		case <-deadline:
			t.Fatal("the watch ended without clearing the row — the spinner would outlive the turn")
		}
	}
}

func TestModelLoadWatchSaysNothingWhenTheModelIsAlreadyIn(t *testing.T) {
	quickLoadWatch(t)
	a, conv, seen, _ := loadWatcher(t, true)

	stop := a.watchModelLoad(context.Background(), conv)
	defer stop()
	select {
	case ev := <-seen:
		t.Fatalf("a resident model drew a loading row (%+v) — every turn after the first would flash one", ev)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestModelLoadWatchIgnoresProvidersThatRunSomewhereElse(t *testing.T) {
	quickLoadWatch(t)
	a, conv, seen, runtime := loadWatcher(t, false)
	// Same endpoint, a provider whose weights are not on this machine: the wait
	// on it is the network's, and this row would misname it.
	conv.cfg = config.Config{ModelProvider: "deepseek", ModelBaseURL: runtime.URL, ModelName: "deepseek-chat"}

	stop := a.watchModelLoad(context.Background(), conv)
	defer stop()
	select {
	case ev := <-seen:
		t.Fatalf("a remote provider drew a local loading row (%+v)", ev)
	case <-time.After(300 * time.Millisecond):
	}
}

// The timings the UI is tuned to are written for human eyes; a test that slept
// through them would spend a second and a half proving nothing.
func quickLoadWatch(t *testing.T) {
	t.Helper()
	poll, grace := modelLoadPoll, modelLoadGrace
	modelLoadPoll, modelLoadGrace = 5*time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { modelLoadPoll, modelLoadGrace = poll, grace })
}
