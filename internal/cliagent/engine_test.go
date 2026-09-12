package cliagent

import (
	"context"
	"testing"
)

// stub is the shape an engine takes, which is also what the desktop's
// translation is tested against. It is the only engine this repository has.
type stub struct {
	provider string
	status   Status
	events   []Event
	result   Result
	err      error
	sawTurn  Turn
}

func (s *stub) Provider() string             { return s.provider }
func (s *stub) Label() string                { return "Stub" }
func (s *stub) Probe(context.Context) Status { return s.status }
func (s *stub) Run(_ context.Context, t Turn, onEvent func(Event)) (Result, error) {
	s.sawTurn = t
	for _, ev := range s.events {
		if onEvent != nil {
			onEvent(ev)
		}
	}
	return s.result, s.err
}

func TestRegistryIsEmptyUntilSomethingRegisters(t *testing.T) {
	// The state this package ships in, and the state the desktop's turn path
	// relies on: no engine registered means every lookup misses and Aetox
	// answers the way it always has.
	for _, name := range Providers() {
		t.Errorf("an engine is registered at rest: %q", name)
	}
	if _, ok := For("anything"); ok {
		t.Error("For hit on an empty registry")
	}
}

func TestRegisterAndFind(t *testing.T) {
	e := &stub{provider: "stub-a"}
	Register(e)
	t.Cleanup(func() { unregister("stub-a") })

	got, ok := For("stub-a")
	if !ok || got != e {
		t.Fatalf("For(stub-a) = %v, %v", got, ok)
	}
	if _, ok := For("Stub-A"); ok {
		// No normalization here on purpose — the caller holds the catalog.
		t.Error("For matched an unnormalized id")
	}
	if names := Providers(); len(names) != 1 || names[0] != "stub-a" {
		t.Errorf("Providers = %v", names)
	}
}

func TestRegisterRefusesDuplicatesAndBlanks(t *testing.T) {
	Register(&stub{provider: "stub-b"})
	t.Cleanup(func() { unregister("stub-b") })
	if !panics(func() { Register(&stub{provider: "stub-b"}) }) {
		t.Error("a duplicate provider was accepted")
	}
	if !panics(func() { Register(&stub{provider: ""}) }) {
		t.Error("an engine with no provider was accepted")
	}
	if !panics(func() { Register(nil) }) {
		t.Error("a nil engine was accepted")
	}
}

func TestNewSessionIDIsUUIDv4(t *testing.T) {
	id := NewSessionID()
	if len(id) != 36 || id[14] != '4' {
		t.Errorf("id = %q", id)
	}
	if id == NewSessionID() {
		t.Error("two ids collided")
	}
}

func unregister(provider string) { UnregisterForTest(provider) }

func panics(f func()) (did bool) {
	defer func() { did = recover() != nil }()
	f()
	return
}
