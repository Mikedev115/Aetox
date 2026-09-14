package cognitive

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// A handoff is compaction the user can see (DECISIONS §282). Compaction folds
// the old half of a conversation into one paragraph the model reads and the
// user never does; a handoff folds the WHOLE conversation into points the user
// picks from before they are carried into a fresh chat. Same summarizer, same
// transcript renderer, a different prompt — because the two outputs are for
// different readers. The compaction summary is prose for a model that will
// keep going; a handoff is a list for a person deciding what still matters.
//
// A list, not JSON. The three providers this app answers with disagree about
// whether a "JSON only" instruction survives a Thai transcript, and a bullet
// list is the one shape every one of them produces on request. parseHandoffPoints
// accepts the markers they actually use.
const handoffPrompt = "The user is ending this conversation and starting a fresh chat that should " +
	"remember what matters. Write the points worth carrying over as a bullet list: one point " +
	"per line, each starting with \"- \", each self-contained (a reader who never saw this " +
	"conversation must understand it alone). Cover, in this order: what the user is trying " +
	"to do; decisions made and why; facts, constraints and preferences the user stated; files " +
	"or paths created or changed; what is unfinished and the agreed next step. Between 3 and " +
	handoffMaxPointsText + " points, most important first, no headings, no preamble, nothing " +
	"after the list. Write in the user's language."

const (
	// handoffMaxPoints caps the list at what fits on one card without
	// scrolling. The prompt asks for the same number; the parser enforces it,
	// because a model that writes twenty is a model that wrote a transcript.
	handoffMaxPoints     = 12
	handoffMaxPointsText = "12"
	// handoffMaxTokens is compactSummaryMaxTokens' sibling: twelve points at
	// a paragraph each is well inside it.
	handoffMaxTokens = 2048
)

// ErrNothingToHandOff is HandoffPoints' answer to a conversation with no turn
// in it. Named so the engine can put its own words on it.
var ErrNothingToHandOff = errors.New("nothing to hand off")

// HandoffPoints asks the model for the points of this conversation worth
// carrying into a new one, whole transcript in, list out.
//
// Whole transcript rather than compact's "everything but the recent six": the
// user is leaving, so there is no recent tail to keep verbatim. A conversation
// that has already been compacted carries its earlier summary as one of its
// messages and that goes in like any other — the summarizer summarizes the
// summary, which is the right depth for something that already happened
// twice ago.
func (a *Agent) HandoffPoints(ctx context.Context) ([]string, error) {
	if a == nil || a.context == nil || a.provider == nil {
		return nil, ErrNothingToHandOff
	}
	messages := a.context.Messages()
	// Index 0 is the system prompt: the assistant's own brief, not something
	// the user said or would want carried.
	if len(messages) < 2 {
		return nil, ErrNothingToHandOff
	}
	defer debuglog.Block(fmt.Sprintf("Agent.HandoffPoints (%d msgs)", len(messages)-1))()
	response, err := a.completeWithReconnect(ctx, model.Request{
		Model: a.model,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: handoffPrompt},
			{Role: model.RoleUser, Content: renderCompactionTranscript(messages[1:])},
		},
		MaxTokens:   handoffMaxTokens,
		Temperature: 0.2,
	}, turn.TurnOptions{})
	if err != nil {
		return nil, err
	}
	points := parseHandoffPoints(response.Text)
	if len(points) == 0 {
		return nil, fmt.Errorf("the model answered with no points")
	}
	return points, nil
}

// parseHandoffPoints reads the list back: one point per line, the marker
// stripped, blank lines and anything that is not a list item dropped. "1."
// and "•" are accepted beside "- " because that is what comes back when a
// model has its own idea of a bullet, and refusing the whole answer over the
// marker would fail the feature on the provider least likely to be retried.
//
// A line with no marker at all is kept only when NO line had one — a model
// that ignored the format entirely and wrote one point per line is still a
// usable answer; a stray sentence between bullets is not a point.
func parseHandoffPoints(text string) []string {
	var marked, bare []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if point, ok := stripListMarker(line); ok {
			marked = append(marked, point)
		} else {
			bare = append(bare, line)
		}
	}
	points := marked
	if len(points) == 0 {
		points = bare
	}
	out := points[:0:0]
	for _, p := range points {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
		if len(out) == handoffMaxPoints {
			break
		}
	}
	return out
}

// stripListMarker takes the bullet or number off a list line and says whether
// there was one.
func stripListMarker(line string) (string, bool) {
	for _, marker := range []string{"- ", "* ", "• ", "– ", "— "} {
		if strings.HasPrefix(line, marker) {
			return strings.TrimSpace(line[len(marker):]), true
		}
	}
	// "1. point" / "12) point"
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i > 0 && i < len(line) && (line[i] == '.' || line[i] == ')') {
		return strings.TrimSpace(line[i+1:]), true
	}
	return line, false
}
