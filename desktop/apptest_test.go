package main

import "github.com/Mikedev115/Aetox/internal/engine"

// newTestApp is the screen a test holds: built the way NewApp builds it, with
// the engine talking back to this very App, and no window — a.ctx stays nil,
// so anything that would reach the Wails runtime reaches the test seams
// (emit, openDir) or nothing.
func newTestApp() *App {
	a := &App{}
	a.eng = engine.NewEngine(appScreen{a})
	return a
}
