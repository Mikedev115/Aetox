package engine

// The UI guide's session (docs/architecture/ui-guide-2026-09-15.md §5.3): a
// conversation at the guide desk that this process holds but the window never
// looks at. It is the one live conversation that is not the chat on screen and
// not a turn the user walked away from — it is held on purpose, beside the
// chat, for as long as the guide figure is open.
//
// Why not NewSessionAt("guide"): that door SHOWS the conversation it opens, so
// the engine's cursor would leave the user's chat, the next message typed
// into the composer would land in the guide, and closing the guide would need
// a LoadSession to find the way back. The turn path never reads the cursor
// (conversation.go, §150), so a conversation can be filed without being shown
// and asked a question by id — which is all the guide needs.
//
// Held, not stored: the row the turn writes carries mode 'guide', and every
// list the window draws leaves that mode out (sessions.go); CloseGuideSession
// deletes it, openDatabase sweeps what a crash left. The guide keeps no memory
// of a person and this is where that is made true — not a rule in a prompt.

import (
	"errors"
	"strings"

	"github.com/Mikedev115/Aetox/internal/mode"
)

var errNoGuide = errors.New("ไม่มีไกด์เปิดอยู่ — เปิดไกด์ใหม่ก่อน")

// OpenGuideSession seats a conversation at the guide desk and files it, not
// shown, and answers with its id. The window keeps the id and asks through
// SendToGuide; the chat on screen is untouched.
//
// provider, model and think are the guide's OWN settings, empty for "whatever
// the chat is using" — which is the ordinary case and the one that keeps
// working when the user changes provider somewhere else. A guide pinned to a
// small local model while the chat talks to a paid one is the point of the
// override: this job is a closed vocabulary over a map, and it is the first
// thing in the app that a 8B model does well enough to ship.
func (a *Engine) OpenGuideSession(provider, model, think string) (string, error) {
	m, _, err := resolveStation(mode.Guide, "", "")
	if err != nil {
		return "", err
	}
	conv := newConversation()
	conv.id = newSessionID()
	conv.desk = m
	a.cur() // builds the manager on a zero Engine, as every door does
	cfg := a.cfg
	// Field by field, so an override of one dial does not silently reset the
	// others to a zero value the provider cannot answer to.
	if p := strings.TrimSpace(provider); p != "" {
		cfg.ModelProvider = p
		// A model belongs to its provider: carrying the chat's model name onto
		// a different provider asks for a model that does not exist there.
		cfg.ModelName = strings.TrimSpace(model)
	}
	if t := strings.TrimSpace(think); t != "" {
		cfg.ThinkLevel = t
	}
	a.applyConfig(conv, cfg)
	a.convs.hold(conv)
	return conv.id, nil
}

// SendToGuide runs one turn in the guide's conversation — the same turn the
// chat runs, on a conversation that is not the one on screen.
func (a *Engine) SendToGuide(id, text string) (TurnReply, error) {
	conv := a.convs.find(strings.TrimSpace(id))
	if conv == nil || conv.desk == nil || conv.desk.DeskName() != mode.Guide {
		return TurnReply{}, errNoGuide
	}
	return a.sendMessageIn(conv, text, "")
}

// CloseGuideSession lets go of the guide's conversation and deletes what its
// turns stored. A guide with a turn still running is refused the way any
// session is (DeleteSession); the window closes the figure and tries again
// when the answer lands.
func (a *Engine) CloseGuideSession(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if conv := a.convs.find(id); conv != nil {
		if a.turnRunningIn(id) {
			return errTurnBusy
		}
		a.letGoOf(conv)
	}
	return a.DeleteSession(id)
}
