package engine

import (
	"reflect"
	"testing"
)

// The factory a delegate uses to think at its own provider (`provider:` in
// AGENT.md, 62ed2bed) reaches the task tool through bootstrap.Options.
// The line that handed it over was lost in the merge that carried that work
// onto the carved engine (305333eb) and nothing said so: a nil factory is the
// CLI's ordinary state, so task.go logs a line and runs the delegate on the
// session's provider — which is the exact fallback the feature exists to
// replace, and it shipped that way in 1.6.0.
//
// Read through reflect because the task tool keeps its options to itself, and
// exporting them for one assertion would be a second door for a test to use.
func TestApplyConfigHandsTheDelegateItsOwnProviderFactory(t *testing.T) {
	a := newSwitchApp(t)

	tool, ok := a.cur().registry.Get("task")
	if !ok {
		t.Fatal("no task tool in a session with delegation on")
	}
	// The registered tool is the packed door (subagent.delegationTool); its
	// `start` half is the taskTool that holds the options.
	v := reflect.ValueOf(tool)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if start := v.FieldByName("start"); start.IsValid() && !start.IsNil() {
		v = start.Elem()
	}
	opts := v.FieldByName("opts")
	if !opts.IsValid() {
		t.Fatalf("the task tool is a %s and has no opts field — update this test with it", v.Type())
	}
	factory := opts.FieldByName("ProviderFor")
	if !factory.IsValid() {
		t.Fatal("TaskOptions has no ProviderFor — the field this test exists to check")
	}
	if factory.IsNil() {
		t.Fatal("applyConfig bootstrapped the session without ProviderFor: a delegate that names a provider will run on the session's instead, and only the log will say so")
	}
}
