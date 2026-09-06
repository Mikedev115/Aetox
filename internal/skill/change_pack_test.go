package skill

import "testing"

// `write` takes the text as `content`, `append` takes it as `replace`: one tool
// asking for the same thing under two names. A model that has just written a
// file with `content` carries it on with `content`, and the round is spent on
// an error naming a field it never sent.
//
// Measured 2026-09-05 on a landing page: write, then seven appends, and the
// first of the seven was this. Every long file pays it once, and a file is long
// whenever the output ceiling made write cut it — which is what append is for.
// So the cost is not occasional, it is attached to the feature.
//
// Repaired here rather than in the edit tool because this is the layer that
// already knows `append` is an action word standing for a flag. Underneath,
// the field is `replace` and stays `replace`.
func TestAppendAcceptsTheFieldNameWriteUses(t *testing.T) {
	s := &changeSkill{}

	out := s.innerArgs("append", map[string]any{"path": "p.html", "content": "tail"})
	if out["replace"] != "tail" {
		t.Fatalf("append did not adopt content: %#v", out)
	}
	if _, still := out["content"]; still {
		t.Errorf("content should not travel on to the edit tool underneath: %#v", out)
	}
	if out["mode"] != "append" {
		t.Errorf("the mode flag this rewrite exists for went missing: %#v", out)
	}

	// A repair, not an override: a call that named both meant the one it named.
	out = s.innerArgs("append", map[string]any{"path": "p.html", "replace": "real", "content": "wrong"})
	if out["replace"] != "real" {
		t.Errorf("an explicit replace was overwritten by content: %#v", out)
	}

	// Every other action passes through untouched, which is what makes a pack a
	// name and not a translation layer.
	in := map[string]any{"path": "p.html", "content": "whole file"}
	if got := s.innerArgs("write", in); got["content"] != "whole file" || got["replace"] != nil {
		t.Errorf("write was rewritten: %#v", got)
	}
}
