package engine

import (
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/hook"
)

// HooksView is the hooks file as the settings page reads it: the rows, where
// they live, and — when the file is there but will not parse — why the app is
// running without them. Bootstrap logs that and carries on (§57: an optional
// convenience must never be the reason the app will not start); the page is
// where a person finds out.
type HooksView struct {
	Path  string      `json:"path"`
	Hooks []hook.Hook `json:"hooks"`
	Err   string      `json:"err,omitempty"`
}

// Hooks reads the hooks file fresh, for the settings page. Fresh rather than
// what bootstrap loaded, for the same reason ListExternalSkills scans: the
// page is the editor of this file, and a hand edit since launch must show.
func (a *Engine) Hooks() HooksView {
	path, err := config.HooksPath()
	if err != nil {
		return HooksView{Err: err.Error()}
	}
	cfg, err := hook.Load(path)
	view := HooksView{Path: path, Hooks: cfg.Hooks}
	if view.Hooks == nil {
		view.Hooks = []hook.Hook{}
	}
	if err != nil {
		view.Err = err.Error()
	}
	return view
}

// SaveHooks writes the hooks file whole and hands the new set to every live
// conversation's runner, so the next tool call in any chat is guarded by what
// was just saved — no relaunch, the way MCP and skills already behave. A set
// hook.Validate refuses is returned unwritten, with which row and why.
func (a *Engine) SaveHooks(hooks []hook.Hook) error {
	cfg := hook.Config{Hooks: hooks}
	if err := config.SaveHooks(cfg); err != nil {
		return err
	}
	// cur() first: it is what builds the manager on a zero Engine, and a
	// test that saves before any chat exists must not nil-deref for it.
	a.cur()
	for _, conv := range a.convs.all() {
		if conv.chat != nil {
			conv.chat.ReloadHooks(cfg)
		}
	}
	return nil
}
