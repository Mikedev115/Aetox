package engine

// WindowPacks builds the two tool packs that act on the window's machine —
// the browser and the computer — for one session. Exported so the screen can
// lend them (Screen.WindowTools) while their implementations still live in
// this package; the moment browser_*.go and computer_*.go move to desktop/
// (§248 B1), this function goes with them and nothing here remembers it.
//
// The browser pack takes the engine for the tab host it drives and the
// project it writes screenshots into; the computer pack for the machine lock
// and the events it raises. Both are the coupling the move undoes.

import "github.com/Mikedev115/Aetox/internal/skill"

func WindowPacks(e *Engine, s Session) []skill.Skill {
	return []skill.Skill{
		// One tool for the browser, nine actions inside it (browser_tool.go).
		// The old per-action names are still what `tools:` and `categories:`
		// speak — they moved from being tools to being the actions' keys.
		&browserSkill{app: e, session: s},
		// Driving programs on this machine (computer_tool.go). Offered always;
		// whether a session gets it is the engine's switch, not the window's.
		newComputerSkill(e, s),
	}
}
