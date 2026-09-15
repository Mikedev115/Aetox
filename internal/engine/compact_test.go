package engine

import (
	"strings"
	"testing"
)

func TestCompactSessionNeedsAModel(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	_, err := a.CompactSession(a.cur().id)
	if err == nil || !strings.Contains(err.Error(), "โมเดล") {
		t.Fatalf("CompactSession err = %v, want no-model refusal", err)
	}
}

func TestCompactSessionUnknownSession(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	_, err := a.CompactSession("non-existent-session-id")
	if err == nil {
		t.Fatal("CompactSession on non-existent session should fail")
	}
}
