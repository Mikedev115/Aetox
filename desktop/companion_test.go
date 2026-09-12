package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The feed is a mirror of the window's presence behind a token on loopback.
// What is guarded: the token is the whole door, the state round-trips with a
// sequence a poller can lean on, and the page is the generated one.

func TestCompanionFeedIsBehindItsToken(t *testing.T) {
	a := &App{}
	c := a.companion()
	h := c.handler()

	for _, path := range []string{"/", "/c/", "/c/wrong/state", "/c/" + c.token + "/other", "/c/" + c.token[:8] + "/state"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: want 404, got %d", path, rec.Code)
		}
	}
}

func TestCompanionStateRoundTripsWithASequence(t *testing.T) {
	a := &App{}
	c := a.companion()
	h := c.handler()

	read := func() CompanionState {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/c/"+c.token+"/state", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("state: want 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("state content type: %s", ct)
		}
		var s CompanionState
		if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
			t.Fatalf("state json: %v", err)
		}
		return s
	}

	if s := read(); s.Seq != 0 || s.Pose != "" {
		t.Fatalf("before any report: %+v", s)
	}
	a.SetCompanionState(CompanionState{Pose: "reading", Report: "อ่าน config แล้ว", On: true, Prefs: CompanionPrefs{Shell: "colour", Accent: "mint", Top: "orb", Face: "neutral"}})
	s := read()
	if s.Seq != 1 || s.Pose != "reading" || s.Report != "อ่าน config แล้ว" || !s.On || s.Prefs.Shell != "colour" || s.Prefs.Accent != "mint" {
		t.Fatalf("after one report: %+v", s)
	}
	if s.At.IsZero() {
		t.Fatal("a report carries the time it was made")
	}
	// The same pose reported again is a new sequence: a poller that saw seq 1
	// must be able to tell a repeat from silence.
	a.SetCompanionState(CompanionState{Pose: "reading", On: true})
	if s := read(); s.Seq != 2 || s.Prefs.Accent != "" {
		t.Fatalf("after a second report: %+v", s)
	}
}

func TestCompanionPageIsTheGeneratedOne(t *testing.T) {
	a := &App{}
	c := a.companion()
	rec := httptest.NewRecorder()
	c.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/c/"+c.token+"/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("page: want 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"<title>Aetox companion</title>", "fetch('state'", "mascotSVG", ".mascot"} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("the page and the state must not be cached")
	}
}

func TestCompanionFeedListensOnLoopbackOnly(t *testing.T) {
	a := &App{}
	info := a.CompanionFeed()
	defer a.StopCompanionFeed()
	if !info.Running || info.Port == 0 {
		t.Fatalf("feed did not start: %+v", info)
	}
	if !strings.HasPrefix(info.URL, "http://127.0.0.1:") || !strings.HasSuffix(info.URL, "/c/"+a.companion().token+"/") {
		t.Fatalf("feed url: %s", info.URL)
	}
	resp, err := http.Get(info.URL + "state")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state over the wire: %d", resp.StatusCode)
	}
	// Starting again is a no-op on the same port; stopping reports nothing running.
	if again := a.CompanionFeed(); again.Port != info.Port {
		t.Fatalf("second start moved the port: %d → %d", info.Port, again.Port)
	}
	if after := a.StopCompanionFeed(); after.Running {
		t.Fatalf("still running after stop: %+v", after)
	}
}
