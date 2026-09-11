package callfault

import (
	"errors"
	"fmt"
	"testing"
)

func TestTheMarkSurvivesWrapping(t *testing.T) {
	base := New("action is required")
	if !Is(base) {
		t.Error("New did not mark")
	}
	if !Is(fmt.Errorf("search: %w", base)) {
		t.Error("the mark was lost under one wrap")
	}
	if !Is(Mark(errors.New("about is required"))) {
		t.Error("Mark did not mark")
	}
	if !Is(Newf("unknown action %q", "grepp")) {
		t.Error("Newf did not mark")
	}
	if Mark(nil) != nil {
		t.Error("Mark(nil) must stay nil")
	}
	if Is(errors.New("the page did not answer with a picture")) {
		t.Error("an unmarked error must not read as a caller fault")
	}
}

func TestTheTextIsUntouched(t *testing.T) {
	err := New("text is required to remember something")
	if err.Error() != "text is required to remember something" {
		t.Errorf("the mark changed the sentence: %q", err.Error())
	}
}
